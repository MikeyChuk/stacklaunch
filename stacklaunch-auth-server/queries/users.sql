-- name: CreateUser :one
INSERT INTO users (
    email,
    hashed_password
) VALUES (
    $1, $2
)
RETURNING id, email, hashed_password, created_at;

-- name: GetUserByEmail :one
SELECT id, email, hashed_password, created_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, hashed_password, created_at
FROM users
WHERE id = $1;