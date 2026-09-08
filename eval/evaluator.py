import asyncio
import aiohttp
import pandas as pd
import time
import argparse
from tqdm.asyncio import tqdm

async def make_request(session, url, prompt):
    start_time = time.time()
    payload = {
        "prompt": prompt,
        "stream": False
    }
    try:
        async with session.post(url, json=payload, timeout=30) as response:
            status = response.status
            if status == 200:
                data = await response.json()
                latency = time.time() - start_time
                return {
                    "success": True,
                    "status": status,
                    "latency": latency,
                    "input_tokens": data.get("input_tokens", 0),
                    "output_tokens": data.get("output_tokens", 0)
                }
            else:
                latency = time.time() - start_time
                text = await response.text()
                return {
                    "success": False,
                    "status": status,
                    "latency": latency,
                    "error": text.strip()
                }
    except Exception as e:
        return {
            "success": False,
            "status": 0,
            "latency": time.time() - start_time,
            "error": str(e)
        }

async def main():
    parser = argparse.ArgumentParser(description="Evaluate OmniRoute Gateway")
    parser.add_argument("--url", default="http://localhost:8787/v1/chat/completions", help="Gateway URL")
    parser.add_argument("--concurrency", type=int, default=5, help="Number of concurrent requests")
    args = parser.parse_args()

    print(f"Loading dataset from eval/dataset.csv...")
    try:
        df = pd.read_csv("eval/dataset.csv")
    except Exception as e:
        print(f"Failed to load dataset: {e}")
        return

    print(f"Loaded {len(df)} prompts. Blasting gateway at {args.url} with concurrency {args.concurrency}...")

    connector = aiohttp.TCPConnector(limit=args.concurrency)
    async with aiohttp.ClientSession(connector=connector) as session:
        tasks = []
        for _, row in df.iterrows():
            tasks.append(make_request(session, args.url, row["prompt"]))

        start_gather = time.time()
        results = await tqdm.gather(*tasks)
        end_gather = time.time()

    # Process results
    total = len(results)
    successes = sum(1 for r in results if r["success"])
    failures = total - successes

    successful_results = [r for r in results if r["success"]]
    avg_latency = sum(r["latency"] for r in successful_results) / len(successful_results) if successful_results else 0
    total_output_tokens = sum(r.get("output_tokens", 0) for r in successful_results)
    
    wall_clock = end_gather - start_gather
    tps = total_output_tokens / wall_clock if wall_clock > 0 else 0

    print("\n--- Evaluation Results ---")
    print(f"Total Requests: {total}")
    print(f"Success Rate:   {successes}/{total} ({(successes/total)*100:.1f}%)")
    print(f"Avg Latency:    {avg_latency:.2f}s")
    print(f"Tokens/Sec TPS: {tps:.2f} tokens/s (based on {total_output_tokens} output tokens)")
    
    if failures > 0:
        print(f"\nSample Error:")
        failed_result = next(r for r in results if not r["success"])
        print(f"Status {failed_result['status']}: {failed_result['error']}")

if __name__ == "__main__":
    asyncio.run(main())
