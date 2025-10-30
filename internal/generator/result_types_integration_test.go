//go:build !short

package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestResultTypes_SimpleSelectNotNull tests that NOT NULL columns generate native Go types
func TestResultTypes_SimpleSelectNotNull(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	// Tables payment_links and payments already exist in test database

	// Create test SQL file with query
	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentLinkBasic :one
SELECT id, status FROM payment_links WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payment_links.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	// Parse and analyze query
	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	// Verify column types
	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	// Test: id should be uuid.UUID (NOT NULL)
	idCol := query.Columns[0]
	if idCol.Name != "id" {
		t.Errorf("Expected column name 'id', got '%s'", idCol.Name)
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL), got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL")
	}

	// Test: status should be string (NOT NULL)
	statusCol := query.Columns[1]
	if statusCol.Name != "status" {
		t.Errorf("Expected column name 'status', got '%s'", statusCol.Name)
	}
	if statusCol.GoType != "string" {
		t.Errorf("Expected status to be string (NOT NULL), got %s", statusCol.GoType)
	}
	if statusCol.IsNullable {
		t.Errorf("Expected status to be NOT NULL")
	}

	t.Logf("EXPECTED FAILURE: This test fails because FieldDescriptions-based nullability detection is not yet implemented")
}

// TestResultTypes_SelectWithNullable tests that nullable columns generate pointer types
func TestResultTypes_SelectWithNullable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	// Tables payment_links and payments already exist in test database

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentLinkWithDescription :one
SELECT id, description FROM payment_links WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payment_links.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	// Test: id should be uuid.UUID (NOT NULL)
	idCol := query.Columns[0]
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL), got %s", idCol.GoType)
	}

	// Test: description should be *string (nullable)
	descCol := query.Columns[1]
	if descCol.Name != "description" {
		t.Errorf("Expected column name 'description', got '%s'", descCol.Name)
	}
	if descCol.GoType != "*string" {
		t.Errorf("Expected description to be *string (nullable), got %s", descCol.GoType)
	}
	if !descCol.IsNullable {
		t.Errorf("Expected description to be nullable")
	}

	t.Logf("EXPECTED FAILURE: This test fails because pointer type generation is not yet implemented")
}

// TestResultTypes_CountAggregate tests that COUNT aggregates are never nullable
func TestResultTypes_CountAggregate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	// Tables payment_links and payments already exist in test database

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentStats :one
SELECT
    COUNT(*) as payment_count,
    SUM(amount) as total_amount
FROM payments
WHERE payment_link_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payments.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	// Test: COUNT(*) should be int (never NULL)
	countCol := findColumn(query.Columns, "payment_count")
	if countCol == nil {
		t.Fatal("payment_count column not found")
	}
	if countCol.GoType != "int" {
		t.Errorf("Expected payment_count to be int (COUNT never NULL), got %s", countCol.GoType)
	}
	if countCol.IsNullable {
		t.Errorf("Expected payment_count to be NOT NULL (COUNT never returns NULL)")
	}

	// Test: SUM should be *int (can be NULL)
	sumCol := findColumn(query.Columns, "total_amount")
	if sumCol == nil {
		t.Fatal("total_amount column not found")
	}
	if sumCol.GoType != "*int" {
		t.Errorf("Expected total_amount to be *int (SUM can be NULL), got %s", sumCol.GoType)
	}
	if !sumCol.IsNullable {
		t.Errorf("Expected total_amount to be nullable (SUM can return NULL)")
	}

	t.Logf("EXPECTED FAILURE: COUNT detection and pointer types not yet implemented")
}

// Helper functions

// findColumn finds a column by name in a slice of columns
func findColumn(columns []Column, name string) *Column {
	for i := range columns {
		if columns[i].Name == name {
			return &columns[i]
		}
	}
	return nil
}
