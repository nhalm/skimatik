-- Posts queries with skimatik annotations

-- name: GetPublishedPosts :many
SELECT p.id, p.title, p.content, p.author_id, p.published_at, p.created_at,
       u.name as author_name
FROM posts p
JOIN users u ON p.author_id = u.id
WHERE p.is_published = true
ORDER BY p.published_at DESC
LIMIT $1;

-- name: GetPostWithAuthor :one
SELECT p.id, p.title, p.content, p.author_id, p.is_published, p.published_at, p.created_at,
       u.name as author_name, u.email as author_email
FROM posts p
JOIN users u ON p.author_id = u.id
WHERE p.id = $1;

-- name: GetUserPosts :many
SELECT id, title, content, author_id, is_published, published_at, created_at
FROM posts
WHERE author_id = $1
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
JOIN users u ON p.author_id = u.id
LEFT JOIN comments c ON p.id = c.post_id AND c.is_approved = true
WHERE p.is_published = true
GROUP BY p.id, p.title, p.author_id, p.published_at, p.created_at, u.name
ORDER BY p.published_at DESC
LIMIT $1; 