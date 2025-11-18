-- name: GetDetectedViolation :one
SELECT * FROM detected_violations
WHERE id = $1 LIMIT 1;

-- name: ListDetectedViolations :many
SELECT * FROM detected_violations
WHERE photo_id = $1
ORDER BY created_at DESC;

-- name: ListDetectedViolationsByStatus :many
SELECT * FROM detected_violations
WHERE photo_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: CreateDetectedViolation :one
INSERT INTO detected_violations (
  photo_id,
  description,
  confidence_score,
  safety_code_id,
  status
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateDetectedViolationStatus :one
UPDATE detected_violations
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateDetectedViolationSafetyCode :one
UPDATE detected_violations
SET safety_code_id = $2
WHERE id = $1
RETURNING *;

-- name: DeleteDetectedViolation :exec
DELETE FROM detected_violations
WHERE id = $1;
