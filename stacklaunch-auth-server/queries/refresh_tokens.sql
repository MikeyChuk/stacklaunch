-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: ConsumeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > NOW()
RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: RevokeRefreshToken :execrows
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token_hash = $1
  AND revoked_at IS NULL;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1
  AND revoked_at IS NULL;