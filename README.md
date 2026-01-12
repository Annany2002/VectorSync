<p align="center"\>
<img src="./assets/Logo.png" alt="VectorSync Logo" width="200" height="300"/\>
</p\>

# VectorSync
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25.4+-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12+-316192?logo=postgresql)](https://www.postgresql.org/)

**A production-ready, self-hostable vector database for semantic search and embedding storage.**

VectorSync is a high-performance vector indexing engine designed for enterprise applications requiring real-time similarity search, hybrid search capabilities, and reliable embedding storage. Built with Go and PostgreSQL with pgvector, it provides both gRPC and HTTP/JSON APIs for seamless integration into modern microservice architectures.

---

## Key Features

### Core Capabilities
- **Vector Similarity Search** - Cosine similarity search with configurable top-K results
- **Hybrid Search** - Combine vector similarity with metadata filters and full-text search
- **Collection Management** - Organize embeddings into collections with fixed dimensions
- **CRUD Operations** - Insert, upsert, retrieve, and delete documents with embeddings
- **Metadata Filtering** - Rich JSONB-based filtering for precise search results

### Production-Ready Infrastructure
- **Dual API Support** - Native gRPC with HTTP/JSON gateway via grpc-gateway
- **Authentication & Authorization** - JWT-based authentication with role-based access control
- **Observability** - Structured logging, Prometheus metrics, and health endpoints
- **Data Durability** - Atomic writes with PostgreSQL persistence and in-memory indexing
- **Graceful Operations** - Startup bootstrapping, graceful shutdown, and connection pooling
- **Docker Support** - Production-ready Dockerfile and docker-compose setup

### Developer Experience
- **Clean API Design** - Well-documented Protocol Buffers with versioned endpoints
- **Type Safety** - Strongly-typed Go implementation with comprehensive error handling
- **Easy Deployment** - Environment-based configuration and containerized deployment
- **Comprehensive Logging** - Request tracing, error tracking, and performance monitoring

---

## Architecture

VectorSync uses a hybrid architecture combining in-memory indexing with persistent storage:

```
┌─────────────────────────────────────────────────┐
│                   Client Apps                   │
└────────────┬────────────────────┬───────────────┘
             │                    │
        gRPC │               HTTP │ (grpc-gateway)
             │                    │
┌────────────┴────────────────────┴───────────────┐
│              VectorSync Server                  │
│  ┌──────────────────────────────────────────┐   │
│  │         In-Memory Vector Index           │   │
│  │    (Loaded at startup, synced on write)  │   │
│  └──────────────────────────────────────────┘   │
│                      │                          │
│                      │ Atomic Writes            │
│                      ▼                          │
│  ┌──────────────────────────────────────────┐   │
│  │   PostgreSQL + pgvector Extension        │   │
│  │   - Collections table                    │   │
│  │   - Documents table (vectors + metadata) │   │
│  └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

**Key Design Decisions:**
- **DB-First Writes**: All writes persist to PostgreSQL first, then update in-memory index (ensures durability)
- **In-Memory Reads**: Search operations use in-memory index for low-latency responses
- **JSONB Metadata**: Flexible metadata storage with GIN indexing for efficient filtering
- **pgvector Integration**: Native PostgreSQL vector operations for reliable persistence

---

## Quick Start

### Prerequisites
- Go 1.25.4+
- PostgreSQL 12+ with pgvector extension
- Docker & Docker Compose (optional)

### Installation

#### Option 1: Docker Compose (Recommended)
```bash
# Clone the repository
git clone https://github.com/Annany2002/vector-sync.git
cd vector-sync

# Start the stack
docker-compose up -d

# VectorSync will be available at:
# - gRPC: localhost:6309
# - HTTP: localhost:8080
```

#### Option 2: Manual Setup
```bash
# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Edit .env with your PostgreSQL connection string

# Run migrations
psql $DB_URL -f migrations/001_enable_pgvector.up.sql
psql $DB_URL -f migrations/002_create_collections.up.sql
psql $DB_URL -f migrations/003_create_documents.up.sql

# Start the server
go run cmd/server/main.go
```

### Usage Example

VectorSync supports both gRPC and HTTP/JSON APIs running on separate ports.

#### Quick Start with HTTP/JSON

```bash
# 1. Create a collection
curl -X POST http://localhost:8080/api/v1/collections \
  -H "Content-Type: application/json" \
  -d '{"name": "embeddings", "vector_dimension": 768}'

# 2. Insert a document
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "your-collection-id",
    "vector": [0.1, 0.2, 0.3],
    "metadata": {"category": "tech"},
    "content": "Sample document"
  }'

# 3. Search for similar vectors
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "your-collection-id",
    "query_vector": [0.1, 0.2, 0.3],
    "top_k": 10
  }'
```

#### Quick Start with gRPC

```bash
# Create a collection
grpcurl -plaintext -d '{
  "name": "embeddings",
  "vector_dimension": 768
}' localhost:6309 collection.CollectionService/CreateCollection
```

**📚 For comprehensive examples, see [API Examples Documentation](./docs/API_EXAMPLES.md)**

---

## API Documentation

VectorSync provides two API interfaces:

### gRPC API
- **Port:** `localhost:6309`
- **CollectionService** - Create, retrieve, list, and delete collections
- **DocumentService** - Insert, retrieve, list, delete, and search documents
- **HealthService** - Liveness and readiness probes (coming soon)

### HTTP/JSON API
- **Port:** `localhost:8080`
- **Base Path:** `/api/v1`
- All gRPC endpoints are automatically exposed via grpc-gateway
- RESTful URL structure following industry best practices
- Full support for JSON request/response payloads

**Available Endpoints:**
```
POST   /api/v1/collections              # Create collection
GET    /api/v1/collections              # List collections
GET    /api/v1/collections/{id}         # Get collection
DELETE /api/v1/collections/{id}         # Delete collection

POST   /api/v1/documents                # Create document
GET    /api/v1/documents                # List documents
GET    /api/v1/documents/{id}           # Get document
DELETE /api/v1/documents/{id}           # Delete document
POST   /api/v1/documents/search         # Search similar vectors
```

**📚 Detailed Usage:** [API Examples](./docs/API_EXAMPLES.md) | API Reference (coming soon)

---

## Configuration

VectorSync is configured via environment variables:

```bash
# Database
DB_URL=postgres://user:password@localhost:5432/vectorsync?sslmode=disable

# Server
GRPC_PORT=6309
HTTP_PORT=8080
LOG_LEVEL=info

# Authentication
JWT_SECRET=your-secret-key
API_KEY_ENABLED=true

# Performance
MAX_DB_CONNECTIONS=10
IDLE_DB_CONNECTIONS=5
```

See [Configuration Guide](./docs/CONFIGURATION.md) for all options.

---

## Observability

### Health Endpoints

VectorSync provides Kubernetes-compatible health check endpoints:

- **`GET /health/live`** - Liveness probe
  - Returns 200 OK if the process is alive and can respond to requests
  - Should always return 200 as long as the server is running
  - Use for container restart policies (Kubernetes liveness probe)

- **`GET /health/ready`** - Readiness probe  
  - Returns 200 OK if the service is ready to serve traffic
  - Verifies database connectivity via ping
  - Returns 503 Service Unavailable if database is unreachable
  - Use for load balancer routing decisions (Kubernetes readiness probe)

**Example:**
```bash
# Check if server is alive
curl http://localhost:8080/health/live
# Response: {"status":"SERVING"}

# Check if server is ready
curl http://localhost:8080/health/ready
# Response: {"status":"SERVING"} (or {"status":"NOT_SERVING"} if DB is down)
```

### Metrics
Prometheus-compatible metrics available at `/metrics`:
- Request count and latency by endpoint
- In-memory index size and document count
- Database connection pool stats
- Error rates and types

### Logging
Structured JSON logs with:
- Request IDs for distributed tracing
- Performance metrics (query latency, vector count)
- Error context and stack traces

---

## Development

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/models -v
```

### Building
```bash
# Build binary
go build -o bin/vectorsync cmd/server/main.go

# Build Docker image
docker build -t vectorsync:latest .
```

### Makefile Commands

VectorSync includes a Makefile for common development tasks:

```bash
# Show all available commands
make help

# Run all checks and build
make all

# Build the production binary
make build

# Run the server (builds first)
make run

# Run in development mode with hot reload (requires air)
make dev

# Run tests
make test

# Format code with goimports
make format

# Run linters
make lint

# Auto-fix linting issues
make fix

# Clean build artifacts
make clean

# Docker commands
make docker-build    # Build Docker image
make docker-up       # Start containers
make docker-down     # Stop containers

# Generate protobuf code
make proto

# Run database migrations
make migrate-up

# Install development tools (goimports, golangci-lint, air)
make install-tools
```

### Code Generation
```bash
# Generate protobuf code
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/proto/v1/*.proto
```

---

## Roadmap

See [ROADMAP.md](./ROADMAP.md) for detailed feature tracking and future plans.

**Upcoming Enhancements:**
- HNSW/IVF indexing for approximate nearest neighbor (ANN) search
- Horizontal scaling with sharding and replication
- Advanced hybrid search with custom ranking algorithms
- Multi-tenancy support with namespace isolation
- Web-based admin dashboard

---

## Performance

**Benchmark Results** (Coming Soon)
- Search latency: < 50ms (p99) for 1M vectors
- Throughput: 1000+ searches/sec on single instance
- Index size: ~4GB for 1M 768-dimensional vectors

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

---

## License

VectorSync is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

```
Copyright 2026 Annany Vishwakarma

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
