-- +goose Up
-- Add severity and location fields to detected_violations table

-- Create severity enum type
CREATE TYPE violation_severity AS ENUM ('critical', 'high', 'medium', 'low');

-- Add severity column
ALTER TABLE detected_violations
ADD COLUMN severity violation_severity NOT NULL DEFAULT 'medium';

-- Add location column
ALTER TABLE detected_violations
ADD COLUMN location TEXT;

-- Add index for severity queries
CREATE INDEX idx_detected_violations_severity ON detected_violations(severity);

-- +goose Down
-- Remove severity and location fields

DROP INDEX IF EXISTS idx_detected_violations_severity;

ALTER TABLE detected_violations
DROP COLUMN IF EXISTS location;

ALTER TABLE detected_violations
DROP COLUMN IF EXISTS severity;

DROP TYPE IF EXISTS violation_severity;
