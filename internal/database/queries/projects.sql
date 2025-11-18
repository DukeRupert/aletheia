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
  name
) VALUES (
  $1, $2
)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET
  name = COALESCE($2, name),
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;
