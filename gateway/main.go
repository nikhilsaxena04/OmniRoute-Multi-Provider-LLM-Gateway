package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nikhilsaxena04/omni-router/config"
	"github.com/nikhilsaxena04/omni-router/handlers"
	"github.com/nikhilsaxena04/omni-router/middleware"
	"github.com/nikhilsaxena04/omni-router/provider"
	"github.com/nikhilsaxena04/omni-router/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 1. Setup Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Load Configuration
	cfg, err := config.Load("config/providers.yaml", "config/routing.yaml", "config/pricing.yaml")
	if err != nil {
		// Log fatal configuration error
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded successfully", "providers_count", len(cfg.Providers))

	// 3. Setup Routes and Router
	mux := http.NewServeMux()

	// Healthz endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	allProviders := []provider.Provider{
		provider.NewOpenAIProvider(cfg.Providers["openai"]),
		provider.NewClaudeProvider(cfg.Providers["claude"]),
		provider.NewGeminiProvider(cfg.Providers["gemini"]),
		provider.NewDeepSeekProvider(cfg.Providers["deepseek"]),
	}
	rtr := router.NewRouter(&cfg.Routing, &cfg.Pricing, allProviders)

	chatHandler := &handlers.ChatHandler{Router: rtr, Pricing: &cfg.Pricing}
	compareHandler := &handlers.CompareHandler{Providers: allProviders, Pricing: &cfg.Pricing}
	gatewayKey := os.Getenv("GATEWAY_API_KEY")

	// Protected endpoints
	mux.Handle("/v1/chat/completions", middleware.Auth(gatewayKey, chatHandler))
	mux.Handle("/v1/compare", middleware.Auth(gatewayKey, compareHandler))

	// Public endpoints
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	})

	// 4. Start Server with Graceful Shutdown
	addr := ":" + cfg.Port
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Starting OmniRoute Gateway", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited properly")
}
