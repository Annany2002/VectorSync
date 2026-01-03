-- Create collections table
-- Stores vector collection metadata and configuration
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    vector_dim INTEGER NOT NULL CHECK (vector_dim > 0),
    metadata_schema JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    document_count BIGINT NOT NULL DEFAULT 0
);

-- Create index on name for faster lookups
CREATE INDEX idx_collections_name ON collections(name);

-- Create index on document_count for sorting/filtering
CREATE INDEX idx_collections_document_count ON collections(document_count);

-- Create trigger function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to collections table
CREATE TRIGGER update_collections_updated_at
    BEFORE UPDATE ON collections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
