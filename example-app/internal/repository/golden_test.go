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

// newGoldenTestDB returns a connected *pgxkit.TestDB. EnableGolden and
// EnableAssertPlan are methods on *TestDB, so the *DB used by pagination_test.go
// is not enough.
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

// fixedIDGen returns a counter-based UUID factory so the parent-row IDs
// skimatik mints are stable across runs (required for stable golden + plan
// baselines).
func fixedIDGen() func() uuid.UUID {
	var counter uint64
	return func() uuid.UUID {
		counter++
		var u uuid.UUID
		binary.BigEndian.PutUint64(u[8:], counter)
		return u
	}
}

const seedUserID = "00000000-0000-0000-0000-000000000001"

func preCleanSeedUser(t *testing.T, db *pgxkit.DB) {
	t.Helper()
	if _, err := db.Exec(context.Background(),
		`DELETE FROM users_audit WHERE parent_id = $1`, seedUserID); err != nil {
		t.Fatalf("pre-clean users_audit: %v", err)
	}
	if _, err := db.Exec(context.Background(),
		`DELETE FROM users WHERE id = $1`, seedUserID); err != nil {
		t.Fatalf("pre-clean users: %v", err)
	}
}

func TestUsersRepository_Golden(t *testing.T) {
	testDB := newGoldenTestDB(t)
	preCleanSeedUser(t, testDB.DB)
	t.Cleanup(func() { preCleanSeedUser(t, testDB.DB) })

	golden := testDB.EnableGolden("TestUsersRepository_Golden")
	repo := generated.NewUsersRepository(fixedIDGen())
	ctx := context.Background()

	now := time.Now()
	created, err := repo.Create(ctx, golden, generated.CreateUsersParams{
		Name:      "Golden Test User",
		Email:     "golden-test@example.com",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Get(ctx, golden, created.Id); err != nil {
		t.Fatalf("Get: %v", err)
	}

	golden.AssertGolden(t, "TestUsersRepository_Golden")
}

func TestUsersRepository_Plan(t *testing.T) {
	testDB := newGoldenTestDB(t)
	preCleanSeedUser(t, testDB.DB)
	t.Cleanup(func() { preCleanSeedUser(t, testDB.DB) })

	plan := testDB.EnableAssertPlan("TestUsersRepository_Plan")
	repo := generated.NewUsersRepository(fixedIDGen())
	ctx := context.Background()

	now := time.Now()
	created, err := repo.Create(ctx, plan, generated.CreateUsersParams{
		Name:      "Plan Test User",
		Email:     "plan-test@example.com",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Get(ctx, plan, created.Id); err != nil {
		t.Fatalf("Get: %v", err)
	}

	plan.AssertPlan(t, "TestUsersRepository_Plan")
}
