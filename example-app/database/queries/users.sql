-- Users queries with skimatik annotations

-- name: GetUserByEmail :one
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users 
WHERE email = $1 AND is_active = true;

-- name: GetActiveUsers :many
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users 
WHERE is_active = true 
ORDER BY created_at DESC 
LIMIT $1;

-- name: GetUserStats :one
SELECT 
    COUNT(DISTINCT p.id) as post_count,
    COUNT(DISTINCT c.id) as comment_count
FROM users u
LEFT JOIN posts p ON u.id = p.author_id AND p.is_published = true
LEFT JOIN comments c ON u.id = c.author_id AND c.is_approved = true
WHERE u.id = $1
GROUP BY u.id;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false WHERE id = $1;

-- name: SearchUsers :many
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users 
WHERE is_active = true 
AND (name ILIKE $1 OR email ILIKE $1)
ORDER BY created_at DESC 
LIMIT $2; 