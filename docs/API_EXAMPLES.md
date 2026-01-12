# VectorSync API Examples

This document provides comprehensive examples for using VectorSync's APIs, including both gRPC and HTTP/JSON interfaces.

---

## Table of Contents

- [Quick Reference](#quick-reference)
- [Health Checks](#health-checks)
- [Collection Operations](#collection-operations)
- [Document Operations](#document-operations)
- [Search Operations](#search-operations)
- [Advanced Usage](#advanced-usage)

---

## Quick Reference

### Server Endpoints

- **gRPC Server**: `localhost:6309`
- **HTTP/JSON API**: `http://localhost:8080`
- **Base Path**: `/api/v1`

### Available Services

**gRPC Services:**
- `collection.CollectionService`
- `document.DocumentService`

**HTTP/JSON Endpoints:**
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

---

## Health Checks

VectorSync provides Kubernetes-compatible health check endpoints for monitoring service health and readiness.

### Liveness Check

**Purpose:** Verify the process is alive and can respond to requests.

#### HTTP

```bash
curl http://localhost:8080/health/live
```

**Response:**
```json
{
  "status": "SERVING"
}
```

#### gRPC

```bash
grpcurl -plaintext -d '{"service": ""}' localhost:6309 health.HealthService/Check
```

**Response:**
```json
{
  "status": "SERVING"
}
```

---

### Readiness Check

**Purpose:** Verify the service is ready to serve traffic (database connected).

#### HTTP

```bash
curl http://localhost:8080/health/ready
```

**Response (Ready):**
```json
{
  "status": "SERVING"
}
```

**Response (Not Ready - database down):**
```json
{
  "status": "NOT_SERVING"
}
```

#### gRPC

```bash
grpcurl -plaintext -d '{"service": "readiness"}' localhost:6309 health.HealthService/Check
```

---

### Kubernetes Integration

**Example pod configuration:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: vectorsync
spec:
  containers:
  - name: vectorsync
    image: vectorsync:latest
    ports:
    - containerPort: 6309
      name: grpc
    - containerPort: 8080
      name: http
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

---

## Collection Operations

### Create a Collection

**Purpose:** Create a new vector collection with specified dimension.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "name": "product_embeddings",
  "vector_dimension": 768,
  "metadata_schema": {
    "category": {"type": "string"},
    "price": {"type": "number"}
  }
}' localhost:6309 collection.CollectionService/CreateCollection
```

#### HTTP/JSON

```bash
curl -X POST http://localhost:8080/api/v1/collections \
  -H "Content-Type: application/json" \
  -d '{
    "name": "product_embeddings",
    "vector_dimension": 768,
    "metadata_schema": {
      "category": {"type": "string"},
      "price": {"type": "number"}
    }
  }'
```

**Response:**
```json
{
  "collection": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "createdAt": "2026-01-10T10:00:00Z",
    "updatedAt": "2026-01-10T10:00:00Z",
    "name": "product_embeddings",
    "vectorDimension": 768,
    "metadataSchema": {
      "category": {"type": "string"},
      "price": {"type": "number"}
    },
    "documentCount": "0"
  }
}
```

---

### Get a Collection

**Purpose:** Retrieve a specific collection by ID.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:6309 collection.CollectionService/ListCollection
```

#### HTTP/JSON

```bash
curl http://localhost:8080/api/v1/collections/550e8400-e29b-41d4-a716-446655440000
```

**Response:**
```json
{
  "collection": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "product_embeddings",
    "vectorDimension": 768,
    "documentCount": "42"
  }
}
```

---

### List Collections

**Purpose:** List all collections with pagination.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "limit": 20,
  "offset": 0
}' localhost:6309 collection.CollectionService/ListCollections
```

#### HTTP/JSON

```bash
# List first 20 collections
curl "http://localhost:8080/api/v1/collections?limit=20&offset=0"

# List next 20 (pagination)
curl "http://localhost:8080/api/v1/collections?limit=20&offset=20"
```

**Response:**
```json
{
  "collections": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "product_embeddings",
      "vectorDimension": 768,
      "documentCount": "42"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "name": "user_profiles",
      "vectorDimension": 512,
      "documentCount": "1234"
    }
  ]
}
```

---

### Delete a Collection

**Purpose:** Delete a collection and all its documents.

⚠️ **Warning:** This operation cascades - all documents in the collection will be deleted!

#### gRPC

```bash
grpcurl -plaintext -d '{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:6309 collection.CollectionService/DeleteCollection
```

#### HTTP/JSON

```bash
curl -X DELETE http://localhost:8080/api/v1/collections/550e8400-e29b-41d4-a716-446655440000
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "deletedAt": "2026-01-10T10:30:00Z",
  "documentDeleted": "42"
}
```

---

## Document Operations

### Insert a Document

**Purpose:** Add a new document with vector embedding to a collection.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "collection_id": "550e8400-e29b-41d4-a716-446655440000",
  "vector": [0.1, 0.2, 0.3, 0.4, ...],  // 768 dimensions
  "metadata": {
    "category": "electronics",
    "price": 599.99,
    "brand": "TechCorp"
  },
  "content": "Premium wireless headphones with noise cancellation"
}' localhost:6309 document.DocumentService/CreateDocument
```

#### HTTP/JSON

```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "550e8400-e29b-41d4-a716-446655440000",
    "vector": [0.1, 0.2, 0.3],
    "metadata": {
      "category": "electronics",
      "price": 599.99,
      "brand": "TechCorp"
    },
    "content": "Premium wireless headphones with noise cancellation"
  }'
```

**Response:**
```json
{
  "document": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "createdAt": "2026-01-10T11:00:00Z",
    "updatedAt": "2026-01-10T11:00:00Z",
    "collectionId": "550e8400-e29b-41d4-a716-446655440000",
    "content": "Premium wireless headphones with noise cancellation",
    "vector": [0.1, 0.2, 0.3],
    "metadata": {
      "category": "electronics",
      "price": 599.99,
      "brand": "TechCorp"
    }
  }
}
```

---

### Get a Document

**Purpose:** Retrieve a specific document by ID.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "id": "770e8400-e29b-41d4-a716-446655440002"
}' localhost:6309 document.DocumentService/ListDocument
```

#### HTTP/JSON

```bash
curl http://localhost:8080/api/v1/documents/770e8400-e29b-41d4-a716-446655440002
```

---

### List Documents in a Collection

**Purpose:** List all documents in a specific collection.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "collection_id": "550e8400-e29b-41d4-a716-446655440000",
  "limit": 50,
  "offset": 0
}' localhost:6309 document.DocumentService/ListDocuments
```

#### HTTP/JSON

```bash
curl "http://localhost:8080/api/v1/documents?collection_id=550e8400-e29b-41d4-a716-446655440000&limit=50&offset=0"
```

---

### Delete a Document

**Purpose:** Remove a document from a collection.

#### gRPC

```bash
grpcurl -plaintext -d '{
  "id": "770e8400-e29b-41d4-a716-446655440002"
}' localhost:6309 document.DocumentService/DeleteDocument
```

#### HTTP/JSON

```bash
curl -X DELETE http://localhost:8080/api/v1/documents/770e8400-e29b-41d4-a716-446655440002
```

**Response:**
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "deletedAt": "2026-01-10T12:00:00Z"
}
```

---

## Search Operations

### Basic Vector Search

**Purpose:** Find documents with similar vectors (k-nearest neighbors).

#### gRPC

```bash
grpcurl -plaintext -d '{
  "collection_id": "550e8400-e29b-41d4-a716-446655440000",
  "query_vector": [0.15, 0.22, 0.31, ...],  // 768 dimensions
  "top_k": 10,
  "include_vector": false
}' localhost:6309 document.DocumentService/SearchDocuments
```

#### HTTP/JSON

```bash
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "550e8400-e29b-41d4-a716-446655440000",
    "query_vector": [0.15, 0.22, 0.31],
    "top_k": 10,
    "include_vector": false
  }'
```

**Response:**
```json
{
  "results": [
    {
      "document": {
        "id": "770e8400-e29b-41d4-a716-446655440002",
        "content": "Premium wireless headphones with noise cancellation",
        "metadata": {
          "category": "electronics",
          "price": 599.99
        }
      },
      "score": 0.9876
    },
    {
      "document": {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "content": "Bluetooth earbuds with case",
        "metadata": {
          "category": "electronics",
          "price": 149.99
        }
      },
      "score": 0.8543
    }
  ]
}
```

---

### Search with Metadata Filters

**Purpose:** Combine vector similarity with metadata filtering.

#### HTTP/JSON

```bash
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "550e8400-e29b-41d4-a716-446655440000",
    "query_vector": [0.15, 0.22, 0.31],
    "top_k": 5,
    "metadata_filter": {
      "category": "electronics",
      "price": {
        "max": 500
      }
    },
    "include_vector": false
  }'
```

**What this does:**
- Finds top 5 similar vectors
- Only includes documents where `category = "electronics"`
- Only includes documents where `price ≤ 500`

---

### Search with Similarity Threshold

**Purpose:** Only return results above a minimum similarity score.

#### HTTP/JSON

```bash
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "550e8400-e29b-41d4-a716-446655440000",
    "query_vector": [0.15, 0.22, 0.31],
    "top_k": 20,
    "min_threshold": 0.75,
    "include_vector": false
  }'
```

**What this does:**
- Searches for top 20 nearest neighbors
- Only returns results with `similarity >= 0.75`
- If fewer than 20 results meet threshold, returns only those matches

---

## Advanced Usage

### Search with Vector Included

**Purpose:** Get the actual vector embeddings in search results (useful for analysis).

```bash
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "550e8400-e29b-41d4-a716-446655440000",
    "query_vector": [0.15, 0.22, 0.31],
    "top_k": 5,
    "include_vector": true
  }'
```

**Response includes vectors:**
```json
{
  "results": [
    {
      "document": {
        "id": "770e8400-e29b-41d4-a716-446655440002",
        "vector": [0.1, 0.2, 0.3, ...],
        "content": "..."
      },
      "score": 0.9876
    }
  ]
}
```

---

### Complex Metadata Filtering

**Purpose:** Advanced filtering with multiple conditions.

```bash
curl -X POST http://localhost:8080/api/v1/documents/search \
  -H "Content-Type: application/json" \
  -d '{
    "collection_id": "550e8400-e29b-41d4-a716-446655440000",
    "query_vector": [0.15, 0.22, 0.31],
    "top_k": 10,
    "metadata_filter": {
      "category": "electronics",
      "brand": "TechCorp",
      "in_stock": true,
      "rating": {
        "min": 4.0
      },
      "price": {
        "min": 100,
        "max": 1000
      }
    }
  }'
```

**Filter conditions:**
- Category must be "electronics"
- Brand must be "TechCorp"
- In stock
- Rating ≥ 4.0
- Price between $100-$1000

---

## Error Handling

### Common Error Responses

**Collection not found:**
```json
{
  "code": 5,
  "message": "collection not found",
  "details": []
}
```

**Invalid vector dimension:**
```json
{
  "code": 3,
  "message": "vector dimension mismatch: expected 768, got 512",
  "details": []
}
```

**Missing required field:**
```json
{
  "code": 3,
  "message": "collection_id is required",
  "details": []
}
```

---

## Tips & Best Practices

### Vector Dimensions
- **Must match collection**: All vectors in a collection must have the same dimension
- **Common sizes**: 384 (sentence transformers), 768 (BERT), 1536 (OpenAI)
- **Set at creation**: Cannot change dimension after collection is created

### Search Performance
- **Start with small top_k**: Use top_k=10 initially, increase as needed
- **Use metadata filters**: Pre-filter with metadata before vector search for better performance
- **Exclude vectors**: Set `include_vector: false` to reduce response size (faster)

### Metadata
- **JSONB storage**: Metadata is stored as PostgreSQL JSONB (flexible schema)
- **Indexed**: GIN index on metadata allows fast filtering
- **Any structure**: Can store nested objects, arrays, etc.

### Pagination
- **Use consistent limit**: Stick to same page size for consistent UX
- **Max recommended**: limit=100 for most use cases
- **Calculate pages**: `total_pages = ceil(total_documents / limit)`

---

## See Also

- [README](../README.md) - Project overview and quick start
- [ROADMAP](../ROADMAP.md) - Feature status and future plans
- [Configuration Guide](./CONFIGURATION.md) - Environment variables and settings
