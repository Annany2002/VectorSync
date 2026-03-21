-- PostgreSQL's tsvector has a hard limit of 1,048,575 bytes.
-- A 1MB content string can produce a tsvector exceeding this limit,
-- causing inserts to fail with "string is too long for tsvector".
--
-- Fix: recreate the FTS index using left(content, 500000) to cap the
-- input to 500KB. This preserves full-text search on the first 500KB
-- of any document's content while accepting arbitrarily large content.

DROP INDEX IF EXISTS idx_documents_content_fts;

CREATE INDEX idx_documents_content_fts
    ON documents
    USING gin(to_tsvector('english', COALESCE(left(content, 500000), '')));
