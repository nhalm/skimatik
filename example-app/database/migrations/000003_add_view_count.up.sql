-- Example migration: Add view count to posts
-- This demonstrates a common schema evolution pattern

-- Add view_count column to posts table
ALTER TABLE posts ADD COLUMN view_count INTEGER NOT NULL DEFAULT 0;

-- Add index for finding popular posts
CREATE INDEX idx_posts_view_count ON posts(view_count DESC) WHERE is_published = true;