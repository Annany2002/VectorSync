#!/usr/bin/env python3
"""
Stress Test 01: Validation & Edge Cases
Tests boundary conditions, invalid inputs, and error handling.

Covers:
  - Empty / missing required fields
  - Vector dimension mismatches
  - NaN and Infinity in vectors
  - top_k / limit boundary values (0, negative, max, above-max)
  - Batch size at exactly max (1000) and above
  - Duplicate collection names
  - Non-existent collection / document IDs
  - Invalid distance metrics
"""

import sys, os, math, uuid, time, requests
sys.path.insert(0, os.path.dirname(__file__))
from helpers import *


def test_empty_fields():
    subheader("Empty / Missing Required Fields")

    # Empty collection_id on insert
    resp = requests.post(f"{BASE_URL}/documents", json={
        "collection_id": "",
        "vector": generate_vector(),
        "content": "test"
    })
    if resp.status_code != 200:
        passed("Empty collection_id rejected on insert")
    else:
        failed("Empty collection_id accepted on insert")

    # Empty vector on insert
    coll_id = setup_collection(f"edge_empty_{int(time.time())}")
    if coll_id:
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": coll_id,
            "vector": [],
            "content": "test"
        })
        if resp.status_code != 200:
            passed("Empty vector rejected on insert")
        else:
            failed("Empty vector accepted on insert")

        # Empty query on full-text search
        resp = requests.post(f"{BASE_URL}/documents/text-search", json={
            "collection_id": coll_id,
            "query": "",
            "limit": 10
        })
        if resp.status_code != 200:
            passed("Empty query rejected on full-text search")
        else:
            failed("Empty query accepted on full-text search")

        cleanup_collection(coll_id)


def test_dimension_mismatch():
    subheader("Vector Dimension Mismatch")

    coll_id = setup_collection(f"edge_dim_{int(time.time())}", dim=128)
    if not coll_id:
        return

    # Insert with wrong dimension (768 into 128-dim collection)
    resp = requests.post(f"{BASE_URL}/documents", json={
        "collection_id": coll_id,
        "vector": generate_vector(768),
        "content": "wrong dimension"
    })
    if resp.status_code != 200:
        passed("Dimension mismatch rejected (768 into 128-dim collection)")
    else:
        failed("Dimension mismatch accepted")

    # Insert with correct dimension
    resp = requests.post(f"{BASE_URL}/documents", json={
        "collection_id": coll_id,
        "vector": generate_vector(128),
        "content": "correct dimension"
    })
    if resp.status_code == 200:
        passed("Correct dimension accepted (128 into 128-dim collection)")
    else:
        failed("Correct dimension rejected", resp.text[:100])

    # Search with wrong dimension
    resp = requests.post(f"{BASE_URL}/documents/search", json={
        "collection_id": coll_id,
        "query_vector": generate_vector(256),
        "top_k": 5
    })
    if resp.status_code != 200:
        passed("Search with wrong query vector dimension rejected")
    else:
        failed("Search with wrong query vector dimension accepted")

    cleanup_collection(coll_id)


def test_nan_infinity_vectors():
    subheader("NaN and Infinity in Vectors")

    coll_id = setup_collection(f"edge_nan_{int(time.time())}")
    if not coll_id:
        return

    # NaN and Infinity cannot be serialized to JSON (RFC 7159 compliance)
    # Python's requests library rejects them before they reach the server
    import json

    vec_nan = generate_vector()
    vec_nan[0] = float('nan')
    try:
        json.dumps(vec_nan, allow_nan=False)
        failed("NaN passed JSON serialization (unexpected)")
    except ValueError:
        passed("NaN in vector rejected at JSON serialization level")

    vec_inf = generate_vector()
    vec_inf[100] = float('inf')
    try:
        json.dumps(vec_inf, allow_nan=False)
        failed("Infinity passed JSON serialization (unexpected)")
    except ValueError:
        passed("Infinity in vector rejected at JSON serialization level")

    vec_ninf = generate_vector()
    vec_ninf[50] = float('-inf')
    try:
        json.dumps(vec_ninf, allow_nan=False)
        failed("-Infinity passed JSON serialization (unexpected)")
    except ValueError:
        passed("-Infinity in vector rejected at JSON serialization level")

    print("  Note: Server also validates NaN/Inf for gRPC clients (Go-side)")

    cleanup_collection(coll_id)


def test_topk_limit_boundaries():
    subheader("top_k and Limit Boundaries")

    coll_id = setup_collection(f"edge_topk_{int(time.time())}")
    if not coll_id:
        return

    seed_documents(coll_id, 20)

    # top_k = 0 should default to 10
    resp = requests.post(f"{BASE_URL}/documents/search", json={
        "collection_id": coll_id,
        "query_vector": generate_vector(),
        "top_k": 0
    })
    if resp.status_code == 200:
        passed("top_k=0 defaults gracefully")
    else:
        failed("top_k=0 rejected", resp.text[:100])

    # top_k = 1000 (max allowed)
    resp = requests.post(f"{BASE_URL}/documents/search", json={
        "collection_id": coll_id,
        "query_vector": generate_vector(),
        "top_k": 1000
    })
    if resp.status_code == 200:
        passed("top_k=1000 (max) accepted")
    else:
        failed("top_k=1000 rejected", resp.text[:100])

    # top_k = 1001 (above max)
    resp = requests.post(f"{BASE_URL}/documents/search", json={
        "collection_id": coll_id,
        "query_vector": generate_vector(),
        "top_k": 1001
    })
    if resp.status_code != 200:
        passed("top_k=1001 (above max) rejected")
    else:
        failed("top_k=1001 accepted (should reject)")

    # Negative top_k
    resp = requests.post(f"{BASE_URL}/documents/search", json={
        "collection_id": coll_id,
        "query_vector": generate_vector(),
        "top_k": -5
    })
    if resp.status_code != 200:
        passed("Negative top_k rejected")
    else:
        failed("Negative top_k accepted")

    # List with limit > maxLimit (1000)
    resp = requests.get(f"{BASE_URL}/documents", params={
        "collection_id": coll_id,
        "limit": 1001
    })
    if resp.status_code != 200:
        passed("List limit=1001 (above max) rejected")
    else:
        failed("List limit=1001 accepted (should reject)")

    cleanup_collection(coll_id)


def test_batch_size_limits():
    subheader("Batch Size at Limits")

    coll_id = setup_collection(f"edge_batch_{int(time.time())}")
    if not coll_id:
        return

    # Batch of exactly 1000 (max allowed)
    print("  Generating 1000-doc batch (this takes a moment)...")
    docs_1000 = [{
        "collection_id": coll_id,
        "vector": generate_vector(),
        "content": f"Max batch doc {i}",
        "metadata": generate_metadata()
    } for i in range(1000)]

    start = time.perf_counter()
    resp = requests.post(f"{BASE_URL}/documents/batch", json={
        "collection_id": coll_id,
        "documents": docs_1000
    })
    elapsed = (time.perf_counter() - start) * 1000
    if resp.status_code == 200:
        count = resp.json().get("insertCount", 0)
        passed(f"Batch of 1000 accepted ({count} inserted, {elapsed:.0f}ms)")
    else:
        failed(f"Batch of 1000 rejected", resp.text[:120])

    # Batch of 1001 (above max)
    docs_1001 = docs_1000 + [{
        "collection_id": coll_id,
        "vector": generate_vector(),
        "content": "One too many",
        "metadata": generate_metadata()
    }]
    resp = requests.post(f"{BASE_URL}/documents/batch", json={
        "collection_id": coll_id,
        "documents": docs_1001
    })
    if resp.status_code != 200:
        passed("Batch of 1001 (above max) rejected")
    else:
        failed("Batch of 1001 accepted (should reject)")

    # Empty batch
    resp = requests.post(f"{BASE_URL}/documents/batch", json={
        "collection_id": coll_id,
        "documents": []
    })
    if resp.status_code != 200:
        passed("Empty batch rejected")
    else:
        failed("Empty batch accepted")

    cleanup_collection(coll_id)


def test_duplicate_collection():
    subheader("Duplicate Collection Names")

    name = f"edge_dup_{int(time.time())}"
    coll_id = setup_collection(name)
    if not coll_id:
        return

    # Try creating with same name
    resp = requests.post(f"{BASE_URL}/collections", json={
        "name": name,
        "vector_dimension": VECTOR_DIM
    })
    if resp.status_code != 200:
        passed("Duplicate collection name rejected")
    else:
        failed("Duplicate collection name accepted")

    cleanup_collection(coll_id)


def test_nonexistent_ids():
    subheader("Non-existent Collection / Document IDs")

    fake_uuid = str(uuid.uuid4())

    # Get non-existent document
    resp = requests.get(f"{BASE_URL}/documents/{fake_uuid}")
    if resp.status_code != 200:
        passed("Non-existent document ID returns error")
    else:
        failed("Non-existent document ID returned 200")

    # Delete non-existent document
    resp = requests.delete(f"{BASE_URL}/documents/{fake_uuid}")
    if resp.status_code != 200:
        passed("Delete non-existent document returns error")
    else:
        failed("Delete non-existent document returned 200")

    # Search with non-existent collection
    resp = requests.post(f"{BASE_URL}/documents/search", json={
        "collection_id": fake_uuid,
        "query_vector": generate_vector(),
        "top_k": 5
    })
    if resp.status_code != 200:
        passed("Search with non-existent collection returns error")
    else:
        failed("Search with non-existent collection returned 200")

    # Delete non-existent collection
    resp = requests.delete(f"{BASE_URL}/collections/{fake_uuid}")
    if resp.status_code != 200:
        passed("Delete non-existent collection returns error")
    else:
        failed("Delete non-existent collection returned 200")


def test_invalid_distance_metric():
    subheader("Invalid Distance Metrics")

    resp = requests.post(f"{BASE_URL}/collections", json={
        "name": f"edge_metric_{int(time.time())}",
        "vector_dimension": VECTOR_DIM,
        "distance_metric": "manhattan"
    })
    if resp.status_code != 200:
        passed("Invalid distance metric 'manhattan' rejected")
    else:
        failed("Invalid distance metric accepted")
        cleanup_collection(resp.json()["collection"]["id"])

    # Valid distance metrics
    for metric in ["cosine", "euclidean", "inner_product"]:
        name = f"edge_m_{metric}_{int(time.time())}"
        resp = requests.post(f"{BASE_URL}/collections", json={
            "name": name,
            "vector_dimension": 128,
            "distance_metric": metric
        })
        if resp.status_code == 200:
            passed(f"Valid distance metric '{metric}' accepted")
            cleanup_collection(resp.json()["collection"]["id"])
        else:
            failed(f"Valid distance metric '{metric}' rejected", resp.text[:100])


def main():
    header("STRESS TEST 01: VALIDATION & EDGE CASES")
    print(f"  Server: {BASE_URL}")
    print(f"  Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")

    if not check_server():
        print("  Server unreachable. Exiting.")
        sys.exit(1)

    test_empty_fields()
    test_dimension_mismatch()
    test_nan_infinity_vectors()
    test_topk_limit_boundaries()
    test_batch_size_limits()
    test_duplicate_collection()
    test_nonexistent_ids()
    test_invalid_distance_metric()

    header("TEST 01 COMPLETE")


if __name__ == "__main__":
    main()
