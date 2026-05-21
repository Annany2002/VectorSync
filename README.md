<p align="center"\>
<img src="./docs/assets/Logo.png" alt="VectorSync Logo" width="200" height="300"/\>
</p\>

# VectorSync

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.4+-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12+-316192?logo=postgresql)](https://www.postgresql.org/)
[![CI](https://github.com/Annany2002/VectorSync/actions/workflows/ci.yaml/badge.svg)](https://github.com/Annany2002/VectorSync/actions/workflows/ci.yaml)

**A self-hostable vector search API built on PostgreSQL + pgvector.**

VectorSync is a Go-based API service for vector storage and retrieval. It wraps PostgreSQL with pgvector, exposing vector similarity search, full-text search, and hybrid search through gRPC and REST endpoints.

## Features

- **Raw Text Ingestion Pipeline** -- Accept raw text, automatically chunk it, generate embeddings via external providers, and store the results (v0.2.0)
- **Multi-Provider Embedding Support** -- OpenAI, Cohere, and local Ollama models for vector generation
- **Pluggable Chunking Engine** -- Fixed-size, sentence-based, and recursive-character text splitting strategies
- **Vector Similarity Search** -- Cosine, Euclidean, and Inner Product distance with configurable top-K and minimum threshold
- **HNSW Indexing** -- Automatic per-collection HNSW index creation via pgvector for fast approximate nearest neighbor search
- **Full-Text Search** -- PostgreSQL tsvector-based keyword search with relevance ranking
- **Hybrid Search** -- Weighted combination of vector similarity and full-text search
- **Configurable Distance Metrics** -- Choose `cosine`, `euclidean`, or `inner_product` per collection at creation time
- **Dual API** -- Native gRPC (port 6309) and HTTP/JSON via grpc-gateway (port 8080)
- **Collection Management** -- Organize embeddings by collection with fixed dimensions and optional embedding config
- **CRUD + Upsert** -- Full document lifecycle with atomic insert-or-update
- **Batch Operations** -- Insert and delete up to 1000 documents per request
- **Metadata Filtering** -- JSONB-based filtering on search queries
- **Optional Vector Returns** -- Exclude vectors from responses to reduce payload by ~97%
- **Prometheus Metrics** -- `/metrics` endpoint on the gateway port: gRPC latency histograms, pool stats, cache hit ratio, HNSW build duration, ingestion counters ([docs](docs/observability.mdx))
- **CI Pipeline** -- Automated linting, formatting, protobuf sync, unit tests, and E2E validation on every PR

## Performance

Benchmarked with 768-dimension vectors on PostgreSQL 17 + pgvector (Docker, local):

| Operation | Throughput | Avg Latency |
|-----------|-----------|-------------|
| Single Insert | ~210 ops/sec | ~5ms |
| Batch Insert (1000 docs) | ~1,100 docs/sec | ~810ms/batch |
| Upsert (new) | ~207 ops/sec | ~5ms |
| Upsert (update) | ~217 ops/sec | ~5ms |
| Vector Search (k=10) | ~250 ops/sec | ~4ms |
| Full-Text Search | ~182 ops/sec | ~6ms |
| Hybrid Search | ~146 ops/sec | ~7ms |
| Concurrent Inserts (50 clients) | ~712 ops/sec | ~17ms |
| Collection Create | ~48 ops/sec | ~21ms |
| Burst After Idle (100 clients) | ~454 ops/sec | ~33ms |

**Stress-tested up to:**
- 100 concurrent clients with zero errors
- 15,000+ documents with no throughput degradation
- 1000-doc max batch size (10,000 docs in 10 batches at ~1,100 docs/sec)
- Vector dimensions up to 3072 (OpenAI `text-embedding-3-large`)
- Content up to 1MB per document
- `include_vector=false` reduces response payload by 55x

Key optimizations: connection pool (50 open / 25 idle), async HNSW index creation, single-lock collection cache, `sync.Pool` for vector serialization, `synchronous_commit=off` for batch inserts, and per-collection partial HNSW indexes with statement-level triggers for O(1) document counting.

## Quick Start

```bash
# Clone and start
git clone https://github.com/Annany2002/VectorSync.git
cd VectorSync

# Configure environment
cp .env.example .env
# Edit .env with your database credentials (optional - defaults work with Docker Compose)

docker-compose up -d

# Create a collection (with optional embedding provider for ingestion)
curl -X POST http://localhost:8080/api/v1/collections \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_embeddings",
    "vector_dimension": 768,
    "distance_metric": "cosine",
    "embedding_provider": "openai",
    "embedding_model": "text-embedding-3-small"
  }'

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

# Ingest raw text (auto-chunk + auto-embed)
curl -X POST http://localhost:8080/api/v1/documents/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "YOUR_COLLECTION_ID",
    "content": "VectorSync is a self-hostable vector search API. It supports multiple embedding providers.",
    "chunking_config": {
      "strategy": "sentence",
      "chunk_size": 500,
      "chunk_overlap": 50
    }
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

All PRs targeting `dev` are automatically validated by CI which runs:
- **Lint & Format** — golangci-lint, gofmt, goimports
- **Protobuf Sync** — Verifies generated `.pb.go` files match `.proto` definitions
- **Unit Tests** — `go test -race ./...` against a pgvector service container
- **E2E Validation** — Python stress tests against a live VectorSync server

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
