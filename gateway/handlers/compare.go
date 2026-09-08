package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/nikhilsaxena04/omni-router/config"
	"github.com/nikhilsaxena04/omni-router/metrics"
	"github.com/nikhilsaxena04/omni-router/middleware"
	"github.com/nikhilsaxena04/omni-router/provider"
)

type CompareRequest struct {
	Prompt string `json:"prompt"`
}

type ProviderResult struct {
	Provider string             `json:"provider"`
	Response *provider.Response `json:"response,omitempty"`
	Error    string             `json:"error,omitempty"`
	Latency  string             `json:"latency"`
	Cost     float64            `json:"cost,omitempty"`
}

type CompareResponse struct {
	Results []ProviderResult `json:"results"`
}

type CompareHandler struct {
	Providers []provider.Provider
	Pricing   *config.PricingConfig
	Logger    *middleware.SupabaseLogger
	Tracer    *middleware.LangfuseTracer
}

func (h *CompareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// 60 seconds strict timeout
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	results := make([]ProviderResult, len(h.Providers))
	var wg sync.WaitGroup

	for i, p := range h.Providers {
		wg.Add(1)
		
		// Capture loop variables
		go func(idx int, prov provider.Provider) {
			defer wg.Done()
			
			start := time.Now()
			resp, err := prov.Complete(ctx, req.Prompt)
			dur := time.Since(start)
			latency := dur.String()

			if err != nil {
				metrics.RequestsTotal.WithLabelValues("/v1/compare", prov.Name(), "500").Inc()
				metrics.LatencyHistogram.WithLabelValues("/v1/compare", prov.Name()).Observe(dur.Seconds())

				results[idx] = ProviderResult{
					Provider: prov.Name(),
					Error:    err.Error(),
					Latency:  latency,
				}
				return
			}

			metrics.RequestsTotal.WithLabelValues("/v1/compare", prov.Name(), "200").Inc()
			metrics.LatencyHistogram.WithLabelValues("/v1/compare", prov.Name()).Observe(dur.Seconds())

			cost := 0.0
			if h.Pricing != nil {
				cost = h.Pricing.CalculateCost(prov.Name(), resp.InputTokens, resp.OutputTokens)
				resp.Cost = cost
				resp.Provider = prov.Name()
			}

			results[idx] = ProviderResult{
				Provider: prov.Name(),
				Response: resp,
				Latency:  latency,
				Cost:     cost,
			}

			// Fire-and-forget: async cloud logging and tracing per provider
			if h.Logger != nil {
				h.Logger.LogRequest(req.Prompt, prov.Name(), resp.InputTokens, resp.OutputTokens, cost, dur.Milliseconds())
			}
			if h.Tracer != nil {
				h.Tracer.TraceGeneration(req.Prompt, resp.Text, prov.Name(), prov.Name(), resp.InputTokens, resp.OutputTokens, cost, dur.Milliseconds())
			}
		}(i, p)
	}

	// Wait for all providers to return (or hit the timeout)
	wg.Wait()

	jsonResponse, _ := json.Marshal(CompareResponse{Results: results})

	// Fire-and-forget: append to local log file
	go func(payload []byte) {
		f, err := os.OpenFile("compare_logs.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			payload = append(payload, '\n')
			f.Write(payload)
		} else {
			slog.Warn("Failed to write to local compare log", "error", err)
		}
	}(jsonResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
