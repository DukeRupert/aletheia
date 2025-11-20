-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1 LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: CreateProject :one
INSERT INTO projects (
  organization_id,
  name,
  description,
  project_type,
  address,
  city,
  state,
  zip_code,
  country
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET
  name = COALESCE(sqlc.narg('name'), name),
  description = COALESCE(sqlc.narg('description'), description),
  project_type = COALESCE(sqlc.narg('project_type'), project_type),
  status = COALESCE(sqlc.narg('status'), status),
  address = COALESCE(sqlc.narg('address'), address),
  city = COALESCE(sqlc.narg('city'), city),
  state = COALESCE(sqlc.narg('state'), state),
  zip_code = COALESCE(sqlc.narg('zip_code'), zip_code),
  country = COALESCE(sqlc.narg('country'), country),
  updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;
