package transcripts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

func (c *Client) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResponse, error) {
	if err := c.validateTranscriptionRequest(req); err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writeMultipartField(writer, "model_id", req.ModelID); err != nil {
		return nil, err
	}
	if err := writeMultipartField(writer, "language_code", req.LanguageCode); err != nil {
		return nil, err
	}
	if err := writeMultipartField(writer, "source_url", req.SourceURL); err != nil {
		return nil, err
	}
	if req.Diarize != nil {
		if err := writeMultipartField(writer, "diarize", strconv.FormatBool(*req.Diarize)); err != nil {
			return nil, err
		}
	}
	if req.DiarizationThreshold != nil {
		if err := writeMultipartField(writer, "diarization_threshold", strconv.FormatFloat(*req.DiarizationThreshold, 'f', -1, 64)); err != nil {
			return nil, err
		}
	}
	if req.TagAudioEvents != nil {
		if err := writeMultipartField(writer, "tag_audio_events", strconv.FormatBool(*req.TagAudioEvents)); err != nil {
			return nil, err
		}
	}
	if req.NumSpeakers != nil {
		if err := writeMultipartField(writer, "num_speakers", strconv.Itoa(*req.NumSpeakers)); err != nil {
			return nil, err
		}
	}
	if err := writeMultipartField(writer, "timestamps_granularity", req.TimestampsGranularity); err != nil {
		return nil, err
	}
	if err := writeMultipartField(writer, "file_format", req.FileFormat); err != nil {
		return nil, err
	}
	if req.Temperature != nil {
		if err := writeMultipartField(writer, "temperature", strconv.FormatFloat(*req.Temperature, 'f', -1, 64)); err != nil {
			return nil, err
		}
	}
	if req.Seed != nil {
		if err := writeMultipartField(writer, "seed", strconv.Itoa(*req.Seed)); err != nil {
			return nil, err
		}
	}
	if len(req.EntityDetection) > 0 {
		value, err := json.Marshal(req.EntityDetection)
		if err != nil {
			return nil, err
		}
		if err = writeMultipartField(writer, "entity_detection", string(value)); err != nil {
			return nil, err
		}
	}
	if err := writeMultipartField(writer, "entity_redaction", req.EntityRedaction); err != nil {
		return nil, err
	}
	if err := writeMultipartField(writer, "entity_redaction_mode", req.EntityRedactionMode); err != nil {
		return nil, err
	}
	if len(req.Keyterms) > 0 {
		value, err := json.Marshal(req.Keyterms)
		if err != nil {
			return nil, err
		}
		if err = writeMultipartField(writer, "keyterms", string(value)); err != nil {
			return nil, err
		}
	}
	if len(req.AdditionalFormats) > 0 {
		value, err := json.Marshal(req.AdditionalFormats)
		if err != nil {
			return nil, err
		}
		if err = writeMultipartField(writer, "additional_formats", string(value)); err != nil {
			return nil, err
		}
	}
	if req.Webhook != nil {
		if err := writeMultipartField(writer, "webhook", strconv.FormatBool(*req.Webhook)); err != nil {
			return nil, err
		}
	}
	if len(req.WebhookMetadata) > 0 {
		value, err := json.Marshal(req.WebhookMetadata)
		if err != nil {
			return nil, err
		}
		if err = writeMultipartField(writer, "webhook_metadata", string(value)); err != nil {
			return nil, err
		}
	}
	if req.File != nil {
		part, err := writer.CreateFormFile("file", req.FileName)
		if err != nil {
			return nil, err
		}
		if _, err = io.Copy(part, req.File); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.getTranscribeURL(), body)
	if err != nil {
		return nil, err
	}
	httpReq.Header = c.getHeaders()
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if req.EnableLogging != nil {
		query := httpReq.URL.Query()
		query.Set("enable_logging", strconv.FormatBool(*req.EnableLogging))
		httpReq.URL.RawQuery = query.Encode()
	}

	resp, err := c.getHTTPClient().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseAPIError(resp)
	}

	var out TranscriptionResponse
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	out.RequestID = resp.Header.Get("request-id")
	out.Headers = resp.Header.Clone()
	return &out, nil
}

func writeMultipartField(writer *multipart.Writer, name, value string) error {
	if value == "" {
		return nil
	}
	return writer.WriteField(name, value)
}

func (c *Client) getHTTPClient() *http.Client {
	if c.config.HTTPClient != nil {
		return c.config.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) getTranscribeURL() string {
	if c.config.HTTPBaseURL != "" && (c.config.HTTPBaseURL != HTTPBaseURL || c.config.BaseURL == BaseUrl) {
		return c.config.HTTPBaseURL
	}
	replacer := strings.NewReplacer("wss://", "https://", "ws://", "http://")
	return strings.TrimSuffix(replacer.Replace(c.config.BaseURL), "/realtime")
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("request-id"),
		Body:       body,
	}

	var payload struct {
		Detail struct {
			Message string `json:"message"`
		} `json:"detail"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		switch {
		case payload.Detail.Message != "":
			apiErr.Message = payload.Detail.Message
		case payload.Message != "":
			apiErr.Message = payload.Message
		}
	}
	if apiErr.Message == "" && len(body) > 0 {
		apiErr.Message = string(body)
	}
	return apiErr
}

func (c *Client) validateTranscriptionRequest(req TranscriptionRequest) error {
	if req.File == nil && req.SourceURL == "" {
		return fmt.Errorf("file or source_url is required")
	}
	if req.File != nil && req.FileName == "" {
		return fmt.Errorf("file name is required")
	}
	return nil
}
