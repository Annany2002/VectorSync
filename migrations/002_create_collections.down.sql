-- Rollback: Remove collections table and related objects
DROP TRIGGER IF EXISTS update_collections_updated_at ON collections;
DROP TABLE IF EXISTS collections CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column();
