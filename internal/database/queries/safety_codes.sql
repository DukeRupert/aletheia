-- name: GetSafetyCode :one
SELECT * FROM safety_codes
WHERE id = $1 LIMIT 1;

-- name: GetSafetyCodeByCode :one
SELECT * FROM safety_codes
WHERE code = $1 LIMIT 1;

-- name: ListSafetyCodes :many
SELECT * FROM safety_codes
ORDER BY code;

-- name: ListSafetyCodesByCountry :many
SELECT * FROM safety_codes
WHERE country = $1
ORDER BY code;

-- name: ListSafetyCodesByStateProvince :many
SELECT * FROM safety_codes
WHERE state_province = $1
ORDER BY code;

-- name: CreateSafetyCode :one
INSERT INTO safety_codes (
  code,
  description,
  country,
  state_province
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateSafetyCode :one
UPDATE safety_codes
SET
  code = COALESCE($2, code),
  description = COALESCE($3, description),
  country = COALESCE($4, country),
  state_province = COALESCE($5, state_province),
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteSafetyCode :exec
DELETE FROM safety_codes
WHERE id = $1;
