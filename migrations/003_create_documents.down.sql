-- Rollback: Remove documents table and related objects

-- Drop document count triggers first
DROP TRIGGER IF EXISTS increment_collection_doc_count ON documents;
DROP TRIGGER IF EXISTS decrement_collection_doc_count ON documents;

-- Drop other triggers
DROP TRIGGER IF EXISTS update_documents_updated_at ON documents;

-- Drop indexes
DROP INDEX IF EXISTS idx_documents_content_fts;
DROP INDEX IF EXISTS idx_documents_metadata;
DROP INDEX IF EXISTS idx_documents_collection_id;

-- Drop the table
DROP TABLE IF EXISTS documents CASCADE;

-- Clean up document count trigger functions
DROP FUNCTION IF EXISTS increment_document_count();
DROP FUNCTION IF EXISTS decrement_document_count();

-- Clean up shared update_updated_at_column function (only safe after documents table is dropped)
-- This function is created in migration 002 but shared by both tables
DROP FUNCTION IF EXISTS update_updated_at_column();
