-- Rollback migration: Remove document_count column and related triggers

-- Drop triggers
DROP TRIGGER IF EXISTS increment_collection_doc_count ON documents;
DROP TRIGGER IF EXISTS decrement_collection_doc_count ON documents;

-- Drop trigger functions
DROP FUNCTION IF EXISTS increment_document_count();
DROP FUNCTION IF EXISTS decrement_document_count();

-- Drop index
DROP INDEX IF EXISTS idx_collections_document_count;

-- Drop column
ALTER TABLE collections DROP COLUMN IF EXISTS document_count;
