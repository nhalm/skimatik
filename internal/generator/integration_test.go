package generator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupIntegrationTest performs common integration test setup
func setupIntegrationTest(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	_ = getTestDB(t)
	return t.TempDir()
}

// TestSystem_EndToEnd tests the complete system workflow:
// Connect to DB → Generate code → Code compiles → Code works
func TestSystem_EndToEnd(t *testing.T) {
	tempDir := setupIntegrationTest(t)

	// Configure for end-to-end generation
	config := &Config{
		DSN:         "postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test",
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
	generator := New(config, "test")
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

// TestSystem_QueryGeneration tests query-based code generation workflow with tables disabled
func TestSystem_QueryGeneration(t *testing.T) {
	tempDir := setupIntegrationTest(t)

	// Create a simple test query directory
	queriesDir := filepath.Join(tempDir, "queries")
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		t.Fatalf("Failed to create queries directory: %v", err)
	}

	// Write a simple test query file with columns that actually exist
	queryContent := `-- name: GetUserByEmail :one
SELECT id, name, email, is_active, created_at, updated_at
FROM users
WHERE email = $1 AND is_active = true;

-- name: GetActiveUsers :many
SELECT id, name, email, is_active, created_at, updated_at
FROM users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false WHERE id = $1;
`
	if err := os.WriteFile(filepath.Join(queriesDir, "users.sql"), []byte(queryContent), 0644); err != nil {
		t.Fatalf("Failed to write query file: %v", err)
	}

	// Configure for query-only generation (tables disabled)
	config := &Config{
		DSN:         "postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		Tables:      false, // Tables disabled - this is the key test case
		QueriesDir:  queriesDir,
		Verbose:     true,
	}

	// Test: System generates query code without errors
	generator := New(config, "test")
	ctx := context.Background()
	err := generator.Generate(ctx)
	if err != nil {
		t.Fatalf("Query generation failed: %v", err)
	}

	// Test: Required shared files are created even with tables disabled
	requiredFiles := []string{
		"errors.go",
		"database_operations.go",
		"users_queries_generated.go",
	}

	for _, filename := range requiredFiles {
		filepath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not generated", filename)
		}
	}

	// Test: Generated code compiles
	if !compileGeneratedCode(t, tempDir) {
		t.Fatal("Generated query code failed to compile")
	}

	// Test: Generated code is properly formatted
	if !verifyCodeFormatting(t, tempDir) {
		t.Fatal("Generated query code is not properly formatted")
	}

	t.Log("✅ Query-only generation test passed: queries work with tables disabled")
}

// TestSystem_CursorColumnsPagination tests cursor_columns annotation and dual function generation
func TestSystem_CursorColumnsPagination(t *testing.T) {
	tempDir := setupIntegrationTest(t)

	// Create queries directory
	queriesDir := filepath.Join(tempDir, "queries")
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		t.Fatalf("Failed to create queries directory: %v", err)
	}

	// Write query with cursor_columns annotation - note: no ORDER BY in the query
	queryContent := `-- name: GetPublishedPosts :many
-- cursor_columns: created_at, id
SELECT id, name, email, is_active, created_at, updated_at
FROM users
WHERE is_active = true;

-- name: SearchUsers :many
-- cursor_columns: created_at, id, name
SELECT id, name, email, is_active, created_at, updated_at
FROM users
WHERE name LIKE $1;
`
	if err := os.WriteFile(filepath.Join(queriesDir, "pagination.sql"), []byte(queryContent), 0644); err != nil {
		t.Fatalf("Failed to write query file: %v", err)
	}

	// Configure for query generation with cursor_columns
	config := &Config{
		DSN:         "postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		Tables:      false,
		QueriesDir:  queriesDir,
		Verbose:     true,
	}

	// Test: Generate code
	generator := New(config, "test")
	ctx := context.Background()
	err := generator.Generate(ctx)
	if err != nil {
		t.Fatalf("Cursor columns generation failed: %v", err)
	}

	// Test: Query file was generated
	generatedFile := filepath.Join(tempDir, "pagination_queries_generated.go")
	if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
		t.Fatalf("Expected file pagination_queries_generated.go was not generated")
	}

	// Test: Verify dual functions are generated (regular + paginated)
	content, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}
	generatedCode := string(content)

	// Check for regular function
	if !strings.Contains(generatedCode, "func (r *PaginationQueries) GetPublishedPosts(") {
		t.Error("Regular GetPublishedPosts function not generated")
	}

	// Check for paginated function
	if !strings.Contains(generatedCode, "func (r *PaginationQueries) GetPublishedPostsPaginated(") {
		t.Error("Paginated GetPublishedPostsPaginated function not generated")
	}

	// Check for orderBy parameter in paginated function
	if !strings.Contains(generatedCode, "orderBy string") {
		t.Error("Paginated function missing orderBy parameter")
	}

	// Check for allowlist validation
	if !strings.Contains(generatedCode, "validOrderBy := map[string]bool{") {
		t.Error("Paginated function missing orderBy allowlist validation")
	}

	// Check that cursor columns are in the allowlist for GetPublishedPosts
	// Note: goimports adds extra whitespace for alignment, so check for key + "true"
	if !strings.Contains(generatedCode, `"created_at":`) || !strings.Contains(generatedCode, "true") {
		t.Error("created_at not in orderBy allowlist")
	}
	if !strings.Contains(generatedCode, `"id":`) {
		t.Error("id not in orderBy allowlist")
	}

	// Check for second query's regular function
	if !strings.Contains(generatedCode, "func (r *PaginationQueries) SearchUsers(") {
		t.Error("Regular SearchUsers function not generated")
	}

	// Check for second query's paginated function
	if !strings.Contains(generatedCode, "func (r *PaginationQueries) SearchUsersPaginated(") {
		t.Error("Paginated SearchUsersPaginated function not generated")
	}

	// Check that SearchUsers has name in allowlist (created_at and id already checked above)
	if !strings.Contains(generatedCode, `"name":`) {
		t.Error("name not in SearchUsers orderBy allowlist")
	}

	// Test: Generated code compiles
	if !compileGeneratedCode(t, tempDir) {
		t.Fatal("Generated cursor_columns code failed to compile")
	}

	// Test: Generated code is properly formatted
	if !verifyCodeFormatting(t, tempDir) {
		t.Fatal("Generated cursor_columns code is not properly formatted")
	}

	t.Log("✅ Cursor columns pagination test passed: dual functions generated correctly")
}

// TestSystem_RealWorldScenarios tests representative table scenarios
func TestSystem_RealWorldScenarios(t *testing.T) {
	setupIntegrationTest(t)

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
				DSN:         "postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test",
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
			generator := New(config, "test")
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

// TestSystem_ArrayColumnSupport tests array column type detection in query results
func TestSystem_ArrayColumnSupport(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	analyzer := NewQueryAnalyzer(db, "public")

	tests := []struct {
		name           string
		query          *Query
		expectedArrays map[string]bool
		expectedTypes  map[string]string
	}{
		{
			name: "data_types_test_array_columns",
			query: &Query{
				Name: "GetDataTypesArrays",
				Type: QueryTypeMany,
				SQL:  "SELECT id, text_array_field, integer_array_field, uuid_array_field FROM data_types_test",
			},
			expectedArrays: map[string]bool{
				"id":                  false,
				"text_array_field":    true,
				"integer_array_field": true,
				"uuid_array_field":    true,
			},
			expectedTypes: map[string]string{
				"id":                  "uuid.UUID",
				"text_array_field":    "[]*string",
				"integer_array_field": "[]*int",
				"uuid_array_field":    "[]*uuid.UUID",
			},
		},
		{
			name: "posts_tags_array",
			query: &Query{
				Name: "GetPostTags",
				Type: QueryTypeMany,
				SQL:  "SELECT id, title, tags FROM posts",
			},
			expectedArrays: map[string]bool{
				"id":    false,
				"title": false,
				"tags":  true,
			},
			expectedTypes: map[string]string{
				"id":    "uuid.UUID",
				"title": "string",
				"tags":  "[]*string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.AnalyzeQuery(ctx, tt.query)
			if err != nil {
				t.Fatalf("Failed to analyze query: %v", err)
			}

			if len(tt.query.Columns) == 0 {
				t.Fatal("No columns detected in query")
			}

			for _, col := range tt.query.Columns {
				expectedIsArray, found := tt.expectedArrays[col.Name]
				if !found {
					t.Errorf("Unexpected column: %s", col.Name)
					continue
				}

				if col.IsArray != expectedIsArray {
					t.Errorf("Column %s: IsArray = %v, want %v", col.Name, col.IsArray, expectedIsArray)
				}

				expectedType, found := tt.expectedTypes[col.Name]
				if !found {
					t.Errorf("No expected type for column: %s", col.Name)
					continue
				}

				if col.GoType != expectedType {
					t.Errorf("Column %s: GoType = %q, want %q", col.Name, col.GoType, expectedType)
				}
			}
		})
	}

	t.Log("✅ Array column support test passed: arrays detected correctly in query results")
}

// TestSystem_ErrorHandling tests system error handling
func TestSystem_ErrorHandling(t *testing.T) {
	setupIntegrationTest(t)

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

		generator := New(config, "test")
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
