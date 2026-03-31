package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Client struct {
	config ClientConfig
}

func NewClient(authKey string) *Client {
	return &Client{
		config: DefaultConfig(authKey),
	}
}

func NewClientWithConfig(config ClientConfig) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) Synthesize(ctx context.Context, req SynthesisRequest) (*SynthesisResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, c.synthesizeURL(req.VoiceID), req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseAPIError(resp)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &SynthesisResponse{
		Audio:          audio,
		ContentType:    resp.Header.Get("Content-Type"),
		RequestID:      resp.Header.Get("request-id"),
		CharacterCount: characterCount(resp.Header),
		Headers:        resp.Header.Clone(),
	}, nil
}

// StreamAudio sends the full text once over HTTP and reads the audio response as a stream.
// This is HTTP audio streaming, not interactive realtime text-input streaming.
func (c *Client) StreamAudio(ctx context.Context, req SynthesisRequest) (*StreamResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, c.streamURL(req.VoiceID), req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	return &StreamResponse{
		Audio:          resp.Body,
		ContentType:    resp.Header.Get("Content-Type"),
		RequestID:      resp.Header.Get("request-id"),
		CharacterCount: characterCount(resp.Header),
		Headers:        resp.Header.Clone(),
	}, nil
}

func (c *Client) newRequest(ctx context.Context, method, url string, req SynthesisRequest) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	query := httpReq.URL.Query()
	if req.OutputFormat != "" {
		query.Set("output_format", string(req.OutputFormat))
	}
	if req.EnableLogging != nil {
		query.Set("enable_logging", strconv.FormatBool(*req.EnableLogging))
	}
	if req.OptimizeStreamingLatency != nil {
		query.Set("optimize_streaming_latency", strconv.Itoa(*req.OptimizeStreamingLatency))
	}
	httpReq.URL.RawQuery = query.Encode()
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", acceptHeader(req.OutputFormat))
	httpReq.Header.Set("xi-api-key", c.config.authKey)
	return httpReq, nil
}

func (c *Client) httpClient() *http.Client {
	if c.config.HTTPClient != nil {
		return c.config.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) synthesizeURL(voiceID string) string {
	return strings.TrimRight(c.config.BaseURL, "/") + "/v1/text-to-speech/" + voiceID
}

func (c *Client) streamURL(voiceID string) string {
	return c.synthesizeURL(voiceID) + "/stream"
}
