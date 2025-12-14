-- Create documents table
-- Stores document embeddings, metadata, and optional content
CREATE TABLE documents (
    id VARCHAR(255) PRIMARY KEY,
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    vector vector NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb NOT NULL,
    content TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Index on collection_id for fast filtering by collection
CREATE INDEX idx_documents_collection_id ON documents(collection_id);

-- GIN index on metadata for fast JSONB queries
-- Enables queries like: WHERE metadata->>'category' = 'tech'
CREATE INDEX idx_documents_metadata ON documents USING gin(metadata);

-- Full-text search index on content (optional for hybrid search)
-- Enables queries like: WHERE to_tsvector('english', content) @@ to_tsquery('search term')
CREATE INDEX idx_documents_content_fts ON documents USING gin(to_tsvector('english', content));

-- Attach updated_at trigger to documents table
CREATE TRIGGER update_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Note: Vector similarity index (IVFFlat/HNSW) can be added later for ANN search
-- For MVP brute-force search, no vector index is needed
-- Example for future: CREATE INDEX ON documents USING ivfflat (vector vector_cosine_ops) WITH (lists = 100);
