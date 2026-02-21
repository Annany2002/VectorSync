-- Create documents table
-- Stores document embeddings, metadata, and optional content
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
-- Enables queries like: WHERE to_tsvector('english', COALESCE(content, '')) @@ to_tsquery('search term')
-- COALESCE handles NULL content gracefully by treating it as empty string
CREATE INDEX idx_documents_content_fts ON documents USING gin(to_tsvector('english', COALESCE(content, '')));

-- Attach updated_at trigger to documents table
CREATE TRIGGER update_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create trigger functions to maintain document_count in collections table
-- Statement-level triggers: fire once per INSERT/DELETE statement, not per row.
-- Uses transition tables (REFERENCING NEW/OLD TABLE) to count affected rows
-- in a single UPDATE per collection_id, eliminating N per-row UPDATEs.

-- Increment document_count when documents are inserted
CREATE OR REPLACE FUNCTION increment_document_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE collections c
    SET document_count = c.document_count + cnt.n
    FROM (
        SELECT collection_id, COUNT(*) AS n
        FROM new_rows
        GROUP BY collection_id
    ) cnt
    WHERE c.id = cnt.collection_id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Decrement document_count when documents are deleted
CREATE OR REPLACE FUNCTION decrement_document_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE collections c
    SET document_count = c.document_count - cnt.n
    FROM (
        SELECT collection_id, COUNT(*) AS n
        FROM old_rows
        GROUP BY collection_id
    ) cnt
    WHERE c.id = cnt.collection_id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Attach statement-level document count triggers to documents table
CREATE TRIGGER increment_collection_doc_count
    AFTER INSERT ON documents
    REFERENCING NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION increment_document_count();

CREATE TRIGGER decrement_collection_doc_count
    AFTER DELETE ON documents
    REFERENCING OLD TABLE AS old_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION decrement_document_count();

-- Note: Vector similarity index (IVFFlat/HNSW) can be added later for ANN search
-- For MVP brute-force search, no vector index is needed
-- Example for future: CREATE INDEX ON documents USING ivfflat (vector vector_cosine_ops) WITH (lists = 100);
