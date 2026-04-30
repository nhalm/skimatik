# blueprint-vet integration

[blueprint-vet](https://github.com/nhalm/blueprint-vet) is a Go static analyzer + SQL linter that enforces conformance rules for services built on the [go-blueprint](https://github.com/nhalm/blueprint) layout. Since blueprint-vet `v0.2.0`, the Go AST analyzers ship as a [golangci-lint Module Plugin](https://golangci-lint.run/plugins/module-plugins/), so skimatik runs them inside `golangci-lint` rather than as a separate binary.

## How it's wired

Two config files drive the integration:

- **`.custom-gcl.yml`** — pins golangci-lint version and the blueprint-vet plugin. `golangci-lint custom` reads this file and produces `./bin/custom-gcl`, a custom golangci-lint binary that bundles the plugin.
- **`.golangci.yml`** — enables `blueprint-vet` alongside the stdlib linters under `linters.settings.custom`.

`make lint` runs:

- `./bin/custom-gcl run ./...` against the **skimatik root module** — runs all stdlib linters plus blueprint-vet's R-1..R-7 / R-11 / R-12 in a single pass.
- `./bin/custom-gcl run ./...` against the **example-app module** (when generated code is present).
- `blueprint-sql-check example-app/database/queries` against the SQL query annotation files. (SQL files aren't lintable by golangci-lint, so this stays a separate binary.)

`.github/workflows/ci.yml`'s lint job mirrors the same steps. The custom-gcl binary builds once per CI run via `golangci-lint custom`.

## Rules in effect

| ID | Rule | Where it applies |
|----|------|------------------|
| R-5 | `repoexecutor` | wrapper repos in `example-app/internal/repository/` must pass `executorFromContext(ctx, r.db)` to generated methods, not `r.db` directly |
| R-9 | `softdelete` | every `:one`/`:many`/`:paginated` query in `example-app/database/queries/*.sql` references `deleted_at` |
| R-10 | `paginatedorderby` | every `:paginated` query has `ORDER BY` (skimatik already requires this for cursor stability) |
| R-12 | `errortranslate` | wrapper repo methods that call generated code do not return bare `err` — they wrap with `fmt.Errorf("...: %w", err)` (or `translateError(err)` if you adopt that helper) |

R-6 (`idtypeuuid`) is satisfied automatically because skimatik uses `uuid.UUID` for ID columns.

R-8 (`nofmtprint`) runs unmodified. The generator assembles source files via `text/template` (see `internal/generator/templates/shared/file.tmpl`) and any user-facing logging happens in `cmd/skimatik/main.go`, which R-8 exempts.

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

Three version pins, all bumped together:

- `.custom-gcl.yml` → `version: v2.11.4` (golangci-lint base)
- `.custom-gcl.yml` → blueprint-vet plugin `version: v0.2.0`
- `Makefile` / `.github/workflows/ci.yml` → `BLUEPRINT_VET_VERSION` for the standalone `blueprint-sql-check` binary (kept on the same release tag as the plugin for consistency).
