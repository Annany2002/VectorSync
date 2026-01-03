-- Rollback: Remove collections table and related objects
DROP TRIGGER IF EXISTS update_collections_updated_at ON collections;
DROP INDEX IF EXISTS idx_collections_document_count;
DROP INDEX IF EXISTS idx_collections_name;
DROP TABLE IF EXISTS collections CASCADE;

-- NOTE: Do NOT drop update_updated_at_column() function here!
-- It's shared with the documents table (migration 003)
-- The function will be cleaned up when the last table using it is dropped
