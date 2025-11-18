-- +goose Up
-- +goose StatementBegin
CREATE TYPE violation_status AS ENUM ('pending', 'confirmed', 'dismissed');

CREATE TABLE IF NOT EXISTS detected_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id UUID NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    confidence_score DECIMAL(5,4),
    status violation_status DEFAULT 'pending' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_detected_violations_photo_id ON detected_violations(photo_id);
CREATE INDEX idx_detected_violations_status ON detected_violations(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS detected_violations;
DROP TYPE IF EXISTS violation_status;
-- +goose StatementEnd
