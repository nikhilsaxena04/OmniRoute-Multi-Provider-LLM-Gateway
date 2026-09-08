package router

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikhilsaxena04/omni-router/config"
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
		
		if !entry.Circuit.Allow() {
			slog.Warn("Circuit breaker open/half-open probe denied, skipping provider", "provider", name)
			continue
		}

		slog.Info("Routing request to provider", "provider", name)
		resp, err := entry.Provider.Complete(ctx, prompt)
		if err != nil {
			slog.Error("Provider failed, recording circuit breaker failure", "provider", name, "error", err)
			entry.Circuit.RecordFailure()
			continue // try next provider in priority list
		}

		entry.Circuit.RecordSuccess()
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
		
		if !entry.Circuit.Allow() {
			slog.Warn("Circuit breaker open/half-open probe denied, skipping provider", "provider", name)
			continue
		}

		slog.Info("Routing stream request to provider", "provider", name)
		
		// Note: in a real system, if it fails mid-stream, failover is complex because 
		// chunks were already sent to the client. Here we just return the error.
		err := entry.Provider.Stream(ctx, prompt, out)
		if err != nil {
			slog.Error("Provider stream failed, recording circuit breaker failure", "provider", name, "error", err)
			entry.Circuit.RecordFailure()
			continue
		}

		entry.Circuit.RecordSuccess()
		return nil
	}

	close(out)
	return fmt.Errorf("all providers failed or unavailable")
}
