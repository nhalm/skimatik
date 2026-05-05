-- Audit companion tables for users and posts.
-- Skimatik's `audit: true` flag emits CTE-based Create/Update on the parent
-- table; the application owns the audit child schema. The shape below is the
-- canonical contract enforced by skimatik's pre-flight validator.

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
