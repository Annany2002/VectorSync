# Changelog

All notable changes to VectorSync are documented in this file.

---

## [v0.2.0] — 2026-05-20

### Added
- **Raw Text Ingestion Pipeline** — `POST /api/v1/documents/ingest` accepts raw text, chunks it, generates embeddings via external providers, and stores document chunks automatically
- **Chunking Engine** (`internal/chunker`) — three strategies: `fixed` (character-based), `sentence` (punctuation-based), and `recursive` (separator hierarchy)
- **Embedding Client Orchestration** (`internal/embedding`) — modular clients for OpenAI, Cohere, and local Ollama with configurable base URLs via `OPENAI_API_BASE`, `COHERE_API_BASE`, and `OLLAMA_HOST`
- **Collection Embedding Config** — `embedding_provider` and `embedding_model` columns on `collections` table (migration 006); configured at collection creation time
- **CI Pipeline** (`.github/workflows/ci.yaml`) — triggered on PRs to `dev`; runs golangci-lint, gofmt, goimports, protobuf sync verification, unit tests with pgvector service container (race detector enabled), and E2E Python stress tests against a live server
- **Centralized Go Tests** — chunker, embedding client, and ingestion pipeline tests consolidated under `tests/`
- **E2E Ingestion Test** (`tests/stress/07_raw_text_ingestion.py`) — validates ingestion endpoint with mock Ollama server

### Changed
- `CollectionCache` now stores `embedding_provider` and `embedding_model` alongside dimension and distance metric
- `CollectionService.CreateCollection` accepts `provider` and `model` parameters
- `CollectionHandler.CreateCollection` gRPC handler passes embedding fields to service layer
- Moved GitHub Actions config from `.github/workflow/` (invalid) to `.github/workflows/` (correct)

---

## [Unreleased]

### Added
- **Prometheus metrics** — `/metrics` endpoint on the gateway port (8080) exposing gRPC RPC latency (histogram, by method + code), in-flight RPCs, HTTP gateway latency, `sql.DB` pool stats (open/in-use/idle/wait), collection cache hit/miss, async HNSW build duration, and ingestion-pipeline counters (chunks, embed calls, embed duration by provider). New package `internal/metrics`. See [docs/observability.mdx](docs/observability.mdx).

### Performance
- **Async HNSW index DROP on collection delete** — index cleanup no longer blocks the delete response
- **Parallel batch serialization** — `vectorToString` + `json.Marshal` fan out across goroutines for batches >= 32 docs
- **Cache warming at startup** — single `SELECT` pre-populates the collection cache, eliminating first-request DB round-trips
- **BatchDelete uses collection cache** — replaced unconditional DB call with cache lookup
- **Remove TOCTOU in CreateCollection** — single INSERT with UNIQUE constraint catch replaces check-then-insert (saves 1 DB round-trip, fixes race condition)

### Fixed
- **DB_URL built from env components** — no longer requires a hardcoded connection string; builds URL from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` with fallback to `DB_URL` if set
- **FullTextSearch `include_vector` exposed** — the handler hardcoded `include_vector=false`; now reads it from the request like vector and hybrid search

### Testing
- **28 service-layer unit tests** — DocumentService (16 tests) and CollectionService (12 tests) with mock repositories; covers validation, error paths, defaults, dimension checks, NaN/Inf handling, batch limits, weight normalization

### Refactored
- **Repository interfaces** — extracted `DocumentRepository` and `CollectionRepository` interfaces; services now depend on interfaces, enabling unit testing with mocks

### Docs
- Configuration docs rewritten with tabbed examples for Docker, local dev, and managed DB setups

---

## [0.1.0-alpha] - 2026-03-21

First tagged release. Core vector search API with full CRUD, three search modes, and two rounds of performance optimization.

### Features
- **Collection management** — create, list, get, delete collections with fixed vector dimensions
- **Document CRUD** — insert, get, list, delete documents with vector embeddings and JSONB metadata
- **Document upsert** — atomic insert-or-update using PostgreSQL `ON CONFLICT`
- **Batch operations** — insert and delete up to 1000 documents per request in a single atomic transaction
- **Vector similarity search** — top-K nearest neighbor search with configurable `min_threshold` and metadata filtering
- **Full-text search** — PostgreSQL `tsvector`/`tsquery`-based keyword search with relevance ranking
- **Hybrid search** — weighted combination of vector similarity and full-text search with configurable `vector_weight` / `text_weight`
- **Configurable distance metrics** — `cosine`, `euclidean`, or `inner_product` per collection at creation time
- **Per-collection HNSW indexing** — automatic partial HNSW index creation (m=16, ef_construction=64) for fast approximate nearest neighbor search
- **Dual API** — native gRPC on port 6309 and HTTP/JSON via grpc-gateway on port 8080
- **Health checks** — Kubernetes-compatible liveness (`/health/live`) and readiness (`/health/ready`) probes
- **Graceful shutdown** — ordered shutdown of HTTP gateway, gRPC server, and database connections on SIGINT/SIGTERM
- **Optional vector exclusion** — `include_vector=false` reduces response payload by ~55x
- **Metadata filtering** — JSONB containment (`@>`) with GIN index for efficient filtered search
- **Docker support** — multi-stage Dockerfile and Docker Compose with pgvector/pg17

### Performance Optimizations
- **Connection pool** — 50 max open / 25 max idle connections (measured: +94% throughput at 50 concurrent clients)
- **`sync.Pool` for vector serialization** — reuses `strings.Builder` instances on the hottest code path, reducing GC pressure
- **Async HNSW index creation** — collection creates no longer block on DDL (measured: create avg 51ms → 21ms, +146% throughput)
- **Single-lock collection cache** — `GetInfo()` returns dimension + distance metric under one read-lock, replacing two separate lock acquisitions per request
- **`SET LOCAL synchronous_commit = OFF` for batches** — skips WAL fsync wait on batch inserts (measured: 1000-doc batch 1086ms → 809ms, -25%)
- **Pre-allocated batch args** — `make([]any, 0, n*4)` eliminates incremental slice re-allocations
- **`strconv.AppendInt` replaces `fmt.Sprintf`** — avoids 1000 format-string parsings per max batch
- **Statement-level triggers** — replaced row-level triggers for document counting, dramatically faster for batch operations
- **Explicit transactions for batch inserts** — reduces WAL overhead and ensures atomicity
- **Lightweight write responses** — write operations return minimal proto responses
- **Vector exclusion from SQL SELECT** — search operations skip the vector column when `include_vector=false`
- **gRPC max message size 32MB** — prevents silent rejection of large payloads (batch inserts, large list responses)

### Fixed
- **FTS tsvector size limit** — capped FTS GIN index input to 500KB via `left(content, 500000)`, fixing `pq: string is too long for tsvector` error for content >= 750KB (migration 005)
- **godotenv.Load() non-fatal** — `.env` loading no longer fails in containers where env vars are injected directly

### Testing
- **6-file stress test suite** covering:
  - Validation & edge cases (27 test cases)
  - Connection pool saturation (up to 100 concurrent clients)
  - High volume writes (5000 sequential + 10x1000 batch inserts)
  - Search under concurrent write load
  - Payload limits (vectors up to 3072-dim, content up to 1MB)
  - Collection lifecycle (100 rapid create/delete cycles, 3 distance metrics)
- All tests pass with zero server errors

### Benchmarks (768-dim vectors, PostgreSQL 17 + pgvector, Docker)

| Operation | Throughput | Avg Latency |
|-----------|-----------|-------------|
| Single Insert | ~210 ops/sec | ~5ms |
| Batch Insert (1000 docs) | ~1,100 docs/sec | ~810ms/batch |
| Upsert | ~207 ops/sec | ~5ms |
| Vector Search (k=10) | ~250 ops/sec | ~4ms |
| Full-Text Search | ~182 ops/sec | ~6ms |
| Hybrid Search | ~146 ops/sec | ~7ms |
| Concurrent Inserts (C=50) | ~712 ops/sec | ~17ms |
| Collection Create | ~133 ops/sec | ~8ms |
| Collection Delete | ~115 ops/sec | ~9ms |
| Burst After 15s Idle (C=100) | ~454 ops/sec | ~33ms |

### Infrastructure
- Go 1.25.4, PostgreSQL 17, pgvector extension
- Multi-stage Docker build with Alpine
- gRPC reflection enabled for `grpcurl` debugging
- Makefile with build, run, dev, format, lint, test, docker, migrate, proto targets
- Mintlify documentation site with introduction, quickstart, architecture, configuration, API reference, and roadmap

### Database Schema
- `collections` — UUID PK, unique name, vector_dim, distance_metric, metadata_schema (JSONB), document_count (trigger-maintained)
- `documents` — UUID PK, collection_id FK (cascade delete), vector (pgvector), metadata (JSONB + GIN), content (TEXT + FTS GIN), timestamps
- 5 migrations: pgvector extension, collections table, documents table + triggers + indexes, distance metric column, FTS index size fix
