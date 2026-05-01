-- Revert: Remove view count from posts

-- Drop the index first
DROP INDEX IF EXISTS idx_posts_view_count;

-- Remove the column
ALTER TABLE posts DROP COLUMN IF EXISTS view_count;