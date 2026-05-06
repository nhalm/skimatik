# Testing skimatik-generated Repositories

skimatik leans on [pgxkit](https://github.com/nhalm/pgxkit) for testing infrastructure. This page covers two complementary patterns for asserting that the SQL skimatik emits stays correct over time:

- **Golden testing** — captures the full DB-event transcript (BEGIN, every Query/Exec with sql/args/rows, COMMIT/ROLLBACK) and diffs against a committed baseline. Catches changes in *which SQL skimatik emits and what it returns*.
- **Plan-regression testing** — captures `EXPLAIN (FORMAT JSON, COSTS OFF)` and diffs against a committed baseline. Catches changes in *plan shape* (seq-scan vs index-scan, nested-loop vs hash-join, new sort nodes, join-order changes).

Both are opt-in. You enable them on a `*pgxkit.TestDB` per scenario; the rest of your tests are unaffected.

## When to use which

| You want to detect… | Use |
|---|---|
| "did skimatik change the SQL it emits / which columns it returns" | Golden |
| "did the planner stop using my index" | Plan |
| "did the new generator version break my repo's contract end-to-end" | Both |

They are not mutually exclusive — pick one or both per scenario.

## Test database setup

Set `TEST_DATABASE_URL` to a writable Postgres before running integration tests. `pgxkit.RequireDB(t)` skips the test cleanly when the variable is unset, so the tests are no-ops on developer machines without a test DB.

```bash
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/test_db?sslmode=disable"
```

```go
testDB := pgxkit.RequireDB(t)
```

The test DB should be migrated to the same schema as production. skimatik's example-app runs migrations from `database/migrations/` via `make setup`.

---

## Golden testing

### What it captures

For every `Query`, `QueryRow`, and `Exec` issued through a `*DB` returned by `EnableGolden`, the recorder writes a step entry containing:

- the SQL text
- the bound args (post-normalization)
- the result rows for `Query` / `QueryRow` (post-normalization)
- the rows-affected count for `Exec`

Wrapping `BEGIN` / `COMMIT` / `ROLLBACK` events are recorded too, so a multi-statement transaction shows up as a single ordered transcript. The transcript is written to `testdata/golden/<test-name>.json`.

### Determinism — the only thing you have to think about

Golden assertions only catch real regressions if the captured transcript is *byte-for-byte stable across runs*. pgxkit's default normalizer handles the common drift sources automatically:

| Source of drift | Default behavior |
|---|---|
| `time.Time` columns/args | Replaced with `<TIMESTAMP>` |
| `uuid.UUID` columns/args | Replaced with `<UUID:N>` (first-seen, per scenario) |
| Integer columns named `id` or `*_id` | Replaced with `<ID:N>` |

Anything else — strings, JSONB blobs, custom types — flows through verbatim. Two patterns keep things stable:

1. **Use constants** for any non-ID arg the test fully controls (names, emails, hashes). Don't bake `time.Now().UnixNano()` into emails or other text fields — that text is not a UUID/timestamp, so the default normalizer won't touch it.
2. **Inject a deterministic ID generator** into the generated repository constructor. skimatik-generated `New<Repo>(idGen func() uuid.UUID)` accepts a UUID factory; passing one that returns a fixed sequence (e.g. `00000000-0000-0000-0000-000000000001`, `…000002`, …) keeps the IDs the *generator* mints stable across runs:

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

If you have a column whose value can't be made deterministic by either of the above (e.g. a free-text field that legitimately varies), register a custom normalizer:

```go
golden := testDB.EnableGolden("TestX", pgxkit.WithGoldenNormalizer(func(v any) (any, bool) {
    if s, ok := v.(string); ok && strings.HasPrefix(s, "session-") {
        return "<SESSION>", true
    }
    return nil, false
}))
```

### Workflow

```go
//go:build integration

func TestUsersRepository_Golden(t *testing.T) {
    testDB := pgxkit.RequireDB(t)

    // Fresh state — pre-clean rows the test will create.
    const seedID = "00000000-0000-0000-0000-000000000001"
    cleanup := func() { /* DELETE FROM users_audit, users WHERE id = seedID */ }
    cleanup()
    t.Cleanup(cleanup)

    golden := testDB.EnableGolden("TestUsersRepository_Golden")
    repo := generated.NewUsersRepository(fixedIDGen())

    ctx := context.Background()
    user, err := repo.Create(ctx, golden, generated.CreateUsersParams{
        Name:  "Golden Test User",
        Email: "golden-test@example.com",
    })
    require.NoError(t, err)

    fetched, err := repo.Get(ctx, golden, user.Id)
    require.NoError(t, err)
    _ = fetched

    golden.AssertGolden(t, "TestUsersRepository_Golden")
}
```

First run: creates `testdata/golden/TestUsersRepository_Golden.json` and logs `created golden baseline: …`. **Commit that file.**

Subsequent runs: read the committed baseline and `t.Errorf` with a unified diff if the transcript changed.

### Regenerating the baseline

When you make an intentional change (e.g. you upgrade skimatik and the generator emits an extra column in `RETURNING`):

```bash
go test -tags=integration -run TestUsersRepository_Golden -overwrite-golden ./...
```

`-overwrite-golden` rewrites *only* baselines for tests that actually run, so you can target a single test and review its diff in isolation.

---

## Plan-regression testing

### What it captures

For every eligible statement (SELECT, INSERT, UPDATE, DELETE, WITH …) issued through a `*DB` returned by `EnableAssertPlan`, the recorder issues `EXPLAIN (FORMAT JSON, COSTS OFF) <sql>` against the same args and writes the resulting plan to `testdata/plans/<test-name>.json`. `AssertPlan` then diffs against `testdata/plans/<test-name>.json.baseline`.

Because the EXPLAIN doesn't `ANALYZE`, the underlying statement is not executed — so plan capture has no side effects.

### Workflow

```go
plan := testDB.EnableAssertPlan("TestUsersRepository_Plan")
repo := generated.NewUsersRepository(fixedIDGen())
// … run scenario against `plan` …
plan.AssertPlan(t, "TestUsersRepository_Plan")
```

First run creates `testdata/plans/TestUsersRepository_Plan.json.baseline` — **commit that.** The non-baseline `.json` capture is overwritten on each run; gitignore it.

### Determinism caveats

Plans are sensitive to:

1. **PostgreSQL version.** Pin your test DB to one major version. Plans differ between PG14 / PG15 / PG16.
2. **Bound parameter values.** PostgreSQL inlines parameter values into custom plans, so a `WHERE id = $1` filter shows the actual UUID/integer literal in the plan. Use a deterministic ID generator (same trick as for golden) so the literal is stable.
3. **Table statistics.** The planner picks bitmap-scan vs seq-scan based on row count. If a scenario's plan is borderline, accumulated test data can flip the choice. For a small, well-isolated scenario this rarely matters; for large or shared tables, either pre-clean to a known state or pick scenarios where the plan is statistics-insensitive.

---

## What skimatik tests for itself

| Test | Type | What it locks in |
|---|---|---|
| `internal/generator/audit_runtime_integration_test.go::TestAuditCTE_Golden` | Golden | The exact CTE SQL the audited Create/Update templates render — a template-correctness regression test |
| `example-app/internal/repository/golden_test.go::TestUsersRepository_Golden` | Golden | The full transcript of `generated.UsersRepository.Create + Get` end-to-end — exercises the actual generated repository, not the rendered template string |
| `example-app/internal/repository/golden_test.go::TestUsersRepository_Plan` | Plan | The structural EXPLAIN plan of the same generated `Create + Get` flow |
| Generated `Test<Struct>Repository_Golden` / `_Plan` (in `templates/tests/repository_test.tmpl`) | Both | Per-table `Create + Get` for users who adopt the test template in their own project |

---

## See also

- [Audit Tables](audit) — the worked example whose SQL `TestAuditCTE_Golden` locks in
- [Embedding Patterns](embedding-patterns) — how user wrappers compose with generated repos
- [pgxkit Testing Guide](https://github.com/nhalm/pgxkit/blob/main/docs/Testing-Guide.md) — upstream reference for `EnableGolden` / `EnableAssertPlan`
