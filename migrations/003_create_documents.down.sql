-- Rollback: Remove documents table and related objects
DROP TRIGGER IF EXISTS update_documents_updated_at ON documents;
DROP INDEX IF EXISTS idx_documents_content_fts;
DROP INDEX IF EXISTS idx_documents_metadata;
DROP INDEX IF EXISTS idx_documents_collection_id;
DROP TABLE IF EXISTS documents CASCADE;
