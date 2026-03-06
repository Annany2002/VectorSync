"""Shared helpers for VectorSync stress tests."""

import requests
import random
import time
import statistics
import sys

BASE_URL = "http://localhost:8080/api/v1"
VECTOR_DIM = 768


def generate_vector(dim=VECTOR_DIM):
    return [random.uniform(-1, 1) for _ in range(dim)]


def generate_metadata():
    return {
        "category": {"stringValue": random.choice(["tech", "science", "business"])},
        "priority": {"stringValue": random.choice(["low", "medium", "high"])},
    }


def percentile(latencies, pct):
    if not latencies:
        return 0.0
    s = sorted(latencies)
    idx = min(int(len(s) * pct / 100), len(s) - 1)
    return s[idx]


def print_stats(label, latencies, errors=0):
    if not latencies:
        print(f"  {label}: NO DATA (errors={errors})")
        return
    total = sum(latencies) / 1000
    throughput = len(latencies) / total if total > 0 else 0
    print(f"  {label}:")
    print(f"    Count: {len(latencies)} | Total: {total:.2f}s | Throughput: {throughput:.2f} ops/sec")
    print(f"    Avg: {statistics.mean(latencies):.1f}ms | p50: {statistics.median(latencies):.1f}ms | p95: {percentile(latencies, 95):.1f}ms | p99: {percentile(latencies, 99):.1f}ms")
    if errors > 0:
        print(f"    ERRORS: {errors}")


def setup_collection(name, dim=VECTOR_DIM, distance_metric="cosine"):
    resp = requests.post(f"{BASE_URL}/collections", json={
        "name": name,
        "vector_dimension": dim,
        "distance_metric": distance_metric,
    })
    if resp.status_code == 200:
        return resp.json()["collection"]["id"]
    print(f"  Failed to create collection: {resp.text[:120]}")
    return None


def cleanup_collection(collection_id):
    requests.delete(f"{BASE_URL}/collections/{collection_id}")


def seed_documents(collection_id, count, dim=VECTOR_DIM):
    """Insert documents via batch for seeding (not measured)."""
    inserted_ids = []
    batch_size = 100
    for start in range(0, count, batch_size):
        batch_count = min(batch_size, count - start)
        documents = []
        for i in range(batch_count):
            documents.append({
                "collection_id": collection_id,
                "vector": generate_vector(dim),
                "content": f"Seed document {start + i} about {random.choice(['AI', 'ML', 'databases', 'vectors'])}",
                "metadata": generate_metadata()
            })
        resp = requests.post(f"{BASE_URL}/documents/batch", json={
            "collection_id": collection_id,
            "documents": documents
        })
        if resp.status_code == 200:
            ids = resp.json().get("documentIds", [])
            inserted_ids.extend(ids)
    return inserted_ids


def check_server():
    """Verify server is reachable."""
    try:
        resp = requests.get(f"{BASE_URL}/collections", timeout=5)
        return resp.status_code == 200
    except requests.ConnectionError:
        return False


def header(text):
    print(f"\n{'=' * 70}")
    print(f"  {text}")
    print(f"{'=' * 70}")


def subheader(text):
    print(f"\n{'-' * 70}")
    print(f"  {text}")
    print(f"{'-' * 70}")


def passed(label):
    print(f"  [PASS] {label}")


def failed(label, detail=""):
    print(f"  [FAIL] {label}" + (f" -- {detail}" if detail else ""))
