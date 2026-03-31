package transcripts

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientTranscribeSendsMultipartRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/v1/speech-to-text"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("enable_logging"), "false"; got != want {
			t.Fatalf("enable_logging = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("xi-api-key"), "test-key"; got != want {
			t.Fatalf("xi-api-key = %s, want %s", got, want)
		}

		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse media type: %v", err)
		}
		if got, want := mediaType, "multipart/form-data"; got != want {
			t.Fatalf("content type = %s, want %s", got, want)
		}
		if err = r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if params["boundary"] == "" {
			t.Fatal("missing multipart boundary")
		}
		if got, want := r.FormValue("model_id"), "scribe_v1"; got != want {
			t.Fatalf("model_id = %s, want %s", got, want)
		}
		if got, want := r.FormValue("language_code"), "en"; got != want {
			t.Fatalf("language_code = %s, want %s", got, want)
		}
		if got, want := r.FormValue("diarize"), "true"; got != want {
			t.Fatalf("diarize = %s, want %s", got, want)
		}
		if got, want := r.FormValue("diarization_threshold"), "0.22"; got != want {
			t.Fatalf("diarization_threshold = %s, want %s", got, want)
		}
		if got, want := r.FormValue("timestamps_granularity"), "character"; got != want {
			t.Fatalf("timestamps_granularity = %s, want %s", got, want)
		}
		if got, want := r.FormValue("file_format"), "pcm_s16le_16"; got != want {
			t.Fatalf("file_format = %s, want %s", got, want)
		}
		if got, want := r.FormValue("temperature"), "0.4"; got != want {
			t.Fatalf("temperature = %s, want %s", got, want)
		}
		if got, want := r.FormValue("seed"), "42"; got != want {
			t.Fatalf("seed = %s, want %s", got, want)
		}
		if got, want := r.FormValue("entity_detection"), `["pii","pci"]`; got != want {
			t.Fatalf("entity_detection = %s, want %s", got, want)
		}
		if got, want := r.FormValue("entity_redaction"), "pii"; got != want {
			t.Fatalf("entity_redaction = %s, want %s", got, want)
		}
		if got, want := r.FormValue("entity_redaction_mode"), "redacted"; got != want {
			t.Fatalf("entity_redaction_mode = %s, want %s", got, want)
		}
		if got, want := r.FormValue("keyterms"), `["ElevenLabs","Golang"]`; got != want {
			t.Fatalf("keyterms = %s, want %s", got, want)
		}
		if got, want := r.FormValue("additional_formats"), `[{"format":"srt"}]`; got != want {
			t.Fatalf("additional_formats = %s, want %s", got, want)
		}
		if got, want := r.FormValue("webhook_metadata"), `{"job_id":"123"}`; got != want {
			t.Fatalf("webhook_metadata = %s, want %s", got, want)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		if got, want := header.Filename, "sample.wav"; got != want {
			t.Fatalf("filename = %s, want %s", got, want)
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read file body: %v", err)
		}
		if got, want := string(body), "hello audio"; got != want {
			t.Fatalf("file body = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(TranscriptionResponse{
			Text:                "hello world",
			LanguageCode:        "en",
			LanguageProbability: 0.98,
			TranscriptionID:     "tr_123",
			AdditionalFormats: []TranscriptAdditionalFormat{
				{
					RequestedFormat: "srt",
					FileExtension:   "srt",
					ContentType:     "text/plain",
				},
			},
			Entities: []TranscriptEntity{
				{
					Text:       "hello",
					EntityType: "other",
					StartChar:  0,
					EndChar:    5,
				},
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL + "/v1/speech-to-text/realtime"
	client := NewClientWithConfig(cfg)
	diarize := true
	enableLogging := false
	diarizationThreshold := 0.22
	temperature := 0.4
	seed := 42

	resp, err := client.Transcribe(context.Background(), TranscriptionRequest{
		ModelID:               "scribe_v1",
		FileName:              "sample.wav",
		File:                  strings.NewReader("hello audio"),
		LanguageCode:          "en",
		Diarize:               &diarize,
		DiarizationThreshold:  &diarizationThreshold,
		TimestampsGranularity: "character",
		FileFormat:            "pcm_s16le_16",
		Temperature:           &temperature,
		Seed:                  &seed,
		EnableLogging:         &enableLogging,
		EntityDetection:       []string{"pii", "pci"},
		EntityRedaction:       "pii",
		EntityRedactionMode:   "redacted",
		Keyterms:              []string{"ElevenLabs", "Golang"},
		AdditionalFormats: []TranscriptOutputFormatRequest{
			{Format: "srt"},
		},
		WebhookMetadata: map[string]any{
			"job_id": "123",
		},
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if got, want := resp.Text, "hello world"; got != want {
		t.Fatalf("resp.Text = %s, want %s", got, want)
	}
	if got, want := resp.LanguageCode, "en"; got != want {
		t.Fatalf("resp.LanguageCode = %s, want %s", got, want)
	}
	if got, want := resp.TranscriptionID, "tr_123"; got != want {
		t.Fatalf("resp.TranscriptionID = %s, want %s", got, want)
	}
	if got, want := len(resp.AdditionalFormats), 1; got != want {
		t.Fatalf("len(resp.AdditionalFormats) = %d, want %d", got, want)
	}
	if got, want := len(resp.Entities), 1; got != want {
		t.Fatalf("len(resp.Entities) = %d, want %d", got, want)
	}
}

func TestClientTranscribeReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("request-id", "req_123")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":{"status":"invalid_api_key","message":"bad key"}}`))
	}))
	defer server.Close()

	cfg := DefaultConfig("bad-key")
	cfg.BaseURL = server.URL + "/v1/speech-to-text/realtime"
	client := NewClientWithConfig(cfg)

	_, err := client.Transcribe(context.Background(), TranscriptionRequest{
		ModelID:  "scribe_v1",
		FileName: "sample.wav",
		File:     strings.NewReader("hello audio"),
	})
	if err == nil {
		t.Fatal("Transcribe() error = nil, want non-nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if got, want := apiErr.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
	if got, want := apiErr.RequestID, "req_123"; got != want {
		t.Fatalf("RequestID = %s, want %s", got, want)
	}
	if got, want := apiErr.Message, "bad key"; got != want {
		t.Fatalf("Message = %s, want %s", got, want)
	}
}

func TestClientTranscribeAcceptsSourceURLWithoutFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if got, want := r.FormValue("source_url"), "https://example.com/audio.mp3"; got != want {
			t.Fatalf("source_url = %s, want %s", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello from source"}`))
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL + "/v1/speech-to-text/realtime"
	client := NewClientWithConfig(cfg)

	resp, err := client.Transcribe(context.Background(), TranscriptionRequest{
		ModelID:   "scribe_v2",
		SourceURL: "https://example.com/audio.mp3",
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if got, want := resp.Text, "hello from source"; got != want {
		t.Fatalf("resp.Text = %s, want %s", got, want)
	}
}
