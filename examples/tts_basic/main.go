package main

import (
	"context"
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
		log.Fatal("usage: go run ./examples/tts_basic <voice-id> <output-file>")
	}

	client := tts.NewClient(authKey)
	resp, err := client.Synthesize(context.Background(), tts.SynthesisRequest{
		VoiceID:      os.Args[1],
		Text:         "Hello from the ElevenLabs Go SDK.",
		ModelID:      "eleven_turbo_v2_5",
		OutputFormat: tts.AudioFormatMP344100128,
	})
	if err != nil {
		log.Fatalf("synthesize: %v", err)
	}

	if err = os.WriteFile(os.Args[2], resp.Audio, 0o644); err != nil {
		log.Fatalf("write output: %v", err)
	}
}
