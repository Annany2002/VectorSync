<p align="center"\>
<img src="./docs/assets/Logo.png" alt="VectorSync Logo" width="200" height="300"/\>
</p\>

# VectorSync

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.4+-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12+-316192?logo=postgresql)](https://www.postgresql.org/)

**A self-hostable vector search API built on PostgreSQL + pgvector.**

VectorSync is a Go-based API service for vector storage and retrieval. It wraps PostgreSQL with pgvector, exposing vector similarity search, full-text search, and hybrid search through gRPC and REST endpoints.

## Features

- **Vector Similarity Search** -- Cosine, Euclidean, and Inner Product distance with configurable top-K and minimum threshold
- **HNSW Indexing** -- Automatic per-collection HNSW index creation via pgvector for fast approximate nearest neighbor search
- **Full-Text Search** -- PostgreSQL tsvector-based keyword search with relevance ranking
- **Hybrid Search** -- Weighted combination of vector similarity and full-text search
- **Configurable Distance Metrics** -- Choose `cosine`, `euclidean`, or `inner_product` per collection at creation time
- **Dual API** -- Native gRPC (port 6309) and HTTP/JSON via grpc-gateway (port 8080)
- **Collection Management** -- Organize embeddings by collection with fixed dimensions
- **CRUD + Upsert** -- Full document lifecycle with atomic insert-or-update
- **Batch Operations** -- Insert and delete up to 100 documents per request
- **Metadata Filtering** -- JSONB-based filtering on search queries
- **Optional Vector Returns** -- Exclude vectors from responses to reduce payload by ~97%

## Performance

Benchmarked with 768-dimension vectors on PostgreSQL with pgvector:

| Operation | Throughput | Avg Latency |
|-----------|-----------|-------------|
| Single Insert | ~7 ops/sec | ~143ms |
| Batch Insert (100 docs) | ~136 docs/sec | ~735ms/batch |
| Batch Insert (500 docs) | ~214 docs/sec | ~2.3s/batch |
| Upsert | ~7 ops/sec | ~148ms |
| Vector Search (k=10) | ~7 ops/sec | ~150ms |
| Full-Text Search | ~7.5 ops/sec | ~134ms |
| Concurrent Inserts (15 clients) | ~47 ops/sec | ~219ms |

Key optimizations: per-collection HNSW indexes, statement-level triggers for document counting, in-memory collection dimension cache, connection pooling (25 open / 10 idle), and explicit transactions for batch operations.

## Quick Start

```bash
# Clone and start
git clone https://github.com/Annany2002/VectorSync.git
cd VectorSync
docker-compose up -d

# Create a collection
curl -X POST http://localhost:8080/api/v1/collections \
  -H "Content-Type: application/json" \
  -d '{"name": "my_embeddings", "vector_dimension": 768, "distance_metric": "cosine"}'

# Insert a document
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "YOUR_COLLECTION_ID",
    "vector": [0.1, 0.2, 0.3, ...],
    "content": "Sample document"
  }'

# Search similar vectors
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "YOUR_COLLECTION_ID",
    "query_vector": [0.1, 0.2, 0.3, ...],
    "top_k": 10
  }'

# Full-text search on content
curl -X POST http://localhost:8080/api/v1/documents/text-search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "YOUR_COLLECTION_ID",
    "query": "sample document",
    "limit": 10
  }'

# Hybrid search (vector + text combined)
curl -X POST http://localhost:8080/api/v1/documents/hybrid-search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "YOUR_COLLECTION_ID",
    "query_vector": [0.1, 0.2, 0.3, ...],
    "query_text": "sample document",
    "top_k": 10,
    "vector_weight": 0.7,
    "text_weight": 0.3
  }'
```


## Documentation

**[View Full Documentation](./docs)** - Complete guides, API reference, and examples.

| Guide | Description |
|-------|-------------|
| [Introduction](./docs/introduction.mdx) | Project overview and features |
| [Quickstart](./docs/quickstart.mdx) | Get running in 5 minutes |
| [Architecture](./docs/architecture.mdx) | System design and schema |
| [Configuration](./docs/configuration.mdx) | Environment variables |
| [API Reference](./docs/api-reference/) | Complete endpoint docs |
| [Roadmap](./docs/roadmap.mdx) | Future plans |

## Server Endpoints

| Protocol | Port | Description |
|----------|------|-------------|
| gRPC | 6309 | Native gRPC API |
| HTTP | 8080 | REST API via grpc-gateway |

## Contributing

Contributions are welcome! Please open an issue first to discuss changes.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
