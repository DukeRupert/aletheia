-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
WHERE status = $1
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (
  email,
  username,
  password_hash,
  first_name,
  last_name
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET
  first_name = COALESCE($2, first_name),
  last_name = COALESCE($3, last_name),
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateUserStatus :one
UPDATE users
SET
  status = $2,
  status_reason = $3,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: SetVerificationToken :exec
UPDATE users
SET
  verification_token = $2,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetUserByVerificationToken :one
SELECT * FROM users
WHERE verification_token = $1
  AND verified_at IS NULL
LIMIT 1;

-- name: VerifyUserEmail :one
UPDATE users
SET
  verified_at = CURRENT_TIMESTAMP,
  verification_token = NULL,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: SetPasswordResetToken :exec
UPDATE users
SET
  reset_token = $2,
  reset_token_expires_at = $3,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetUserByResetToken :one
SELECT * FROM users
WHERE reset_token = $1
  AND reset_token_expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- name: ResetUserPassword :one
UPDATE users
SET
  password_hash = $2,
  reset_token = NULL,
  reset_token_expires_at = NULL,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
