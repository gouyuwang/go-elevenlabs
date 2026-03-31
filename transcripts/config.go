package transcripts

import "net/http"

const (
	// BaseUrl is the base URL for the elevenlabs Realtime API.
	BaseUrl = "wss://api.elevenlabs.io/v1/speech-to-text/realtime"
	// HTTPBaseURL is the base URL for the elevenlabs speech-to-text HTTP API.
	HTTPBaseURL = "https://api.elevenlabs.io/v1/speech-to-text"
)

// ClientConfig is the configuration for the client.
type ClientConfig struct {
	authKey     string
	BaseURL     string       // Base URL for the realtime API.
	HTTPBaseURL string       // Base URL for the HTTP API.
	HTTPClient  *http.Client // HTTP client for non-streaming transcription.
}

func DefaultConfig(authKey string) ClientConfig {
	return ClientConfig{
		authKey:     authKey,
		BaseURL:     BaseUrl,
		HTTPBaseURL: HTTPBaseURL,
		HTTPClient:  http.DefaultClient,
	}
}
