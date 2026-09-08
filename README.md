# Omni-Router

A production-grade, highly observable LLM Gateway written in Go.

## Features
- **Concurrent Fan-Out:** Send requests to OpenAI, Claude, Gemini, and DeepSeek simultaneously.
- **Resilience:** Built-in Token Bucket Rate Limiting and Sliding-Window Circuit Breakers.
- **Cost-Aware Routing:** YAML-driven routing policies to failover to cheaper models automatically.
- **Dual-Layer Observability:** Prometheus/Grafana for system health, Langfuse for LLM traces.
- **Evaluation Engine:** Python benchmark script to calculate deterministic quality, latency, and cost ROI.

*(Architecture diagram and setup instructions will be added in Phase 9)*
