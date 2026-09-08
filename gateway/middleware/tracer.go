package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"crypto/rand"
	mrand "math/rand"
)

// LangfuseTracer sends trace + generation events to the LangFuse Cloud ingestion API.
type LangfuseTracer struct {
	host      string
	publicKey string
	secretKey string
	client    *http.Client
	enabled   bool
}

// NewLangfuseTracer creates a new LangFuse tracer.
// If any of LANGFUSE_HOST, LANGFUSE_PUBLIC_KEY, or LANGFUSE_SECRET_KEY are empty,
// the tracer is disabled (no-ops silently).
func NewLangfuseTracer(host, publicKey, secretKey string) *LangfuseTracer {
	enabled := host != "" && publicKey != "" && secretKey != ""
	if enabled {
		slog.Info("LangFuse tracing enabled", "host", host)
	} else {
		slog.Warn("LangFuse tracing disabled (missing LANGFUSE_HOST, LANGFUSE_PUBLIC_KEY, or LANGFUSE_SECRET_KEY)")
	}
	return &LangfuseTracer{
		host:      host,
		publicKey: publicKey,
		secretKey: secretKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: enabled,
	}
}

// TraceGeneration asynchronously sends a trace + generation event to LangFuse.
// This method is fire-and-forget — it never blocks or crashes the request path.
func (t *LangfuseTracer) TraceGeneration(prompt, responseText, providerName, model string, inputTokens, outputTokens int, costUSD float64, latencyMs int64) {
	if !t.enabled {
		return
	}

	go func() {
		now := time.Now().UTC()
		traceID := generateUUID()
		genID := generateUUID()

		startTime := now.Add(-time.Duration(latencyMs) * time.Millisecond).Format(time.RFC3339Nano)
		endTime := now.Format(time.RFC3339Nano)
		timestamp := now.Format(time.RFC3339Nano)

		batch := map[string]any{
			"batch": []map[string]any{
				{
					"id":        generateUUID(),
					"timestamp": timestamp,
					"type":      "trace-create",
					"body": map[string]any{
						"id":   traceID,
						"name": "omni-router-completion",
						"metadata": map[string]any{
							"provider": providerName,
							"cost_usd": costUSD,
						},
					},
				},
				{
					"id":        generateUUID(),
					"timestamp": timestamp,
					"type":      "observation-create",
					"body": map[string]any{
						"id":        genID,
						"traceId":   traceID,
						"type":      "GENERATION",
						"name":      "llm-generation",
						"model":     model,
						"startTime": startTime,
						"endTime":   endTime,
						"input":     map[string]any{"prompt": prompt},
						"output":    responseText,
						"usage": map[string]any{
							"promptTokens":     inputTokens,
							"completionTokens": outputTokens,
							"totalTokens":      inputTokens + outputTokens,
						},
					},
				},
			},
		}

		body, err := json.Marshal(batch)
		if err != nil {
			slog.Warn("LangFuse: failed to marshal batch", "error", err)
			return
		}

		url := fmt.Sprintf("%s/api/public/ingestion", t.host)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			slog.Warn("LangFuse: failed to create request", "error", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(t.publicKey, t.secretKey)

		resp, err := t.client.Do(req)
		if err != nil {
			slog.Warn("LangFuse: failed to send trace", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			slog.Warn("LangFuse: unexpected status", "status", resp.StatusCode)
		} else {
			slog.Debug("LangFuse: trace sent successfully", "traceId", traceID, "provider", providerName)
		}
	}()
}

// generateUUID creates a random UUID v4 string.
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to math/rand if crypto/rand fails (shouldn't happen)
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
