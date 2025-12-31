package transcripts

import (
	"context"
	"net/http"
	"net/url"
)

const (
	// ModelScribeV2Realtime the only supported model at the moment
	ModelScribeV2Realtime string = "scribe_v2_realtime"
)

type Client struct {
	config ClientConfig
}

// NewClient creates new OpenAI Realtime API client for specified auth token.
func NewClient(authKey string) *Client {
	return &Client{
		config: DefaultConfig(authKey),
	}
}

// NewClientWithConfig creates new OpenAI Realtime API client for specified config.
func NewClientWithConfig(config ClientConfig) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) getURL(queries map[string]string) string {
	query := url.Values{}

	if nil != queries {
		for k, v := range queries {
			query.Set(k, v)
		}
	}

	return c.config.BaseURL + "?" + query.Encode()
}

func (c *Client) getHeaders() http.Header {
	headers := http.Header{}
	headers.Set("xi-api-key", c.config.authKey)
	return headers
}

type connectOption struct {
	dialer  WebSocketDialer
	logger  Logger
	queries map[string]string
}
type ConnectOption func(*connectOption)

// WithQuery sets the query parameters for the connection.
// Required Parameters
//
// model_id (string)
// The ID of the model to be used for transcription. This parameter is required.
// Notice: the only supported model is scribe_v2_realtime at the moment
//
// # Optional Parameters
//
// token (string)
// The authorization bearer token for authentication, typically an API key.
//
// include_timestamps (boolean)
// Default: false
// Whether to include word-level timestamps in the final transcription. If set to true, the transcription will include timestamps.
//
// include_language_detection (boolean)
// Default: false
// Whether to include the detected language code in the transcription. If set to true, the API will return the detected language code.
//
// audio_format (enum)
// Default: pcm_16000
// The audio encoding format used for speech-to-text. This parameter can support multiple values, but they are not listed here.
// Allowed values: pcm_8000  pcm_16000  pcm_22050 pcm_24000 pcm_44100 pcm_48000 ulaw_8000
//
// language_code (string)
// Default: None
// The language code of the audio content, in ISO 639-1 or ISO 639-3 format. For example, en for English, fr for French.
//
// commit_strategy (enum)
// Default: manual
// Defines the strategy for committing transcriptions.
// Allowed values: manual vad
//
// manual: Manual submission of the transcription.
//
// vad: Automatically commits transcription using Voice Activity Detection (VAD).
//
// vad_silence_threshold_secs (double)
// Default: 1.5
// Sets the silence threshold in seconds for Voice Activity Detection (VAD). Silence lasting longer than this duration will be considered as a pause.
//
// vad_threshold (double)
// Default: 0.4
// Sets the threshold for VAD to determine when speech activity starts or stops.
//
// min_speech_duration_ms (integer)
// Default: 250 milliseconds
// The minimum speech duration (in milliseconds) required to be considered valid. Speech shorter than this duration will be ignored.
//
// min_silence_duration_ms (integer)
// Default: 2500 milliseconds
// The minimum silence duration (in milliseconds) that VAD will recognize as a pause.
//
// enable_logging (boolean)
// Default: true
// Whether to enable logging. If set to false, the "zero retention mode" will be enabled, meaning the request history will not be stored. This mode is typically available only for enterprise customers.
func WithQuery(query map[string]string) ConnectOption {
	return func(opts *connectOption) {
		if nil == opts.queries {
			opts.queries = make(map[string]string)
		}
		for k, v := range query {
			opts.queries[k] = v
		}
	}
}

// WithDialer sets the dialer for the connection.
func WithDialer(dialer WebSocketDialer) ConnectOption {
	return func(opts *connectOption) {
		opts.dialer = dialer
	}
}

// WithLogger sets the logger for the connection.
func WithLogger(logger Logger) ConnectOption {
	return func(opts *connectOption) {
		opts.logger = logger
	}
}

// Connect connects to the Realtime API.
func (c *Client) Connect(ctx context.Context, opts ...ConnectOption) (*Conn, error) {
	connectOpts := connectOption{
		logger: NopLogger{},
		queries: map[string]string{
			"model_id":           ModelScribeV2Realtime,
			"audio_format":       string(AudioFormatPcm_16000),
			"include_timestamps": "true",
		},
	}
	for _, opt := range opts {
		opt(&connectOpts)
	}
	if connectOpts.dialer == nil {
		connectOpts.dialer = DefaultDialer()
	}

	// default headers
	headers := c.getHeaders()

	// get url by model
	uri := c.getURL(connectOpts.queries)

	// dial
	conn, err := connectOpts.dialer.Dial(ctx, uri, headers)
	if err != nil {
		return nil, err
	}

	return &Conn{conn: conn, logger: connectOpts.logger}, nil
}
