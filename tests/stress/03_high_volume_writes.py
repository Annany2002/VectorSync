#!/usr/bin/env python3
"""
Stress Test 03: High Volume Writes
Tests sustained write performance at scale and batch throughput limits.

Covers:
  - 5000 sequential inserts (tracking throughput degradation over time)
  - Max batch size (1000 docs) repeated
  - Rapid batch fire (10 x 1000-doc batches back-to-back)
  - Upsert storm (insert + update cycles at scale)
  - Write throughput under data volume (large existing dataset)
"""

import sys, os, time, statistics, uuid, random
sys.path.insert(0, os.path.dirname(__file__))
from helpers import *


def test_sequential_insert_at_scale(collection_id):
    """5000 sequential inserts -- track throughput in 500-doc windows."""
    subheader("5000 Sequential Inserts (windowed throughput)")

    total = 5000
    window = 500
    all_latencies = []
    errors = 0

    for win_start in range(0, total, window):
        win_latencies = []
        win_start_time = time.perf_counter()

        for i in range(win_start, min(win_start + window, total)):
            start = time.perf_counter()
            resp = requests.post(f"{BASE_URL}/documents", json={
                "collection_id": collection_id,
                "vector": generate_vector(),
                "content": f"Volume doc {i}",
                "metadata": generate_metadata()
            })
            elapsed = (time.perf_counter() - start) * 1000
            if resp.status_code == 200:
                win_latencies.append(elapsed)
            else:
                errors += 1

        win_time = time.perf_counter() - win_start_time
        all_latencies.extend(win_latencies)
        count = len(win_latencies)
        throughput = count / win_time if win_time > 0 else 0
        avg = statistics.mean(win_latencies) if win_latencies else 0
        p95 = percentile(win_latencies, 95)

        print(f"  Docs {win_start:>5}-{win_start + window:>5}: "
              f"{throughput:>6.1f} ops/sec | "
              f"Avg: {avg:>6.1f}ms | "
              f"p95: {p95:>6.1f}ms | "
              f"Errors: {errors}")

    total_time = sum(all_latencies) / 1000
    print(f"\n  Total: {len(all_latencies)} docs in {total_time:.1f}s "
          f"({len(all_latencies) / total_time:.1f} overall ops/sec)")


def test_max_batch_repeated(collection_id):
    """10 consecutive max-batch (1000 docs each) inserts."""
    subheader("10 x 1000-doc Batch Inserts (10,000 docs total)")

    batch_latencies = []
    errors = 0

    for batch_num in range(10):
        docs = [{
            "collection_id": collection_id,
            "vector": generate_vector(),
            "content": f"Max batch {batch_num}-{i}",
            "metadata": generate_metadata()
        } for i in range(1000)]

        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents/batch", json={
            "collection_id": collection_id,
            "documents": docs
        })
        elapsed = (time.perf_counter() - start) * 1000

        if resp.status_code == 200:
            count = resp.json().get("insertCount", 0)
            batch_latencies.append(elapsed)
            docs_sec = 1000 / (elapsed / 1000) if elapsed > 0 else 0
            print(f"  Batch {batch_num + 1:>2}/10: {elapsed:>8.1f}ms | "
                  f"{count} docs | {docs_sec:.0f} docs/sec")
        else:
            errors += 1
            print(f"  Batch {batch_num + 1:>2}/10: FAILED -- {resp.text[:80]}")

    if batch_latencies:
        print(f"\n  Summary: {len(batch_latencies)} batches, "
              f"Avg: {statistics.mean(batch_latencies):.0f}ms, "
              f"Total docs: {len(batch_latencies) * 1000}")
    if errors:
        print(f"  Batch errors: {errors}")


def test_upsert_storm(collection_id):
    """100 upserts (new) then 100 upserts (update same IDs)."""
    subheader("Upsert Storm (100 new + 100 updates)")

    doc_ids = [str(uuid.uuid4()) for _ in range(100)]

    # Phase 1: Insert via upsert
    insert_lats = []
    for i, doc_id in enumerate(doc_ids):
        start = time.perf_counter()
        resp = requests.put(f"{BASE_URL}/documents/{doc_id}", json={
            "collection_id": collection_id,
            "vector": generate_vector(),
            "content": f"Upsert insert {i}",
            "metadata": generate_metadata()
        })
        elapsed = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            insert_lats.append(elapsed)

    # Phase 2: Update via upsert
    update_lats = []
    for i, doc_id in enumerate(doc_ids):
        start = time.perf_counter()
        resp = requests.put(f"{BASE_URL}/documents/{doc_id}", json={
            "collection_id": collection_id,
            "vector": generate_vector(),
            "content": f"Upsert update {i}",
            "metadata": generate_metadata()
        })
        elapsed = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            update_lats.append(elapsed)

    print_stats("Upsert (new docs)", insert_lats)
    print_stats("Upsert (existing docs)", update_lats)


def test_write_throughput_under_volume(collection_id):
    """Insert 50 docs into a collection already holding 15,000+ docs."""
    subheader("Write Throughput Under Data Volume (15k+ existing docs)")

    latencies = []
    errors = 0

    for i in range(50):
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": collection_id,
            "vector": generate_vector(),
            "content": f"Post-volume doc {i}",
            "metadata": generate_metadata()
        })
        elapsed = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            latencies.append(elapsed)
        else:
            errors += 1

    print_stats("Insert into 15k+ doc collection", latencies, errors)


def main():
    header("STRESS TEST 03: HIGH VOLUME WRITES")
    print(f"  Server: {BASE_URL}")
    print(f"  Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")

    if not check_server():
        print("  Server unreachable. Exiting.")
        sys.exit(1)

    coll_name = f"volume_stress_{int(time.time())}"
    coll_id = setup_collection(coll_name)
    if not coll_id:
        return

    test_sequential_insert_at_scale(coll_id)
    test_max_batch_repeated(coll_id)
    test_upsert_storm(coll_id)
    test_write_throughput_under_volume(coll_id)

    cleanup_collection(coll_id)
    header("TEST 03 COMPLETE")


if __name__ == "__main__":
    main()
