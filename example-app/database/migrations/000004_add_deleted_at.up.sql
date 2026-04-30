-- Add soft-delete columns
-- blueprint-vet R-9 (softdelete) requires SELECT queries to filter on deleted_at.
-- Rows with deleted_at IS NOT NULL are considered deleted and excluded from normal reads.

ALTER TABLE users    ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE posts    ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE comments ADD COLUMN deleted_at TIMESTAMPTZ;

CREATE INDEX idx_users_deleted_at    ON users(deleted_at)    WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_deleted_at    ON posts(deleted_at)    WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_deleted_at ON comments(deleted_at) WHERE deleted_at IS NULL;
