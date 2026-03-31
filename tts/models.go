package tts

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.modelsURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", c.config.authKey)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseAPIError(resp)
	}

	var models []Model
	if err = json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}
	return models, nil
}

func TextToSpeechModels(models []Model) []Model {
	filtered := make([]Model, 0, len(models))
	for _, model := range models {
		if model.CanDoTextToSpeech == nil || !*model.CanDoTextToSpeech {
			continue
		}
		filtered = append(filtered, model)
	}
	return filtered
}

func (c *Client) modelsURL() string {
	return strings.TrimRight(c.config.BaseURL, "/") + "/v1/models"
}
