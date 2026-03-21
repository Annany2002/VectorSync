-- Revert to original unbounded tsvector index.
-- WARNING: inserting content > ~750KB will fail after this rollback.

DROP INDEX IF EXISTS idx_documents_content_fts;

CREATE INDEX idx_documents_content_fts
    ON documents
    USING gin(to_tsvector('english', COALESCE(content, '')));
