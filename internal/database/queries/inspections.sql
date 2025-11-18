-- name: GetInspection :one
SELECT * FROM inspections
WHERE id = $1 LIMIT 1;

-- name: ListInspections :many
SELECT * FROM inspections
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: ListInspectionsByInspector :many
SELECT * FROM inspections
WHERE inspector_id = $1
ORDER BY created_at DESC;

-- name: ListInspectionsByStatus :many
SELECT * FROM inspections
WHERE project_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: CreateInspection :one
INSERT INTO inspections (
  project_id,
  inspector_id,
  status
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: UpdateInspectionStatus :one
UPDATE inspections
SET
  status = $2,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteInspection :exec
DELETE FROM inspections
WHERE id = $1;
