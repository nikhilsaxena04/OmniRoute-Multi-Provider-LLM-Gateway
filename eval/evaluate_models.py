import json, statistics, requests
import os

GATEWAY_URL = "http://localhost:8787/v1/compare"
GATEWAY_KEY = os.getenv("GATEWAY_API_KEY", "my-super-secret-key")

def load_dataset(path="eval_dataset.json"):
    # Ensure relative paths work from project root
    if not os.path.exists(path) and os.path.exists(os.path.join("eval", path)):
        path = os.path.join("eval", path)
        
    with open(path) as f:
        return json.load(f)

def score_answer(response_text: str, expected_keywords: list[str]) -> float:
    """Fraction of expected keywords present, case-insensitive. Simple, but defensible."""
    if not response_text:
        return 0.0
    text = response_text.lower()
    hits = sum(1 for kw in expected_keywords if kw.lower() in text)
    return hits / len(expected_keywords) if expected_keywords else 0.0

def run_eval():
    dataset = load_dataset()
    per_provider = {}

    headers = {
        "Authorization": f"Bearer {GATEWAY_KEY}",
        "Content-Type": "application/json"
    }

    print(f"Running evaluation on {len(dataset)} prompts via {GATEWAY_URL}...")
    
    for item in dataset:
        try:
            resp = requests.post(GATEWAY_URL, json={"query": item["question"]}, headers=headers, timeout=30)
            resp.raise_for_status()
            
            for r in resp.json()["results"]:
                provider = r.get("provider", "unknown")
                bucket = per_provider.setdefault(provider, {"scores": [], "latencies": [], "costs": []})
                
                if r.get("error"):
                    continue
                    
                response_text = r.get("response", "")
                bucket["scores"].append(score_answer(response_text, item.get("expected_keywords", [])))
                bucket["latencies"].append(r.get("latency_ms", 0))
                bucket["costs"].append(r.get("cost_usd", 0.0))
        except Exception as e:
            print(f"Request failed: {e}")

    print("\n" + "="*70)
    print(f"{'Model':<20}{'Quality %':<12}{'Avg Cost $':<14}{'P95 Latency'}")
    print("-" * 70)
    
    for provider, m in per_provider.items():
        quality = statistics.mean(m["scores"]) * 100 if m["scores"] else 0
        avg_cost = statistics.mean(m["costs"]) if m["costs"] else 0
        p95 = sorted(m["latencies"])[max(0, int(len(m["latencies"]) * 0.95) - 1)] if m["latencies"] else 0
        print(f"{provider:<20}{quality:<12.1f}{avg_cost:<14.6f}{p95}ms")
    print("=" * 70 + "\n")

if __name__ == "__main__":
    run_eval()
