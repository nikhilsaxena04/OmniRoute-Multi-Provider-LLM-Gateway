package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/nikhilsaxena04/omni-router/config"
	"github.com/nikhilsaxena04/omni-router/handlers"
	"github.com/nikhilsaxena04/omni-router/provider"
	"github.com/nikhilsaxena04/omni-router/router"
)

func main() {
	// 1. Setup Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Load Configuration
	cfgPath := "config/providers.yaml"
	routingPath := "config/routing.yaml"
	cfg, err := config.Load(cfgPath, routingPath)
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
	rtr := router.NewRouter(&cfg.Routing, allProviders)

	chatHandler := &handlers.ChatHandler{Router: rtr}
	mux.Handle("/v1/chat/completions", chatHandler)

	// 4. Start Server
	addr := ":" + cfg.Port
	slog.Info("Starting OmniRoute Gateway", "port", cfg.Port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("Server crashed", "error", err)
		os.Exit(1)
	}
}
