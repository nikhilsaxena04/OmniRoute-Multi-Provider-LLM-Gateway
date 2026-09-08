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
	providers map[string]*ProviderEntry
}

func NewRouter(cfg *config.RoutingConfig, allProviders []provider.Provider) *Router {
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
		providers: pm,
	}
}

func (r *Router) ExecuteComplete(ctx context.Context, prompt string) (*provider.Response, error) {
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

		slog.Info("Routing request to provider", "provider", name)
		
		start := time.Now()
		resp, err := entry.Provider.Complete(ctx, prompt)
		duration := time.Since(start).Seconds()
		
		if err != nil {
			metrics.RequestsTotal.WithLabelValues("/v1/chat/completions", name, "500").Inc()
			metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions", name).Observe(duration)
			
			slog.Error("Provider failed, recording circuit breaker failure", "provider", name, "error", err.Error())
			entry.Circuit.RecordFailure()
			continue
		}

		metrics.RequestsTotal.WithLabelValues("/v1/chat/completions", name, "200").Inc()
		metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions", name).Observe(duration)

		entry.Circuit.RecordSuccess()
		resp.Provider = name
		return resp, nil
	}

	return nil, fmt.Errorf("all providers failed or unavailable")
}

func (r *Router) ExecuteStream(ctx context.Context, prompt string, out chan<- provider.Chunk) error {
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

		slog.Info("Routing stream request to provider", "provider", name)
		
		start := time.Now()
		err := entry.Provider.Stream(ctx, prompt, out)
		duration := time.Since(start).Seconds()

		if err != nil {
			metrics.RequestsTotal.WithLabelValues("/v1/chat/completions (stream)", name, "500").Inc()
			metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions (stream)", name).Observe(duration)

			slog.Error("Provider stream failed, recording circuit breaker failure", "provider", name, "error", err)
			entry.Circuit.RecordFailure()
			continue
		}

		metrics.RequestsTotal.WithLabelValues("/v1/chat/completions (stream)", name, "200").Inc()
		metrics.LatencyHistogram.WithLabelValues("/v1/chat/completions (stream)", name).Observe(duration)

		entry.Circuit.RecordSuccess()
		return nil
	}

	close(out)
	return fmt.Errorf("all providers failed or unavailable")
}
