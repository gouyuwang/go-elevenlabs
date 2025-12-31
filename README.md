# ElevenLabs Go SDK

A Go client library for ElevenLabs Realtime Speech-to-Text API.

## Overview

This SDK provides a Go interface to interact with ElevenLabs' real-time speech-to-text service. It allows you to stream audio data and receive real-time transcriptions with support for various audio formats and configuration options.

## Features

- Real-time speech-to-text conversion
- WebSocket-based streaming
- Support for PCM audio formats (8kHz to 48kHz)
- Configurable commit strategies (manual or VAD - Voice Activity Detection)
- Event-based handling for session start, partial transcripts, and final transcriptions
- Error handling for various scenarios (quota exceeded, rate limits, etc.)

## Installation

```bash
go get github.com/gouyuwang/go-elevenlabs
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/gouyuwang/go-elevenlabs/transcripts"
)

func main() {
    authToken := "YOUR_API_KEY"
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
    
    recognizer := transcripts.NewRecognizer(ctx, conn)
    
    // Add event handlers
    recognizer.Start() // Start the recognizer in a goroutine
    
    // Add event handlers after starting
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // Handle events in a separate goroutine if needed
            }
        }
    }()

    // Send audio data (PCM format, 16kHz sample rate)
    // Example: recognizer.Send(audioData)
    
    // Wait for completion or error
    if err := <-recognizer.Err(); err != nil {
        fmt.Printf("recognizer error: %v\n", err)
    }
}
```

### Streaming PCM Audio

```go
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
```

## Audio Requirements

- **Format**: PCM (uncompressed)
- **Sample Rates Supported**: 8000Hz, 16000Hz, 22050Hz, 24000Hz, 44100Hz, 48000Hz
- **Default**: 16000Hz (pcm_16000)
- **Channels**: Mono (1 channel)

## Configuration Options

You can configure the connection with various query parameters:

```go
client.Connect(ctx, transcripts.WithQuery(map[string]string{
    "language_code":           "eng",              // Language code (ISO 639-1/3)
    "commit_strategy":         "vad",              // Commit strategy: "manual" or "vad"
    "min_silence_duration_ms": "1000",             // Minimum silence duration in ms
    "audio_format":            "pcm_16000",        // Audio format
    "include_timestamps":      "true",             // Include word-level timestamps
}))
```

## Event Types

The SDK supports several event types that can be handled in your event handler:

- `SessionStartEventArgs`: Fired when the session starts
- `SpeechRecognizingEventArgs`: Fired for partial transcripts
- `SpeechRecognizedEventArgs`: Fired for committed transcripts
- `SpeechRecognizedWithTimestampEventArgs`: Fired for transcripts with word-level timestamps
- `SpeechRecognitionCanceledEventArgs`: Fired when recognition is canceled (with error details)

## API Methods

- `NewClient(authKey string) *Client`: Creates a new client with the provided auth key
- `Client.Connect()`: Establishes a connection to the API
- `NewRecognizer(ctx context.Context, conn *Conn, handlers ...ServerEventHandler) *Recognizer`: Creates a new recognizer
- `recognizer.Start()`: Starts the recognizer to listen for events
- `recognizer.Send(pcm []byte) error`: Sends PCM audio data to the API
- `recognizer.Commit() error`: Commits the current audio for processing
- `recognizer.Stop() error`: Stops the recognizer and closes the connection
- `recognizer.Err() <-chan error`: Returns a channel for receiving errors

## Error Handling

The SDK handles various error types:

- `QuotaExceededError`: When API quota is exceeded
- `RateLimitedError`: When rate limits are hit
- `AuthError`: When authentication fails
- `InputError`: When input format is invalid
- `TranscriberError`: When transcription fails

## Dependencies

- `github.com/coder/websocket`: WebSocket implementation
- `encoding/json`: JSON marshaling/unmarshaling
- `encoding/base64`: Audio data encoding

## License

This project is licensed under the MIT License - see the LICENSE file for details.