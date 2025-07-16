package generator

import (
	"context"
	"strings"
	"testing"
)

func TestUUIDValidation_ValidTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool := getTestDB(t)
	defer pool.Shutdown(context.Background())

	introspector := NewIntrospector(pool, "public")
	typeMapper := NewTypeMapper(nil)
	ctx := context.Background()

	// Get all tables from the database
	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables() error = %v", err)
	}

	// Tables that should have valid UUID primary keys
	validUUIDTables := []string{
		"users",
		"profiles",
		"posts",
		"comments",
		"categories",
		"post_categories",
		"files",
		"data_types_test",
	}

	// Create table map
	tableMap := make(map[string]Table)
	for _, table := range tables {
		tableMap[table.Name] = table
	}

	for _, tableName := range validUUIDTables {
		t.Run(tableName, func(t *testing.T) {
			table, exists := tableMap[tableName]
			if !exists {
				t.Fatalf("Table %s not found", tableName)
			}

			// Get primary key column
			pkCol := table.GetPrimaryKeyColumn()
			if pkCol == nil {
				t.Fatalf("Table %s has no single-column primary key", tableName)
			}

			// Validate UUID primary key
			err := typeMapper.ValidateUUIDPrimaryKey(pkCol)
			if err != nil {
				t.Errorf("Table %s UUID validation failed: %v", tableName, err)
			}

			// Verify the column is actually UUID type
			if !pkCol.IsUUID() {
				t.Errorf("Table %s primary key is not UUID type: %s", tableName, pkCol.Type)
			}

			// Verify the column is not nullable
			if pkCol.IsNullable {
				t.Errorf("Table %s primary key should not be nullable", tableName)
			}

			// Verify the column is not an array
			if pkCol.IsArray {
				t.Errorf("Table %s primary key should not be an array", tableName)
			}
		})
	}
}

func TestUUIDValidation_InvalidTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool := getTestDB(t)
	defer pool.Shutdown(context.Background())

	introspector := NewIntrospector(pool, "public")
	typeMapper := NewTypeMapper(nil)
	ctx := context.Background()

	// Get all tables from the database
	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables() error = %v", err)
	}

	// Create table map
	tableMap := make(map[string]Table)
	for _, table := range tables {
		tableMap[table.Name] = table
	}

	// Test the invalid_pk_table which has a serial primary key
	t.Run("invalid_pk_table_serial", func(t *testing.T) {
		table, exists := tableMap["invalid_pk_table"]
		if !exists {
			t.Skip("invalid_pk_table not found")
		}

		pkCol := table.GetPrimaryKeyColumn()
		if pkCol == nil {
			t.Fatalf("Table invalid_pk_table has no single-column primary key")
		}

		// This should fail UUID validation
		err := typeMapper.ValidateUUIDPrimaryKey(pkCol)
		if err == nil {
			t.Error("invalid_pk_table should fail UUID validation")
		}

		// Verify the error message is clear
		expectedSubstrings := []string{"must be UUID type", "id"}
		for _, substring := range expectedSubstrings {
			if err != nil && !strings.Contains(err.Error(), substring) {
				t.Errorf("Error message should contain '%s', got: %s", substring, err.Error())
			}
		}
	})

	// Test composite primary key table
	t.Run("composite_pk_table", func(t *testing.T) {
		table, exists := tableMap["composite_pk_table"]
		if !exists {
			t.Skip("composite_pk_table not found")
		}

		// This table has a composite primary key, so GetPrimaryKeyColumn should return nil
		pkCol := table.GetPrimaryKeyColumn()
		if pkCol != nil {
			t.Error("composite_pk_table should not have single-column primary key")
		}

		// Since there's no single primary key column, we can't validate it
		// This is expected behavior for composite keys
	})
}

func TestUUIDValidation_Integration_AllTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool := getTestDB(t)
	defer pool.Shutdown(context.Background())

	introspector := NewIntrospector(pool, "public")
	typeMapper := NewTypeMapper(nil)
	ctx := context.Background()

	// Get all tables from the database
	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables() error = %v", err)
	}

	validUUIDCount := 0
	invalidUUIDCount := 0
	compositeKeyCount := 0

	for _, table := range tables {
		t.Run(table.Name, func(t *testing.T) {
			pkCol := table.GetPrimaryKeyColumn()

			if pkCol == nil {
				// Composite primary key or no primary key
				compositeKeyCount++
				t.Logf("Table %s has composite or no primary key", table.Name)
				return
			}

			err := typeMapper.ValidateUUIDPrimaryKey(pkCol)
			if err != nil {
				invalidUUIDCount++
				t.Logf("Table %s has invalid UUID primary key: %v", table.Name, err)

				// Verify that non-UUID tables are properly identified
				if pkCol.IsUUID() {
					t.Errorf("Table %s has UUID type but validation failed: %v", table.Name, err)
				}
			} else {
				validUUIDCount++
				t.Logf("Table %s has valid UUID primary key", table.Name)

				// Verify that valid tables actually have UUID type
				if !pkCol.IsUUID() {
					t.Errorf("Table %s passed validation but is not UUID type: %s", table.Name, pkCol.Type)
				}
			}
		})
	}

	// Log summary
	t.Logf("Summary: %d valid UUID, %d invalid UUID, %d composite/no PK",
		validUUIDCount, invalidUUIDCount, compositeKeyCount)

	// We should have at least some valid UUID tables from our test schema
	if validUUIDCount == 0 {
		t.Error("Expected at least some tables with valid UUID primary keys")
	}

	// We should have at least one invalid table (invalid_pk_table)
	if invalidUUIDCount == 0 {
		t.Error("Expected at least one table with invalid UUID primary key")
	}
}

func TestUUIDValidation_PRDRequirement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// This test validates the specific PRD requirement:
	// "All primary keys must be UUID v7 for consistent time-ordered pagination"

	pool := getTestDB(t)
	defer pool.Shutdown(context.Background())

	introspector := NewIntrospector(pool, "public")
	typeMapper := NewTypeMapper(nil)
	ctx := context.Background()

	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables() error = %v", err)
	}

	// Tables that are expected to be used for code generation
	// (excluding test tables with intentionally invalid PKs)
	productionTables := []string{
		"users",
		"profiles",
		"posts",
		"comments",
		"categories",
		"post_categories",
		"files",
	}

	tableMap := make(map[string]Table)
	for _, table := range tables {
		tableMap[table.Name] = table
	}

	for _, tableName := range productionTables {
		t.Run(tableName, func(t *testing.T) {
			table, exists := tableMap[tableName]
			if !exists {
				t.Fatalf("Production table %s not found", tableName)
			}

			pkCol := table.GetPrimaryKeyColumn()
			if pkCol == nil {
				t.Fatalf("Production table %s must have single-column primary key", tableName)
			}

			// This is the core PRD requirement validation
			err := typeMapper.ValidateUUIDPrimaryKey(pkCol)
			if err != nil {
				t.Errorf("PRD VIOLATION: Table %s does not meet UUID primary key requirement: %v",
					tableName, err)
			}

			// Additional checks for pagination requirements
			if !pkCol.IsUUID() {
				t.Errorf("PRD VIOLATION: Table %s primary key is not UUID (required for time-ordered pagination)",
					tableName)
			}

			if pkCol.IsNullable {
				t.Errorf("PRD VIOLATION: Table %s primary key is nullable (incompatible with pagination)",
					tableName)
			}

			// Note: We don't validate UUID v7 specifically here because that would require
			// parsing the UUID format, which is beyond the scope of schema introspection.
			// The PRD requirement for UUID v7 is enforced at the application level when
			// generating UUIDs, not at the database schema level.
		})
	}
}
