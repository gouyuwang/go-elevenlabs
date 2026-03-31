package transcripts

import (
	"fmt"
	"io"
	"net/http"
)

type TranscriptionRequest struct {
	ModelID               string
	FileName              string
	File                  io.Reader
	SourceURL             string
	LanguageCode          string
	TagAudioEvents        *bool
	NumSpeakers           *int
	Diarize               *bool
	DiarizationThreshold  *float64
	TimestampsGranularity string
	FileFormat            string
	Temperature           *float64
	Seed                  *int
	EnableLogging         *bool
	Webhook               *bool
	WebhookMetadata       map[string]any
	EntityDetection       []string
	EntityRedaction       string
	EntityRedactionMode   string
	Keyterms              []string
	AdditionalFormats     []TranscriptOutputFormatRequest
}

type TranscriptionWord struct {
	Text  string  `json:"text"`
	Start float64 `json:"start,omitempty"`
	End   float64 `json:"end,omitempty"`
}

type TranscriptAdditionalFormat struct {
	RequestedFormat string `json:"requested_format,omitempty"`
	FileExtension   string `json:"file_extension,omitempty"`
	ContentType     string `json:"content_type,omitempty"`
}

type TranscriptEntity struct {
	Text       string `json:"text,omitempty"`
	EntityType string `json:"entity_type,omitempty"`
	StartChar  int    `json:"start_char,omitempty"`
	EndChar    int    `json:"end_char,omitempty"`
}

type TranscriptOutputFormatRequest struct {
	Format string `json:"format"`
}

type TranscriptionResponse struct {
	Text                string                       `json:"text"`
	LanguageCode        string                       `json:"language_code,omitempty"`
	LanguageProbability float64                      `json:"language_probability,omitempty"`
	Words               []TranscriptionWord          `json:"words,omitempty"`
	TranscriptionID     string                       `json:"transcription_id,omitempty"`
	AdditionalFormats   []TranscriptAdditionalFormat `json:"additional_formats,omitempty"`
	Entities            []TranscriptEntity           `json:"entities,omitempty"`
	RequestID           string                       `json:"-"`
	Headers             http.Header                  `json:"-"`
}

type APIError struct {
	StatusCode int
	RequestID  string
	Message    string
	Body       []byte
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
