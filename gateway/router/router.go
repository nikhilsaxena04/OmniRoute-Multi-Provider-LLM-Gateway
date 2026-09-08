package router

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikhilsaxena04/omni-router/config"
	"github.com/nikhilsaxena04/omni-router/metrics"
	"github.com/nikhilsaxena04/omni-router/provider"
	"github.com/nikhilsaxena04/omni-router/resilience"
)

type ProviderEntry struct {
	Provider provider.Provider
	Circuit  *resilience.CircuitBreaker
}

type Router struct {
	cfg       *config.RoutingConfig
	pricing   *config.PricingConfig
	providers map[string]*ProviderEntry
}

func NewRouter(cfg *config.RoutingConfig, pricing *config.PricingConfig, allProviders []provider.Provider) *Router {
	pm := make(map[string]*ProviderEntry)
	for _, p := range allProviders {
		pm[p.Name()] = &ProviderEntry{
			Provider: p,
			// 3 failures, 60s sliding window, 30s open duration
			Circuit:  resilience.NewCircuitBreaker(3, time.Second*60, time.Second*30),
		}
	}
	return &Router{
		cfg:       cfg,
		pricing:   pricing,
		providers: pm,
	}
}

func (r *Router) getCheapestAvailable() (string, *ProviderEntry) {
	var cheapestName string
	var cheapestEntry *ProviderEntry
	var lowestPrice float64 = -1

	// Fallback logic: find cheapest provider that is under threshold and available
	for _, name := range r.cfg.Priorities {
		entry, ok := r.providers[name]
		if !ok || !entry.Circuit.Allow() {
			continue
		}
		
		price := 0.0
		if r.pricing != nil {
			if p, ok := r.pricing.Pricing[name]; ok {
				price = p.InputPer1M
			}
		}

		if lowestPrice == -1 || price < lowestPrice {
			lowestPrice = price
			cheapestName = name
			cheapestEntry = entry
		}
	}
	return cheapestName, cheapestEntry
}

func (r *Router) ExecuteComplete(ctx context.Context, prompt string) (*provider.Response, error) {
	var selectedName string
	var selectedEntry *ProviderEntry

	for _, name := range r.cfg.Priorities {
		entry, ok := r.providers[name]
		if !ok {
			continue
		}
		
		metrics.CircuitState.WithLabelValues(name).Set(float64(entry.Circuit.State()))

		if !entry.Circuit.Allow() {
			slog.Warn("Circuit breaker open/half-open probe denied, skipping provider", "provider", name)
			continue
		}

		// Check cost threshold
		if r.cfg.CostThreshold > 0 && r.pricing != nil {
			if p, ok := r.pricing.Pricing[name]; ok {
				if p.InputPer1M > r.cfg.CostThreshold {
					slog.Warn("Provider exceeds cost threshold, skipping for cheaper alternative", "provider", name, "price", p.InputPer1M, "threshold", r.cfg.CostThreshold)
					continue
				}
			}
		}

		selectedName = name
		selectedEntry = entry
		break
	}

	// If no provider under threshold was found, just find the absolute cheapest available
	if selectedEntry == nil && r.cfg.CostThreshold > 0 {
		slog.Warn("No available providers under cost threshold, falling back to absolute cheapest available")
		selectedName, selectedEntry = r.getCheapestAvailable()
	}

	if selectedEntry != nil {
		slog.Info("Routing request to provider", "provider", selectedName)
		
		start := time.Now()
		resp, err := selectedEntry.Provider.Complete(ctx, prompt)
		duration := time.Since(start).Seconds()
		
		if err != nil {
			metrics.RequestsTotal.WithLabelValues("/v1/chat/completions", selectedName, "500").Inc()
			metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions", selectedName).Observe(duration)
			
			slog.Error("Provider failed, recording circuit breaker failure", "provider", selectedName, "error", err.Error())
			selectedEntry.Circuit.RecordFailure()
			return nil, err // Let client retry, we tried the best one
		}

		metrics.RequestsTotal.WithLabelValues("/v1/chat/completions", selectedName, "200").Inc()
		metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions", selectedName).Observe(duration)

		selectedEntry.Circuit.RecordSuccess()
		resp.Provider = selectedName
		return resp, nil
	}

	return nil, fmt.Errorf("all providers failed or unavailable")
}

func (r *Router) ExecuteStream(ctx context.Context, prompt string, out chan<- provider.Chunk) error {
	var selectedName string
	var selectedEntry *ProviderEntry

	for _, name := range r.cfg.Priorities {
		entry, ok := r.providers[name]
		if !ok {
			continue
		}
		
		metrics.CircuitState.WithLabelValues(name).Set(float64(entry.Circuit.State()))

		if !entry.Circuit.Allow() {
			slog.Warn("Circuit breaker open/half-open probe denied, skipping provider", "provider", name)
			continue
		}

		// Check cost threshold
		if r.cfg.CostThreshold > 0 && r.pricing != nil {
			if p, ok := r.pricing.Pricing[name]; ok {
				if p.InputPer1M > r.cfg.CostThreshold {
					slog.Warn("Provider exceeds cost threshold, skipping for cheaper alternative", "provider", name, "price", p.InputPer1M, "threshold", r.cfg.CostThreshold)
					continue
				}
			}
		}

		selectedName = name
		selectedEntry = entry
		break
	}

	// If no provider under threshold was found, just find the absolute cheapest available
	if selectedEntry == nil && r.cfg.CostThreshold > 0 {
		slog.Warn("No available providers under cost threshold, falling back to absolute cheapest available")
		selectedName, selectedEntry = r.getCheapestAvailable()
	}

	if selectedEntry != nil {
		slog.Info("Routing stream request to provider", "provider", selectedName)
		
		start := time.Now()
		err := selectedEntry.Provider.Stream(ctx, prompt, out)
		duration := time.Since(start).Seconds()

		if err != nil {
			metrics.RequestsTotal.WithLabelValues("/v1/chat/completions (stream)", selectedName, "500").Inc()
			metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions (stream)", selectedName).Observe(duration)

			slog.Error("Provider stream failed, recording circuit breaker failure", "provider", selectedName, "error", err)
			selectedEntry.Circuit.RecordFailure()
			return err
		}

		metrics.RequestsTotal.WithLabelValues("/v1/chat/completions (stream)", selectedName, "200").Inc()
		metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions (stream)", selectedName).Observe(duration)

		selectedEntry.Circuit.RecordSuccess()
		return nil
	}

	close(out)
	return fmt.Errorf("all providers failed or unavailable")
}
