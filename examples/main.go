package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gouyuwang/go-elevenlabs/transcripts"
)

func main() {
	authToken := "Your key"
	client := transcripts.NewClient(authToken)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := client.Connect(ctx, transcripts.WithQuery(map[string]string{
		"language_code": "eng",
	}))
	if err != nil {
		fmt.Println("connect error:", err)
		return
	}

	fmt.Printf("connecting: %+v\n", conn)
	recognizer := transcripts.NewRecognizer(ctx, conn,
		func(ctx context.Context, event transcripts.ServerEvent) {
			switch e := event.(type) {
			case transcripts.SessionStartEventArgs:
				fmt.Printf("session start: %+v\n", e)
			case transcripts.SpeechRecognizingEventArgs:
				fmt.Printf("speech recognizing: %+v\n", e)
			case transcripts.SpeechRecognizedEventArgs:
				fmt.Printf("speech recognized: %+v\n", e)
			case transcripts.SpeechRecognizedWithTimestampEventArgs:
				fmt.Printf("speech recognized with timestamp: %+v\n", e)
			case transcripts.SpeechRecognitionCanceledEventArgs:
				fmt.Printf("speech recognition canceled: %+v\n", e)
			}
		})

	fmt.Printf("start continuous recognition...\n")
	if outcome := <-recognizer.StartContinuousRecognitionAsync(); outcome != nil {
		fmt.Printf("connect error: %+v\n", outcome)
		return
	}
	defer func() {
		if outcome := <-recognizer.StopContinuousRecognitionAsync(); outcome != nil {
			fmt.Printf("stop continuous recognition error: %+v\n", outcome)
		} else {
			fmt.Println("stop continuous recognition done.")
		}
	}()

	fmt.Printf("Mock send pcm stream...\n")
	interval := 300 * time.Millisecond
	chunkSize := int(16000 * 2 * interval.Seconds())
	if err = StreamPCMWithChannel(ctx, recognizer, "./examples/simple/nicole.pcm", chunkSize, interval); err != nil {
		fmt.Printf("stream PCM error: %+v\n", err)
	} else {
		fmt.Printf("send data done...\n")
	}
}

func StreamPCMWithChannel(ctx context.Context, recognizer *transcripts.Recognizer, pcmFile string, chunkSize int, interval time.Duration) error {
	file, err := os.Open(pcmFile)
	if err != nil {
		return fmt.Errorf("failed to open PCM file: %w", err)
	}
	defer func(file *os.File) {
		if err = file.Close(); err != nil {
			fmt.Printf("close file error: %+v\n", err)
		}
	}(file)

	buffer := make([]byte, chunkSize)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var n int
			if n, err = file.Read(buffer); n > 0 {
				if sendErr := recognizer.Send(buffer[:n]); sendErr != nil {
					return fmt.Errorf("failed to send audio chunk: %w", sendErr)
				}
				fmt.Printf("Sent %d bytes\n", n)
			}

			if err == io.EOF {
				if sendErr := recognizer.Commit(); sendErr != nil {
					return fmt.Errorf("failed to commit audio: %w", sendErr)
				}
				fmt.Println("Finished sending PCM data")
				return nil
			}

			if err != nil && err != io.EOF {
				return fmt.Errorf("error reading PCM file: %w", err)
			}
		}
	}
}
