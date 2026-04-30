# blueprint-vet integration

[blueprint-vet](https://github.com/nhalm/blueprint-vet) is a Go static analyzer + SQL linter that enforces conformance rules for services built on the [go-blueprint](https://github.com/nhalm/blueprint) layout. skimatik runs both binaries as part of `make lint` and CI to keep generated code and the example-app aligned with those rules.

## What gets checked

`make lint` runs:

- `blueprint-vet -nofmtprint=false ./...` against the **skimatik root module** (the generator itself).
- `blueprint-vet -nofmtprint=false ./...` against the **example-app module** (consumer + generated code).
- `blueprint-sql-check example-app/database/queries` against the SQL query annotation files.

Same three steps run in `.github/workflows/ci.yml`'s lint job.

## Rules in effect

| ID | Rule | Where it applies |
|----|------|------------------|
| R-5 | `repoexecutor` | wrapper repos in `example-app/internal/repository/` must pass `executorFromContext(ctx, r.db)` to generated methods, not `r.db` directly |
| R-9 | `softdelete` | every `:one`/`:many`/`:paginated` query in `example-app/database/queries/*.sql` references `deleted_at` |
| R-10 | `paginatedorderby` | every `:paginated` query has `ORDER BY` (skimatik already requires this for cursor stability) |
| R-12 | `errortranslate` | wrapper repo methods that call generated code do not return bare `err` — they wrap with `fmt.Errorf("...: %w", err)` (or `translateError(err)` if you adopt that helper) |

R-6 (`idtypeuuid`) is satisfied automatically because skimatik uses `uuid.UUID` for ID columns.

## Why `-nofmtprint=false`

R-8 (`nofmtprint`) bans the entire `fmt.Print*/Sprint*/Fprint*` family on the assumption the call is a runtime log that should go through canonlog. skimatik is a code generator: `fmt.Fprintf(&code, ...)` is how the generator writes Go source, and generated cursor helpers use `fmt.Sprintf("%v", value)` for value formatting. None of that is logging, so the rule does not apply. CI and `make lint` keep R-8 disabled for both modules.

If you adopt skimatik in a service that genuinely uses canonlog, leave R-8 on for your own packages — only the generator and generated code need the exemption.

## The wrapper executor pattern

Generated repository methods take `pgxkit.Executor` as the second parameter. Wrappers must route through `executorFromContext` so the same wrapper instance works inside or outside a transaction:

```go
// example-app/internal/repository/executor.go
type txKey struct{}

func WithTx(ctx context.Context, tx pgxkit.Executor) context.Context {
    return context.WithValue(ctx, txKey{}, tx)
}

func executorFromContext(ctx context.Context, db pgxkit.Executor) pgxkit.Executor {
    if tx, ok := ctx.Value(txKey{}).(pgxkit.Executor); ok && tx != nil {
        return tx
    }
    return db
}
```

Wrapper call site:

```go
// Bad — bypasses an active *pgxkit.Tx attached to ctx.
results, err := r.UsersQueries.GetActiveUsers(ctx, r.db, limit)

// Good — picks up the transaction from context if present.
results, err := r.UsersQueries.GetActiveUsers(ctx, executorFromContext(ctx, r.db), limit)
```

R-5 only fires on receivers whose type name ends in `Repository`. Service types (`*BlogService`, etc.) can pass `tx` directly because they hold the transaction explicitly.

## Soft-delete column convention

R-9 expects every read-shape query (`:one`, `:many`, `:paginated`) to reference a `deleted_at` column. example-app's migration `000004_add_deleted_at.up.sql` adds nullable `deleted_at TIMESTAMPTZ` to `users`, `posts`, `comments`, plus partial indexes on `deleted_at IS NULL`. Queries filter with `AND deleted_at IS NULL` (or join on it).

Opt-outs (when a query genuinely needs to read deleted rows):

- Suffix the query name with `IncludingDeleted`, `Audit`, `Trash`, or `AllVersions`.
- Or add `-- blueprint-vet:skip softdelete` in the SQL file.

## Path scoping

blueprint-vet's analyzers key off path substrings: `/internal/api`, `/internal/models`, `/internal/repository/generated`, `/internal/errors`, `/internal/config`, `/cmd/`. example-app keeps generated code at `internal/repository/generated/` and wrappers at `internal/repository/` so R-5 and R-12 fire. `api/`, `domain/`, and `service/` stay at the top level — moving them under `internal/` would activate R-1/R-2/R-6/R-11 and is left to consumers who follow the full blueprint layout.

## Versions

Pinned to `v0.1.0` in both `Makefile` (`BLUEPRINT_VET_VERSION`) and `.github/workflows/ci.yml`. Bump both together.
