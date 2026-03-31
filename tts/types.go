package tts

import (
	"fmt"
	"io"
	"net/http"
)

type AudioFormat string

const (
	AudioFormatMP344100128 AudioFormat = "mp3_44100_128"
	AudioFormatPCM44100    AudioFormat = "pcm_44100"
)

const (
	ModelElevenV3             = "eleven_v3"
	ModelElevenMultilingualV2 = "eleven_multilingual_v2"
	ModelElevenFlashV25       = "eleven_flash_v2_5"
	ModelElevenTurboV25       = "eleven_turbo_v2_5"
)

type VoiceSettings struct {
	Stability       *float64 `json:"stability,omitempty"`
	SimilarityBoost *float64 `json:"similarity_boost,omitempty"`
	Style           *float64 `json:"style,omitempty"`
	UseSpeakerBoost *bool    `json:"use_speaker_boost,omitempty"`
}

type SynthesisRequest struct {
	VoiceID                  string         `json:"-"`
	Text                     string         `json:"text"`
	ModelID                  string         `json:"model_id,omitempty"`
	OutputFormat             AudioFormat    `json:"-"`
	LanguageCode             string         `json:"language_code,omitempty"`
	VoiceSettings            *VoiceSettings `json:"voice_settings,omitempty"`
	EnableLogging            *bool          `json:"-"`
	OptimizeStreamingLatency *int           `json:"-"`
	Seed                     *int           `json:"seed,omitempty"`
	PreviousText             string         `json:"previous_text,omitempty"`
	NextText                 string         `json:"next_text,omitempty"`
	PreviousRequestIDs       []string       `json:"previous_request_ids,omitempty"`
	NextRequestIDs           []string       `json:"next_request_ids,omitempty"`
}

type SynthesisResponse struct {
	Audio          []byte
	ContentType    string
	RequestID      string
	CharacterCount string
	Headers        http.Header
}

type StreamResponse struct {
	Audio          io.ReadCloser
	ContentType    string
	RequestID      string
	CharacterCount string
	Headers        http.Header
}

type APIError struct {
	StatusCode int
	RequestID  string
	Message    string
	Body       []byte
}

type Model struct {
	ModelID           string          `json:"model_id"`
	Name              string          `json:"name,omitempty"`
	Description       string          `json:"description,omitempty"`
	CanDoTextToSpeech *bool           `json:"can_do_text_to_speech,omitempty"`
	Languages         []ModelLanguage `json:"languages,omitempty"`
	ModelRates        ModelRates      `json:"model_rates,omitempty"`
	ConcurrencyGroup  string          `json:"concurrency_group,omitempty"`
	MaxCharacters     int             `json:"max_characters,omitempty"`
}

type ModelLanguage struct {
	LanguageID string `json:"language_id,omitempty"`
	Name       string `json:"name,omitempty"`
}

type ModelRates struct {
	CharacterCostMultiplier float64 `json:"character_cost_multiplier,omitempty"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("elevenlabs api error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("elevenlabs api error (%d)", e.StatusCode)
}
