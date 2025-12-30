package transcripts

import "encoding/json"

// ClientEventType is the type of client event.
type ClientEventType string

const (
	ClientEventTypeSessionUpdate ClientEventType = "input_audio_chunk"
)

// ClientEvent is the interface for client event.
type ClientEvent interface {
	ClientEventType() ClientEventType
}

// EventBase is the base struct for all client events.
type EventBase struct {
	Type ClientEventType `json:"message_type"`
}

type InputAudioChunkEvent struct {
	EventBase
	// Required. Base64 encoded audio data.
	Audio string `json:"audio_base_64"`
	// Required. Whether to commit the transcription.
	Commit bool `json:"commit"`
	// Required. Sample rate of the audio in Hz.
	SampleRate int64 `json:"sample_rate"`
	// Optional. Send text context to the model. Can only be sent alongside the first audio chunk. If sent in a subsequent chunk, an error will be returned.
	PreviousTxt string `json:"previous_txt,omitempty"`
}

func (m InputAudioChunkEvent) ClientEventType() ClientEventType {
	return ClientEventTypeSessionUpdate
}

func (m InputAudioChunkEvent) MarshalJSON() ([]byte, error) {
	type inputAudioChunkEvent InputAudioChunkEvent
	v := struct {
		*inputAudioChunkEvent
		Type ClientEventType `json:"message_type"`
	}{
		inputAudioChunkEvent: (*inputAudioChunkEvent)(&m),
		Type:                 m.ClientEventType(),
	}
	return json.Marshal(v)
}

// MarshalClientEvent marshals the client event to JSON.
func MarshalClientEvent(event ClientEvent) ([]byte, error) {
	return json.Marshal(event)
}
