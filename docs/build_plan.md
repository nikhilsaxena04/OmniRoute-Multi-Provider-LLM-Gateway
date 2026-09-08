# OmniRoute — Build Plan

What got built, what got cut, and why.

---

## Scope

The original spec had Qdrant, RAG with hybrid search, gRPC, an embedded web UI, and self-hosted LangFuse with its own Postgres. All of that got cut.

What stayed:

- **Go gateway** — routing, circuit breakers, rate limiting, concurrent fan-out across providers
- **Python eval script** — runs a fixed question set through the gateway, produces cost/latency/quality numbers
- **Prometheus + Grafana** for system metrics, **LangFuse Cloud** for per-request LLM traces
- **Supabase** for async request logging (fire-and-forget, never blocks the response)

3 Docker containers. Under 1 GB RAM.

---

## What Got Cut

| Feature | Why it's not here |
|---|---|
| Qdrant + hybrid search + RRF | Retrieval quality math adds complexity that doesn't serve the gateway's actual job (routing, cost, reliability). Would add it as a separate project. |
| gRPC RAG microservice | Second service + protobufs + gRPC wiring for something that isn't the core of this project. |
| Embedded web UI | Terminal + curl + a live Grafana dashboard is the demo. A hand-rolled frontend adds nothing here. |
| Self-hosted LangFuse | LangFuse Cloud free tier gives the same traces. Saves 2 containers and a bunch of docker-networking busywork. |
| Semantic caching | Real feature (embedding + vector similarity + cache store), but marginal value for a demo. Next thing to add would be: embed the query, check Redis, return cached response above a similarity threshold. |
| Kubernetes | Docker Compose is fine for 3 containers. The gateway already exposes `/healthz` so moving to k8s is a small step, not a redesign. |

---

## Architecture

```
                    Client / curl
                         │
                         ▼
              ┌─────────────────────┐
              │   Go Gateway         │
              │   (single binary)    │
              │                      │
              │  /v1/chat/completions│  ← routes to 1 provider, streams via SSE
              │  /v1/compare         │  ← fans out to all providers concurrently
              │                      │
              │  Middleware:         │
              │  • API key auth      │
              │  • Rate limiter      │  (token bucket, per provider)
              │  • Circuit breaker   │  (per provider, sliding window)
              │  • Metrics exporter  │  (Prometheus)
              │  • LangFuse tracer   │  (async, HTTP)
              │  • Supabase logger   │  (async, fire-and-forget)
              └──────────┬───────────┘
                         │
         ┌───────────────┼───────────────┬───────────────┐
         ▼               ▼               ▼               ▼
      OpenAI          Claude          Gemini         DeepSeek
      (+ any OpenAI-compat: Groq, Cerebras, etc.)

Local: gateway, prometheus, grafana (Docker Compose)
Cloud: LangFuse, Supabase, LLM provider APIs
```

---

## Component Breakdown

Instead of a monolithic service, the gateway is broken down into modular, testable layers:

- **Core Gateway & Routing**
  - **Provider Adapters:** A common `Provider` interface (`Complete()`, `Stream()`) implemented for OpenAI, Claude, Gemini, and DeepSeek. DeepSeek reuses the OpenAI client since it speaks the same wire format.
  - **Router:** Config-driven via `routing.yaml`. Walks a priority list, skips providers with open circuits or pricing above a threshold, and falls back to the cheapest available.
  - **Compare Fan-Out:** `/v1/compare` hits all providers concurrently using goroutines + `sync.WaitGroup` with `context.WithTimeout`. A failure from one provider returns a clean error field in the JSON, rather than hanging or killing the whole response.

- **Resilience Layer (Zero Dependencies)**
  - **Circuit Breaker:** A Closed → Open → HalfOpen → Closed state machine using a sliding window. Hand-rolled to avoid pulling in heavy third-party libraries for a simple systems problem.
  - **Rate Limiter:** Token bucket implementation (capacity + refill-per-second) with mutex-guarded math.

- **Observability Stack**
  - **System Metrics:** Prometheus counters, histograms, and gauges. Exposes `/metrics` for Grafana scraping.
  - **LLM Traces:** LangFuse Cloud. One trace per request, one span per provider call via async HTTP ingestion.
  - **Audit Logging:** Supabase PostgREST HTTP POST in a fire-and-forget goroutine. Ensures zero impact on client response latency.

- **Security & Polish**
  - Bearer token auth middleware.
  - `log/slog` for structured JSON logging.
  - Graceful shutdown via `signal.NotifyContext`.
  - Config-driven `pricing.yaml` (prices change too fast to hardcode).

---

## Eval Methodology

The evaluation engine scores LLM responses by checking what fraction of expected keywords appear in the response, case-insensitive. The question set has known-correct factual answers.

```json
{
  "question": "What HTTP status code means 'Too Many Requests'?",
  "expected_keywords": ["429"]
}
```

It's simple, but every number it produces is reproducible and comes from actually running the script — not from a made-up table.

---

## Future Additions (Not Started)

- Semantic caching (embedding similarity + Redis)
- LLM-as-judge scoring mode (MT-Bench style, with rubric prompt)
- CI/CD pipeline (GitHub Actions → GHCR)
- Live deployment on Fly.io or Render free tier
