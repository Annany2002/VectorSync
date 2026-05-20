#!/usr/bin/env python3
"""
Stress Test 07: Raw Text Ingestion Pipeline
Tests raw document ingestion with automatic chunking and embedding generation.
"""

import sys
import os
import time
import requests
import http.server
import socketserver
import threading
import json

sys.path.insert(0, os.path.dirname(__file__))
from helpers import *

# Background mock Ollama server to serve embeddings on the default Ollama port if free
class MockOllamaHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, format, *args):
        pass

    def do_POST(self):
        # Read request body
        content_length = int(self.headers['Content-Length'])
        body = self.rfile.read(content_length)
        req_data = json.loads(body.decode('utf-8'))
        
        # Determine number of embeddings to return based on input size
        inputs = req_data.get("input", [])
        if isinstance(inputs, str):
            inputs = [inputs]
        elif not inputs:
            prompt = req_data.get("prompt", "")
            inputs = [prompt] if prompt else []
            
        embeddings = [[0.1, 0.2, 0.3] for _ in inputs]
        
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        
        if self.path == "/api/embed":
            self.wfile.write(json.dumps({"embeddings": embeddings}).encode('utf-8'))
        else:
            # Fallback legacy endpoint
            self.wfile.write(json.dumps({"embedding": embeddings[0] if embeddings else [0.1, 0.2, 0.3]}).encode('utf-8'))

def start_mock_ollama():
    try:
        # Default Ollama port
        server = socketserver.TCPServer(("127.0.0.1", 11434), MockOllamaHandler)
        t = threading.Thread(target=server.serve_forever)
        t.daemon = True
        t.start()
        print("  Started background mock Ollama server on 127.0.0.1:11434")
        return server
    except Exception as e:
        print(f"  Could not start mock Ollama server (probably already running/port in use): {e}")
        return None

def test_ingestion_without_provider():
    subheader("Ingestion Without Embedding Provider Configured")
    
    # Create collection with no embedding config
    coll_id = setup_collection(f"ingest_no_prov_{int(time.time())}", dim=3)
    if not coll_id:
        failed("Failed to setup test collection")
        return

    # Try to ingest document -> should be rejected because no provider is configured
    resp = requests.post(f"{BASE_URL}/documents/ingest", json={
        "collection_id": coll_id,
        "content": "This is raw document content to test validation.",
        "chunking_config": {
            "strategy": "fixed_size",
            "chunk_size": 10,
            "chunk_overlap": 0
        }
    })
    
    if resp.status_code != 200:
        passed(f"Ingestion rejected successfully for unconfigured collection (Status: {resp.status_code})")
    else:
        failed("Ingestion accepted unexpectedly for collection without embedding provider configuration")
        
    cleanup_collection(coll_id)

def test_ingestion_strategies():
    subheader("Ingestion Strategies and Flow Verification")
    
    # Create collection configured with Ollama (dim=3 to match mock server vectors)
    coll_name = f"ingest_strat_{int(time.time())}"
    resp = requests.post(f"{BASE_URL}/collections", json={
        "name": coll_name,
        "vector_dimension": 3,
        "distance_metric": "cosine",
        "embedding_provider": "ollama",
        "embedding_model": "nomic-embed-text"
    })
    
    if resp.status_code != 200:
        failed("Failed to create collection with embedding configuration", resp.text)
        return
        
    coll_id = resp.json()["collection"]["id"]
    passed(f"Created collection with Ollama embedding config: {coll_id}")
    
    # 1. Test Fixed Size Strategy Ingestion
    print("\n  Testing fixed_size strategy...")
    resp = requests.post(f"{BASE_URL}/documents/ingest", json={
        "collection_id": coll_id,
        "content": "abcdefghij",
        "chunking_config": {
            "strategy": "fixed_size",
            "chunk_size": 3,
            "chunk_overlap": 0
        }
    })
    
    if resp.status_code == 200:
        data = resp.json()
        chunk_count = data.get("chunkCount", 0)
        doc_ids = data.get("documentIds", [])
        if chunk_count == 4 and len(doc_ids) == 4:
            passed(f"Fixed size strategy ingested {chunk_count} chunks successfully")
        else:
            failed(f"Expected 4 chunks, got {chunk_count} ({len(doc_ids)} IDs returned)")
    else:
        failed("Fixed size strategy ingestion failed", resp.text)
        
    # 2. Test Sentence Strategy Ingestion
    print("\n  Testing sentence strategy...")
    resp = requests.post(f"{BASE_URL}/documents/ingest", json={
        "collection_id": coll_id,
        "content": "First sentence. Second sentence.",
        "chunking_config": {
            "strategy": "sentence",
            "chunk_size": 100,
            "chunk_overlap": 0
        }
    })
    
    if resp.status_code == 200:
        data = resp.json()
        chunk_count = data.get("chunkCount", 0)
        if chunk_count == 2:
            passed(f"Sentence strategy ingested {chunk_count} sentences successfully")
        else:
            failed(f"Expected 2 chunks, got {chunk_count}")
    else:
        failed("Sentence strategy ingestion failed", resp.text)

    # 3. Verify chunks were persisted
    print("\n  Verifying database persistence...")
    resp = requests.get(f"{BASE_URL}/documents", params={
        "collection_id": coll_id,
        "limit": 10,
        "include_vector": True
    })
    
    if resp.status_code == 200:
        docs = resp.json().get("documents", [])
        if len(docs) == 6:  # 4 from fixed_size + 2 from sentence
            passed(f"Verified all 6 generated chunks are stored in DB")
            # Verify the vector dimension is 3
            first_vector = docs[0].get("vector", [])
            if len(first_vector) == 3:
                passed("Verified embedding vector dimension is correct (3)")
            else:
                failed(f"Expected vector dimension 3, got {len(first_vector)}")
        else:
            failed(f"Expected 6 documents total in collection, got {len(docs)}")
    else:
        failed("Failed to list stored documents", resp.text)

    cleanup_collection(coll_id)

def main():
    header("STRESS TEST 07: RAW TEXT INGESTION PIPELINE")
    print(f"  Server: {BASE_URL}")
    print(f"  Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")

    if not check_server():
        print("  Server unreachable. Make sure the VectorSync server is running.")
        sys.exit(1)

    mock_server = start_mock_ollama()
    
    try:
        test_ingestion_without_provider()
        test_ingestion_strategies()
    finally:
        if mock_server:
            mock_server.shutdown()
            print("  Stopped background mock Ollama server.")

    header("TEST 07 COMPLETE")


if __name__ == "__main__":
    main()
