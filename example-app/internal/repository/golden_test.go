//go:build integration

package repository

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
	"github.com/nhalm/skimatik/v2/example-app/internal/repository/generated"
)

// newGoldenTestDB connects a *pgxkit.TestDB to TEST_DATABASE_URL or DATABASE_URL.
// EnableGolden / EnableAssertPlan are methods on *TestDB, so the regular
// *pgxkit.DB used by pagination_test.go is not enough.
func newGoldenTestDB(t *testing.T) *pgxkit.TestDB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:15987/blog?sslmode=disable"
	}

	testDB := pgxkit.NewTestDB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := testDB.Connect(ctx, dsn); err != nil {
		t.Skipf("Skipping test: could not connect to database: %v", err)
		return nil
	}

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = testDB.Shutdown(shutdownCtx)
	})

	return testDB
}

// fixedIDGen returns a closure that produces a deterministic sequence of
// UUIDs (00000000-0000-0000-0000-000000000001, ...000002, ...). Pass it to
// generated.NewUsersRepository so the parent-row UUID skimatik mints during
// Create is the same across runs — the prerequisite for a stable golden
// transcript. The audit-row UUID is still pgxkit-normalized by the default
// recorder, so it does not need to be deterministic.
func fixedIDGen() func() uuid.UUID {
	var counter uint64
	return func() uuid.UUID {
		counter++
		var u uuid.UUID
		binary.BigEndian.PutUint64(u[8:], counter)
		return u
	}
}

// TestUsersRepository_Golden locks in the DB-event transcript of a
// Create + Get scenario against the *generated* skimatik repository — no
// HTTP, no service-layer wrapper. The transcript contains exactly the SQL
// skimatik emits, the args it binds, and the rows it scans back.
//
// A diff against the committed baseline at testdata/golden/TestUsersRepository_Golden.json
// catches any change to:
//   - The audited Create CTE skimatik renders for the users table
//   - The Get-by-id query skimatik renders
//   - The columns either statement returns
//
// Determinism: the parent users.id is supplied by fixedIDGen(); the audit
// row id, the params Email, and any returned timestamps/UUIDs are
// normalized by pgxkit's default normalizers.
func TestUsersRepository_Golden(t *testing.T) {
	testDB := newGoldenTestDB(t)

	// Pre-clean any leftover row at the deterministic ID so the scenario
	// starts from a known state. fixedIDGen() emits 00000000-...-000001 first.
	const seedID = "00000000-0000-0000-0000-000000000001"
	preClean := func() {
		_, err := testDB.Exec(context.Background(),
			`DELETE FROM users_audit WHERE parent_id = $1`, seedID)
		if err != nil {
			t.Fatalf("pre-clean users_audit: %v", err)
		}
		_, err = testDB.Exec(context.Background(),
			`DELETE FROM users WHERE id = $1`, seedID)
		if err != nil {
			t.Fatalf("pre-clean users: %v", err)
		}
	}
	preClean()
	t.Cleanup(preClean)

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
	if created == nil {
		t.Fatalf("Create returned nil user")
	}

	fetched, err := repo.Get(ctx, golden, created.Id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched == nil {
		t.Fatalf("Get returned nil user")
	}

	golden.AssertGolden(t, "TestUsersRepository_Golden")
}

// resetPlanCapture removes the per-run plan capture file (NOT the
// .baseline) before EnableAssertPlan re-records. Works around pgxkit's
// behavior of appending to the capture across runs, which would otherwise
// produce doubled-up plan counts on the second invocation.
func resetPlanCapture(t *testing.T, testName string) {
	t.Helper()
	path := fmt.Sprintf("testdata/plans/%s.json", testName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("reset plan capture %s: %v", path, err)
	}
}

// TestUsersRepository_Plan locks in the structural EXPLAIN plan of the
// Create + Get scenario. A diff against the committed baseline at
// testdata/plans/TestUsersRepository_Plan.json.baseline catches plan-shape
// regressions: seq-scan vs index-scan, nested-loop vs hash-join, new sort
// nodes, join-order changes.
//
// Determinism: same fixedIDGen() and pre-clean as the golden test, so the
// per-run UUIDs that would otherwise inline into the plan's filter
// literals (PG's custom-plan inlining) are now stable across runs. Plans
// remain PostgreSQL-version-sensitive — pin the test image.
func TestUsersRepository_Plan(t *testing.T) {
	testDB := newGoldenTestDB(t)

	const seedID = "00000000-0000-0000-0000-000000000001"
	preClean := func() {
		_, err := testDB.Exec(context.Background(),
			`DELETE FROM users_audit WHERE parent_id = $1`, seedID)
		if err != nil {
			t.Fatalf("pre-clean users_audit: %v", err)
		}
		_, err = testDB.Exec(context.Background(),
			`DELETE FROM users WHERE id = $1`, seedID)
		if err != nil {
			t.Fatalf("pre-clean users: %v", err)
		}
	}
	preClean()
	t.Cleanup(preClean)

	resetPlanCapture(t, "TestUsersRepository_Plan")
	plan := testDB.EnableAssertPlan("TestUsersRepository_Plan")
	repo := generated.NewUsersRepository(fixedIDGen())

	ctx := context.Background()

	created, err := repo.Create(ctx, plan, generated.CreateUsersParams{
		Name:  "Plan Test User",
		Email: "plan-test@example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Get(ctx, plan, created.Id); err != nil {
		t.Fatalf("Get: %v", err)
	}

	plan.AssertPlan(t, "TestUsersRepository_Plan")
}
