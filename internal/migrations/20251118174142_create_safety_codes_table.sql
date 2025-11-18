-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS safety_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    country VARCHAR(2),
    state_province VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_safety_codes_country ON safety_codes(country);
CREATE INDEX idx_safety_codes_state_province ON safety_codes(state_province);
CREATE INDEX idx_safety_codes_code ON safety_codes(code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS safety_codes;
-- +goose StatementEnd
