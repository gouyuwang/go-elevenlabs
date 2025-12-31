package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/gouyuwang/go-elevenlabs/transcripts"
)

func init() {
	log.SetFlags(log.LstdFlags)
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := transcripts.StdLogger{}
	authKey := "Your key"
	client := transcripts.NewClient(authKey)
	conn, err := client.Connect(ctx, transcripts.WithQuery(map[string]string{
		"language_code": "eng", //iso 639-1 or iso 639-3
	}), transcripts.WithLogger(logger))
	if err != nil {
		logger.Errorf("connect error: %+v\n", err)
		return
	}
	logger.Debugf("connected.\n")

	recognizer := transcripts.NewRecognizer(ctx, conn,
		func(ctx context.Context, event transcripts.ServerEvent) {
			switch e := event.(type) {
			case transcripts.SessionStartEventArgs:
				logger.Debugf("session start: %+v\n", e)
			case transcripts.SpeechRecognizingEventArgs:
				logger.Debugf("speech recognizing: %+v\n", e)
			case transcripts.SpeechRecognizedEventArgs:
				logger.Debugf("speech recognized: %+v\n", e)
			case transcripts.SpeechRecognizedWithTimestampEventArgs:
				logger.Debugf("speech recognized with timestamp: %+v\n", e)
			case transcripts.SpeechRecognitionCanceledEventArgs:
				logger.Debugf("speech recognition canceled: %+v\n", e)
			}
		})

	recognizer.Start()
	defer func() {
		if err = recognizer.Stop(); err != nil {
			logger.Errorf("stop continuous recognition error: %+v\n", err)
			return
		}
		logger.Debugf("stop continuous recognition done.\n")
	}()

	logger.Debugf("Mock send pcm stream...\n")
	interval := 300 * time.Millisecond
	chunkSize := int(16000 * 2 * interval.Seconds())
	if err = StreamPCMWithChannel(ctx, recognizer, "./examples/simple/nicole.pcm", chunkSize, interval); err != nil {
		logger.Errorf("stream PCM error: %+v\n", err)
		return
	}
	logger.Debugf("send data done...\n")

	for {
		select {
		case <-ctx.Done():
			return
		case err = <-recognizer.Err():
			logger.Errorf("conn handler error: %+v\n", err)
			return
		case <-time.After(20 * time.Second):
			return
		}
	}
}

func StreamPCMWithChannel(ctx context.Context, recognizer *transcripts.Recognizer, pcmFile string, chunkSize int, interval time.Duration) error {
	file, err := os.Open(pcmFile)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Printf("close file error: %+v\n", err)
		}
	}()

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
				log.Printf("Sent %d bytes\n", n)
			}

			if err != nil {
				if err == io.EOF {
					if sendErr := recognizer.Commit(); sendErr != nil {
						return fmt.Errorf("failed to commit audio: %w", sendErr)
					}
					log.Println("Finished sending PCM data")
					return nil
				}
				return fmt.Errorf("error reading PCM file: %w", err)
			}
		}
	}
}
