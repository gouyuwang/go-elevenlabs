package transcripts

const (
	// BaseUrl is the base URL for the elevenlabs Realtime API.
	BaseUrl = "wss://api.elevenlabs.io/v1/speech-to-text/realtime"
)

// ClientConfig is the configuration for the client.
type ClientConfig struct {
	authKey string
	BaseURL string // Base URL for the API.
}

func DefaultConfig(authKey string) ClientConfig {
	return ClientConfig{
		authKey: authKey,
		BaseURL: BaseUrl,
	}
}
