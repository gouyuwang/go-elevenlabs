package transcripts

import (
	"encoding/json"
	"fmt"
)

type ServerEventType string

const (
	ServerEventSessionStarted                    ServerEventType = "session_started"
	ServerEventPartialTranscript                 ServerEventType = "partial_transcript"
	ServerEventCommittedTranscript               ServerEventType = "committed_transcript"
	ServerEventCommittedTranscriptWithTimestamps ServerEventType = "committed_transcript_with_timestamps"
	ServerEventError                             ServerEventType = "error"
	ServerEventAuthError                         ServerEventType = "auth_error"
	ServerEventQuotaExceededError                ServerEventType = "quota_exceeded"
	ServerEventCommitThrottledError              ServerEventType = "commit_throttled"
	ServerEventUnacceptedTermsError              ServerEventType = "unaccepted_terms"
	ServerEventRateLimitedError                  ServerEventType = "rate_limited"
	ServerEventQueueOverflowError                ServerEventType = "queue_overflow"
	ServerEventResourceExhaustedError            ServerEventType = "resource_exhausted"
	ServerEventSessionTimeLimitExceededError     ServerEventType = "session_time_limit_exceeded"
	ServerEventInputError                        ServerEventType = "input_error"
	ServerEventChunkSizeExceededError            ServerEventType = "chunk_size_exceeded"
	ServerEventInsufficientAudioActivityError    ServerEventType = "insufficient_audio_activity"
	ServerEventTranscriberError                  ServerEventType = "transcriber_error"
	ServerEventInvalidRequestError               ServerEventType = "invalid_request"
)

// ServerEvent is the interface for server event.
type ServerEvent interface {
	ServerEventType() ServerEventType
}

func unmarshalServerEvent[T ServerEvent](data []byte) (T, error) {
	var t T
	err := json.Unmarshal(data, &t)
	if err != nil {
		return t, err
	}
	return t, nil
}

// UnmarshalServerEvent unmarshal the server event from the given JSON data.
func UnmarshalServerEvent(data []byte) (ServerEvent, error) {
	var eventType struct {
		Type ServerEventType `json:"message_type"`
	}
	err := json.Unmarshal(data, &eventType)
	if err != nil {
		return nil, err
	}
	switch eventType.Type {
	case ServerEventSessionStarted:
		return unmarshalServerEvent[SessionStartEventArgs](data)
	case ServerEventPartialTranscript:
		return unmarshalServerEvent[SpeechRecognizingEventArgs](data)
	case ServerEventCommittedTranscript:
		return unmarshalServerEvent[SpeechRecognizedEventArgs](data)
	case ServerEventCommittedTranscriptWithTimestamps:
		return unmarshalServerEvent[SpeechRecognizedWithTimestampEventArgs](data)
	case ServerEventError,
		ServerEventAuthError,
		ServerEventQuotaExceededError,
		ServerEventCommitThrottledError,
		ServerEventUnacceptedTermsError,
		ServerEventRateLimitedError,
		ServerEventQueueOverflowError,
		ServerEventResourceExhaustedError,
		ServerEventSessionTimeLimitExceededError,
		ServerEventInputError,
		ServerEventChunkSizeExceededError,
		ServerEventInsufficientAudioActivityError,
		ServerEventTranscriberError,
		ServerEventInvalidRequestError:
		return unmarshalServerEvent[SpeechRecognitionCanceledEventArgs](data)

	default:
		// This should never happen.
		return nil, fmt.Errorf("unknown client event type: %s", eventType.Type)
	}
}

type RecognitionEventArgs struct {
	// The message type identifier.
	Type ServerEventType `json:"message_type"`
}

func (r RecognitionEventArgs) ServerEventType() ServerEventType {
	return r.Type
}

type SessionStartConfig struct {
	// Optional. Sample rate of the audio in Hz.
	SampleRate int64 `json:"sample_rate,omitempty"`
	// Optional. Audio format of the audio. Defaults to pcm_16000
	AudioFormat AudioFormat `json:"audio_format,omitempty"`
	// Optional. Language code in ISO 639-1 or ISO 639-3 format.
	LanguageCode string `json:"language_code,omitempty"`
	// Optional. Strategy for committing transcriptions.
	CommitStrategy CommitStrategy `json:"commit_strategy,omitempty"`
	// Optional. Granularity for timestamps in the committed transcript.
	TimestampsGranularity string `json:"timestamps_granularity,omitempty"`
	// Optional. Silence threshold in seconds.
	VadSilenceThresholdSecs float64 `json:"vad_silence_threshold_secs,omitempty"`
	// Optional. Threshold for voice activity detection.
	VadThreshold float64 `json:"vad_threshold,omitempty"`
	// Optional. Minimum duration of speech in milliseconds.
	MinSpeechDurationMs int `json:"min_speech_duration_ms,omitempty"`
	// Optional. Minimum duration of silence in milliseconds.
	MinSilenceDurationMs int `json:"min_silence_duration_ms,omitempty"`
	// Optional. Number of tokens to recompute around each commit boundary.
	MaxTokensToRecompute int `json:"max_tokens_to_recompute,omitempty"`
	// Optional. ID of the model to use for transcription.
	ModelID string `json:"model_id,omitempty"`
	// Optional. When enable_logging is set to false zero retention mode will be used for the request.
	// This will mean history features are unavailable for this request. Zero retention mode may only be used by enterprise customers.
	EnableLogging bool `json:"enable_logging,omitempty"`
	// Optional. Whether the session will include word-level timestamps in the committed transcript.
	IncludeTimestamps bool `json:"include_timestamps,omitempty"`
	// Optional. Whether the session will include language detection in the committed transcript.
	IncludeLanguageDetection bool `json:"include_language_detection,omitempty"`
	// Optional. Keyterms the model is biased towards.
	Keyterms []string `json:"keyterms,omitempty"`
	// Optional. Whether filler words and disfluencies are removed from the transcript.
	NoVerbatim bool `json:"no_verbatim,omitempty"`
}

func (c *SessionStartConfig) UnmarshalJSON(data []byte) error {
	type sessionStartConfig SessionStartConfig
	var raw struct {
		sessionStartConfig
		VADCommitStrategy *bool `json:"vad_commit_strategy,omitempty"`
		DisableLogging    *bool `json:"disable_logging,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = SessionStartConfig(raw.sessionStartConfig)
	if raw.VADCommitStrategy != nil {
		if *raw.VADCommitStrategy {
			c.CommitStrategy = CommitStrategyVAD
		} else if c.CommitStrategy == "" {
			c.CommitStrategy = CommitStrategyManual
		}
	}
	if raw.DisableLogging != nil {
		c.EnableLogging = !*raw.DisableLogging
	}
	return nil
}

type SessionStartEventArgs struct {
	RecognitionEventArgs
	// Unique identifier for the session.
	SessionID string `json:"session_id"`
	// Configuration parameters for the session.
	Config SessionStartConfig `json:"config"`
}

type SpeechRecognizingEventArgs struct {
	RecognitionEventArgs
	// Committed transcription text.
	Text string `json:"text"`
}

type SpeechRecognizedEventArgs struct {
	RecognitionEventArgs
	// Committed transcription text.
	Text string `json:"text"`
}

type SpeechRecognizedWithTimestampEventArgs struct {
	RecognitionEventArgs
	// Committed transcription text.
	Text string `json:"text"`
	// Detected or specified language code.
	Language string `json:"language_code,omitempty"`
	// Word-level information with timestamps.
	Words []RealtimeTranscriptWord `json:"words"`
}

type RealtimeTranscriptWord struct {
	Text       string   `json:"text,omitempty"`
	Start      float64  `json:"start,omitempty"`
	End        float64  `json:"end,omitempty"`
	Type       string   `json:"type,omitempty"`
	SpeakerID  string   `json:"speaker_id,omitempty"`
	LogProb    float64  `json:"logprob,omitempty"`
	Characters []string `json:"characters,omitempty"`
}

type SpeechRecognitionCanceledEventArgs struct {
	RecognitionEventArgs
	Error string `json:"error"`
}
