-- Comments queries with skimatik annotations
-- Soft-delete: rows with deleted_at IS NOT NULL are treated as deleted (blueprint-vet R-9).

-- name: GetPostComments :many
SELECT c.id, c.content, c.created_at, c.is_approved,
       u.name as author_name, u.email as author_email
FROM comments c
JOIN users u ON c.author_id = u.id AND u.deleted_at IS NULL
WHERE c.post_id = $1 AND c.is_approved = true AND c.deleted_at IS NULL
ORDER BY c.created_at ASC;

-- name: GetUnapprovedComments :many
SELECT c.id, c.content, c.created_at, c.post_id,
       u.name as author_name,
       p.title as post_title
FROM comments c
JOIN users u ON c.author_id = u.id AND u.deleted_at IS NULL
JOIN posts p ON c.post_id = p.id AND p.deleted_at IS NULL
WHERE c.is_approved = false AND c.deleted_at IS NULL
ORDER BY c.created_at DESC;

-- name: ApproveComment :exec
UPDATE comments
SET is_approved = true
WHERE id = $1;

-- name: GetUserCommentCount :one
SELECT COUNT(*) as comment_count
FROM comments
WHERE author_id = $1 AND is_approved = true AND deleted_at IS NULL;
