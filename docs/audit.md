# Audit Tables (SCD Type 2 History)

## Overview

Skimatik can generate Go repositories that maintain an SCD Type 2 audit history alongside any parent table. When a table is opted in via `audit: true`, every Create and Update on that table is rewritten as a single PostgreSQL CTE so the parent row and a row in the companion `<table>_audit` table are written atomically. The resulting history records the full pre- and post-image of every mutation; each version of a row carries a monotonically increasing `version` counter and is delimited by a `valid_from`/`valid_to` window.

The audit child is a normal table you own: skimatik validates its shape but does not create or migrate it. This keeps audit history visible to your migration tooling, your DBAs, and any downstream consumers (analytics, archives, GDPR exports) that already consume your schema.

## Configuration

Set `audit: true` on any parent table in `skimatik.yaml`:

```yaml
tables:
  users:
    audit: true
  posts:
    audit: true
  comments:  # not audited; standard CRUD is generated
```

The flag is per-table. Default-functions (`create`, `get`, `update`, `delete`, `list`, `paginate`) still apply — `audit: true` only changes how `create` and `update` are emitted. Get / List / Paginate are unchanged.

## Required Audit Table

You own the audit table. Add it through your normal migration tooling. The shape below is the contract enforced by skimatik's pre-flight validator (see [Strict validation](#strict-validation)):

```sql
CREATE TABLE <parent>_audit (
  id          UUID         PRIMARY KEY,
  parent_id   <parent_pk>  NOT NULL REFERENCES <parent>(<pk_col>),
  version     INTEGER      NOT NULL,
  snapshot    JSONB        NOT NULL,
  valid_from  TIMESTAMPTZ  NOT NULL,
  valid_to    TIMESTAMPTZ
);
CREATE INDEX ON <parent>_audit (parent_id);
CREATE UNIQUE INDEX ON <parent>_audit (parent_id, version);
```

Notes:
- `parent_id` must match the parent's primary key type. For a `BIGINT`-keyed parent, `parent_id` is `BIGINT`; for `UUID`-keyed it is `UUID`.
- `version` is a monotonically increasing 1-based counter scoped to a single `parent_id`. Create writes `version = 1`; each Update writes `version = MAX(prior) + 1`.
- `snapshot` carries the post-image of the parent row as JSONB (`to_jsonb(parent.*)`).
- `valid_to IS NULL` means the row is the currently open version; on Update, the previously open row's `valid_to` is set to `NOW()` and a new open row is inserted.
- The leading-column index on `parent_id` is required because every audit lookup filters by it.
- The UNIQUE index on `(parent_id, version)` is required as a defensive backstop. The audited Update CTE relies on the parent UPDATE row-locking the parent row to serialize concurrent updates; the unique constraint guarantees correctness even if that assumption is ever wrong (and surfaces a hard error rather than silently double-numbering).

Skimatik does **not** generate this DDL. The application owns the migration so the audit table participates fully in your schema-management workflow.

## What Gets Generated

For an audited parent table, skimatik emits CTE-based mutations:

**Create** — single statement, parent INSERT and audit INSERT share `NOW()`. Initial `version` is hardcoded to `1`:

```sql
WITH inserted AS (
    INSERT INTO users (id, name, email, ...)
    VALUES ($1, $2, $3, ...)
    RETURNING id, name, email, ...
),
audited AS (
    INSERT INTO users_audit (id, parent_id, version, snapshot, valid_from)
    SELECT gen_random_uuid(), id, 1, to_jsonb(inserted.*), NOW() FROM inserted
)
SELECT id, name, email, ... FROM inserted
```

**Update** — single statement, closes the prior open audit row, applies the parent UPDATE, opens a new audit row with `version = MAX(prior) + 1`, all sharing one statement-level `NOW()`:

```sql
WITH closed AS (
    UPDATE users_audit
    SET valid_to = NOW()
    WHERE parent_id = $1 AND valid_to IS NULL
),
updated AS (
    UPDATE users
    SET name = $2, email = $3, ...
    WHERE id = $1
    RETURNING id, name, email, ...
),
audited AS (
    INSERT INTO users_audit (id, parent_id, version, snapshot, valid_from)
    SELECT gen_random_uuid(), updated.id,
           COALESCE((SELECT MAX(version) FROM users_audit WHERE parent_id = updated.id), 0) + 1,
           to_jsonb(updated.*),
           NOW()
    FROM updated
)
SELECT id, name, email, ... FROM updated
```

The `COALESCE(MAX(version), 0) + 1` pattern is concurrency-safe because the parent UPDATE in the same statement row-locks the parent row, serializing concurrent updates to the same `parent_id`. By the time a second transaction's MAX subquery runs, the first transaction has committed and its audit row is visible. The UNIQUE index on `(parent_id, version)` is the defensive backstop in case the lock assumption is ever wrong.

**Get / List / Paginate**: unchanged. They read the parent table only.

**Custom `.sql` queries** (`:one`, `:many`, `:paginated`, `:exec`): unchanged. Audit only rewrites the table-driven CRUD templates.

**Delete**: generated normally. Whether deletion is permitted on an audited table is a database-level concern enforced via PostgreSQL roles (`REVOKE DELETE`, row-level security, or restricted application credentials). Skimatik does not opinionate this — if you need delete-prevention, do it where it belongs, in the database.

## Strict Validation

Before any code is generated, skimatik validates each audited parent's companion `<parent>_audit` table. Validation runs as a pre-flight gate: if any audited table fails, **no file is written**. Errors across all audited tables are aggregated into a single message so you can fix everything in one pass; the canonical DDL is appended for each failing parent so you can copy-paste a working schema.

The validator checks:

- The `<parent>_audit` table exists.
- It has the six required columns: `id`, `parent_id`, `version`, `snapshot`, `valid_from`, `valid_to`.
- Each column has the expected type and nullability (`id`, `parent_id`, `version`, `snapshot`, `valid_from` are NOT NULL; `valid_to` is NULL).
- `id` is the primary key.
- `parent_id` carries a foreign-key constraint referencing `<parent>(<pk_col>)`.
- An index leads with `parent_id`.
- A UNIQUE index covers `(parent_id, version)`.

The validator is permissive about extra columns — your audit table is allowed to have additional columns (a `who` field, a `request_id`, etc.). Only the canonical contract is enforced.

## Constraints and Known Limits

- **One audit per parent.** A parent table has at most one companion audit table; the `<parent>_audit` naming is fixed and not configurable.
- **`gen_random_uuid()` is required.** Audit row IDs are generated via `gen_random_uuid()`, which is built into PostgreSQL 13+ (no extension needed).
- **Single-column primary keys only.** Audit follows skimatik's general constraint that parent tables must have a single-column primary key.
- **Audit rows are append-only by design.** The Update CTE never UPDATEs `snapshot` on a prior row — it only sets `valid_to`. The post-image lives on a fresh row.
- **`:paginated`, `:one`, `:many`, `:exec` queries are not audited.** Custom `.sql` queries do not flow through CRUD templates and are emitted unchanged.

## Example-App Demonstration

The example-app demonstrates audit end-to-end on two parent tables with different shapes:

- Migration: [`example-app/internal/database/migrations/000005_add_audit_tables.up.sql`](https://github.com/nhalm/skimatik/tree/main/example-app/internal/database/migrations/000005_add_audit_tables.up.sql) — creates `users_audit` and `posts_audit`.
- Configuration: [`example-app/skimatik.yaml`](https://github.com/nhalm/skimatik/tree/main/example-app/skimatik.yaml) — sets `audit: true` on `users` and `posts`.
- Generated code: `example-app/internal/repository/generated/users_generated.go` and `posts_generated.go` (regenerated on `make generate`) — contain the CTE-based Create and Update.
- Runtime exercise: `example-app/Makefile`'s `test` target — POSTs a new user, PATCHes the name, then GETs `/api/users/{id}/audit` and asserts that exactly two audit rows exist (one closed, one open) carrying the original and updated payloads.

To run the demonstration locally:

```bash
cd example-app
make test
```

## Related Documentation

- [Configuration Reference](configuration-reference) — full `skimatik.yaml` schema.
- [Embedding Patterns](embedding-patterns) — composing repositories that wrap audited generated code.
- [Database Migrations](database-migrations) — managing your own schema, including audit tables, with golang-migrate.
