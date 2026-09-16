-- name: CreateUser :one
INSERT INTO users (email, password_hash, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users WHERE id = $1;