package tts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/coder/websocket"
	"github.com/gouyuwang/go-elevenlabs/transcripts"
)

type StreamInputRequest struct {
	VoiceID                         string
	ModelID                         string
	OutputFormat                    AudioFormat
	LanguageCode                    string
	VoiceSettings                   *VoiceSettings
	EnableLogging                   *bool
	OptimizeStreamingLatency        *int
	EnableSSMLParsing               *bool
	InactivityTimeout               *int
	SyncAlignment                   *bool
	AutoMode                        *bool
	ApplyTextNormalization          TextNormalizationMode
	Seed                            *int
	GenerationConfig                *GenerationConfig
	PronunciationDictionaryLocators []PronunciationDictionaryLocator
}

type StreamTextMessage struct {
	Text                            string                           `json:"text"`
	TryTriggerGeneration            *bool                            `json:"try_trigger_generation,omitempty"`
	Flush                           *bool                            `json:"flush,omitempty"`
	GenerationConfig                *GenerationConfig                `json:"generation_config,omitempty"`
	PronunciationDictionaryLocators []PronunciationDictionaryLocator `json:"pronunciation_dictionary_locators,omitempty"`
}

type StreamEvent interface{}

type AudioEvent struct {
	Audio   []byte
	IsFinal bool
}

type DoneEvent struct {
	IsFinal bool
}

type ErrorEvent struct {
	Message string
}

type StreamEventHandler func(ctx context.Context, event StreamEvent)

type Conn struct {
	logger transcripts.Logger
	conn   transcripts.WebSocketConn
}

type connectOption struct {
	dialer transcripts.WebSocketDialer
	logger transcripts.Logger
}

type ConnectOption func(*connectOption)

func WithDialer(dialer transcripts.WebSocketDialer) ConnectOption {
	return func(opts *connectOption) {
		opts.dialer = dialer
	}
}

func WithLogger(logger transcripts.Logger) ConnectOption {
	return func(opts *connectOption) {
		opts.logger = logger
	}
}

// ConnectRealtime opens an interactive websocket TTS session.
// Unlike StreamAudio, this supports incremental text input and event-based audio output.
func (c *Client) ConnectRealtime(ctx context.Context, req StreamInputRequest, opts ...ConnectOption) (*Conn, error) {
	connectOpts := connectOption{
		dialer: transcripts.DefaultDialer(),
		logger: transcripts.NopLogger{},
	}
	for _, opt := range opts {
		opt(&connectOpts)
	}

	uri, err := c.streamInputURL(req)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set("xi-api-key", c.config.authKey)
	wsConn, err := connectOpts.dialer.Dial(ctx, uri, headers)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		logger: connectOpts.logger,
		conn:   wsConn,
	}

	initMessage := struct {
		Text                            string                           `json:"text"`
		XIAPIKey                        string                           `json:"xi_api_key"`
		ModelID                         string                           `json:"model_id,omitempty"`
		LanguageCode                    string                           `json:"language_code,omitempty"`
		VoiceSettings                   *VoiceSettings                   `json:"voice_settings,omitempty"`
		GenerationConfig                *GenerationConfig                `json:"generation_config,omitempty"`
		PronunciationDictionaryLocators []PronunciationDictionaryLocator `json:"pronunciation_dictionary_locators,omitempty"`
	}{
		Text:                            " ",
		XIAPIKey:                        c.config.authKey,
		ModelID:                         req.ModelID,
		LanguageCode:                    req.LanguageCode,
		VoiceSettings:                   req.VoiceSettings,
		GenerationConfig:                req.GenerationConfig,
		PronunciationDictionaryLocators: req.PronunciationDictionaryLocators,
	}
	if err = conn.Send(ctx, initMessage); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

// ConnectStreamInput is kept for backward compatibility.
// Deprecated: use ConnectRealtime for interactive websocket TTS streaming.
func (c *Client) ConnectStreamInput(ctx context.Context, req StreamInputRequest, opts ...ConnectOption) (*Conn, error) {
	return c.ConnectRealtime(ctx, req, opts...)
}

func (c *Client) streamInputURL(req StreamInputRequest) (string, error) {
	base := strings.TrimRight(c.config.BaseURL, "/")
	if strings.HasPrefix(base, "https://") {
		base = "wss://" + strings.TrimPrefix(base, "https://")
	} else if strings.HasPrefix(base, "http://") {
		base = "ws://" + strings.TrimPrefix(base, "http://")
	} else if !strings.HasPrefix(base, "wss://") && !strings.HasPrefix(base, "ws://") {
		base = "wss://" + strings.TrimPrefix(base, "/")
	}

	u, err := url.Parse(base + "/v1/text-to-speech/" + req.VoiceID + "/stream-input")
	if err != nil {
		return "", err
	}
	query := u.Query()
	if req.OutputFormat != "" {
		query.Set("output_format", string(req.OutputFormat))
	}
	if req.EnableLogging != nil {
		query.Set("enable_logging", strconvFormatBool(*req.EnableLogging))
	}
	if req.OptimizeStreamingLatency != nil {
		query.Set("optimize_streaming_latency", strconvFormatInt(*req.OptimizeStreamingLatency))
	}
	if req.EnableSSMLParsing != nil {
		query.Set("enable_ssml_parsing", strconvFormatBool(*req.EnableSSMLParsing))
	}
	if req.InactivityTimeout != nil {
		query.Set("inactivity_timeout", strconvFormatInt(*req.InactivityTimeout))
	}
	if req.SyncAlignment != nil {
		query.Set("sync_alignment", strconvFormatBool(*req.SyncAlignment))
	}
	if req.AutoMode != nil {
		query.Set("auto_mode", strconvFormatBool(*req.AutoMode))
	}
	if req.ApplyTextNormalization != "" {
		query.Set("apply_text_normalization", string(req.ApplyTextNormalization))
	}
	if req.Seed != nil {
		query.Set("seed", strconvFormatInt(*req.Seed))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) Send(ctx context.Context, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(ctx, transcripts.MessageText, data)
}

func (c *Conn) ReadEvent(ctx context.Context) (StreamEvent, error) {
	messageType, data, err := c.conn.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	if messageType != transcripts.MessageText {
		return nil, transcripts.ErrUnsupportedMessageType
	}
	return unmarshalStreamEvent(data)
}

type Streamer struct {
	ctx      context.Context
	conn     *Conn
	handlers []StreamEventHandler
	errCh    chan error
}

type RealtimeSynthesizer = Streamer

// NewRealtimeSynthesizer creates an event-driven websocket TTS runner.
func NewRealtimeSynthesizer(ctx context.Context, conn *Conn, handlers ...StreamEventHandler) *RealtimeSynthesizer {
	return &Streamer{
		ctx:      ctx,
		conn:     conn,
		handlers: handlers,
		errCh:    make(chan error, 1),
	}
}

// NewStreamer is kept for backward compatibility.
// Deprecated: use NewRealtimeSynthesizer for interactive websocket TTS streaming.
func NewStreamer(ctx context.Context, conn *Conn, handlers ...StreamEventHandler) *Streamer {
	return NewRealtimeSynthesizer(ctx, conn, handlers...)
}

func (s *Streamer) Start() {
	go func() {
		err := s.run()
		if err != nil {
			s.errCh <- err
		}
		close(s.errCh)
	}()
}

func (s *Streamer) Err() <-chan error {
	return s.errCh
}

func (s *Streamer) Send(msg StreamTextMessage) error {
	return s.conn.Send(s.ctx, msg)
}

// SendText sends one incremental text chunk to the realtime websocket session.
func (s *Streamer) SendText(text string) error {
	return s.Send(StreamTextMessage{
		Text: text,
	})
}

// Flush asks the realtime websocket session to generate audio for buffered text.
func (s *Streamer) Flush() error {
	flush := true
	return s.Send(StreamTextMessage{
		Text:  "",
		Flush: &flush,
	})
}

// CloseInput closes the logical text input side of the realtime websocket session.
// It currently sends a final flush frame and keeps reading output events until completion.
func (s *Streamer) CloseInput() error {
	return s.Flush()
}

func (s *Streamer) Close() error {
	return s.conn.Close()
}

func (s *Streamer) run() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		event, err := s.conn.ReadEvent(s.ctx)
		if err != nil {
			var permanent *transcripts.PermanentError
			if errors.As(err, &permanent) {
				if websocket.CloseStatus(permanent.Err) == websocket.StatusNormalClosure {
					return nil
				}
				return permanent.Err
			}
			return err
		}
		for _, handler := range s.handlers {
			handler(s.ctx, event)
		}
		if done, ok := event.(DoneEvent); ok && done.IsFinal {
			return nil
		}
	}
}

func unmarshalStreamEvent(data []byte) (StreamEvent, error) {
	var probe struct {
		Audio   string `json:"audio"`
		IsFinal bool   `json:"isFinal"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	if probe.Error != "" {
		return ErrorEvent{Message: probe.Error}, nil
	}
	if probe.Message != "" && probe.Audio == "" && !probe.IsFinal {
		return ErrorEvent{Message: probe.Message}, nil
	}
	if probe.Audio != "" {
		audio, err := base64.StdEncoding.DecodeString(probe.Audio)
		if err != nil {
			return nil, err
		}
		return AudioEvent{
			Audio:   audio,
			IsFinal: probe.IsFinal,
		}, nil
	}
	if probe.IsFinal {
		return DoneEvent{IsFinal: true}, nil
	}
	return nil, errors.New("unknown stream event")
}

func strconvFormatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func strconvFormatInt(value int) string {
	return strconv.Itoa(value)
}
