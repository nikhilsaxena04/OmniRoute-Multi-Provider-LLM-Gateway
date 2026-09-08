package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nikhilsaxena04/omni-router/config"
)

type GeminiProvider struct {
	cfg    *config.ProviderConfig
	client *http.Client
}

func NewGeminiProvider(cfg *config.ProviderConfig) *GeminiProvider {
	return &GeminiProvider{
		cfg:    cfg,
		client: &http.Client{},
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Complete(ctx context.Context, prompt string) (*Response, error) {
	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// URL format: base_url/models/model:generateContent?key=api_key
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.cfg.BaseURL, p.cfg.Model, p.cfg.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned in response")
	}

	return &Response{
		Text:         result.Candidates[0].Content.Parts[0].Text,
		InputTokens:  result.UsageMetadata.PromptTokenCount,
		OutputTokens: result.UsageMetadata.CandidatesTokenCount,
	}, nil
}

func (p *GeminiProvider) Stream(ctx context.Context, prompt string, out chan<- Chunk) error {
	return fmt.Errorf("Stream not implemented for Gemini")
}
