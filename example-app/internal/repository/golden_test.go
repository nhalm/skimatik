//go:build integration

package repository

import (
	"context"
	"encoding/binary"
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
//
// NOTE: currently blocked by https://github.com/nhalm/pgxkit/issues/77 —
// pgxkit v2.1.0's golden recorder cannot replay UUID columns through Scan
// (returns "unable to scan type [16]uint8 into UUID"). The Skip below comes
// out once that issue is resolved and pgxkit is re-bumped in this branch.
func TestUsersRepository_Golden(t *testing.T) {
	t.Skip("blocked on pgxkit#77 — golden replay cannot Scan UUID columns")

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
