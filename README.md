<p align="center"\>
<img src="./docs/assets/Logo.png" alt="VectorSync Logo" width="200" height="300"/\>
</p\>

# VectorSync

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.4+-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12+-316192?logo=postgresql)](https://www.postgresql.org/)

**A production-ready, self-hostable vector database for semantic search and embedding storage.**

VectorSync is a high-performance vector indexing engine designed for enterprise applications requiring real-time similarity search, hybrid search capabilities, and reliable embedding storage. Built with Go and PostgreSQL with pgvector.

## Features

- **Vector Similarity Search** - Cosine similarity with configurable top-K results
- **Full-Text Search** - Keyword search on document content using PostgreSQL tsvector
- **Hybrid Search** - Combine vector similarity and full-text search with weighted scoring
- **Dual API Support** - Native gRPC and HTTP/JSON via grpc-gateway
- **Collection Management** - Organize embeddings with fixed dimensions
- **CRUD + Upsert** - Full document operations with atomic upsert
- **Batch Operations** - Insert and delete multiple documents atomically in a single request
- **Metadata Filtering** - Rich JSONB-based filtering
- **Health Checks** - Kubernetes-compatible liveness and readiness probes
- **Graceful Shutdown** - Proper signal handling and resource cleanup

## Performance

Benchmarked with 768-dimension vectors on PostgreSQL with pgvector:

| Operation | Throughput | Avg Latency |
|-----------|-----------|-------------|
| Single Insert | ~6 ops/sec | ~164ms |
| Batch Insert (100 docs) | ~64 docs/sec | ~1.6s per batch |
| Upsert (new) | ~6 ops/sec | ~159ms |
| Upsert (update) | ~5 ops/sec | ~205ms |

Write operations use in-memory collection caching and optimized vector serialization to minimize overhead. Response payloads for writes exclude vector data, reducing transfer size by ~3KB per document.

## Quick Start

```bash
# Clone and start
git clone https://github.com/Annany2002/VectorSync.git
cd VectorSync
docker-compose up -d

# Create a collection
curl -X POST http://localhost:8080/api/v1/collections \
  -H "Content-Type: application/json" \
  -d '{"name": "my_embeddings", "vector_dimension": 768}'

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
