-- Add document_count column to collections table
-- This denormalizes the count for better query performance
ALTER TABLE collections
ADD COLUMN document_count BIGINT NOT NULL DEFAULT 0;

-- Initialize document_count for existing collections
-- Count existing documents and update the collections table
UPDATE collections c
SET document_count = (
    SELECT COUNT(*)
    FROM documents d
    WHERE d.collection_id = c.id
);

-- Create trigger function to increment document_count on document insert
CREATE OR REPLACE FUNCTION increment_document_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE collections
    SET document_count = document_count + 1
    WHERE id = NEW.collection_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger function to decrement document_count on document delete
CREATE OR REPLACE FUNCTION decrement_document_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE collections
    SET document_count = document_count - 1
    WHERE id = OLD.collection_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Attach triggers to documents table
CREATE TRIGGER increment_collection_doc_count
    AFTER INSERT ON documents
    FOR EACH ROW
    EXECUTE FUNCTION increment_document_count();

CREATE TRIGGER decrement_collection_doc_count
    AFTER DELETE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION decrement_document_count();

-- Create index on document_count for potential sorting/filtering
CREATE INDEX idx_collections_document_count ON collections(document_count);
