package tts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListModelsReturnsModels(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/v1/models"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("xi-api-key"), "test-key"; got != want {
			t.Fatalf("xi-api-key = %s, want %s", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{
				"model_id":"eleven_multilingual_v2",
				"name":"Eleven Multilingual v2",
				"can_do_text_to_speech":true,
				"languages":[{"language_id":"en","name":"English"}],
				"model_rates":{"character_cost_multiplier":1},
				"concurrency_group":"standard_eleven_multilingual_v2"
			},
			{
				"model_id":"scribe_v2",
				"name":"Scribe v2",
				"can_do_text_to_speech":false
			}
		]`)
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := len(models), 2; got != want {
		t.Fatalf("len(models) = %d, want %d", got, want)
	}
	if got, want := models[0].ModelID, ModelElevenMultilingualV2; got != want {
		t.Fatalf("models[0].ModelID = %s, want %s", got, want)
	}
	if got, want := models[0].Languages[0].LanguageID, "en"; got != want {
		t.Fatalf("models[0].Languages[0].LanguageID = %s, want %s", got, want)
	}
	if got, want := models[0].ModelRates.CharacterCostMultiplier, 1.0; got != want {
		t.Fatalf("models[0].ModelRates.CharacterCostMultiplier = %v, want %v", got, want)
	}
}

func TestClientListModelsReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("request-id", "req_models_bad")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":{"message":"bad key"}}`)
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	_, err := client.ListModels(context.Background())
	if err == nil {
		t.Fatal("ListModels() error = nil, want non-nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if got, want := apiErr.RequestID, "req_models_bad"; got != want {
		t.Fatalf("RequestID = %s, want %s", got, want)
	}
}

func TestTextToSpeechModelsFiltersOnlyTTSModels(t *testing.T) {
	t.Parallel()

	models := []Model{
		{ModelID: ModelElevenMultilingualV2, CanDoTextToSpeech: boolPtr(true)},
		{ModelID: "scribe_v2", CanDoTextToSpeech: boolPtr(false)},
		{ModelID: ModelElevenFlashV25, CanDoTextToSpeech: nil},
	}

	filtered := TextToSpeechModels(models)
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("len(filtered) = %d, want %d", got, want)
	}
	if got, want := filtered[0].ModelID, ModelElevenMultilingualV2; got != want {
		t.Fatalf("filtered[0].ModelID = %s, want %s", got, want)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
