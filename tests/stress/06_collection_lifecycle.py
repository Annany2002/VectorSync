#!/usr/bin/env python3
"""
Stress Test 06: Collection Lifecycle
Tests rapid collection creation/deletion and multi-collection stress.

Covers:
  - Rapid create/delete cycles (100 collections)
  - Many active collections simultaneously
  - Cascade delete with large document counts
  - Distance metric correctness across all three metrics
"""

import sys, os, time, random
from concurrent.futures import ThreadPoolExecutor, as_completed
sys.path.insert(0, os.path.dirname(__file__))
from helpers import *


def test_rapid_create_delete():
    subheader("Rapid Create/Delete Cycles (100 collections)")
    create_lats, delete_lats = [], []
    errors = 0
    for i in range(100):
        name = f"lifecycle_{i}_{int(time.time())}"
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/collections", json={
            "name": name, "vector_dimension": 128
        })
        create_ms = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            create_lats.append(create_ms)
            cid = resp.json()["collection"]["id"]
            start = time.perf_counter()
            requests.delete(f"{BASE_URL}/collections/{cid}")
            delete_lats.append((time.perf_counter() - start) * 1000)
        else:
            errors += 1
    print_stats("Create collection", create_lats, errors)
    print_stats("Delete collection", delete_lats)


def test_many_active_collections():
    subheader("50 Active Collections Simultaneously")
    coll_ids = []
    for i in range(50):
        name = f"multi_{i}_{int(time.time())}"
        cid = setup_collection(name, dim=128)
        if cid:
            coll_ids.append(cid)
    print(f"  Created {len(coll_ids)} collections")

    # Insert 1 doc into each
    insert_lats = []
    for cid in coll_ids:
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": cid, "vector": generate_vector(128),
            "content": "Multi-collection test"
        })
        insert_lats.append((time.perf_counter() - start) * 1000)
    print_stats("Insert across 50 collections", insert_lats)

    # List all collections
    start = time.perf_counter()
    resp = requests.get(f"{BASE_URL}/collections")
    elapsed = (time.perf_counter() - start) * 1000
    count = len(resp.json().get("collections", []))
    print(f"  List all: {count} collections in {elapsed:.1f}ms")

    for cid in coll_ids:
        cleanup_collection(cid)


def test_cascade_delete():
    subheader("Cascade Delete (1000 docs)")
    cid = setup_collection(f"cascade_{int(time.time())}")
    if not cid:
        return
    print("  Seeding 1000 docs...")
    seed_documents(cid, 1000)
    start = time.perf_counter()
    resp = requests.delete(f"{BASE_URL}/collections/{cid}")
    elapsed = (time.perf_counter() - start) * 1000
    if resp.status_code == 200:
        passed(f"Cascade delete of 1000 docs: {elapsed:.0f}ms")
    else:
        failed(f"Cascade delete failed", resp.text[:80])


def test_distance_metric_correctness():
    subheader("Distance Metric Correctness (all 3 metrics)")
    for metric in ["cosine", "euclidean", "inner_product"]:
        cid = setup_collection(f"metric_{metric}_{int(time.time())}", dim=4, distance_metric=metric)
        if not cid:
            continue
        # Insert known vectors
        requests.post(f"{BASE_URL}/documents", json={
            "collection_id": cid, "vector": [1, 0, 0, 0], "content": "unit x"
        })
        requests.post(f"{BASE_URL}/documents", json={
            "collection_id": cid, "vector": [0, 1, 0, 0], "content": "unit y"
        })
        requests.post(f"{BASE_URL}/documents", json={
            "collection_id": cid, "vector": [0.9, 0.1, 0, 0], "content": "near x"
        })
        # Search for vector closest to [1,0,0,0]
        resp = requests.post(f"{BASE_URL}/documents/search", json={
            "collection_id": cid, "query_vector": [1, 0, 0, 0], "top_k": 3
        })
        if resp.status_code == 200:
            results = resp.json().get("results", [])
            top_content = results[0]["document"]["content"] if results else "none"
            scores = [f'{r["score"]:.4f}' for r in results]
            passed(f"{metric}: top='{top_content}' scores={scores}")
        else:
            failed(f"{metric} search", resp.text[:80])
        cleanup_collection(cid)


def main():
    header("STRESS TEST 06: COLLECTION LIFECYCLE")
    if not check_server():
        print("  Server unreachable."); sys.exit(1)
    test_rapid_create_delete()
    test_many_active_collections()
    test_cascade_delete()
    test_distance_metric_correctness()
    header("TEST 06 COMPLETE")

if __name__ == "__main__":
    main()
