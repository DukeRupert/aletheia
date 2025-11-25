-- name: GetReport :one
SELECT * FROM reports
WHERE id = $1 LIMIT 1;

-- name: ListReports :many
SELECT * FROM reports
WHERE inspection_id = $1
ORDER BY created_at DESC;

-- name: CreateReport :one
INSERT INTO reports (
  inspection_id,
  storage_url
) VALUES (
  $1, $2
)
RETURNING *;

-- name: DeleteReport :exec
DELETE FROM reports
WHERE id = $1;

-- name: GetReportCountByOrganizationAndDateRange :one
SELECT COUNT(*) as count
FROM reports r
JOIN inspections i ON i.id = r.inspection_id
JOIN projects p ON p.id = i.project_id
WHERE p.organization_id = $1
  AND r.created_at >= $2
  AND r.created_at < $3;
