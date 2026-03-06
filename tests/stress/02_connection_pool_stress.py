#!/usr/bin/env python3
"""
Stress Test 02: Connection Pool Saturation
Tests database connection pool under extreme concurrent load.

Pool config: MaxOpenConns=25, MaxIdleConns=10

Covers:
  - Incremental concurrency ramp (1 -> 50 -> 100 concurrent clients)
  - Sustained high concurrency for extended periods
  - Mixed read/write concurrent load
  - Burst traffic after idle period
"""

import sys, os, time, statistics, random
from concurrent.futures import ThreadPoolExecutor, as_completed
sys.path.insert(0, os.path.dirname(__file__))
from helpers import *


def test_concurrency_ramp(collection_id):
    """Incrementally ramp concurrency to find the breaking point."""
    subheader("Concurrency Ramp (1 -> 100 concurrent clients)")
    print("  Pool: MaxOpenConns=25, MaxIdleConns=10\n")

    request_count = 50  # Requests per concurrency level

    for concurrency in [1, 5, 10, 15, 20, 25, 30, 40, 50, 75, 100]:
        latencies = []
        errors = 0

        def do_insert(_):
            start = time.perf_counter()
            resp = requests.post(f"{BASE_URL}/documents", json={
                "collection_id": collection_id,
                "vector": generate_vector(),
                "content": f"Concurrency test",
                "metadata": generate_metadata()
            })
            elapsed = (time.perf_counter() - start) * 1000
            return elapsed, resp.status_code == 200

        wall_start = time.perf_counter()
        with ThreadPoolExecutor(max_workers=concurrency) as pool:
            futures = [pool.submit(do_insert, i) for i in range(request_count)]
            for f in as_completed(futures):
                elapsed, ok = f.result()
                if ok:
                    latencies.append(elapsed)
                else:
                    errors += 1
        wall_time = time.perf_counter() - wall_start

        if latencies:
            print(f"  C={concurrency:>3}: "
                  f"Wall={wall_time:.2f}s | "
                  f"Throughput={request_count / wall_time:.1f} ops/sec | "
                  f"Avg={statistics.mean(latencies):.0f}ms | "
                  f"p95={percentile(latencies, 95):.0f}ms | "
                  f"p99={percentile(latencies, 99):.0f}ms | "
                  f"Errors={errors}")
        else:
            print(f"  C={concurrency:>3}: ALL REQUESTS FAILED ({errors} errors)")


def test_sustained_load(collection_id):
    """Sustain high concurrency for 30 seconds."""
    subheader("Sustained High Concurrency (30 workers, 30 seconds)")

    concurrency = 30
    duration_sec = 30
    latencies = []
    errors = 0
    completed = 0

    def do_insert():
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": collection_id,
            "vector": generate_vector(),
            "content": "Sustained load test",
            "metadata": generate_metadata()
        })
        elapsed = (time.perf_counter() - start) * 1000
        return elapsed, resp.status_code == 200

    end_time = time.time() + duration_sec
    wall_start = time.perf_counter()

    with ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = set()

        # Keep submitting work until time runs out
        while time.time() < end_time:
            # Top up to concurrency level
            while len(futures) < concurrency and time.time() < end_time:
                futures.add(pool.submit(do_insert))

            # Collect completed ones
            done = {f for f in futures if f.done()}
            for f in done:
                elapsed, ok = f.result()
                if ok:
                    latencies.append(elapsed)
                else:
                    errors += 1
                completed += 1
            futures -= done
            time.sleep(0.01)

        # Drain remaining
        for f in as_completed(futures):
            elapsed, ok = f.result()
            if ok:
                latencies.append(elapsed)
            else:
                errors += 1
            completed += 1

    wall_time = time.perf_counter() - wall_start

    print(f"  Duration: {wall_time:.1f}s")
    print(f"  Total requests: {completed}")
    print(f"  Throughput: {completed / wall_time:.1f} ops/sec")
    if latencies:
        print(f"  Avg: {statistics.mean(latencies):.1f}ms | "
              f"p50: {statistics.median(latencies):.1f}ms | "
              f"p95: {percentile(latencies, 95):.1f}ms | "
              f"p99: {percentile(latencies, 99):.1f}ms")
    print(f"  Errors: {errors} ({errors / completed * 100:.1f}%)" if completed else "")


def test_mixed_concurrent_load(collection_id, doc_ids):
    """Mixed read/write load -- 50% inserts, 25% searches, 25% gets."""
    subheader("Mixed Concurrent Read/Write Load (40 workers)")

    concurrency = 40
    request_count = 200
    insert_lats, search_lats, get_lats = [], [], []
    errors = 0

    def do_work(i):
        op = i % 4  # 0,1 = insert, 2 = search, 3 = get
        start = time.perf_counter()
        if op <= 1:
            resp = requests.post(f"{BASE_URL}/documents", json={
                "collection_id": collection_id,
                "vector": generate_vector(),
                "content": f"Mixed load doc {i}",
                "metadata": generate_metadata()
            })
            elapsed = (time.perf_counter() - start) * 1000
            return "insert", elapsed, resp.status_code == 200
        elif op == 2:
            resp = requests.post(f"{BASE_URL}/documents/search", json={
                "collection_id": collection_id,
                "query_vector": generate_vector(),
                "top_k": 10
            })
            elapsed = (time.perf_counter() - start) * 1000
            return "search", elapsed, resp.status_code == 200
        else:
            doc_id = random.choice(doc_ids) if doc_ids else "nonexistent"
            resp = requests.get(f"{BASE_URL}/documents/{doc_id}")
            elapsed = (time.perf_counter() - start) * 1000
            return "get", elapsed, resp.status_code == 200

    wall_start = time.perf_counter()
    with ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = [pool.submit(do_work, i) for i in range(request_count)]
        for f in as_completed(futures):
            op, elapsed, ok = f.result()
            if ok:
                if op == "insert":
                    insert_lats.append(elapsed)
                elif op == "search":
                    search_lats.append(elapsed)
                else:
                    get_lats.append(elapsed)
            else:
                errors += 1

    wall_time = time.perf_counter() - wall_start

    print(f"  Wall time: {wall_time:.2f}s | Total throughput: {request_count / wall_time:.1f} ops/sec")
    print_stats("Inserts (50%)", insert_lats)
    print_stats("Searches (25%)", search_lats)
    print_stats("Gets (25%)", get_lats)
    print(f"  Total errors: {errors}")


def test_burst_after_idle(collection_id):
    """Idle for 15 seconds, then burst 100 requests at max concurrency."""
    subheader("Burst After Idle (15s idle -> 100 concurrent requests)")

    print("  Idling for 15 seconds (connections should go idle)...")
    time.sleep(15)

    latencies = []
    errors = 0

    def do_insert(_):
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": collection_id,
            "vector": generate_vector(),
            "content": "Burst test",
            "metadata": generate_metadata()
        })
        elapsed = (time.perf_counter() - start) * 1000
        return elapsed, resp.status_code == 200

    wall_start = time.perf_counter()
    with ThreadPoolExecutor(max_workers=100) as pool:
        futures = [pool.submit(do_insert, i) for i in range(100)]
        for f in as_completed(futures):
            elapsed, ok = f.result()
            if ok:
                latencies.append(elapsed)
            else:
                errors += 1
    wall_time = time.perf_counter() - wall_start

    print(f"  Burst wall time: {wall_time:.2f}s")
    print(f"  Throughput: {100 / wall_time:.1f} ops/sec")
    if latencies:
        print(f"  First request: {latencies[0]:.1f}ms (cold connection)")
        print(f"  Avg: {statistics.mean(latencies):.1f}ms | "
              f"p95: {percentile(latencies, 95):.1f}ms | "
              f"p99: {percentile(latencies, 99):.1f}ms")
    print(f"  Errors: {errors}")


def main():
    header("STRESS TEST 02: CONNECTION POOL SATURATION")
    print(f"  Server: {BASE_URL}")
    print(f"  Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")

    if not check_server():
        print("  Server unreachable. Exiting.")
        sys.exit(1)

    coll_name = f"pool_stress_{int(time.time())}"
    coll_id = setup_collection(coll_name)
    if not coll_id:
        return

    print(f"\n  Seeding 100 documents...")
    doc_ids = seed_documents(coll_id, 100)
    print(f"  Seeded {len(doc_ids)} documents.\n")

    test_concurrency_ramp(coll_id)
    test_sustained_load(coll_id)
    test_mixed_concurrent_load(coll_id, doc_ids)
    test_burst_after_idle(coll_id)

    cleanup_collection(coll_id)
    header("TEST 02 COMPLETE")


if __name__ == "__main__":
    main()
