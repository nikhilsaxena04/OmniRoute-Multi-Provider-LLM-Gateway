package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nikhilsaxena04/omni-router/config"
	"github.com/nikhilsaxena04/omni-router/metrics"
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

	// 15 seconds strict timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
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
		}(i, p)
	}

	// Wait for all providers to return (or hit the timeout)
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CompareResponse{Results: results})
}
