package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gouyuwang/go-elevenlabs/transcripts"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./examples/transcribe_file <audio-file>")
	}

	authKey := os.Getenv("ELEVENLABS_API_KEY")
	if authKey == "" {
		log.Fatal("missing ELEVENLABS_API_KEY")
	}

	audioPath := os.Args[1]
	file, err := os.Open(audioPath)
	if err != nil {
		log.Fatalf("open audio file: %v", err)
	}
	defer file.Close()

	client := transcripts.NewClient(authKey)
	resp, err := client.Transcribe(context.Background(), transcripts.TranscriptionRequest{
		ModelID:      "scribe_v1",
		FileName:     filepath.Base(audioPath),
		File:         file,
		LanguageCode: "en",
	})
	if err != nil {
		log.Fatalf("transcribe: %v", err)
	}

	fmt.Println(resp.Text)
}
