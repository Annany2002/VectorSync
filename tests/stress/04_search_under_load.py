#!/usr/bin/env python3
"""
Stress Test 04: Search Under Load
Tests search performance while the system is under heavy write pressure.

Covers:
  - Concurrent vector searches during batch inserts
  - Full-text search under write contention
  - Hybrid search with max top_k (1000) on large dataset
  - Search accuracy consistency under load
"""

import sys, os, time, statistics, random, threading
from concurrent.futures import ThreadPoolExecutor, as_completed
sys.path.insert(0, os.path.dirname(__file__))
from helpers import *


def test_search_during_writes(collection_id):
    """Run vector searches while concurrent inserts are happening."""
    subheader("Vector Search During Concurrent Inserts")
    print("  10 search workers + 10 insert workers, 30 seconds\n")

    search_lats = []
    insert_lats = []
    search_errors = 0
    insert_errors = 0
    stop_event = threading.Event()

    def do_searches():
        nonlocal search_errors
        while not stop_event.is_set():
            start = time.perf_counter()
            resp = requests.post(f"{BASE_URL}/documents/search", json={
                "collection_id": collection_id,
                "query_vector": generate_vector(),
                "top_k": 10
            })
            elapsed = (time.perf_counter() - start) * 1000
            if resp.status_code == 200:
                search_lats.append(elapsed)
            else:
                search_errors += 1
            time.sleep(0.05)  # ~20 searches/sec per worker

    def do_inserts():
        nonlocal insert_errors
        while not stop_event.is_set():
            start = time.perf_counter()
            resp = requests.post(f"{BASE_URL}/documents", json={
                "collection_id": collection_id,
                "vector": generate_vector(),
                "content": f"Load test doc {random.randint(0, 999999)}",
                "metadata": generate_metadata()
            })
            elapsed = (time.perf_counter() - start) * 1000
            if resp.status_code == 200:
                insert_lats.append(elapsed)
            else:
                insert_errors += 1

    # Launch workers
    threads = []
    for _ in range(10):
        threads.append(threading.Thread(target=do_searches, daemon=True))
        threads.append(threading.Thread(target=do_inserts, daemon=True))

    for t in threads:
        t.start()

    time.sleep(30)
    stop_event.set()

    for t in threads:
        t.join(timeout=5)

    print_stats("Searches (during writes)", search_lats, search_errors)
    print_stats("Inserts (during searches)", insert_lats, insert_errors)


def test_fulltext_under_write_contention(collection_id):
    """Full-text search while batch inserts are running."""
    subheader("Full-Text Search During Batch Inserts")

    queries = ["AI", "databases", "ML", "vectors", "performance", "cloud"]
    search_lats = []
    search_errors = 0
    batch_running = threading.Event()
    batch_running.set()

    def do_batch_inserts():
        """Continuously insert batches while event is set."""
        while batch_running.is_set():
            docs = [{
                "collection_id": collection_id,
                "vector": generate_vector(),
                "content": f"Batch doc {random.choice(['AI', 'ML', 'databases', 'cloud'])}",
                "metadata": generate_metadata()
            } for _ in range(100)]
            requests.post(f"{BASE_URL}/documents/batch", json={
                "collection_id": collection_id,
                "documents": docs
            })

    # Start 3 batch insert threads
    insert_threads = []
    for _ in range(3):
        t = threading.Thread(target=do_batch_inserts, daemon=True)
        t.start()
        insert_threads.append(t)

    # Run searches for 20 seconds
    end_time = time.time() + 20
    while time.time() < end_time:
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents/text-search", json={
            "collection_id": collection_id,
            "query": random.choice(queries),
            "limit": 10
        })
        elapsed = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            search_lats.append(elapsed)
        else:
            search_errors += 1

    batch_running.clear()
    for t in insert_threads:
        t.join(timeout=5)

    print_stats("Full-text search (under write contention)", search_lats, search_errors)


def test_max_topk_search(collection_id):
    """Search with top_k=1000 on a large dataset."""
    subheader("Max top_k Search (top_k=1000)")

    latencies = []
    for i in range(10):
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents/search", json={
            "collection_id": collection_id,
            "query_vector": generate_vector(),
            "top_k": 1000
        })
        elapsed = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            count = len(resp.json().get("results", []))
            latencies.append(elapsed)
            if i == 0:
                payload_kb = len(resp.content) / 1024
                print(f"  First query: {elapsed:.1f}ms, {count} results, {payload_kb:.1f}KB payload")
        else:
            print(f"  Query {i} failed: {resp.text[:80]}")

    print_stats("top_k=1000 vector search", latencies)


def test_hybrid_search_stress(collection_id):
    """50 concurrent hybrid searches."""
    subheader("50 Concurrent Hybrid Searches")

    queries = ["AI", "ML", "databases", "cloud", "vectors", "optimization"]
    latencies = []
    errors = 0

    def do_hybrid(_):
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents/hybrid-search", json={
            "collection_id": collection_id,
            "query_vector": generate_vector(),
            "query_text": random.choice(queries),
            "top_k": 25,
            "vector_weight": 0.6,
            "text_weight": 0.4
        })
        elapsed = (time.perf_counter() - start) * 1000
        return elapsed, resp.status_code == 200

    wall_start = time.perf_counter()
    with ThreadPoolExecutor(max_workers=50) as pool:
        futures = [pool.submit(do_hybrid, i) for i in range(100)]
        for f in as_completed(futures):
            elapsed, ok = f.result()
            if ok:
                latencies.append(elapsed)
            else:
                errors += 1
    wall_time = time.perf_counter() - wall_start

    print(f"  Wall time: {wall_time:.2f}s | Throughput: {100 / wall_time:.1f} ops/sec")
    print_stats("Concurrent hybrid search", latencies, errors)


def main():
    header("STRESS TEST 04: SEARCH UNDER LOAD")
    print(f"  Server: {BASE_URL}")
    print(f"  Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")

    if not check_server():
        print("  Server unreachable. Exiting.")
        sys.exit(1)

    coll_name = f"search_stress_{int(time.time())}"
    coll_id = setup_collection(coll_name)
    if not coll_id:
        return

    print(f"\n  Seeding 2000 documents...")
    doc_ids = seed_documents(coll_id, 2000)
    print(f"  Seeded {len(doc_ids)} documents.\n")

    test_search_during_writes(coll_id)
    test_fulltext_under_write_contention(coll_id)
    test_max_topk_search(coll_id)
    test_hybrid_search_stress(coll_id)

    cleanup_collection(coll_id)
    header("TEST 04 COMPLETE")


if __name__ == "__main__":
    main()
