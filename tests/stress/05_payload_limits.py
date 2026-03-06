#!/usr/bin/env python3
"""
Stress Test 05: Payload Limits
Tests system behavior with extreme payload sizes.
"""

import sys, os, time, random, string
sys.path.insert(0, os.path.dirname(__file__))
from helpers import *


def test_large_vector_dimensions():
    subheader("Large Vector Dimensions")
    for dim in [128, 384, 768, 1536, 3072]:
        name = f"payload_dim_{dim}_{int(time.time())}"
        coll_id = setup_collection(name, dim=dim)
        if not coll_id:
            continue
        vec = generate_vector(dim)
        payload_kb = len(str(vec)) / 1024
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": coll_id, "vector": vec,
            "content": f"Dim test {dim}", "metadata": generate_metadata()
        })
        insert_ms = (time.perf_counter() - start) * 1000
        if resp.status_code == 200:
            start = time.perf_counter()
            resp = requests.post(f"{BASE_URL}/documents/search", json={
                "collection_id": coll_id, "query_vector": generate_vector(dim), "top_k": 1
            })
            search_ms = (time.perf_counter() - start) * 1000
            passed(f"dim={dim:>5}: Insert={insert_ms:.0f}ms | Search={search_ms:.0f}ms | ~{payload_kb:.1f}KB")
        else:
            failed(f"dim={dim}", resp.text[:80])
        cleanup_collection(coll_id)


def test_large_metadata():
    subheader("Large Metadata Objects")
    coll_id = setup_collection(f"payload_meta_{int(time.time())}")
    if not coll_id:
        return
    wide_meta = {f"key_{i}": {"stringValue": f"value_{i}_" + "x" * 100} for i in range(50)}
    start = time.perf_counter()
    resp = requests.post(f"{BASE_URL}/documents", json={
        "collection_id": coll_id, "vector": generate_vector(),
        "content": "Wide metadata", "metadata": wide_meta
    })
    elapsed = (time.perf_counter() - start) * 1000
    (passed if resp.status_code == 200 else failed)(f"50-key metadata: {elapsed:.0f}ms")
    cleanup_collection(coll_id)


def test_large_content():
    subheader("Large Content Strings")
    coll_id = setup_collection(f"payload_content_{int(time.time())}")
    if not coll_id:
        return
    for label, size in [("1KB", 1024), ("10KB", 10240), ("100KB", 102400), ("500KB", 512000), ("1MB", 1048576)]:
        content = ''.join(random.choices(string.ascii_letters + " ", k=size))
        start = time.perf_counter()
        resp = requests.post(f"{BASE_URL}/documents", json={
            "collection_id": coll_id, "vector": generate_vector(),
            "content": content, "metadata": generate_metadata()
        })
        elapsed = (time.perf_counter() - start) * 1000
        (passed if resp.status_code == 200 else failed)(f"Content {label}: {elapsed:.0f}ms")
    cleanup_collection(coll_id)


def test_response_payload_scaling():
    subheader("Response Payload Scaling (1536-dim vectors)")
    coll_id = setup_collection(f"payload_resp_{int(time.time())}", dim=1536)
    if not coll_id:
        return
    print("  Seeding 200 documents (1536-dim)...")
    seed_documents(coll_id, 200, dim=1536)
    for include_vec in ["true", "false"]:
        tag = "" if include_vec == "true" else " (no vectors)"
        for limit in [10, 50, 100, 200]:
            start = time.perf_counter()
            resp = requests.get(f"{BASE_URL}/documents", params={
                "collection_id": coll_id, "limit": limit, "include_vector": include_vec
            })
            elapsed = (time.perf_counter() - start) * 1000
            if resp.status_code == 200:
                kb = len(resp.content) / 1024
                n = len(resp.json().get("documents", []))
                print(f"  List limit={limit:>3}{tag}: {n} docs | {kb:.0f}KB | {elapsed:.0f}ms")
        print()
    cleanup_collection(coll_id)


def main():
    header("STRESS TEST 05: PAYLOAD LIMITS")
    if not check_server():
        print("  Server unreachable."); sys.exit(1)
    test_large_vector_dimensions()
    test_large_metadata()
    test_large_content()
    test_response_payload_scaling()
    header("TEST 05 COMPLETE")

if __name__ == "__main__":
    main()
