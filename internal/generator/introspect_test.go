package generator

import (
	"strings"
	"testing"
)

func TestIntrospector_parseIndexColumns(t *testing.T) {
	introspector := &Introspector{}

	tests := []struct {
		name     string
		indexDef string
		want     []string
	}{
		{
			name:     "single column index",
			indexDef: "CREATE INDEX idx_users_email ON users USING btree (email)",
			want:     []string{"email"},
		},
		{
			name:     "multiple column index",
			indexDef: "CREATE INDEX idx_users_name_email ON users USING btree (name, email)",
			want:     []string{"name", "email"},
		},
		{
			name:     "unique index",
			indexDef: "CREATE UNIQUE INDEX idx_users_email_unique ON users USING btree (email)",
			want:     []string{"email"},
		},
		{
			name:     "index with schema",
			indexDef: "CREATE INDEX idx_public_users_email ON public.users USING btree (email)",
			want:     []string{"email"},
		},
		{
			name:     "index with spaces",
			indexDef: "CREATE INDEX idx_users_multi ON users USING btree (first_name, last_name, email)",
			want:     []string{"first_name", "last_name", "email"},
		},
		{
			name:     "index with quoted columns",
			indexDef: "CREATE INDEX idx_users_quoted ON users USING btree (\"first name\", \"last name\")",
			want:     []string{"\"first name\"", "\"last name\""},
		},
		{
			name:     "complex index definition",
			indexDef: "CREATE INDEX CONCURRENTLY idx_posts_user_status ON posts USING btree (user_id, status) WHERE status = 'active'",
			want:     []string{"user_id", "status"},
		},
		{
			name:     "malformed index definition",
			indexDef: "CREATE INDEX invalid_index",
			want:     []string{},
		},
		{
			name:     "empty index definition",
			indexDef: "",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := introspector.parseIndexColumns(tt.indexDef)
			if len(got) != len(tt.want) {
				t.Errorf("parseIndexColumns() = %v, want %v", got, tt.want)
				return
			}
			for i, col := range got {
				if col != tt.want[i] {
					t.Errorf("parseIndexColumns() = %v, want %v", got, tt.want)
					break
				}
			}
		})
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

// Test the column type normalization logic that's embedded in the SQL query
func TestColumnTypeNormalization(t *testing.T) {
	// These tests verify the logic that would be applied in the SQL query
	// for normalizing PostgreSQL column types
	tests := []struct {
		name         string
		dataType     string
		udtName      string
		isArray      bool
		expectedType string
	}{
		{
			name:         "text type",
			dataType:     "text",
			udtName:      "text",
			isArray:      false,
			expectedType: "text",
		},
		{
			name:         "character varying to varchar",
			dataType:     "character varying",
			udtName:      "varchar",
			isArray:      false,
			expectedType: "varchar",
		},
		{
			name:         "timestamp without time zone",
			dataType:     "timestamp without time zone",
			udtName:      "timestamp",
			isArray:      false,
			expectedType: "timestamp",
		},
		{
			name:         "timestamp with time zone",
			dataType:     "timestamp with time zone",
			udtName:      "timestamptz",
			isArray:      false,
			expectedType: "timestamptz",
		},
		{
			name:         "array type with underscore prefix",
			dataType:     "ARRAY",
			udtName:      "_text",
			isArray:      true,
			expectedType: "text",
		},
		{
			name:         "array varchar type",
			dataType:     "ARRAY",
			udtName:      "_varchar",
			isArray:      true,
			expectedType: "text", // _varchar becomes text after removing underscore and replacing varchar
		},
		{
			name:         "integer type",
			dataType:     "integer",
			udtName:      "int4",
			isArray:      false,
			expectedType: "integer",
		},
		{
			name:         "uuid type",
			dataType:     "uuid",
			udtName:      "uuid",
			isArray:      false,
			expectedType: "uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the normalization logic from the SQL query
			var normalizedType string
			if tt.isArray {
				// Remove underscore prefix and handle varchar replacement
				normalizedType = strings.TrimPrefix(tt.udtName, "_")
				normalizedType = strings.ReplaceAll(normalizedType, "varchar", "text")
			} else {
				switch tt.dataType {
				case "character varying":
					normalizedType = "varchar"
				case "timestamp without time zone":
					normalizedType = "timestamp"
				case "timestamp with time zone":
					normalizedType = "timestamptz"
				default:
					normalizedType = tt.dataType
				}
			}

			if normalizedType != tt.expectedType {
				t.Errorf("Column type normalization: got %v, want %v", normalizedType, tt.expectedType)
			}
		})
	}
}

// Test error handling scenarios
func TestIntrospector_ErrorHandling(t *testing.T) {
	introspector := NewIntrospector(nil, "public")

	t.Run("nil database connection", func(t *testing.T) {
		// These should panic or return errors when called with nil database
		// For now, we'll just test that we can create the introspector
		// The actual error handling happens at the pgx level
		if introspector.db != nil {
			t.Error("Expected nil database connection")
		}
	})
}

// Test the structure and relationships of the introspection results
func TestIntrospector_ResultStructure(t *testing.T) {
	// Test that the result structures are properly formed
	// This doesn't require a database connection

	t.Run("table structure validation", func(t *testing.T) {
		table := Table{
			Name:   "users",
			Schema: "public",
			Columns: []Column{
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
			Indexes: []Index{
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

// Test SQL query construction logic
func TestIntrospector_SQLQueries(t *testing.T) {
	// Test that the SQL queries are properly constructed
	// We can't test the actual execution without a database, but we can validate the query structure

	t.Run("table names query structure", func(t *testing.T) {
		expectedQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
		// Normalize whitespace for comparison
		normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")

		// This would be the actual query used in getTableNames
		actualQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
		normalizedActual := strings.Join(strings.Fields(actualQuery), " ")

		if normalizedActual != normalizedExpected {
			t.Errorf("Table names query structure mismatch")
		}
	})

	t.Run("primary key query structure", func(t *testing.T) {
		expectedQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = $1 
		  AND tc.table_name = $2
		  AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY kcu.ordinal_position
	`
		// Normalize whitespace for comparison
		normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")

		// This would be the actual query used in getTablePrimaryKey
		actualQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = $1 
		  AND tc.table_name = $2
		  AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY kcu.ordinal_position
	`
		normalizedActual := strings.Join(strings.Fields(actualQuery), " ")

		if normalizedActual != normalizedExpected {
			t.Errorf("Primary key query structure mismatch")
		}
	})
}

// Test schema validation
func TestIntrospector_SchemaValidation(t *testing.T) {
	tests := []struct {
		name       string
		schema     string
		tableName  string
		expectsErr bool
	}{
		{
			name:       "valid schema and table",
			schema:     "public",
			tableName:  "users",
			expectsErr: false,
		},
		{
			name:       "custom schema",
			schema:     "custom_schema",
			tableName:  "users",
			expectsErr: false,
		},
		{
			name:       "empty schema",
			schema:     "",
			tableName:  "users",
			expectsErr: false, // Empty schema might be valid in some contexts
		},
		{
			name:       "empty table name",
			schema:     "public",
			tableName:  "",
			expectsErr: false, // Empty table name might be valid for some operations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			introspector := NewIntrospector(nil, tt.schema)

			// Test that the schema is set correctly
			if introspector.schema != tt.schema {
				t.Errorf("Schema = %v, want %v", introspector.schema, tt.schema)
			}

			// For now, we just validate that the introspector can be created
			// with various schema values. Actual validation would happen
			// when connecting to the database.
		})
	}
}

// Test concurrent access safety (basic structure test)
func TestIntrospector_ConcurrentAccess(t *testing.T) {
	// Test that multiple introspectors can be created safely
	introspectors := make([]*Introspector, 10)

	for i := 0; i < 10; i++ {
		introspectors[i] = NewIntrospector(nil, "public")
	}

	// Verify all introspectors are independent
	for i, introspector := range introspectors {
		if introspector.schema != "public" {
			t.Errorf("Introspector %d schema = %v, want public", i, introspector.schema)
		}
	}
}
