-- name: GetPhoto :one
SELECT * FROM photos
WHERE id = $1 LIMIT 1;

-- name: ListPhotos :many
SELECT * FROM photos
WHERE inspection_id = $1
ORDER BY created_at DESC;

-- name: CreatePhoto :one
INSERT INTO photos (
  inspection_id,
  storage_url,
  thumbnail_url
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: DeletePhoto :exec
DELETE FROM photos
WHERE id = $1;

-- name: GetPhotoCountByOrganizationAndDateRange :one
SELECT COUNT(*) as count
FROM photos ph
JOIN inspections i ON i.id = ph.inspection_id
JOIN projects p ON p.id = i.project_id
WHERE p.organization_id = $1
  AND ph.created_at >= $2
  AND ph.created_at < $3;
