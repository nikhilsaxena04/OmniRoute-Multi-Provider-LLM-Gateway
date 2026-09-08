# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]
- Initialized project directories and learning log.
- Added Phase 0 foundations: `config` loader using `yaml` and `godotenv`, `slog` structured logging, and `/healthz` endpoint.
- Added Phase 1: Standard `Provider` interface and four SDK-less HTTP adapters (OpenAI, Claude, Gemini, DeepSeek).
- Added Phase 2: Created `/v1/chat/completions` endpoint supporting both non-streaming and Server-Sent Events (SSE) streaming modes.
- Added Phase 3: Hand-rolled DSA implementations of Token Bucket rate limiter and Sliding-Window Circuit Breaker with complete test coverage.
- Added Phase 4: Implemented a configuration-driven `Router` with automatic failover between models utilizing the Circuit Breaker pattern.
- Added Phase 5: Built an asynchronous Python evaluation script using `aiohttp` to load-test the Go gateway and calculate TPS metrics.
- Added Phase 6: Created the `/v1/compare` endpoint featuring concurrent fan-out to all providers using goroutines, a strict timeout context, and graceful partial-failure handling.
- Added Phase 7: Integrated `prometheus/client_golang` for metrics (latency, requests, circuit state) and configured a `docker-compose.yml` stack with Prometheus and a pre-built Grafana dashboard.
- Added Phase 8: Implemented a token-based Cost Calculator driven by `pricing.yaml` to dynamically inject costs into API responses.
- Added Phase 9: Added API security with Bearer Token `AuthMiddleware`, an active `/healthz` endpoint, and graceful server shutdown using `signal.NotifyContext`.
- Added Phase 10: Implemented zero-dependency async observability with Supabase Cloud Logging (PostgREST) and LangFuse Cloud Tracing (Ingestion API).
