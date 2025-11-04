//go:build !short

package generator

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nhalm/pgxkit"
)

func TestMain(m *testing.M) {
	// Setup: Create shared test database connection
	if os.Getenv("TEST_DATABASE_URL") != "" {
		db := pgxkit.NewDB()
		err := db.Connect(context.Background(), os.Getenv("TEST_DATABASE_URL"))
		if err != nil {
			panic("Failed to connect to test database: " + err.Error())
		}
		packageTestDB = db
	}

	// Run tests
	code := m.Run()

	// Teardown: Close database connection
	if packageTestDB != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := packageTestDB.Shutdown(shutdownCtx); err != nil {
			panic("Failed to shutdown test database: " + err.Error())
		}
	}

	os.Exit(code)
}
