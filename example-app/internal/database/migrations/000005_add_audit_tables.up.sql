-- Add audit companion tables for users and posts.
--
-- Skimatik's `audit: true` flag generates CTE-based Create/Update on the
-- parent table that maintains an SCD Type 2 history in this <parent>_audit
-- child. Skimatik does NOT create these tables; the application owns the
-- migration. The shape below is the canonical contract enforced by
-- skimatik's pre-flight audit validator (issue #144):
--
--   id          UUID         PRIMARY KEY
--   parent_id   <parent_pk>  NOT NULL REFERENCES <parent>(<pk>)
--   data        JSONB        NOT NULL
--   start_date  TIMESTAMPTZ  NOT NULL
--   end_date    TIMESTAMPTZ  (NULL means the row is currently open)
--   + an index leading with parent_id
--
-- Two parents are audited here to demonstrate shape-coverage: `users` is a
-- simple flat row, `posts` adds an FK to users and a nullable timestamp
-- (`published_at`) so the JSONB pre/post-image carries a richer mix.

CREATE TABLE users_audit (
    id          UUID         PRIMARY KEY,
    parent_id   UUID         NOT NULL REFERENCES users(id),
    data        JSONB        NOT NULL,
    start_date  TIMESTAMPTZ  NOT NULL,
    end_date    TIMESTAMPTZ
);
CREATE INDEX idx_users_audit_parent_id ON users_audit (parent_id);

CREATE TABLE posts_audit (
    id          UUID         PRIMARY KEY,
    parent_id   UUID         NOT NULL REFERENCES posts(id),
    data        JSONB        NOT NULL,
    start_date  TIMESTAMPTZ  NOT NULL,
    end_date    TIMESTAMPTZ
);
CREATE INDEX idx_posts_audit_parent_id ON posts_audit (parent_id);
