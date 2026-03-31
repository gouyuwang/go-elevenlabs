package tts

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func acceptHeader(format AudioFormat) string {
	if format == "" {
		return "audio/mpeg"
	}
	switch {
	case strings.HasPrefix(string(format), "mp3_"):
		return "audio/mpeg"
	case strings.HasPrefix(string(format), "pcm_"):
		return "audio/pcm"
	case strings.HasPrefix(string(format), "ulaw_"), strings.HasPrefix(string(format), "alaw_"):
		return "audio/basic"
	case strings.HasPrefix(string(format), "opus_"):
		return "audio/ogg"
	default:
		return "application/octet-stream"
	}
}

func characterCount(header http.Header) string {
	if value := header.Get("x-character-count"); value != "" {
		return value
	}
	return header.Get("character-cost")
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
