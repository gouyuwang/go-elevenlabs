package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSynthesizeReturnsAudioAndMetadata(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/v1/text-to-speech/voice_123"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("xi-api-key"), "test-key"; got != want {
			t.Fatalf("xi-api-key = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("Accept"), "audio/mpeg"; got != want {
			t.Fatalf("accept = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("output_format"), "mp3_44100_128"; got != want {
			t.Fatalf("output_format = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("enable_logging"), "false"; got != want {
			t.Fatalf("enable_logging = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("optimize_streaming_latency"), "3"; got != want {
			t.Fatalf("optimize_streaming_latency = %s, want %s", got, want)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var payload map[string]any
		if err = json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if got, want := payload["text"], "hello world"; got != want {
			t.Fatalf("text = %v, want %v", got, want)
		}
		if got, want := payload["model_id"], "eleven_turbo_v2_5"; got != want {
			t.Fatalf("model_id = %v, want %v", got, want)
		}
		if got, want := payload["language_code"], "en"; got != want {
			t.Fatalf("language_code = %v, want %v", got, want)
		}
		if got, want := payload["seed"], float64(7); got != want {
			t.Fatalf("seed = %v, want %v", got, want)
		}
		if got, want := payload["previous_text"], "previous"; got != want {
			t.Fatalf("previous_text = %v, want %v", got, want)
		}
		if got, want := payload["next_text"], "next"; got != want {
			t.Fatalf("next_text = %v, want %v", got, want)
		}
		voiceSettings, ok := payload["voice_settings"].(map[string]any)
		if !ok {
			t.Fatalf("voice_settings type = %T, want map[string]any", payload["voice_settings"])
		}
		if got, want := voiceSettings["stability"], 0.3; got != want {
			t.Fatalf("voice_settings.stability = %v, want %v", got, want)
		}

		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("request-id", "req_tts")
		w.Header().Set("x-character-count", "11")
		_, _ = w.Write([]byte("mp3-bytes"))
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)
	enableLogging := false
	optimizeStreamingLatency := 3
	stability := 0.3
	similarityBoost := 0.8
	style := 0.1
	useSpeakerBoost := true
	seed := 7

	resp, err := client.Synthesize(context.Background(), SynthesisRequest{
		VoiceID:                  "voice_123",
		Text:                     "hello world",
		ModelID:                  "eleven_turbo_v2_5",
		OutputFormat:             AudioFormatMP344100128,
		LanguageCode:             "en",
		EnableLogging:            &enableLogging,
		OptimizeStreamingLatency: &optimizeStreamingLatency,
		Seed:                     &seed,
		PreviousText:             "previous",
		NextText:                 "next",
		VoiceSettings: &VoiceSettings{
			Stability:       &stability,
			SimilarityBoost: &similarityBoost,
			Style:           &style,
			UseSpeakerBoost: &useSpeakerBoost,
		},
	})
	if err != nil {
		t.Fatalf("Synthesize() error = %v", err)
	}
	if got, want := string(resp.Audio), "mp3-bytes"; got != want {
		t.Fatalf("resp.Audio = %s, want %s", got, want)
	}
	if got, want := resp.ContentType, "audio/mpeg"; got != want {
		t.Fatalf("ContentType = %s, want %s", got, want)
	}
	if got, want := resp.RequestID, "req_tts"; got != want {
		t.Fatalf("RequestID = %s, want %s", got, want)
	}
	if got, want := resp.CharacterCount, "11"; got != want {
		t.Fatalf("CharacterCount = %s, want %s", got, want)
	}
}

func TestClientStreamAudioReturnsReadableAudio(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v1/text-to-speech/voice_123/stream"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("output_format"), "pcm_44100"; got != want {
			t.Fatalf("output_format = %s, want %s", got, want)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("request-id", "req_stream")
		_, _ = w.Write([]byte("stream-bytes"))
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	resp, err := client.StreamAudio(context.Background(), SynthesisRequest{
		VoiceID:      "voice_123",
		Text:         "hello stream",
		OutputFormat: AudioFormatPCM44100,
	})
	if err != nil {
		t.Fatalf("StreamAudio() error = %v", err)
	}
	defer resp.Audio.Close()

	body, err := io.ReadAll(resp.Audio)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if got, want := string(body), "stream-bytes"; got != want {
		t.Fatalf("stream body = %s, want %s", got, want)
	}
	if got, want := resp.RequestID, "req_stream"; got != want {
		t.Fatalf("RequestID = %s, want %s", got, want)
	}
}

func TestClientStreamAudioReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("request-id", "req_stream_bad")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.Copy(w, strings.NewReader(`{"detail":{"message":"invalid voice"}}`))
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	_, err := client.StreamAudio(context.Background(), SynthesisRequest{
		VoiceID: "missing",
		Text:    "hello",
	})
	if err == nil {
		t.Fatal("StreamAudio() error = nil, want non-nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if got, want := apiErr.RequestID, "req_stream_bad"; got != want {
		t.Fatalf("RequestID = %s, want %s", got, want)
	}
	if got, want := apiErr.Message, "invalid voice"; got != want {
		t.Fatalf("Message = %s, want %s", got, want)
	}
}

func TestClientSynthesizeReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("request-id", "req_bad")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.Copy(w, strings.NewReader(`{"detail":{"message":"voice not found"}}`))
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	_, err := client.Synthesize(context.Background(), SynthesisRequest{
		VoiceID: "missing",
		Text:    "hello",
	})
	if err == nil {
		t.Fatal("Synthesize() error = nil, want non-nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if got, want := apiErr.StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got, want := apiErr.RequestID, "req_bad"; got != want {
		t.Fatalf("RequestID = %s, want %s", got, want)
	}
	if got, want := apiErr.Message, "voice not found"; got != want {
		t.Fatalf("Message = %s, want %s", got, want)
	}
}
