-- Add distance_metric column to collections table
-- Default to 'cosine' for backward compatibility with existing collections
ALTER TABLE collections ADD COLUMN distance_metric VARCHAR(20) NOT NULL DEFAULT 'cosine';

-- Validate only supported values
ALTER TABLE collections ADD CONSTRAINT check_distance_metric
    CHECK (distance_metric IN ('cosine', 'euclidean', 'inner_product'));
