package transcripts

const (
	// BaseUrl is the base URL for the elevenlabs Realtime API.
	BaseUrl = "wss://api.elevenlabs.io/v1/speech-to-text/realtime"
)

// ClientConfig is the configuration for the client.
type ClientConfig struct {
	authToken string
	BaseURL   string // Base URL for the API.
}

func DefaultConfig(authToken string) ClientConfig {
	return ClientConfig{
		authToken: authToken,
		BaseURL:   BaseUrl,
	}
}
