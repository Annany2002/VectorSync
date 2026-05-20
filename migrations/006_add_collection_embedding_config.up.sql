-- Add embedding_provider and embedding_model columns to collections table
-- Nullable: if null, the collection uses manually computed vector embeddings (backward compatible)
ALTER TABLE collections ADD COLUMN embedding_provider VARCHAR(50);
ALTER TABLE collections ADD COLUMN embedding_model VARCHAR(100);

-- Validate supported embedding providers
ALTER TABLE collections ADD CONSTRAINT check_embedding_provider
    CHECK (embedding_provider IS NULL OR embedding_provider IN ('openai', 'ollama', 'cohere'));
