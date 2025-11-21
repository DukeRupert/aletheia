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
  status,
  severity,
  location
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
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

-- name: ListDetectedViolationsByInspection :many
SELECT dv.* FROM detected_violations dv
JOIN photos p ON dv.photo_id = p.id
WHERE p.inspection_id = $1
ORDER BY dv.created_at DESC;

-- name: ListDetectedViolationsByInspectionAndStatus :many
SELECT dv.* FROM detected_violations dv
JOIN photos p ON dv.photo_id = p.id
WHERE p.inspection_id = $1 AND dv.status = $2
ORDER BY dv.created_at DESC;

-- name: UpdateDetectedViolationNotes :one
UPDATE detected_violations
SET
  status = COALESCE(sqlc.narg(status), status),
  description = COALESCE(sqlc.narg(description), description)
WHERE id = $1
RETURNING *;

-- name: CountDetectedViolationsByInspection :one
SELECT COUNT(*) FROM detected_violations dv
JOIN photos p ON dv.photo_id = p.id
WHERE p.inspection_id = $1;

-- name: DeletePendingViolationsByPhoto :exec
DELETE FROM detected_violations
WHERE photo_id = $1 AND status = 'pending';

-- name: DeletePendingAndDismissedViolationsByPhoto :exec
DELETE FROM detected_violations
WHERE photo_id = $1 AND status IN ('pending', 'dismissed');
