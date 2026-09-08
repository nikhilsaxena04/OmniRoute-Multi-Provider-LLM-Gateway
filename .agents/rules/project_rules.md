# OmniRoute — Project Map & Rules

> This file is auto-loaded by every agent in this workspace. It is the single source of truth.

---

## 🗺️ Read Order (start here)

| Priority | File | Purpose |
|----------|------|---------|
| 1st | **This file** (`.agents/rules/project_rules.md`) | Project map, standards, constraints |
| 2nd | `CLAUDE.md` | Same standards (for Claude Code CLI compatibility) |
| 3rd | `learning_log.md` | Phase history, concepts learned, DSA mappings |
| 4th | `CHANGELOG.md` | What changed and when |
| 5th | `.env.example` | All required environment variables |

---

## 📁 Folder Structure

```
OmniRoute/
├── .agents/                  # Agent definitions (this system)
│   ├── architect.md          # @architect — system design, API contracts
│   ├── developer.md          # @developer — Go implementation
│   ├── intern.md             # @intern — docs (CHANGELOG, README)
│   ├── reviewer.md           # @reviewer — code review + learning log
│   └── rules/
│       └── project_rules.md  # ← YOU ARE HERE
│
├── gateway/                  # Go source code (core of the project)
│   ├── main.go               # Entrypoint
│   ├── config/               # YAML loaders (providers, routing, pricing)
│   ├── provider/             # Provider interface + adapters (openai, claude, gemini, deepseek)
│   ├── resilience/           # Token bucket, circuit breaker (hand-rolled)
│   ├── router/               # Routing engine (priority + cost-aware)
│   ├── handlers/             # HTTP handlers (chat, compare, healthz)
│   └── middleware/           # Auth, metrics, tracer, logger
│
├── eval/                     # Python evaluation engine (async benchmarks)
├── prometheus/               # Prometheus scrape config
├── grafana/                  # Pre-built Grafana dashboard JSON
├── docker-compose.yml        # 3 containers: Gateway, Prometheus, Grafana
│
├── CLAUDE.md                 # Project rules (Claude Code CLI)
├── CHANGELOG.md              # Keep a Changelog format
├── README.md                 # Project overview + setup instructions
├── learning_log.md           # Phase-by-phase knowledge base / interview cheat sheet
├── .env.example              # Template for required env vars
└── .gitignore
```

---

## 📐 Key Files Index

| File | Owner | What it contains |
|------|-------|------------------|
| `gateway/**/*.go` | `@developer` | All Go implementation code |
| `CHANGELOG.md` | `@intern` | Versioned change history |
| `README.md` | `@intern` | Project overview, setup, features |
| `learning_log.md` | `@reviewer` | Phase concepts, DSA mappings, interview prep |
| `.env.example` | `@intern` | Environment variable template |
| `CLAUDE.md` | `@architect` (approval needed) | Canonical project rules |
| `docker-compose.yml` | `@architect` | Infrastructure definition |
| `eval/*.py` | separate concern | Python benchmarks (not managed by these agents) |

---

## ⚙️ Language & Runtime

- **Primary:** Go (Golang) 1.22+
- **Secondary:** Python 3.11+ (Evaluation Engine only)
- **Architecture:** 3 local containers (Gateway, Prometheus, Grafana) + Cloud APIs (LangFuse, Supabase, LLM providers)

---

## 📏 Go Coding Standards

- **Logging:** Use `log/slog` for structured JSON logging. Never use `fmt.Println` for system logs.
- **Error Handling:** Always wrap errors with context: `fmt.Errorf("failed to call %s: %w", provider, err)`.
- **Concurrency:** Use `sync.Mutex` for shared state (circuit breaker, rate limiter). Use `sync.WaitGroup` for fan-out patterns.
- **Context Propagation:** Every function that touches I/O must accept `context.Context` as its first parameter. Check `ctx.Done()` in loops and streaming.
- **Dependencies:** Build core algorithms from scratch (Token Bucket, Circuit Breaker, LRU Cache). No `gobreaker`, no `x/time/rate` for these.
- **Config:** All config must be loaded from YAML files or environment variables. No hardcoded API keys, pricing, or provider URLs in Go code.

## 🐍 Python Coding Standards

- **Async:** Use `asyncio` + `aiohttp` for concurrent benchmark requests. Use `asyncio.Semaphore` to cap concurrency.
- **Scoring:** Quality scoring must be deterministic (keyword recall), not LLM-judged.

---

## 🚫 What We Are NOT Building

- No RAG pipeline, no Qdrant, no vector search.
- No gRPC, no Protobuf. All APIs are REST (JSON over HTTP).
- No embedded web UI or frontend.
- No local LLMs (Ollama). All models are accessed via cloud APIs.
- No SQLite or local Postgres. Logs go to Supabase Cloud.

---

## 🔄 Phase Workflow

```
@architect  →  design & approve    →  "✅ APPROVED ARCHITECTURE"
@developer  →  implement & compile →  "go build ✓"
@reviewer   →  audit + learning log →  "🚀 SHIP / 🔧 FIX / 🚫 BLOCK"
@intern     →  CHANGELOG + README   →  "📋 Phase complete"
                                         ↓
                                    Next phase → @architect
```
