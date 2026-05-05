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
--   version     INTEGER      NOT NULL
--   snapshot    JSONB        NOT NULL
--   valid_from  TIMESTAMPTZ  NOT NULL
--   valid_to    TIMESTAMPTZ  (NULL means the row is currently open)
--   + a regular index on (parent_id)
--   + a UNIQUE index on (parent_id, version)
--
-- The UNIQUE index on (parent_id, version) is the defensive backstop for the
-- audited Update CTE's `COALESCE(MAX(version), 0) + 1` pattern; the parent
-- row-lock taken by UPDATE serializes concurrent updates to the same parent,
-- but the unique constraint guarantees correctness even if that assumption
-- is ever wrong.
--
-- Two parents are audited here to demonstrate shape-coverage: `users` is a
-- simple flat row, `posts` adds an FK to users and a nullable timestamp
-- (`published_at`) so the JSONB pre/post-image carries a richer mix.

CREATE TABLE users_audit (
    id          UUID         PRIMARY KEY,
    parent_id   UUID         NOT NULL REFERENCES users(id),
    version     INTEGER      NOT NULL,
    snapshot    JSONB        NOT NULL,
    valid_from  TIMESTAMPTZ  NOT NULL,
    valid_to    TIMESTAMPTZ
);
CREATE INDEX idx_users_audit_parent_id ON users_audit (parent_id);
CREATE UNIQUE INDEX uq_users_audit_parent_id_version ON users_audit (parent_id, version);

CREATE TABLE posts_audit (
    id          UUID         PRIMARY KEY,
    parent_id   UUID         NOT NULL REFERENCES posts(id),
    version     INTEGER      NOT NULL,
    snapshot    JSONB        NOT NULL,
    valid_from  TIMESTAMPTZ  NOT NULL,
    valid_to    TIMESTAMPTZ
);
CREATE INDEX idx_posts_audit_parent_id ON posts_audit (parent_id);
CREATE UNIQUE INDEX uq_posts_audit_parent_id_version ON posts_audit (parent_id, version);
