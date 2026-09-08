# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]
- Initialized project directories and learning log.
- Added Phase 0 foundations: `config` loader using `yaml` and `godotenv`, `slog` structured logging, and `/healthz` endpoint.
- Added Phase 1: Standard `Provider` interface and four SDK-less HTTP adapters (OpenAI, Claude, Gemini, DeepSeek).
- Added Phase 2: Created `/v1/chat/completions` endpoint supporting both non-streaming and Server-Sent Events (SSE) streaming modes.
- Added Phase 3: Hand-rolled DSA implementations of Token Bucket rate limiter and Sliding-Window Circuit Breaker with complete test coverage.
