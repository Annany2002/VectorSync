<p align="center"\>
<img src="./assets/Logo.png" alt="VectorSync Logo" width="200" height="300"/\>
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
- **Dual API Support** - Native gRPC and HTTP/JSON via grpc-gateway
- **Collection Management** - Organize embeddings with fixed dimensions
- **CRUD + Upsert** - Full document operations with atomic upsert
- **Metadata Filtering** - Rich JSONB-based filtering
- **Health Checks** - Kubernetes-compatible liveness and readiness probes
- **Graceful Shutdown** - Proper signal handling and resource cleanup

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
