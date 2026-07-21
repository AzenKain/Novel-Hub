-- Add location_cfi and location_type to reading_progress
ALTER TABLE reading_progress ADD COLUMN location_cfi TEXT;
ALTER TABLE reading_progress ADD COLUMN location_type TEXT;
