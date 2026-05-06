# Testing skimatik-generated Repositories

skimatik uses [pgxkit](https://github.com/nhalm/pgxkit)'s testing helpers to lock in the SQL the generator emits.

- **`AssertGolden`** — captures the per-run DB-event transcript and diffs it against a committed `testdata/golden/<name>.json` baseline. Catches changes in *which SQL skimatik emits*.
- **`AssertPlan`** — captures `EXPLAIN (FORMAT JSON, COSTS OFF)` and diffs against `testdata/plans/<name>.json`. Catches plan-shape regressions.

See pgxkit's [Testing Guide](https://github.com/nhalm/pgxkit/blob/main/docs/Testing-Guide.md) for the full upstream reference. This page covers the skimatik-specific patterns.

## Setup

Set `TEST_DATABASE_URL` to a writable Postgres. `pgxkit.RequireDB(t)` skips when it's unset.

```bash
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/test_db?sslmode=disable"
```

## Determinism — pass a counter-based ID generator

Generated `New<Repo>(idGen func() uuid.UUID)` mints UUIDs for every `Create`. Per-run UUIDs make baselines flake — so for golden/plan tests, inject a counter:

```go
func fixedIDGen() func() uuid.UUID {
    var counter uint64
    return func() uuid.UUID {
        counter++
        var u uuid.UUID
        binary.BigEndian.PutUint64(u[8:], counter)
        return u
    }
}
```

pgxkit normalizes UUID/timestamp/`id`-typed args automatically; everything else (free-text, JSON) flows through verbatim, so use constants.

## Worked example

A complete file users can copy. The same shape works for `EnableAssertPlan` / `AssertPlan` — swap the calls and the test name.

```go
//go:build integration

package repository

import (
    "context"
    "encoding/binary"
    "testing"

    "github.com/google/uuid"
    "github.com/nhalm/pgxkit/v2"
    "github.com/your/project/repository/generated"
)

func fixedIDGen() func() uuid.UUID {
    var counter uint64
    return func() uuid.UUID {
        counter++
        var u uuid.UUID
        binary.BigEndian.PutUint64(u[8:], counter)
        return u
    }
}

// seedUserID is the first UUID fixedIDGen() mints. Pre-clean by this so the
// scenario starts from a known state across runs.
const seedUserID = "00000000-0000-0000-0000-000000000001"

func preCleanSeedUser(t *testing.T, db *pgxkit.DB) {
    t.Helper()
    if _, err := db.Exec(context.Background(),
        `DELETE FROM users WHERE id = $1`, seedUserID); err != nil {
        t.Fatalf("pre-clean: %v", err)
    }
}

func TestUsersRepository_Golden(t *testing.T) {
    testDB := pgxkit.RequireDB(t)
    preCleanSeedUser(t, testDB.DB)
    t.Cleanup(func() { preCleanSeedUser(t, testDB.DB) })

    golden := testDB.EnableGolden("TestUsersRepository_Golden")
    repo := generated.NewUsersRepository(fixedIDGen())
    ctx := context.Background()

    created, err := repo.Create(ctx, golden, generated.CreateUsersParams{
        Name:  "Golden Test User",
        Email: "golden-test@example.com",
    })
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    if _, err := repo.Get(ctx, golden, created.Id); err != nil {
        t.Fatalf("Get: %v", err)
    }

    golden.AssertGolden(t, "TestUsersRepository_Golden")
}
```

First run writes `testdata/golden/TestUsersRepository_Golden.json`; commit it. Subsequent runs assert against it. Regenerate intentionally with `go test -overwrite-golden` (or `-overwrite-plan` for plan tests).

The example-app integration suite at [`example-app/internal/repository/golden_test.go`](https://github.com/nhalm/skimatik/blob/main/example-app/internal/repository/golden_test.go) is the same pattern with `_Plan` alongside `_Golden`.

## Plan-regression caveats

Plans are PostgreSQL-version-sensitive — pin your test image. PG inlines parameter values into custom plans, so unstable args become unstable plan literals; `fixedIDGen()` covers UUID args. For statistics-driven plan flips (seq-scan ↔ bitmap-scan around a row-count threshold), pre-clean to a known state or pick a scenario where the plan is statistics-insensitive.

## What skimatik tests for itself

| Test | Type | What it locks in |
|---|---|---|
| `internal/generator/audit_runtime_integration_test.go::TestAuditCTE_Golden` | Golden | The CTE SQL the audited Create/Update templates render |
| `example-app/internal/repository/golden_test.go::TestUsersRepository_Golden` | Golden | `generated.UsersRepository.Create + Get` end-to-end |
| `example-app/internal/repository/golden_test.go::TestUsersRepository_Plan` | Plan | EXPLAIN plan of the same generated `Create + Get` |

## See also

- [Audit Tables](audit) — the worked example whose SQL `TestAuditCTE_Golden` locks in
- [Embedding Patterns](embedding-patterns) — how user wrappers compose with generated repos
- [pgxkit Testing Guide](https://github.com/nhalm/pgxkit/blob/main/docs/Testing-Guide.md)
