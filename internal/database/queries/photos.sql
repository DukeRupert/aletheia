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
  storage_url
) VALUES (
  $1, $2
)
RETURNING *;

-- name: DeletePhoto :exec
DELETE FROM photos
WHERE id = $1;
