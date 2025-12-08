# VectorSync

VectorSync is a high-performance, self-hostable vector indexing engine built for real-time ingestion and low-latency similarity search. It provides clean APIs for creating collections, inserting/updating documents with embeddings, and performing vector/hybrid search with metadata filters.

---

## MVP Scope

### Core Features
- [ ] Create and manage collections (name, vector_dim, metadata schema)
- [ ] Insert and upsert documents (id, vector, metadata)
- [ ] Delete documents
- [ ] In-memory index loaded from persistent storage at startup
- [ ] Cosine similarity search (brute-force for MVP)
- [ ] Top-K search with optional metadata filters
- [ ] Basic keyword search using DB text index (optional in MVP)
- [ ] Clean HTTP API for all operations
- [ ] Basic gRPC API (optional in MVP)
- [ ] Basic authentication (API key)

### Storage & Persistence
- [ ] PostgreSQL schema for collections, documents, and metadata
- [ ] Atomic writes (DB first, then in-memory index)
- [ ] Startup bootstrap: load all vectors into memory

### Observability
- [ ] Structured logging (request logs, errors, slow queries)
- [ ] Prometheus-compatible metrics (request count, latency, index size)
- [ ] Health endpoints (`/health/live`, `/health/ready`)

### Infrastructure & Packaging
- [ ] Environment-based config (DB URL, port, log level)
- [ ] Dockerfile for service
- [ ] docker-compose setup (service + PostgreSQL)
- [ ] Graceful shutdown handling
- [ ] CI placeholder (lint + build)

---

## Roadmap (Post-MVP)
- [ ] ANN indexing (HNSW/IVF)
- [ ] Async ingestion pipeline (Redis / WAL-based worker)
- [ ] Real-time replicas and sharding
- [ ] Hybrid search (vector + keyword + metadata ranking)
- [ ] Query planner and scoring strategy improvements
- [ ] Admin dashboard (index stats, latency graphs, replicas)
- [ ] Multi-tenant isolation
- [ ] Collection-level retention policies
- [ ] Background index rebuilds
- [ ] Zero-downtime rolling updates
