package generator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSystem_EndToEnd tests the complete system workflow:
// Connect to DB → Generate code → Code compiles → Code works
func TestSystem_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool := getTestDB(t)
	defer pool.Close()

	tempDir := t.TempDir()

	// Configure for end-to-end generation
	config := &Config{
		DSN:         "postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		Tables:      true,
		Include:     []string{"users", "posts", "data_types_test"},
		TableConfigs: map[string]TableConfig{
			"users":           {Functions: []string{"create", "get", "update", "delete", "list", "paginate"}},
			"posts":           {Functions: []string{"create", "get", "update", "delete", "list", "paginate"}},
			"data_types_test": {Functions: []string{"create", "get", "update", "delete", "list", "paginate"}},
		},
		Verbose: false,
	}

	// Test: System generates code without errors
	generator := New(config)
	ctx := context.Background()
	err := generator.Generate(ctx)
	if err != nil {
		t.Fatalf("System failed to generate code: %v", err)
	}

	// Test: All expected files are created
	expectedFiles := []string{
		"users_generated.go",
		"posts_generated.go",
		"data_types_test_generated.go",
		"pagination.go",
	}

	for _, filename := range expectedFiles {
		filepath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not generated", filename)
		}
	}

	// Test: Generated code compiles
	if !compileGeneratedCode(t, tempDir) {
		t.Fatal("Generated code failed to compile")
	}

	// Test: Generated code is properly formatted
	if !verifyCodeFormatting(t, tempDir) {
		t.Fatal("Generated code is not properly formatted")
	}

	t.Log("✅ End-to-end system test passed: DB → Generation → Compilation → Formatting")
}

// TestSystem_QueryGeneration tests query-based code generation workflow
func TestSystem_QueryGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool := getTestDB(t)
	defer pool.Close()

	tempDir := t.TempDir()

	// Create test SQL files
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	// Create test queries
	testQueries := `-- name: GetUserByID :one
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListActiveUsers :many
SELECT id, name, email FROM users WHERE is_active = true ORDER BY name;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES ($1, $2);

-- name: GetUsersPaginated :paginated
SELECT id, name, email FROM users ORDER BY id ASC LIMIT $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(testQueries), 0644)
	if err != nil {
		t.Fatalf("Failed to write test queries: %v", err)
	}

	// Configure for query generation
	config := &Config{
		DSN:         "postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		QueriesDir:  sqlDir,
		Verbose:     false,
	}

	// Test: System generates query code without errors
	generator := New(config)
	ctx := context.Background()
	err = generator.Generate(ctx)
	if err != nil {
		t.Fatalf("System failed to generate query code: %v", err)
	}

	// Test: Query file is created
	queryFile := filepath.Join(tempDir, "users_queries_generated.go")
	if _, err := os.Stat(queryFile); os.IsNotExist(err) {
		t.Error("Expected users_queries_generated.go was not created")
	}

	// Test: Generated query code compiles
	if !compileGeneratedCode(t, tempDir) {
		t.Fatal("Generated query code failed to compile")
	}

	t.Log("✅ Query generation test passed: SQL → Analysis → Generation → Compilation")
}

// TestSystem_RealWorldScenarios tests representative table scenarios
func TestSystem_RealWorldScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool := getTestDB(t)
	defer pool.Close()

	scenarios := []struct {
		name        string
		table       string
		description string
	}{
		{
			name:        "simple_table",
			table:       "users",
			description: "Basic table with standard columns",
		},
		{
			name:        "complex_relationships",
			table:       "posts",
			description: "Table with foreign keys and relationships",
		},
		{
			name:        "diverse_data_types",
			table:       "data_types_test",
			description: "Table with various PostgreSQL data types",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()

			config := &Config{
				DSN:         "postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test",
				Schema:      "public",
				OutputDir:   tempDir,
				PackageName: "testgen",
				Tables:      true,
				Include:     []string{scenario.table},
				TableConfigs: map[string]TableConfig{
					scenario.table: {Functions: []string{"create", "get", "update", "delete", "list", "paginate"}},
				},
				Verbose: false,
			}

			// Test: Table-specific generation works
			generator := New(config)
			ctx := context.Background()
			err := generator.Generate(ctx)
			if err != nil {
				t.Fatalf("Failed to generate code for %s (%s): %v", scenario.table, scenario.description, err)
			}

			// Test: Generated code compiles
			if !compileGeneratedCode(t, tempDir) {
				t.Fatalf("Generated code for %s failed to compile", scenario.table)
			}

			t.Logf("✅ %s scenario passed: %s", scenario.name, scenario.description)
		})
	}
}

// TestSystem_ErrorHandling tests system error handling
func TestSystem_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Run("invalid_database_connection", func(t *testing.T) {
		tempDir := t.TempDir()

		config := &Config{
			DSN:         "postgres://invalid:invalid@localhost:9999/invalid",
			Schema:      "public",
			OutputDir:   tempDir,
			PackageName: "testgen",
			Tables:      true,
			Include:     []string{"users"},
			Verbose:     false,
		}

		generator := New(config)
		ctx := context.Background()
		err := generator.Generate(ctx)

		// Test: System handles invalid database gracefully
		if err == nil {
			t.Error("Expected error for invalid database connection")
		}

		// Test: Error message is helpful
		if !strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "connection") {
			t.Errorf("Error message should mention connection issue: %v", err)
		}
	})

	t.Run("invalid_primary_key_table", func(t *testing.T) {
		pool := getTestDB(t)
		defer pool.Close()

		tempDir := t.TempDir()

		config := &Config{
			DSN:         "postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test",
			Schema:      "public",
			OutputDir:   tempDir,
			PackageName: "testgen",
			Tables:      true,
			Include:     []string{"invalid_pk_table"}, // This table has serial PK, not UUID
			TableConfigs: map[string]TableConfig{
				"invalid_pk_table": {Functions: []string{"create", "get", "list"}},
			},
			Verbose: false,
		}

		generator := New(config)
		ctx := context.Background()
		err := generator.Generate(ctx)

		// Test: System should succeed but skip tables without UUID primary keys
		if err != nil {
			t.Errorf("Expected success when skipping invalid tables, got error: %v", err)
		}

		// Test: No files should be generated for invalid tables
		files, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("Failed to read temp directory: %v", err)
		}

		// Should only have shared pagination files, no table-specific files
		var hasTableFiles bool
		for _, file := range files {
			if strings.Contains(file.Name(), "invalid_pk_table") {
				hasTableFiles = true
				break
			}
		}

		if hasTableFiles {
			t.Error("Expected no files to be generated for invalid_pk_table")
		}
	})
}

// Helper function to compile generated code
func compileGeneratedCode(t *testing.T, tempDir string) bool {
	// Create go.mod file
	goModContent := `module testgen

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.5
)
`

	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Errorf("Failed to create go.mod: %v", err)
		return false
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Env = append(os.Environ(), "GO111MODULE=on")

	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Errorf("go mod tidy failed: %v\nOutput: %s", err, string(output))
		return false
	}

	// Compile the code
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = tempDir
	buildCmd.Env = append(os.Environ(), "GO111MODULE=on")

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Errorf("Generated code compilation failed: %v\nOutput: %s", err, string(output))
		return false
	}

	return true
}

// Helper function to verify code formatting
func verifyCodeFormatting(t *testing.T, tempDir string) bool {
	// Run go fmt to check formatting
	fmtCmd := exec.Command("go", "fmt", "./...")
	fmtCmd.Dir = tempDir
	fmtCmd.Env = append(os.Environ(), "GO111MODULE=on")

	output, err := fmtCmd.CombinedOutput()
	if err != nil {
		t.Errorf("go fmt failed: %v\nOutput: %s", err, string(output))
		return false
	}

	// If go fmt produces output, it means files were not properly formatted
	if len(output) > 0 {
		t.Errorf("Generated code is not properly formatted. go fmt output: %s", string(output))
		return false
	}

	return true
}
