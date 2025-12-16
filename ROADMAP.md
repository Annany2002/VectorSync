# VectorSync Roadmap

This document tracks the detailed feature implementation status and future plans for VectorSync.

---

## Current Release: v0.1.0-alpha (In Development)

### Core Vector Operations
- [ ] **Collection Management**
  - [ ] Create collection with name, vector dimension, optional metadata schema
  - [ ] Get collection by ID
  - [ ] Get collection by name
  - [ ] List all collections with pagination
  - [ ] Delete collection (cascade delete all documents)
  - [ ] Validate vector dimension on all operations

- [ ] **Document Operations**
  - [ ] Insert single document (vector + metadata + optional content)
  - [ ] Upsert document (insert or update if exists)
  - [ ] Batch insert documents (multiple docs in one request)
  - [ ] Get document by ID
  - [ ] Delete document by ID
  - [ ] Batch delete documents

- [ ] **Vector Search**
  - [ ] Cosine similarity search (brute-force)
  - [ ] Top-K results with configurable K
  - [ ] Metadata filtering (JSONB queries)
  - [ ] Full-text search on content field
  - [ ] Hybrid search (vector + metadata + text combined)
  - [ ] Return similarity scores with results

### API & Protocol
- [ ] **gRPC API**
  - [ ] Protocol Buffer definitions (v1)
  - [ ] CollectionService implementation
  - [ ] DocumentService implementation
  - [ ] HealthService implementation
  - [ ] Comprehensive error handling with status codes
  - [ ] Request validation and sanitization

- [ ] **HTTP/JSON API**
  - [ ] grpc-gateway integration
  - [ ] RESTful endpoint mapping
  - [ ] OpenAPI/Swagger documentation generation
  - [ ] CORS configuration

### Storage & Persistence
- [ ] **Database Layer**
  - [ ] PostgreSQL schema with pgvector extension
  - [ ] Collections table with indexes
  - [ ] Documents table with vector column
  - [ ] JSONB metadata with GIN index
  - [ ] Full-text search with tsvector and GIN index
  - [ ] Atomic write guarantees (DB-first pattern)
  - [ ] Connection pooling with configurable limits
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
  - [ ] Input validation and sanitization
  - [ ] SQL injection prevention
  - [ ] Rate limiting per API key
  - [ ] TLS/SSL support for gRPC and HTTP
  - [ ] Request size limits
  - [ ] Vector dimension validation

### Observability & Operations
- [ ] **Logging**
  - [ ] Structured logging with Logrus
  - [ ] Request ID propagation
  - [ ] Log levels (DEBUG, INFO, WARN, ERROR)
  - [ ] Log rotation and retention
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
  - [ ] Startup health checks
  - [ ] Configuration validation at startup
  - [ ] Environment-based configuration
  - [ ] Signal handling (SIGTERM, SIGINT)

### Infrastructure & Deployment
- [ ] **Containerization**
  - [ ] Optimized Dockerfile (multi-stage build)
  - [ ] docker-compose.yml (service + PostgreSQL + optional monitoring)
  - [ ] Health check configuration in Docker
  - [ ] Volume mounting for logs and data

- [ ] **Configuration Management**
  - [ ] Environment variable loading with godotenv
  - [ ] Configuration validation and defaults
  - [ ] .env.example template
  - [ ] Config documentation

- [ ] **Build & CI**
  - [ ] Makefile for common tasks
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
  - [ ] Getting Started guide
  - [ ] API reference (auto-generated from protos)
  - [ ] Configuration guide
  - [ ] Deployment guide (Docker, Kubernetes)
  - [ ] Authentication setup guide

- [ ] **Developer Documentation**
  - [ ] Architecture overview
  - [ ] Contributing guidelines
  - [ ] Code style guide
  - [ ] Migration guide
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

---

## Contributing

Want to help implement features from this roadmap? Check out [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines on how to contribute.

Feature requests and feedback are welcome via GitHub Issues!
