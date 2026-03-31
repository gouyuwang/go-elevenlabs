package tts

import "net/http"

const (
	BaseURL = "https://api.elevenlabs.io"
)

type ClientConfig struct {
	authKey    string
	BaseURL    string
	HTTPClient *http.Client
}

func DefaultConfig(authKey string) ClientConfig {
	return ClientConfig{
		authKey:    authKey,
		BaseURL:    BaseURL,
		HTTPClient: http.DefaultClient,
	}
}
