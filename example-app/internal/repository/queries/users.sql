-- Users queries with skimatik annotations
-- Soft-delete: rows with deleted_at IS NOT NULL are treated as deleted (blueprint-vet R-9).

-- name: GetUserByEmail :one
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users
WHERE email = $1 AND is_active = true AND deleted_at IS NULL;

-- name: GetActiveUsers :many
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users
WHERE is_active = true AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1;

-- name: GetUserStats :one
SELECT
    COUNT(DISTINCT p.id) as post_count,
    COUNT(DISTINCT c.id) as comment_count
FROM users u
LEFT JOIN posts p ON u.id = p.author_id AND p.is_published = true AND p.deleted_at IS NULL
LEFT JOIN comments c ON u.id = c.author_id AND c.is_approved = true AND c.deleted_at IS NULL
WHERE u.id = $1 AND u.deleted_at IS NULL
GROUP BY u.id;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false WHERE id = $1;

-- name: SearchUsers :many
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users
WHERE is_active = true
AND deleted_at IS NULL
AND (name ILIKE $1 OR email ILIKE $1)
ORDER BY created_at DESC
LIMIT $2;
