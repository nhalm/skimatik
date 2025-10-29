-- Test queries for nullable parameter support

-- name: ListUsersWithOptionalFilters :many
-- param: $1 limit int
-- param: $2 is_active *bool
-- param: $3 name_filter *string
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users
WHERE ($2::boolean IS NULL OR is_active = $2)
  AND ($3::text IS NULL OR name ILIKE $3)
ORDER BY created_at DESC
LIMIT $1;

-- name: SearchPostsByDateRange :many
-- param: $1 author_id uuid.UUID
-- param: $2 start_date *time.Time
-- param: $3 end_date *time.Time
-- param: $4 is_published *bool
-- param: $5 limit int
SELECT id, title, content, author_id, is_published, created_at
FROM posts
WHERE author_id = $1
  AND ($2::timestamptz IS NULL OR created_at >= $2)
  AND ($3::timestamptz IS NULL OR created_at <= $3)
  AND ($4::boolean IS NULL OR is_published = $4)
ORDER BY created_at DESC
LIMIT $5;

-- name: GetUserByOptionalEmail :one
-- param: $1 id uuid.UUID
-- param: $2 email *string
SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users
WHERE id = $1
  AND ($2::text IS NULL OR email = $2);
