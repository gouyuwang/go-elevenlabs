# ElevenLabs Go SDK

A Go client library for ElevenLabs speech APIs with:

- ASR realtime streaming over WebSocket
- ASR file transcription over HTTP
- TTS synchronous synthesis over HTTP
- TTS HTTP audio streaming

## Installation

```bash
go get github.com/gouyuwang/go-elevenlabs
```

## Packages

- `github.com/gouyuwang/go-elevenlabs/transcripts`
  - realtime ASR with `Client.Connect(...)` and `Recognizer`
  - file or source URL transcription with `Client.Transcribe(...)`
- `github.com/gouyuwang/go-elevenlabs/tts`
  - full audio synthesis with `Client.Synthesize(...)`
  - HTTP audio response streaming with `Client.StreamAudio(...)`
  - websocket realtime TTS with `Client.ConnectRealtime(...)` + `NewRealtimeSynthesizer(...)`
  - model discovery with `Client.ListModels(...)`

## Authentication

Set your API key with the `ELEVENLABS_API_KEY` environment variable:

```bash
export ELEVENLABS_API_KEY=your_api_key
```

On PowerShell:

```powershell
$env:ELEVENLABS_API_KEY="your_api_key"
```

## ASR Realtime Streaming

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/gouyuwang/go-elevenlabs/transcripts"
)

func main() {
	authKey := "YOUR_API_KEY"
	client := transcripts.NewClient(authKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Connect(ctx, transcripts.WithQuery(map[string]string{
		"language_code": "eng",
		"audio_format":  string(transcripts.AudioFormatPcm_16000),
	}))
	if err != nil {
		log.Fatal(err)
	}

	recognizer := transcripts.NewRecognizer(ctx, conn, func(ctx context.Context, event transcripts.ServerEvent) {
		switch e := event.(type) {
		case transcripts.SpeechRecognizingEventArgs:
			log.Printf("partial: %s", e.Text)
		case transcripts.SpeechRecognizedEventArgs:
			log.Printf("final: %s", e.Text)
		}
	})

	recognizer.Start()
	defer recognizer.Stop()

	_ = recognizer.Send([]byte("pcm-bytes"))
	_ = recognizer.Commit()

	select {
	case err = <-recognizer.Err():
		if err != nil {
			log.Fatal(err)
		}
	case <-time.After(3 * time.Second):
	}
}
```

See `examples/main.go`.

## ASR File Transcription

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gouyuwang/go-elevenlabs/transcripts"
)

func main() {
	file, err := os.Open("sample.wav")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	client := transcripts.NewClient(os.Getenv("ELEVENLABS_API_KEY"))
	resp, err := client.Transcribe(context.Background(), transcripts.TranscriptionRequest{
		ModelID:      "scribe_v1",
		FileName:     filepath.Base("sample.wav"),
		File:         file,
		LanguageCode: "en",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Text)
}
```

See `examples/transcribe_file/main.go`.

`TranscriptionRequest` supports common official fields such as `SourceURL`, `Diarize`, `DiarizationThreshold`, `TimestampsGranularity`, `EntityDetection`, `Keyterms`, `AdditionalFormats`, and `WebhookMetadata`.

## TTS Synchronous Synthesis

```go
package main

import (
	"context"
	"os"

	"github.com/gouyuwang/go-elevenlabs/tts"
)

func main() {
	client := tts.NewClient(os.Getenv("ELEVENLABS_API_KEY"))
	resp, err := client.Synthesize(context.Background(), tts.SynthesisRequest{
		VoiceID:      "voice_id",
		Text:         "Hello from ElevenLabs.",
		ModelID:      "eleven_turbo_v2_5",
		OutputFormat: tts.AudioFormatMP344100128,
	})
	if err != nil {
		panic(err)
	}

	_ = os.WriteFile("speech.mp3", resp.Audio, 0o644)
}
```

See `examples/tts_basic/main.go`.

`SynthesisRequest` supports common official fields such as `LanguageCode`, `VoiceSettings`, `GenerationConfig`, `PronunciationDictionaryLocators`, `Seed`, `PreviousText`, `NextText`, `PreviousRequestIDs`, `NextRequestIDs`, `EnableLogging`, and `OptimizeStreamingLatency`.

Common TTS model constants are available in the `tts` package:

- `tts.ModelElevenV3`
- `tts.ModelElevenMultilingualV2`
- `tts.ModelElevenFlashV25`
- `tts.ModelElevenTurboV25`

## TTS Models

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/gouyuwang/go-elevenlabs/tts"
)

func main() {
	client := tts.NewClient(os.Getenv("ELEVENLABS_API_KEY"))
	models, err := client.ListModels(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, model := range tts.TextToSpeechModels(models) {
		log.Printf("%s %s", model.ModelID, model.Name)
	}
}
```

## TTS WebSocket Realtime Streaming

This mode is closer to the Azure push-style synthesizer: connect once, send incremental text chunks, and handle audio chunks as realtime events.

`StreamInputRequest` supports common `stream-input` query parameters such as `OutputFormat`, `EnableSSMLParsing`, `InactivityTimeout`, `SyncAlignment`, `AutoMode`, `ApplyTextNormalization`, `Seed`, `EnableLogging`, and `OptimizeStreamingLatency`.

The realtime websocket initialization message body also supports fields such as `VoiceSettings`, `GenerationConfig`, and `PronunciationDictionaryLocators`. Per-message bodies support text chunks plus optional `GenerationConfig`, `Flush`, and `TryTriggerGeneration`.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/gouyuwang/go-elevenlabs/tts"
)

func main() {
	client := tts.NewClient(os.Getenv("ELEVENLABS_API_KEY"))
	conn, err := client.ConnectRealtime(context.Background(), tts.StreamInputRequest{
		VoiceID:      "voice_id",
		ModelID:      tts.ModelElevenTurboV25,
		OutputFormat: tts.AudioFormatMP344100128,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	streamer := tts.NewRealtimeSynthesizer(context.Background(), conn, func(_ context.Context, event tts.StreamEvent) {
		switch e := event.(type) {
		case tts.AudioEvent:
			log.Printf("audio chunk: %d bytes", len(e.Audio))
		case tts.DoneEvent:
			log.Printf("done: %v", e.IsFinal)
		case tts.ErrorEvent:
			log.Printf("server error: %s", e.Message)
		}
	})
	streamer.Start()

	_ = streamer.SendText("Hello from websocket streaming.")
}
```

See `examples/tts_ws_stream/main.go`.

For backward compatibility, `Client.ConnectStreamInput(...)` and `NewStreamer(...)` still exist as aliases, but new code should prefer `ConnectRealtime(...)` and `NewRealtimeSynthesizer(...)`.

## TTS HTTP Audio Streaming

This mode sends the full text once and reads the audio response as a chunked HTTP stream. It is HTTP audio streaming, not websocket realtime text-input streaming.

```go
package main

import (
	"context"
	"io"
	"os"

	"github.com/gouyuwang/go-elevenlabs/tts"
)

func main() {
	client := tts.NewClient(os.Getenv("ELEVENLABS_API_KEY"))
	resp, err := client.StreamAudio(context.Background(), tts.SynthesisRequest{
		VoiceID: "voice_id",
		Text:    "This response is streamed as audio.",
	})
	if err != nil {
		panic(err)
	}
	defer resp.Audio.Close()

	file, err := os.Create("speech.mp3")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, _ = io.Copy(file, resp.Audio)
}
```

See `examples/tts_stream/main.go`.

## Audio Formats

### Realtime ASR input

- `pcm_8000`
- `pcm_16000`
- `pcm_22050`
- `pcm_24000`
- `pcm_44100`
- `pcm_48000`
- `ulaw_8000`

### TTS output

- `mp3_22050_32`
- `mp3_44100_32`
- `mp3_44100_64`
- `mp3_44100_96`
- `mp3_44100_128`
- `mp3_44100_192`
- `pcm_8000`
- `pcm_16000`
- `pcm_22050`
- `pcm_24000`
- `pcm_44100`
- `ulaw_8000`
- `alaw_8000`
- `opus_48000_32`
- `opus_48000_64`
- `opus_48000_96`
- `opus_48000_128`
- `opus_48000_192`

## Error Handling

- Realtime ASR continues to use the existing WebSocket error flow
- HTTP ASR and TTS return typed API errors with HTTP status, request ID, message, and raw body when available

## Examples

- `examples/main.go`
- `examples/transcribe_file/main.go`
- `examples/tts_basic/main.go`
- `examples/tts_stream/main.go`
- `examples/tts_ws_stream/main.go`
