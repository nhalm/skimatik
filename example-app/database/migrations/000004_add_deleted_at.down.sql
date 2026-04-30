DROP INDEX IF EXISTS idx_comments_deleted_at;
DROP INDEX IF EXISTS idx_posts_deleted_at;
DROP INDEX IF EXISTS idx_users_deleted_at;

ALTER TABLE comments DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE posts    DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users    DROP COLUMN IF EXISTS deleted_at;
