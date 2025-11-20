-- +goose Up
-- +goose StatementBegin
ALTER TABLE projects
ADD COLUMN description TEXT,
ADD COLUMN project_type VARCHAR(50),
ADD COLUMN status VARCHAR(20) DEFAULT 'active',
ADD COLUMN address VARCHAR(255),
ADD COLUMN city VARCHAR(100),
ADD COLUMN state VARCHAR(2),
ADD COLUMN zip_code VARCHAR(10),
ADD COLUMN country VARCHAR(2) DEFAULT 'US';

CREATE INDEX idx_projects_state ON projects(state);
CREATE INDEX idx_projects_status ON projects(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_projects_status;
DROP INDEX IF EXISTS idx_projects_state;

ALTER TABLE projects
DROP COLUMN country,
DROP COLUMN zip_code,
DROP COLUMN state,
DROP COLUMN city,
DROP COLUMN address,
DROP COLUMN status,
DROP COLUMN project_type,
DROP COLUMN description;
-- +goose StatementEnd
