package transcripts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

type captureDialer struct {
	url    string
	header http.Header
	conn   *captureWebSocketConn
}

func (d *captureDialer) Dial(_ context.Context, rawURL string, header http.Header) (WebSocketConn, error) {
	d.url = rawURL
	d.header = header.Clone()
	if d.conn == nil {
		d.conn = &captureWebSocketConn{}
	}
	return d.conn, nil
}

type captureWebSocketConn struct {
	writes [][]byte
}

func (c *captureWebSocketConn) ReadMessage(context.Context) (MessageType, []byte, error) {
	return MessageText, nil, nil
}

func (c *captureWebSocketConn) WriteMessage(_ context.Context, _ MessageType, data []byte) error {
	c.writes = append(c.writes, append([]byte(nil), data...))
	return nil
}

func (c *captureWebSocketConn) Close() error {
	return nil
}

func (c *captureWebSocketConn) Response() *http.Response {
	return &http.Response{Header: make(http.Header)}
}

func (c *captureWebSocketConn) Ping(context.Context) error {
	return nil
}

func TestClientConnectEncodesLatestRealtimeQueryParameters(t *testing.T) {
	t.Parallel()

	dialer := &captureDialer{}
	includeTimestamps := false
	includeLanguageDetection := true
	enableLogging := false
	noVerbatim := true
	vadSilenceThreshold := 1.8
	vadThreshold := 0.6
	minSpeechDuration := 120
	minSilenceDuration := 200

	client := NewClient("test-key")
	_, err := client.Connect(context.Background(),
		WithDialer(dialer),
		WithRealtimeConfig(RealtimeConfig{
			Token:                    "single-use-token",
			IncludeTimestamps:        &includeTimestamps,
			IncludeLanguageDetection: &includeLanguageDetection,
			AudioFormat:              AudioFormatPcm_24000,
			LanguageCode:             "eng",
			CommitStrategy:           CommitStrategyVAD,
			Keyterms:                 []string{"ElevenLabs", "Golang"},
			NoVerbatim:               &noVerbatim,
			VadSilenceThresholdSecs:  &vadSilenceThreshold,
			VadThreshold:             &vadThreshold,
			MinSpeechDurationMs:      &minSpeechDuration,
			MinSilenceDurationMs:     &minSilenceDuration,
			EnableLogging:            &enableLogging,
		}),
	)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	u, err := url.Parse(dialer.url)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	query := u.Query()
	if got, want := query.Get("model_id"), ModelScribeV2Realtime; got != want {
		t.Fatalf("model_id = %s, want %s", got, want)
	}
	if got, want := query.Get("token"), "single-use-token"; got != want {
		t.Fatalf("token = %s, want %s", got, want)
	}
	if got, want := query.Get("include_timestamps"), "false"; got != want {
		t.Fatalf("include_timestamps = %s, want %s", got, want)
	}
	if got, want := query.Get("include_language_detection"), "true"; got != want {
		t.Fatalf("include_language_detection = %s, want %s", got, want)
	}
	if got, want := query.Get("audio_format"), string(AudioFormatPcm_24000); got != want {
		t.Fatalf("audio_format = %s, want %s", got, want)
	}
	if got, want := query.Get("language_code"), "eng"; got != want {
		t.Fatalf("language_code = %s, want %s", got, want)
	}
	if got, want := query.Get("commit_strategy"), string(CommitStrategyVAD); got != want {
		t.Fatalf("commit_strategy = %s, want %s", got, want)
	}
	if got, want := query.Get("no_verbatim"), "true"; got != want {
		t.Fatalf("no_verbatim = %s, want %s", got, want)
	}
	if got, want := query.Get("vad_silence_threshold_secs"), "1.8"; got != want {
		t.Fatalf("vad_silence_threshold_secs = %s, want %s", got, want)
	}
	if got, want := query.Get("vad_threshold"), "0.6"; got != want {
		t.Fatalf("vad_threshold = %s, want %s", got, want)
	}
	if got, want := query.Get("min_speech_duration_ms"), "120"; got != want {
		t.Fatalf("min_speech_duration_ms = %s, want %s", got, want)
	}
	if got, want := query.Get("min_silence_duration_ms"), "200"; got != want {
		t.Fatalf("min_silence_duration_ms = %s, want %s", got, want)
	}
	if got, want := query.Get("enable_logging"), "false"; got != want {
		t.Fatalf("enable_logging = %s, want %s", got, want)
	}

	keyterms := query["keyterms"]
	if got, want := len(keyterms), 2; got != want {
		t.Fatalf("len(keyterms) = %d, want %d", got, want)
	}
	if got, want := keyterms[0], "ElevenLabs"; got != want {
		t.Fatalf("keyterms[0] = %s, want %s", got, want)
	}
	if got, want := keyterms[1], "Golang"; got != want {
		t.Fatalf("keyterms[1] = %s, want %s", got, want)
	}

	if got, want := dialer.header.Get("xi-api-key"), "test-key"; got != want {
		t.Fatalf("xi-api-key = %s, want %s", got, want)
	}
}

func TestClientConnectAllowsTokenOnlyAuthentication(t *testing.T) {
	t.Parallel()

	dialer := &captureDialer{}
	client := NewClientWithConfig(DefaultConfig(""))

	_, err := client.Connect(context.Background(),
		WithDialer(dialer),
		WithRealtimeConfig(RealtimeConfig{
			Token: "token-only",
		}),
	)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if got := dialer.header.Get("xi-api-key"); got != "" {
		t.Fatalf("xi-api-key = %q, want empty", got)
	}
}

func TestInputAudioChunkEventMarshalUsesPreviousText(t *testing.T) {
	t.Parallel()

	data, err := MarshalClientEvent(InputAudioChunkEvent{
		Audio:        "aGVsbG8=",
		Commit:       true,
		SampleRate:   16000,
		PreviousText: "legacy field",
	})
	if err != nil {
		t.Fatalf("MarshalClientEvent() error = %v", err)
	}

	var got map[string]any
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if _, ok := got["previous_txt"]; ok {
		t.Fatal("previous_txt should not be present in marshaled payload")
	}
	if value, ok := got["previous_text"].(string); !ok || value != "legacy field" {
		t.Fatalf("previous_text = %#v, want %q", got["previous_text"], "legacy field")
	}
}

func TestRecognizerSendAndCommitEmitRequiredRealtimeFields(t *testing.T) {
	t.Parallel()

	wsConn := &captureWebSocketConn{}
	recognizer := NewRecognizer(context.Background(), &Conn{
		conn:       wsConn,
		logger:     NopLogger{},
		sampleRate: 24000,
	})

	if err := recognizer.Send([]byte("pcm-data")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if err := recognizer.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got, want := len(wsConn.writes), 2; got != want {
		t.Fatalf("len(writes) = %d, want %d", got, want)
	}

	var sendPayload map[string]any
	if err := json.Unmarshal(wsConn.writes[0], &sendPayload); err != nil {
		t.Fatalf("Unmarshal(send) error = %v", err)
	}
	if got, want := sendPayload["message_type"], string(ClientEventTypeSessionUpdate); got != want {
		t.Fatalf("send message_type = %v, want %s", got, want)
	}
	if got, want := sendPayload["commit"], false; got != want {
		t.Fatalf("send commit = %v, want %v", got, want)
	}
	if got, want := sendPayload["sample_rate"], float64(24000); got != want {
		t.Fatalf("send sample_rate = %v, want %v", got, want)
	}
	if got, want := sendPayload["audio_base_64"], base64.StdEncoding.EncodeToString([]byte("pcm-data")); got != want {
		t.Fatalf("send audio_base_64 = %v, want %s", got, want)
	}

	var commitPayload map[string]any
	if err := json.Unmarshal(wsConn.writes[1], &commitPayload); err != nil {
		t.Fatalf("Unmarshal(commit) error = %v", err)
	}
	if got, want := commitPayload["commit"], true; got != want {
		t.Fatalf("commit commit = %v, want %v", got, want)
	}
	if got, want := commitPayload["sample_rate"], float64(24000); got != want {
		t.Fatalf("commit sample_rate = %v, want %v", got, want)
	}
}

func TestUnmarshalServerEventSupportsLatestRealtimeSchemas(t *testing.T) {
	t.Parallel()

	sessionData := []byte(`{
		"message_type":"session_started",
		"session_id":"sess_123",
		"config":{
			"sample_rate":24000,
			"audio_format":"pcm_24000",
			"language_code":"en",
			"timestamps_granularity":"word",
			"vad_commit_strategy":true,
			"vad_silence_threshold_secs":1.5,
			"vad_threshold":0.4,
			"min_speech_duration_ms":100,
			"min_silence_duration_ms":100,
			"max_tokens_to_recompute":5,
			"model_id":"scribe_v2_realtime",
			"disable_logging":true,
			"include_timestamps":true,
			"include_language_detection":true,
			"keyterms":["ElevenLabs"],
			"no_verbatim":true
		}
	}`)

	event, err := UnmarshalServerEvent(sessionData)
	if err != nil {
		t.Fatalf("UnmarshalServerEvent(session_started) error = %v", err)
	}
	session, ok := event.(SessionStartEventArgs)
	if !ok {
		t.Fatalf("event type = %T, want SessionStartEventArgs", event)
	}
	if got, want := session.Config.CommitStrategy, CommitStrategyVAD; got != want {
		t.Fatalf("CommitStrategy = %s, want %s", got, want)
	}
	if got, want := session.Config.EnableLogging, false; got != want {
		t.Fatalf("EnableLogging = %v, want %v", got, want)
	}
	if got, want := session.Config.TimestampsGranularity, "word"; got != want {
		t.Fatalf("TimestampsGranularity = %s, want %s", got, want)
	}
	if got, want := session.Config.MaxTokensToRecompute, 5; got != want {
		t.Fatalf("MaxTokensToRecompute = %d, want %d", got, want)
	}
	if got, want := len(session.Config.Keyterms), 1; got != want {
		t.Fatalf("len(Keyterms) = %d, want %d", got, want)
	}
	if got, want := session.Config.NoVerbatim, true; got != want {
		t.Fatalf("NoVerbatim = %v, want %v", got, want)
	}

	timestampData := []byte(`{
		"message_type":"committed_transcript_with_timestamps",
		"text":"hello world",
		"language_code":"en",
		"words":[
			{
				"text":"hello",
				"start":0.0,
				"end":0.4,
				"type":"word",
				"speaker_id":"speaker_1",
				"logprob":-0.05,
				"characters":["h","e","l","l","o"]
			},
			{
				"text":" ",
				"start":0.4,
				"end":0.42,
				"type":"spacing"
			}
		]
	}`)

	event, err = UnmarshalServerEvent(timestampData)
	if err != nil {
		t.Fatalf("UnmarshalServerEvent(committed_transcript_with_timestamps) error = %v", err)
	}
	transcript, ok := event.(SpeechRecognizedWithTimestampEventArgs)
	if !ok {
		t.Fatalf("event type = %T, want SpeechRecognizedWithTimestampEventArgs", event)
	}
	if got, want := len(transcript.Words), 2; got != want {
		t.Fatalf("len(Words) = %d, want %d", got, want)
	}
	if got, want := transcript.Words[0].Type, "word"; got != want {
		t.Fatalf("Words[0].Type = %s, want %s", got, want)
	}
	if got, want := transcript.Words[0].SpeakerID, "speaker_1"; got != want {
		t.Fatalf("Words[0].SpeakerID = %s, want %s", got, want)
	}
	if got, want := transcript.Words[0].LogProb, -0.05; got != want {
		t.Fatalf("Words[0].LogProb = %v, want %v", got, want)
	}
	if got, want := len(transcript.Words[0].Characters), 5; got != want {
		t.Fatalf("len(Words[0].Characters) = %d, want %d", got, want)
	}
}
