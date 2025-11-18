-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1 LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY created_at DESC;

-- name: CreateOrganization :one
INSERT INTO organizations (
  name
) VALUES (
  $1
)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations
SET
  name = COALESCE($2, name),
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;

-- name: SearchOrganizationsByName :many
SELECT * FROM organizations
WHERE name ILIKE '%' || $1 || '%'
ORDER BY name;
