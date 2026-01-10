# VectorSync Roadmap

This document tracks the detailed feature implementation status and future plans for VectorSync.

---

## Progress Summary

**v0.1.0-alpha:**

**Completed:**
- Collection CRUD (Create, Get, List, Delete) - 4/4 methods
- Document CRUD (Create, Get, List, Delete) - 4/4 methods
- Vector search with cosine similarity
- Database schema & migrations (Collections + Documents tables)
- Document count tracking with database triggers
- Repository pattern with JSONB and pgvector handling
- Service layer with validation & business logic
- Pagination support for list operations
- gRPC server with reflection (port 6309)
- HTTP/JSON REST API via grpc-gateway (port 8080)
- RESTful endpoint mapping at /api/v1/*
- Logging infrastructure
- Configuration management
- Security basics (input validation, SQL injection prevention)

**In Progress:**
- Advanced document operations (upsert, batch operations)
- Full-text search and hybrid search capabilities

---

## Current Release: v0.1.0-alpha (In Development)

### Core Vector Operations
- [x] **Collection Management**
  - [x] Create collection with name, vector dimension, optional metadata schema
  - [x] Get collection by ID
  - [x] Get collection by name (implemented for duplicate checking)
  - [x] List all collections with pagination
  - [x] Delete collection (cascade delete all documents with count returned)
  - [x] Validate vector dimension on all operations
  - [x] Document count tracking with database triggers

- [ ] **Document Operations**
  - [x] Insert single document (vector + metadata + optional content)
  - [x] List documents from collection with pagination
  - [x] Get single document by ID
  - [x] Delete document by ID
  - [ ] Upsert document (insert or update if exists)
  - [ ] Batch insert documents (multiple docs in one request)
  - [ ] Batch delete documents

- [x] **Vector Search**
  - [x] Cosine similarity search (brute-force)
  - [x] Top-K results with configurable K
  - [x] Metadata filtering (JSONB queries)
  - [ ] Full-text search on content field
  - [ ] Hybrid search (vector + metadata + text combined)
  - [x] Return similarity scores with results

### API & Protocol
- [ ] **gRPC API**
  - [x] Protocol Buffer definitions (v1)
  - [x] CollectionService implementation (4/4 methods complete: Create, Get, List, Delete)
  - [x] DocumentService implementation (5/5 methods complete: CreateDocument, ListDocument, ListDocuments, DeleteDocument, SearchDocuments)
  - [ ] HealthService implementation
  - [x] Comprehensive error handling with status codes
  - [x] Request validation and sanitization

- [ ] **HTTP/JSON API**
  - [x] grpc-gateway integration
  - [x] RESTful endpoint mapping (all endpoints at /api/v1/*)
  - [ ] OpenAPI/Swagger documentation generation
  - [ ] CORS configuration

### Storage & Persistence
- [ ] **Database Layer**
  - [x] PostgreSQL schema with pgvector extension
  - [x] Collections table with indexes
  - [x] Documents table with vector column
  - [x] JSONB metadata with GIN index
  - [x] Full-text search with tsvector and GIN index
  - [x] Atomic write guarantees (DB-first pattern)
  - [x] Connection pooling with configurable limits
  - [ ] Transaction management for batch operations

- [ ] **In-Memory Index**
  - [ ] Load all vectors at startup
  - [ ] Sync writes to in-memory index
  - [ ] Brute-force cosine similarity implementation
  - [ ] Memory-efficient vector storage
  - [ ] Index rebuild capability

### Authentication & Security
- [ ] **Authentication**
  - [ ] JWT token generation and validation
  - [ ] API key authentication
  - [ ] Token expiration and refresh
  - [ ] gRPC metadata interceptor for auth

- [ ] **Authorization**
  - [ ] Role-based access control (RBAC)
  - [ ] Collection-level permissions
  - [ ] Admin vs. user roles
  - [ ] Permission checking middleware

- [ ] **Security Hardening**
  - [x] Input validation and sanitization
  - [x] SQL injection prevention (using parameterized queries)
  - [ ] Rate limiting per API key
  - [ ] TLS/SSL support for gRPC and HTTP
  - [ ] Request size limits
  - [x] Vector dimension validation

### Observability & Operations
- [ ] **Logging**
  - [x] Structured logging with Logrus
  - [ ] Request ID propagation
  - [x] Log levels (DEBUG, INFO, WARN, ERROR)
  - [x] Log rotation and retention
  - [ ] Sensitive data masking in logs

- [ ] **Metrics**
  - [ ] Prometheus metrics endpoint
  - [ ] Request count by endpoint and status
  - [ ] Request latency histograms (p50, p95, p99)
  - [ ] In-memory index size metrics
  - [ ] Database connection pool metrics
  - [ ] Error rate tracking
  - [ ] Active collections and documents count

- [ ] **Health Checks**
  - [ ] Liveness probe (is process running?)
  - [ ] Readiness probe (can serve traffic?)
  - [ ] Database connectivity check
  - [ ] Memory index status check

- [ ] **Operational Features**
  - [ ] Graceful shutdown (complete in-flight requests)
  - [x] Startup health checks
  - [x] Configuration validation at startup
  - [x] Environment-based configuration
  - [ ] Signal handling (SIGTERM, SIGINT)

### Infrastructure & Deployment
- [ ] **Containerization**
  - [ ] Optimized Dockerfile (multi-stage build)
  - [ ] docker-compose.yml (service + PostgreSQL + optional monitoring)
  - [ ] Health check configuration in Docker
  - [ ] Volume mounting for logs and data

- [x] **Configuration Management**
  - [x] Environment variable loading with godotenv
  - [x] Configuration validation and defaults
  - [x] .env.example template
  - [x] Config documentation

- [ ] **Build & CI**
  - [x] Makefile for common tasks
  - [ ] GitHub Actions workflow (lint, test, build)
  - [ ] golangci-lint configuration
  - [ ] Code formatting checks
  - [ ] Dependency vulnerability scanning

### Testing
- [ ] **Unit Tests**
  - [ ] Model validation tests
  - [ ] Database layer tests (with test DB)
  - [ ] In-memory index tests
  - [ ] API handler tests (with mocks)
  - [ ] 80%+ code coverage

- [ ] **Integration Tests**
  - [ ] End-to-end API tests
  - [ ] Database migration tests
  - [ ] Authentication flow tests
  - [ ] Search accuracy tests

- [ ] **Performance Tests**
  - [ ] Load testing with k6 or similar
  - [ ] Latency benchmarks
  - [ ] Throughput measurements
  - [ ] Memory profiling

### Documentation
- [ ] **User Documentation**
  - [x] Getting Started guide (README.md)
  - [ ] API reference (auto-generated from protos)
  - [x] Configuration guide
  - [ ] Deployment guide (Docker, Kubernetes)
  - [ ] Authentication setup guide

- [ ] **Developer Documentation**
  - [x] Architecture overview
  - [ ] Contributing guidelines
  - [ ] Code style guide
  - [x] Migration guide (migrations/README.md)
  - [ ] Testing guide

---

## v0.2.0: Performance & Scalability (Post-MVP)

### Approximate Nearest Neighbor (ANN) Search
- [ ] HNSW index implementation using pgvector
- [ ] IVFFlat index support
- [ ] Configurable index type per collection
- [ ] Index building on collection creation
- [ ] Background index rebuild without downtime
- [ ] A/B testing ANN vs brute-force accuracy

### Async Ingestion Pipeline
- [ ] Write-ahead log (WAL) for ingestion
- [ ] Redis-based queue for async processing
- [ ] Batch processing worker
- [ ] Retry logic with exponential backoff
- [ ] Dead letter queue for failed inserts
- [ ] Ingestion status tracking API

### Horizontal Scaling
- [ ] Stateless server design (externalize state)
- [ ] Read replicas support
- [ ] Collection-based sharding strategy
- [ ] Consistent hashing for shard assignment
- [ ] Shard rebalancing tooling
- [ ] Cross-shard search aggregation

### Advanced Search Features
- [ ] Custom scoring functions (weighted hybrid search)
- [ ] Query expansion and reranking
- [ ] Multi-vector search (multiple query vectors)
- [ ] Filtered ANN (metadata filters applied during ANN)
- [ ] Diversified search results
- [ ] Explain API (why documents matched)

---

## v0.3.0: Enterprise Features

### Multi-Tenancy
- [ ] Namespace isolation per tenant
- [ ] Tenant-level quotas and rate limits
- [ ] Per-tenant authentication
- [ ] Tenant usage analytics
- [ ] Data isolation guarantees
- [ ] Tenant migration tooling

### Advanced Operations
- [ ] Backup and restore functionality
- [ ] Point-in-time recovery
- [ ] Collection snapshots
- [ ] Collection cloning
- [ ] Retention policies (auto-delete old documents)
- [ ] Data export (CSV, JSONL, Parquet)

### Admin Dashboard
- [ ] Web UI for monitoring
- [ ] Collection management interface
- [ ] Real-time metrics visualization
- [ ] Query analytics and slow query log
- [ ] Index health monitoring
- [ ] User and permission management

### Compliance & Governance
- [ ] Audit logging (who did what, when)
- [ ] Data encryption at rest
- [ ] Field-level encryption for sensitive metadata
- [ ] GDPR compliance tooling (right to deletion, data export)
- [ ] Compliance reports

---

## v0.4.0: Advanced Capabilities

### Machine Learning Integration
- [ ] Built-in embedding generation (via external model APIs)
- [ ] Model version tracking per collection
- [ ] Automatic re-embedding on model updates
- [ ] A/B testing for embeddings

### Advanced Indexing
- [ ] Product quantization (PQ) for compression
- [ ] Disk-based ANN indexes (for datasets > RAM)
- [ ] GPU-accelerated search
- [ ] Multiple metric support (Euclidean, dot product, etc.)

### Streaming & Real-Time
- [ ] Streaming inserts (bidirectional gRPC streaming)
- [ ] Real-time index updates (no restart required)
- [ ] Change data capture (CDC) for external systems
- [ ] Webhook notifications on document changes

---

## Future Considerations

- Federated search across multiple VectorSync instances
- Support for sparse vectors and hybrid dense-sparse search
- Graph-based vector search
- Integration with popular ML frameworks (LangChain, LlamaIndex)
- Kubernetes Operator for automated deployment
- Cloud provider integrations (managed service)
