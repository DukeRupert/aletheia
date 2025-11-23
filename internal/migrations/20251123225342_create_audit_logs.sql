-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_logs (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL REFERENCES users(id),
      organization_id UUID NOT NULL REFERENCES organizations(id),
      action VARCHAR(50) NOT NULL,
      resource_type VARCHAR(50) NOT NULL,
      resource_id UUID NOT NULL,
      old_values JSONB,
      new_values JSONB,
      ip_address INET,
      user_agent TEXT,
      request_id VARCHAR(100),
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
  );

  CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
  CREATE INDEX idx_audit_logs_org_id ON audit_logs(organization_id);
  CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
  CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE audit_logs;
-- +goose StatementEnd
