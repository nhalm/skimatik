-- Posts queries with skimatik annotations
-- Soft-delete: rows with deleted_at IS NOT NULL are treated as deleted (blueprint-vet R-9).

-- name: GetPublishedPosts :many
SELECT p.id, p.title, p.content, p.author_id, p.published_at, p.created_at,
       u.name as author_name
FROM posts p
JOIN users u ON p.author_id = u.id AND u.deleted_at IS NULL
WHERE p.is_published = true AND p.deleted_at IS NULL
ORDER BY p.published_at DESC
LIMIT $1;

-- name: GetPostWithAuthor :one
SELECT p.id, p.title, p.content, p.author_id, p.is_published, p.published_at, p.created_at,
       u.name as author_name, u.email as author_email
FROM posts p
JOIN users u ON p.author_id = u.id AND u.deleted_at IS NULL
WHERE p.id = $1 AND p.deleted_at IS NULL;

-- name: GetUserPosts :many
SELECT id, title, content, author_id, is_published, published_at, created_at
FROM posts
WHERE author_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: PublishPost :exec
UPDATE posts
SET is_published = true, published_at = NOW()
WHERE id = $1 AND is_published = false;

-- name: GetPostsWithCommentCount :many
SELECT p.id, p.title, p.author_id, p.published_at, p.created_at,
       u.name as author_name,
       COUNT(c.id) as comment_count
FROM posts p
JOIN users u ON p.author_id = u.id AND u.deleted_at IS NULL
LEFT JOIN comments c ON p.id = c.post_id AND c.is_approved = true AND c.deleted_at IS NULL
WHERE p.is_published = true AND p.deleted_at IS NULL
GROUP BY p.id, p.title, p.author_id, p.published_at, p.created_at, u.name
ORDER BY p.published_at DESC
LIMIT $1;

-- name: GetPublishedPostsPaginated :paginated
SELECT p.id, p.title, p.content, p.published_at, p.created_at
FROM posts p
WHERE p.is_published = true AND p.deleted_at IS NULL
ORDER BY p.published_at DESC

-- name: GetOldestPostsPaginated :paginated
SELECT p.id, p.title, p.content, p.published_at, p.created_at
FROM posts p
WHERE p.is_published = true AND p.deleted_at IS NULL
ORDER BY p.published_at ASC

-- name: GetPostsByAuthorPaginated :paginated
-- param: author_id uuid
SELECT id, title, content, published_at, created_at
FROM posts
WHERE author_id = $1 AND is_published = true AND deleted_at IS NULL
ORDER BY published_at DESC
