# Audit Tables (SCD Type 2 History)

## Overview

Skimatik can generate Go repositories that maintain an SCD Type 2 audit history alongside any parent table. When a table is opted in via `audit: true`, every Create and Update on that table is rewritten as a single PostgreSQL CTE so the parent row and a row in the companion `<table>_audit` table are written atomically. The resulting history records the full pre- and post-image of every mutation, with each version of a row delimited by a `start_date`/`end_date` window.

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
  data        JSONB        NOT NULL,
  start_date  TIMESTAMPTZ  NOT NULL,
  end_date    TIMESTAMPTZ
);
CREATE INDEX ON <parent>_audit (parent_id);
```

Notes:
- `parent_id` must match the parent's primary key type. For a `BIGINT`-keyed parent, `parent_id` is `BIGINT`; for `UUID`-keyed it is `UUID`.
- `data` carries the post-image of the parent row as JSONB (`to_jsonb(parent.*)`).
- `end_date IS NULL` means the row is the currently open version; on Update, the previously open row's `end_date` is set to `NOW()` and a new open row is inserted.
- The leading-column index on `parent_id` is required because every audit lookup filters by it.

Skimatik does **not** generate this DDL. The application owns the migration so the audit table participates fully in your schema-management workflow.

## What Gets Generated

For an audited parent table, skimatik emits CTE-based mutations:

**Create** — single statement, parent INSERT and audit INSERT share `NOW()`:

```sql
WITH inserted AS (
    INSERT INTO users (id, name, email, ...)
    VALUES ($1, $2, $3, ...)
    RETURNING id, name, email, ...
),
audited AS (
    INSERT INTO users_audit (id, parent_id, data, start_date)
    SELECT gen_random_uuid(), id, to_jsonb(inserted.*), NOW() FROM inserted
)
SELECT id, name, email, ... FROM inserted
```

**Update** — single statement, closes the prior open audit row, applies the parent UPDATE, opens a new audit row, all sharing one statement-level `NOW()`:

```sql
WITH closed AS (
    UPDATE users_audit
    SET end_date = NOW()
    WHERE parent_id = $1 AND end_date IS NULL
),
updated AS (
    UPDATE users
    SET name = $2, email = $3, ...
    WHERE id = $1
    RETURNING id, name, email, ...
),
audited AS (
    INSERT INTO users_audit (id, parent_id, data, start_date)
    SELECT gen_random_uuid(), id, to_jsonb(updated.*), NOW() FROM updated
)
SELECT id, name, email, ... FROM updated
```

**Get / List / Paginate**: unchanged. They read the parent table only.

**Custom `.sql` queries** (`:one`, `:many`, `:paginated`, `:exec`): unchanged. Audit only rewrites the table-driven CRUD templates.

**Delete**: generated normally. Whether deletion is permitted on an audited table is a database-level concern enforced via PostgreSQL roles (`REVOKE DELETE`, row-level security, or restricted application credentials). Skimatik does not opinionate this — if you need delete-prevention, do it where it belongs, in the database.

## Strict Validation

Before any code is generated, skimatik validates each audited parent's companion `<parent>_audit` table. Validation runs as a pre-flight gate: if any audited table fails, **no file is written**. Errors across all audited tables are aggregated into a single message so you can fix everything in one pass; the canonical DDL is appended for each failing parent so you can copy-paste a working schema.

The validator checks:

- The `<parent>_audit` table exists.
- It has exactly the five required columns: `id`, `parent_id`, `data`, `start_date`, `end_date`.
- Each column has the expected type and nullability (`id` and `parent_id` and `data` and `start_date` are NOT NULL; `end_date` is NULL).
- `id` is the primary key.
- `parent_id` carries a foreign-key constraint referencing `<parent>(<pk_col>)`.
- An index leads with `parent_id`.

The validator is permissive about extra columns — your audit table is allowed to have additional columns (a `who` field, a `request_id`, etc.). Only the canonical contract is enforced.

## Constraints and Known Limits

- **One audit per parent.** A parent table has at most one companion audit table; the `<parent>_audit` naming is fixed and not configurable.
- **`gen_random_uuid()` is required.** Audit row IDs are generated via `gen_random_uuid()`, which is built into PostgreSQL 13+ (no extension needed).
- **Single-column primary keys only.** Audit follows skimatik's general constraint that parent tables must have a single-column primary key.
- **Audit rows are append-only by design.** The Update CTE never UPDATEs `data` on a prior row — it only sets `end_date`. The post-image lives on a fresh row.
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
