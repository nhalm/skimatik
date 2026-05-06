package generator

import (
	"testing"
)

func TestTable_HasIndexLeadingWith(t *testing.T) {
	table := table{
		Indexes: []index{
			{Name: "idx_users_email", Columns: []string{"email"}},
			{Name: "idx_users_active_created", Columns: []string{"is_active", "created_at"}},
			{Name: "idx_users_expr", Columns: []string{""}}, // expression index, no leading column
		},
	}

	tests := []struct {
		name   string
		column string
		want   bool
	}{
		{"single-column leading match", "email", true},
		{"composite leading match", "is_active", true},
		{"non-leading column does not count", "created_at", false},
		{"missing column", "name", false},
		{"empty column does not match expression index", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := table.hasIndexLeadingWith(tt.column); got != tt.want {
				t.Errorf("HasIndexLeadingWith(%q) = %v, want %v", tt.column, got, tt.want)
			}
		})
	}
}

func TestTable_HasForeignKeyTo(t *testing.T) {
	table := table{
		Name: "users_audit",
		ForeignKeys: []foreignKey{
			{
				ConstraintName:   "users_audit_parent_id_fkey",
				ColumnName:       "parent_id",
				ReferencedTable:  "users",
				ReferencedColumn: "id",
			},
		},
	}

	if !table.hasForeignKeyTo("parent_id", "users", "id") {
		t.Error("expected HasForeignKeyTo to find parent_id -> users.id")
	}
	if table.hasForeignKeyTo("parent_id", "users", "name") {
		t.Error("HasForeignKeyTo should reject mismatched referenced column")
	}
	if table.hasForeignKeyTo("parent_id", "posts", "id") {
		t.Error("HasForeignKeyTo should reject mismatched referenced table")
	}
	if table.hasForeignKeyTo("user_id", "users", "id") {
		t.Error("HasForeignKeyTo should reject mismatched child column")
	}
}

func TestNewIntrospector(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		want   string
	}{
		{
			name:   "public schema",
			schema: "public",
			want:   "public",
		},
		{
			name:   "custom schema",
			schema: "custom_schema",
			want:   "custom_schema",
		},
		{
			name:   "empty schema",
			schema: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			introspector := NewIntrospector(nil, tt.schema)
			if introspector.schema != tt.want {
				t.Errorf("NewIntrospector() schema = %v, want %v", introspector.schema, tt.want)
			}
			if introspector.db != nil {
				t.Errorf("NewIntrospector() db should be nil in this test")
			}
		})
	}
}

// Test the structure and relationships of the introspection results
func TestIntrospector_ResultStructure(t *testing.T) {
	// Test that the result structures are properly formed
	// This doesn't require a database connection

	t.Run("table structure validation", func(t *testing.T) {
		table := table{
			Name:   "users",
			Schema: "public",
			Columns: []column{
				{
					Name:       "id",
					Type:       "uuid",
					IsNullable: false,
					IsArray:    false,
				},
				{
					Name:       "name",
					Type:       "text",
					IsNullable: true,
					IsArray:    false,
				},
			},
			PrimaryKey: []string{"id"},
			Indexes: []index{
				{
					Name:     "idx_users_name",
					Columns:  []string{"name"},
					IsUnique: false,
				},
			},
		}

		// Validate the structure
		if table.Name != "users" {
			t.Errorf("Table name = %v, want users", table.Name)
		}
		if table.Schema != "public" {
			t.Errorf("Table schema = %v, want public", table.Schema)
		}
		if len(table.Columns) != 2 {
			t.Errorf("Table columns length = %v, want 2", len(table.Columns))
		}
		if len(table.PrimaryKey) != 1 {
			t.Errorf("Table primary key length = %v, want 1", len(table.PrimaryKey))
		}
		if len(table.Indexes) != 1 {
			t.Errorf("Table indexes length = %v, want 1", len(table.Indexes))
		}

		// Test column structure
		idCol := table.Columns[0]
		if idCol.Name != "id" || idCol.Type != "uuid" || idCol.IsNullable {
			t.Errorf("ID column structure incorrect: %+v", idCol)
		}

		nameCol := table.Columns[1]
		if nameCol.Name != "name" || nameCol.Type != "text" || !nameCol.IsNullable {
			t.Errorf("Name column structure incorrect: %+v", nameCol)
		}

		// Test index structure
		index := table.Indexes[0]
		if index.Name != "idx_users_name" || len(index.Columns) != 1 || index.Columns[0] != "name" {
			t.Errorf("Index structure incorrect: %+v", index)
		}
	})
}
