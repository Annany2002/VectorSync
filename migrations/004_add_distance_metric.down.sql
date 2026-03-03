-- Remove distance_metric constraint and column
ALTER TABLE collections DROP CONSTRAINT IF EXISTS check_distance_metric;
ALTER TABLE collections DROP COLUMN IF EXISTS distance_metric;
