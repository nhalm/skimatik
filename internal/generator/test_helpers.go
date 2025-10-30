package generator

import (
	"testing"

	"github.com/nhalm/pgxkit"
)

// packageTestDB is the shared test database connection for all integration tests
// It is initialized in TestMain (main_test.go)
var packageTestDB *pgxkit.DB

// getTestDB returns the shared test database connection
// This function is used across multiple integration test files
func getTestDB(t *testing.T) *pgxkit.DB {
	if testing.Short() {
		t.Skip("Skipping test database connection in short mode")
		return nil
	}

	if packageTestDB == nil {
		t.Skip("Test database not available (TEST_DATABASE_URL not set)")
		return nil
	}

	return packageTestDB
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
