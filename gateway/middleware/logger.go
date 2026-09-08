package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// SupabaseLogger logs LLM request data to a Supabase PostgREST table asynchronously.
type SupabaseLogger struct {
	supabaseURL string
	anonKey     string
	client      *http.Client
	enabled     bool
}

type requestLog struct {
	Prompt       string  `json:"prompt"`
	Provider     string  `json:"provider"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	LatencyMs    int64   `json:"latency_ms"`
}

// NewSupabaseLogger creates a new Supabase logger.
// If SUPABASE_URL or SUPABASE_ANON_KEY are empty, the logger is disabled (no-ops silently).
func NewSupabaseLogger(supabaseURL, anonKey string) *SupabaseLogger {
	enabled := supabaseURL != "" && anonKey != ""
	if enabled {
		slog.Info("Supabase logging enabled", "url", supabaseURL)
	} else {
		slog.Warn("Supabase logging disabled (missing SUPABASE_URL or SUPABASE_ANON_KEY)")
	}
	return &SupabaseLogger{
		supabaseURL: supabaseURL,
		anonKey:     anonKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: enabled,
	}
}

// LogRequest asynchronously logs an LLM request to the Supabase 'requests' table.
// This method is fire-and-forget — it never blocks or crashes the request path.
func (l *SupabaseLogger) LogRequest(prompt, provider string, inputTokens, outputTokens int, costUSD float64, latencyMs int64) {
	if !l.enabled {
		return
	}

	go func() {
		log := requestLog{
			Prompt:       prompt,
			Provider:     provider,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			CostUSD:      costUSD,
			LatencyMs:    latencyMs,
		}

		body, err := json.Marshal(log)
		if err != nil {
			slog.Warn("Supabase: failed to marshal log", "error", err)
			return
		}

		url := fmt.Sprintf("%s/rest/v1/requests", l.supabaseURL)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			slog.Warn("Supabase: failed to create request", "error", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", l.anonKey)
		req.Header.Set("Authorization", "Bearer "+l.anonKey)
		req.Header.Set("Prefer", "return=minimal")

		resp, err := l.client.Do(req)
		if err != nil {
			slog.Warn("Supabase: failed to send log", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			slog.Warn("Supabase: unexpected status", "status", resp.StatusCode)
		} else {
			slog.Debug("Supabase: log sent successfully", "provider", provider)
		}
	}()
}
