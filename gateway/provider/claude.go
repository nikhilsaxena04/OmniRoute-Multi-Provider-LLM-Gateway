package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nikhilsaxena04/omni-router/config"
)

type ClaudeProvider struct {
	cfg    *config.ProviderConfig
	client *http.Client
	name   string
}

func NewClaudeProvider(cfg *config.ProviderConfig, name string) *ClaudeProvider {
	return &ClaudeProvider{
		cfg:    cfg,
		client: &http.Client{},
		name:   name,
	}
}

func (p *ClaudeProvider) Name() string {
	return p.name
}

func (p *ClaudeProvider) Complete(ctx context.Context, prompt string) (*Response, error) {
	// If type is "openai", use OpenAI-compatible endpoint (e.g. Groq)
	if p.cfg.Type == "openai" {
		return completeOpenAICompat(ctx, p.client, p.cfg, prompt)
	}

	// Default: native Anthropic API
	reqBody := map[string]any{
		"model":      p.cfg.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/messages", p.cfg.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("no content returned in response")
	}

	return &Response{
		Text:         result.Content[0].Text,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}, nil
}

func (p *ClaudeProvider) Stream(ctx context.Context, prompt string, out chan<- Chunk) error {
	return fmt.Errorf("Stream not implemented for Claude")
}
