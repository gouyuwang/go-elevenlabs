package transcripts

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
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

func (c *Client) getURL(query url.Values) string {
	if encoded := query.Encode(); encoded != "" {
		return c.config.BaseURL + "?" + encoded
	}
	return c.config.BaseURL
}

func (c *Client) getHeaders() http.Header {
	headers := http.Header{}
	if c.config.authKey != "" {
		headers.Set("xi-api-key", c.config.authKey)
	}
	return headers
}

type connectOption struct {
	dialer     WebSocketDialer
	logger     Logger
	queries    map[string]string
	keyterms   []string
	sampleRate int64
}
type ConnectOption func(*connectOption)

type RealtimeConfig struct {
	Token                    string
	IncludeTimestamps        *bool
	IncludeLanguageDetection *bool
	AudioFormat              AudioFormat
	LanguageCode             string
	CommitStrategy           CommitStrategy
	Keyterms                 []string
	NoVerbatim               *bool
	VadSilenceThresholdSecs  *float64
	VadThreshold             *float64
	MinSpeechDurationMs      *int
	MinSilenceDurationMs     *int
	EnableLogging            *bool
}

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
			switch k {
			case "audio_format":
				if sampleRate, ok := sampleRateForAudioFormat(AudioFormat(v)); ok {
					opts.sampleRate = sampleRate
				}
			case "keyterms":
				opts.keyterms = append(opts.keyterms, v)
			}
		}
	}
}

// WithRealtimeConfig sets the documented realtime websocket query parameters with typed fields.
func WithRealtimeConfig(cfg RealtimeConfig) ConnectOption {
	return func(opts *connectOption) {
		if opts.queries == nil {
			opts.queries = make(map[string]string)
		}
		if cfg.Token != "" {
			opts.queries["token"] = cfg.Token
		}
		if cfg.IncludeTimestamps != nil {
			opts.queries["include_timestamps"] = strconv.FormatBool(*cfg.IncludeTimestamps)
		}
		if cfg.IncludeLanguageDetection != nil {
			opts.queries["include_language_detection"] = strconv.FormatBool(*cfg.IncludeLanguageDetection)
		}
		if cfg.AudioFormat != "" {
			opts.queries["audio_format"] = string(cfg.AudioFormat)
			if sampleRate, ok := sampleRateForAudioFormat(cfg.AudioFormat); ok {
				opts.sampleRate = sampleRate
			}
		}
		if cfg.LanguageCode != "" {
			opts.queries["language_code"] = cfg.LanguageCode
		}
		if cfg.CommitStrategy != "" {
			opts.queries["commit_strategy"] = string(cfg.CommitStrategy)
		}
		if cfg.NoVerbatim != nil {
			opts.queries["no_verbatim"] = strconv.FormatBool(*cfg.NoVerbatim)
		}
		if cfg.VadSilenceThresholdSecs != nil {
			opts.queries["vad_silence_threshold_secs"] = strconv.FormatFloat(*cfg.VadSilenceThresholdSecs, 'f', -1, 64)
		}
		if cfg.VadThreshold != nil {
			opts.queries["vad_threshold"] = strconv.FormatFloat(*cfg.VadThreshold, 'f', -1, 64)
		}
		if cfg.MinSpeechDurationMs != nil {
			opts.queries["min_speech_duration_ms"] = strconv.Itoa(*cfg.MinSpeechDurationMs)
		}
		if cfg.MinSilenceDurationMs != nil {
			opts.queries["min_silence_duration_ms"] = strconv.Itoa(*cfg.MinSilenceDurationMs)
		}
		if cfg.EnableLogging != nil {
			opts.queries["enable_logging"] = strconv.FormatBool(*cfg.EnableLogging)
		}
		if len(cfg.Keyterms) > 0 {
			opts.keyterms = append([]string(nil), cfg.Keyterms...)
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
		sampleRate: 16000,
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
	query := url.Values{}
	for k, v := range connectOpts.queries {
		query.Set(k, v)
	}
	for _, keyterm := range connectOpts.keyterms {
		query.Add("keyterms", keyterm)
	}
	uri := c.getURL(query)

	// dial
	conn, err := connectOpts.dialer.Dial(ctx, uri, headers)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn:       conn,
		logger:     connectOpts.logger,
		sampleRate: connectOpts.sampleRate,
	}, nil
}

func sampleRateForAudioFormat(format AudioFormat) (int64, bool) {
	switch format {
	case AudioFormatPcm_8000, AudioFormatUlaw_8000:
		return 8000, true
	case AudioFormatPcm_16000:
		return 16000, true
	case AudioFormatPcm_22050:
		return 22050, true
	case AudioFormatPcm_24000:
		return 24000, true
	case AudioFormatPcm_44100:
		return 44100, true
	case AudioFormatPcm_48000:
		return 48000, true
	default:
		return 0, false
	}
}
