-- +goose Up
-- +goose StatementBegin
ALTER TABLE photos ADD COLUMN thumbnail_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE photos DROP COLUMN thumbnail_url;
-- +goose StatementEnd
