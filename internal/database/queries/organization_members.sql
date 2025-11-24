-- name: GetOrganizationMember :one
SELECT * FROM organization_members
WHERE id = $1 LIMIT 1;

-- name: GetOrganizationMemberByUserAndOrg :one
SELECT * FROM organization_members
WHERE organization_id = $1 AND user_id = $2 LIMIT 1;

-- name: ListOrganizationMembers :many
SELECT * FROM organization_members
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: ListUserOrganizations :many
SELECT * FROM organization_members
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListUserOrganizationsWithDetails :many
SELECT
  o.id,
  o.name,
  o.created_at,
  o.updated_at,
  om.role
FROM organizations o
INNER JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1
ORDER BY o.created_at DESC;

-- name: AddOrganizationMember :one
INSERT INTO organization_members (
  organization_id,
  user_id,
  role
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: UpdateOrganizationMemberRole :one
UPDATE organization_members
SET role = $2
WHERE id = $1
RETURNING *;

-- name: RemoveOrganizationMember :exec
DELETE FROM organization_members
WHERE id = $1;
