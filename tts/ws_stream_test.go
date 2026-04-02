package tts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestClientConnectRealtimeAndReceiveAudio(t *testing.T) {
	t.Parallel()

	type receivedMessage struct {
		Text          string         `json:"text"`
		XIAPIKey      string         `json:"xi_api_key,omitempty"`
		ModelID       string         `json:"model_id,omitempty"`
		VoiceSettings *VoiceSettings `json:"voice_settings,omitempty"`
	}

	var (
		mu       sync.Mutex
		messages []receivedMessage
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v1/text-to-speech/voice_123/stream-input"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("output_format"), "mp3_44100_128"; got != want {
			t.Fatalf("output_format = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("enable_ssml_parsing"), "true"; got != want {
			t.Fatalf("enable_ssml_parsing = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("inactivity_timeout"), "45"; got != want {
			t.Fatalf("inactivity_timeout = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("sync_alignment"), "true"; got != want {
			t.Fatalf("sync_alignment = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("auto_mode"), "true"; got != want {
			t.Fatalf("auto_mode = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("apply_text_normalization"), "on"; got != want {
			t.Fatalf("apply_text_normalization = %s, want %s", got, want)
		}
		if got, want := r.URL.Query().Get("seed"), "7"; got != want {
			t.Fatalf("seed = %s, want %s", got, want)
		}

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("accept websocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := r.Context()
		for i := 0; i < 2; i++ {
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("read message: %v", err)
			}
			var msg receivedMessage
			if err = json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			mu.Lock()
			messages = append(messages, msg)
			mu.Unlock()
		}

		if err = conn.Write(ctx, websocket.MessageText, []byte(`{"audio":"aGVsbG8=","isFinal":false}`)); err != nil {
			t.Fatalf("write audio event: %v", err)
		}
		if err = conn.Write(ctx, websocket.MessageText, []byte(`{"isFinal":true}`)); err != nil {
			t.Fatalf("write final event: %v", err)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	stability := 0.3
	enableSSMLParsing := true
	inactivityTimeout := 45
	syncAlignment := true
	autoMode := true
	seed := 7
	conn, err := client.ConnectRealtime(context.Background(), StreamInputRequest{
		VoiceID:                "voice_123",
		ModelID:                ModelElevenTurboV25,
		OutputFormat:           AudioFormatMP344100128,
		EnableSSMLParsing:      &enableSSMLParsing,
		InactivityTimeout:      &inactivityTimeout,
		SyncAlignment:          &syncAlignment,
		AutoMode:               &autoMode,
		ApplyTextNormalization: TextNormalizationOn,
		Seed:                   &seed,
		VoiceSettings: &VoiceSettings{
			Stability: stability,
		},
	})
	if err != nil {
		t.Fatalf("ConnectRealtime() error = %v", err)
	}
	defer conn.Close()

	var gotEvents []StreamEvent
	streamer := NewRealtimeSynthesizer(context.Background(), conn, func(_ context.Context, event StreamEvent) {
		gotEvents = append(gotEvents, event)
	})
	streamer.Start()

	if err = streamer.Send(StreamTextMessage{Text: "hello world"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case err = <-streamer.Err():
		if err != nil {
			t.Fatalf("streamer error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
	}

	mu.Lock()
	defer mu.Unlock()
	if got, want := len(messages), 2; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got, want := messages[0].XIAPIKey, "test-key"; got != want {
		t.Fatalf("init xi_api_key = %s, want %s", got, want)
	}
	if got, want := messages[0].ModelID, ModelElevenTurboV25; got != want {
		t.Fatalf("init model_id = %s, want %s", got, want)
	}
	if messages[0].VoiceSettings == nil {
		t.Fatal("missing voice settings in init message")
	}
	if got, want := messages[1].Text, "hello world"; got != want {
		t.Fatalf("send text = %s, want %s", got, want)
	}
	if got, want := len(gotEvents), 2; got != want {
		t.Fatalf("len(gotEvents) = %d, want %d", got, want)
	}
	audioEvent, ok := gotEvents[0].(AudioEvent)
	if !ok {
		t.Fatalf("gotEvents[0] type = %T, want AudioEvent", gotEvents[0])
	}
	if got, want := string(audioEvent.Audio), "hello"; got != want {
		t.Fatalf("audio bytes = %s, want %s", got, want)
	}
	finalEvent, ok := gotEvents[1].(DoneEvent)
	if !ok {
		t.Fatalf("gotEvents[1] type = %T, want DoneEvent", gotEvents[1])
	}
	if !finalEvent.IsFinal {
		t.Fatal("final event should be final")
	}
}

func TestAudioFormatConstantsCoverDocumentedValues(t *testing.T) {
	t.Parallel()

	formats := []AudioFormat{
		AudioFormatMP32205032,
		AudioFormatMP34410032,
		AudioFormatMP34410064,
		AudioFormatMP34410096,
		AudioFormatMP344100128,
		AudioFormatMP344100192,
		AudioFormatPCM8000,
		AudioFormatPCM16000,
		AudioFormatPCM22050,
		AudioFormatPCM24000,
		AudioFormatPCM44100,
		AudioFormatULAW8000,
		AudioFormatALAW8000,
		AudioFormatOpus4800032,
		AudioFormatOpus4800064,
		AudioFormatOpus4800096,
		AudioFormatOpus48000128,
		AudioFormatOpus48000192,
	}

	if got, want := len(formats), 18; got != want {
		t.Fatalf("len(formats) = %d, want %d", got, want)
	}
}

func TestClientConnectStreamInputAliasAndNewStreamerAlias(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("accept websocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := r.Context()
		for i := 0; i < 2; i++ {
			if _, _, err = conn.Read(ctx); err != nil {
				t.Fatalf("read message: %v", err)
			}
		}
		if err = conn.Write(ctx, websocket.MessageText, []byte(`{"isFinal":true}`)); err != nil {
			t.Fatalf("write final event: %v", err)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	conn, err := client.ConnectStreamInput(context.Background(), StreamInputRequest{
		VoiceID: "voice_123",
	})
	if err != nil {
		t.Fatalf("ConnectStreamInput() error = %v", err)
	}
	defer conn.Close()

	streamer := NewStreamer(context.Background(), conn)
	streamer.Start()
	if err = streamer.Send(StreamTextMessage{Text: "alias"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	select {
	case err = <-streamer.Err():
		if err != nil {
			t.Fatalf("streamer error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for streamer completion")
	}
}

func TestRealtimeSynthesizerHelperMethods(t *testing.T) {
	t.Parallel()

	type receivedMessage struct {
		Text  string `json:"text"`
		Flush *bool  `json:"flush,omitempty"`
	}

	var (
		mu       sync.Mutex
		messages []receivedMessage
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("accept websocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := r.Context()
		for i := 0; i < 5; i++ {
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("read message: %v", err)
			}
			var msg receivedMessage
			if err = json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			mu.Lock()
			messages = append(messages, msg)
			mu.Unlock()
		}
		if err = conn.Write(ctx, websocket.MessageText, []byte(`{"isFinal":true}`)); err != nil {
			t.Fatalf("write final event: %v", err)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig("test-key")
	cfg.BaseURL = server.URL
	client := NewClientWithConfig(cfg)

	conn, err := client.ConnectRealtime(context.Background(), StreamInputRequest{
		VoiceID: "voice_123",
	})
	if err != nil {
		t.Fatalf("ConnectRealtime() error = %v", err)
	}
	defer conn.Close()

	streamer := NewRealtimeSynthesizer(context.Background(), conn)
	streamer.Start()
	if err = streamer.SendText("first"); err != nil {
		t.Fatalf("SendText(first) error = %v", err)
	}
	if err = streamer.SendText("second"); err != nil {
		t.Fatalf("SendText(second) error = %v", err)
	}
	if err = streamer.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if err = streamer.CloseInput(); err != nil {
		t.Fatalf("CloseInput() error = %v", err)
	}

	select {
	case err = <-streamer.Err():
		if err != nil {
			t.Fatalf("streamer error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for streamer completion")
	}

	mu.Lock()
	defer mu.Unlock()
	if got, want := len(messages), 5; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got, want := messages[1].Text, "first"; got != want {
		t.Fatalf("messages[1].Text = %s, want %s", got, want)
	}
	if got, want := messages[2].Text, "second"; got != want {
		t.Fatalf("messages[2].Text = %s, want %s", got, want)
	}
	if messages[3].Flush == nil || !*messages[3].Flush {
		t.Fatal("messages[3] should be flush")
	}
	if got, want := messages[3].Text, ""; got != want {
		t.Fatalf("messages[3].Text = %q, want %q", got, want)
	}
	if messages[4].Flush == nil || !*messages[4].Flush {
		t.Fatal("messages[4] should be close-input flush")
	}
	if got, want := messages[4].Text, ""; got != want {
		t.Fatalf("messages[4].Text = %q, want %q", got, want)
	}
}
