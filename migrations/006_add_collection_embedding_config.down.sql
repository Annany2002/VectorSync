-- Remove embedding_provider and embedding_model columns from collections table
ALTER TABLE collections DROP CONSTRAINT IF EXISTS check_embedding_provider;
ALTER TABLE collections DROP COLUMN IF EXISTS embedding_provider;
ALTER TABLE collections DROP COLUMN IF EXISTS embedding_model;
