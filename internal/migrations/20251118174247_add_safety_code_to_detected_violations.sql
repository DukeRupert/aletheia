-- +goose Up
-- +goose StatementBegin
ALTER TABLE detected_violations
ADD COLUMN safety_code_id UUID REFERENCES safety_codes(id) ON DELETE SET NULL;

CREATE INDEX idx_detected_violations_safety_code_id ON detected_violations(safety_code_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_detected_violations_safety_code_id;
ALTER TABLE detected_violations DROP COLUMN safety_code_id;
-- +goose StatementEnd
