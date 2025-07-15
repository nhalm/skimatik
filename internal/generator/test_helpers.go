package generator

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// getTestDB creates a connection to the test database
// This function is used across multiple integration test files
func getTestDB(t *testing.T) *pgxpool.Pool {
	if testing.Short() {
		t.Skip("Skipping test database connection in short mode")
		return nil
	}

	// Use environment variable if set, otherwise use default test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("Failed to ping test database: %v", err)
	}

	return pool
}

// getTestTable returns a standardized test table for code generation tests
func getTestTable() Table {
	return Table{
		Name:   "users",
		Schema: "public",
		Columns: []Column{
			{
				Name:         "id",
				Type:         "uuid",
				GoType:       "uuid.UUID",
				IsNullable:   false,
				DefaultValue: "",
				IsArray:      false,
			},
			{
				Name:         "name",
				Type:         "text",
				GoType:       "string",
				IsNullable:   false,
				DefaultValue: "",
				IsArray:      false,
			},
			{
				Name:         "email",
				Type:         "text",
				GoType:       "string",
				IsNullable:   false,
				DefaultValue: "",
				IsArray:      false,
			},
			{
				Name:         "is_active",
				Type:         "boolean",
				GoType:       "pgtype.Bool",
				IsNullable:   true,
				DefaultValue: "true",
				IsArray:      false,
			},
			{
				Name:         "created_at",
				Type:         "timestamptz",
				GoType:       "time.Time",
				IsNullable:   false,
				DefaultValue: "now()",
				IsArray:      false,
			},
			{
				Name:         "metadata",
				Type:         "jsonb",
				GoType:       "pgtype.JSONB",
				IsNullable:   true,
				DefaultValue: "",
				IsArray:      false,
			},
		},
		PrimaryKey: []string{"id"},
		Indexes:    []Index{},
	}
}

// getTestConfig returns a standardized test configuration
func getTestConfig() *Config {
	return &Config{
		OutputDir:   "/tmp/test",
		PackageName: "repositories",
		Verbose:     false,
	}
}

// getTestConfigWithTempDir returns a test configuration with a temporary directory
func getTestConfigWithTempDir(t *testing.T) *Config {
	return &Config{
		OutputDir:   t.TempDir(),
		PackageName: "repositories",
		Verbose:     false,
	}
}
