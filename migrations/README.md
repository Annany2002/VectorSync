# Database Migrations

This directory contains SQL migration files for VectorSync's PostgreSQL schema.

## Migration Files

Migrations are numbered and come in pairs:
- `*.up.sql` - Applies the migration (creates tables, indexes, etc.)
- `*.down.sql` - Rolls back the migration (drops tables, indexes, etc.)

### Current Migrations

1. **001_enable_pgvector** - Enables the pgvector extension for vector operations
2. **002_create_collections** - Creates the collections table with auto-updating timestamps
3. **003_create_documents** - Creates the documents table with vector column and indexes
4. **004_add_distance_metric** - Adds configurable distance metric column to collections

## Prerequisites

Your PostgreSQL database must support:
- PostgreSQL 12+ (for `gen_random_uuid()`)
- pgvector extension installed

### Installing pgvector

```bash
# Ubuntu/Debian
sudo apt install postgresql-15-pgvector

# macOS with Homebrew
brew install pgvector

# Docker (use official pgvector image)
docker pull pgvector/pgvector:pg15
```

## Running Migrations

### Option 1: Manual Execution

```bash
# Apply migrations in order
psql $DB_URL -f migrations/001_enable_pgvector.up.sql
psql $DB_URL -f migrations/002_create_collections.up.sql
psql $DB_URL -f migrations/003_create_documents.up.sql
```

### Option 2: Using golang-migrate

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path ./migrations -database "$DB_URL" up

# Rollback migrations
migrate -path ./migrations -database "$DB_URL" down
```

### Option 3: Programmatic (Go code)

Add migration runner in your application startup:

```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dbURL string) error {
    m, err := migrate.New("file://migrations", dbURL)
    if err != nil {
        return err
    }
    return m.Up()
}
```

## Schema Overview

### collections
- Stores collection metadata
- Each collection has a fixed `vector_dim` (all vectors must match this dimension)
- Supports optional `metadata_schema` for validation

### documents
- Stores document embeddings and metadata
- `vector` column uses pgvector's VECTOR type
- `metadata` is JSONB for flexible filtering
- `content` is optional text for keyword search
- Foreign key to `collections` with CASCADE delete

## Indexes

- **collections.name** - Unique index for fast name lookups
- **documents.collection_id** - Fast filtering by collection
- **documents.metadata** - GIN index for JSONB queries
- **documents.content** - Full-text search (GIN with tsvector)

No vector index is created initially (brute-force search for MVP). Post-MVP, add IVFFlat or HNSW index for ANN search.
