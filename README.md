# OmniRoute - Multi-Provider LLM Gateway 🚀

OmniRoute is an enterprise-grade API gateway designed to unify, route, and observe requests to multiple Large Language Model (LLM) providers like OpenAI, Anthropic (Claude), Google (Gemini), and DeepSeek.

## Features ✨

- **Unified API Surface:** Use a single API format to interact with any underlying LLM provider.
- **Intelligent Routing & Failover:** Config-driven priority routing. If the primary provider (e.g., OpenAI) goes down or rate-limits, requests automatically failover to Claude or Gemini using the **Circuit Breaker** pattern.
- **Compare Mode (Fan-Out):** A `/v1/compare` endpoint that fires concurrent requests to all configured providers, executing partial-failure handling to ensure one failing API key doesn't block the other providers.
- **Token Bucket Rate Limiting:** Built-in rate limiting to prevent abuse.
- **Cost Calculator:** Configurable `pricing.yaml` that dynamically tracks the real-time USD cost of each request based on input/output tokens.
- **Observability Stack:** Out-of-the-box Prometheus metrics (Latencies, Request Counts, Circuit States) and a pre-configured Grafana dashboard for instant visualization.
- **Security:** Built-in Bearer Auth Middleware to protect your endpoints. Graceful shutdown to prevent dropping active connections during redeploys.

## Quick Start 🛠️

### 1. Configure Environment
Copy `.env.example` to `.env` and fill in your API keys:
```bash
GATEWAY_API_KEY=your-secure-gateway-token
OPENAI_API_KEY=...
CLAUDE_API_KEY=...
GEMINI_API_KEY=...
DEEPSEEK_API_KEY=...
```

### 2. Run Locally (Docker)
The easiest way to start the Gateway + Prometheus + Grafana stack:
```bash
make docker-up
```
- Gateway: `http://localhost:8787`
- Grafana: `http://localhost:3000` (No auth required)
- Prometheus: `http://localhost:9090`

### 3. API Usage

**Standard Completion (with Failover):**
```bash
curl -X POST http://localhost:8787/v1/chat/completions \
  -H "Authorization: Bearer your-secure-gateway-token" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Hello!"}'
```

**Compare Mode (All Models Concurrently):**
```bash
curl -X POST http://localhost:8787/v1/compare \
  -H "Authorization: Bearer your-secure-gateway-token" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Explain quantum computing in one sentence."}'
```

## Architecture 🏗️
- **Golang** for high concurrency and performance.
- SDK-less HTTP implementations to avoid dependency hell.
- Pure Data Structures: Sliding-window circuit breakers and token-bucket limiters built from scratch.

## Project Structure
- `gateway/handlers`: HTTP handlers and endpoints.
- `gateway/router`: The core routing engine and failover logic.
- `gateway/resilience`: Rate limiters and circuit breakers.
- `gateway/provider`: Provider adapters.
- `gateway/config`: YAML parsing and cost calculation.
- `eval/`: Asynchronous Python script (`evaluator.py`) to run load tests against the gateway.
