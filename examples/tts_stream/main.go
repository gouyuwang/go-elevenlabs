package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/gouyuwang/go-elevenlabs/tts"
)

func main() {
	authKey := os.Getenv("ELEVENLABS_API_KEY")
	if authKey == "" {
		log.Fatal("missing ELEVENLABS_API_KEY")
	}
	if len(os.Args) < 3 {
		log.Fatal("usage: go run ./examples/tts_stream <voice-id> <output-file>")
	}

	client := tts.NewClient(authKey)
	resp, err := client.Stream(context.Background(), tts.SynthesisRequest{
		VoiceID: os.Args[1],
		Text:    "This audio is streamed from ElevenLabs.",
	})
	if err != nil {
		log.Fatalf("stream: %v", err)
	}
	defer resp.Audio.Close()

	file, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, resp.Audio); err != nil {
		log.Fatalf("write streamed audio: %v", err)
	}
}
