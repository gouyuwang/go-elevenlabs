package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gouyuwang/go-elevenlabs/tts"
)

func main() {
	authKey := os.Getenv("ELEVENLABS_API_KEY")
	if authKey == "" {
		log.Fatal("missing ELEVENLABS_API_KEY")
	}
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./examples/tts_ws_stream <voice-id>")
	}

	client := tts.NewClient(authKey)
	conn, err := client.ConnectRealtime(context.Background(), tts.StreamInputRequest{
		VoiceID:      os.Args[1],
		ModelID:      tts.ModelElevenTurboV25,
		OutputFormat: tts.AudioFormatMP344100128,
	})
	if err != nil {
		log.Fatalf("connect websocket stream: %v", err)
	}
	defer conn.Close()

	streamer := tts.NewRealtimeSynthesizer(context.Background(), conn, func(_ context.Context, event tts.StreamEvent) {
		switch e := event.(type) {
		case tts.AudioEvent:
			log.Printf("audio chunk: %d bytes, final=%v", len(e.Audio), e.IsFinal)
		case tts.DoneEvent:
			log.Printf("stream done: final=%v", e.IsFinal)
		case tts.ErrorEvent:
			log.Printf("stream error event: %s", e.Message)
		}
	})
	streamer.Start()

	if err = streamer.SendText("你好，这是第一段。"); err != nil {
		log.Fatalf("send first chunk: %v", err)
	}
	if err = streamer.SendText("This is the second sentence."); err != nil {
		log.Fatalf("send second chunk: %v", err)
	}
	if err = streamer.Flush(); err != nil {
		log.Fatalf("flush stream: %v", err)
	}

	select {
	case err = <-streamer.Err():
		if err != nil {
			log.Fatalf("streamer error: %v", err)
		}
	case <-time.After(10 * time.Second):
		log.Println("stream timeout")
	}
}
