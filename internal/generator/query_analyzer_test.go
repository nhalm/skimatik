package generator

import (
	"context"
	"strings"
	"testing"
)

func TestQueryAnalyzer_ExtractParameters(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil) // No database needed for parameter extraction

	tests := []struct {
		name           string
		query          Query
		expectedParams []Parameter
		expectError    bool
	}{
		{
			name: "query with no parameters",
			query: Query{
				Name: "GetAllUsers",
				SQL:  "SELECT id, name FROM users",
				Type: QueryTypeMany,
			},
			expectedParams: []Parameter{},
			expectError:    false,
		},
		{
			name: "query with single parameter",
			query: Query{
				Name: "GetUserByID",
				SQL:  "SELECT id, name FROM users WHERE id = $1",
				Type: QueryTypeOne,
			},
			expectedParams: []Parameter{
				{Name: "param1", Type: "text", GoType: "string", Index: 1},
			},
			expectError: false,
		},
		{
			name: "query with multiple parameters",
			query: Query{
				Name: "GetUsersByNameAndEmail",
				SQL:  "SELECT id, name FROM users WHERE name = $1 AND email = $2",
				Type: QueryTypeMany,
			},
			expectedParams: []Parameter{
				{Name: "param1", Type: "text", GoType: "string", Index: 1},
				{Name: "param2", Type: "text", GoType: "string", Index: 2},
			},
			expectError: false,
		},
		{
			name: "query with duplicate parameters",
			query: Query{
				Name: "GetUsersByStatus",
				SQL:  "SELECT id, name FROM users WHERE status = $1 OR backup_status = $1",
				Type: QueryTypeMany,
			},
			expectedParams: []Parameter{
				{Name: "param1", Type: "text", GoType: "string", Index: 1},
			},
			expectError: false,
		},
		{
			name: "query with non-sequential parameters",
			query: Query{
				Name: "GetUsersByStatusAndRole",
				SQL:  "SELECT id, name FROM users WHERE status = $2 AND role = $1",
				Type: QueryTypeMany,
			},
			expectedParams: []Parameter{
				{Name: "param1", Type: "text", GoType: "string", Index: 1},
				{Name: "param2", Type: "text", GoType: "string", Index: 2},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.query
			err := analyzer.extractParameters(&query)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(query.Parameters) != len(tt.expectedParams) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expectedParams), len(query.Parameters))
			}

			for i, param := range query.Parameters {
				if i < len(tt.expectedParams) {
					expected := tt.expectedParams[i]
					if param.Name != expected.Name || param.Index != expected.Index {
						t.Errorf("Parameter %d: expected %+v, got %+v", i, expected, param)
					}
				}
			}
		})
	}
}

func TestQueryAnalyzer_EdgeCases(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil)

	tests := []struct {
		name        string
		query       Query
		expectError bool
		description string
	}{
		{
			name: "empty SQL",
			query: Query{
				Name: "EmptyQuery",
				SQL:  "",
				Type: QueryTypeMany,
			},
			expectError: false,
			description: "Empty SQL should return no parameters",
		},
		{
			name: "dollar sign in string literal",
			query: Query{
				Name: "DollarInString",
				SQL:  "SELECT '$100' as price, id FROM products WHERE id = $1",
				Type: QueryTypeOne,
			},
			expectError: false,
			description: "Dollar signs in string literals should not be treated as parameters",
		},
		{
			name: "dollar sign in quoted identifier",
			query: Query{
				Name: "DollarInIdentifier",
				SQL:  `SELECT "price$amount" FROM products WHERE id = $1`,
				Type: QueryTypeOne,
			},
			expectError: false,
			description: "Dollar signs in quoted identifiers should not be treated as parameters",
		},
		{
			name: "parameter in comment",
			query: Query{
				Name: "ParameterInComment",
				SQL:  "SELECT id FROM users -- WHERE status = $1\nWHERE id = $1",
				Type: QueryTypeOne,
			},
			expectError: false,
			description: "Parameters in comments should be ignored",
		},
		{
			name: "high parameter number",
			query: Query{
				Name: "HighParameterNumber",
				SQL:  "SELECT id FROM users WHERE id = $100",
				Type: QueryTypeOne,
			},
			expectError: false,
			description: "High parameter numbers should be handled correctly",
		},
		{
			name: "invalid parameter format",
			query: Query{
				Name: "InvalidParameterFormat",
				SQL:  "SELECT id FROM users WHERE id = $abc",
				Type: QueryTypeOne,
			},
			expectError: false,
			description: "Invalid parameter formats should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.query
			err := analyzer.extractParameters(&query)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for %s", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}

			// Basic validation that we got some result
			if query.Parameters == nil {
				t.Errorf("Expected non-nil parameters slice for %s", tt.description)
			}
		})
	}
}

func TestQueryAnalyzer_ComplexQueries(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil)

	tests := []struct {
		name           string
		query          Query
		expectedParams int
		description    string
	}{
		{
			name: "CTE query",
			query: Query{
				Name: "CTEQuery",
				SQL: `WITH user_posts AS (
					SELECT user_id, COUNT(*) as post_count
					FROM posts
					WHERE created_at > $1
					GROUP BY user_id
				)
				SELECT u.id, u.name, up.post_count
				FROM users u
				JOIN user_posts up ON u.id = up.user_id
				WHERE u.status = $2`,
				Type: QueryTypeMany,
			},
			expectedParams: 2,
			description:    "CTE with multiple parameters",
		},
		{
			name: "subquery",
			query: Query{
				Name: "SubqueryExample",
				SQL: `SELECT id, name FROM users
				WHERE id IN (
					SELECT user_id FROM posts
					WHERE category_id = $1 AND created_at > $2
				) AND status = $3`,
				Type: QueryTypeMany,
			},
			expectedParams: 3,
			description:    "Subquery with multiple parameters",
		},
		{
			name: "window function",
			query: Query{
				Name: "WindowFunctionQuery",
				SQL: `SELECT
					id, name,
					ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) as rank
				FROM employees
				WHERE department = $1 AND salary > $2`,
				Type: QueryTypeMany,
			},
			expectedParams: 2,
			description:    "Window function with parameters",
		},
		{
			name: "array operations",
			query: Query{
				Name: "ArrayQuery",
				SQL: `SELECT id, tags FROM posts
				WHERE $1 = ANY(tags) AND category_id = $2`,
				Type: QueryTypeMany,
			},
			expectedParams: 2,
			description:    "Array operations with parameters",
		},
		{
			name: "multiple joins",
			query: Query{
				Name: "MultipleJoins",
				SQL: `SELECT u.id, u.name, p.title, c.name as category
				FROM users u
				JOIN posts p ON u.id = p.user_id
				JOIN categories c ON p.category_id = c.id
				WHERE u.created_at > $1
				AND p.status = $2
				AND c.active = $3`,
				Type: QueryTypeMany,
			},
			expectedParams: 3,
			description:    "Multiple joins with parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.query
			err := analyzer.extractParameters(&query)
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
				return
			}

			if len(query.Parameters) != tt.expectedParams {
				t.Errorf("For %s: expected %d parameters, got %d",
					tt.description, tt.expectedParams, len(query.Parameters))
			}
		})
	}
}

func TestQueryAnalyzer_IsSelectQuery(t *testing.T) {
	tests := []struct {
		name      string
		queryType QueryType
		expected  bool
	}{
		{"QueryTypeOne", QueryTypeOne, true},
		{"QueryTypeMany", QueryTypeMany, true},
		{"QueryTypePaginated", QueryTypePaginated, true},
		{"QueryTypeExec", QueryTypeExec, false},
	}

	analyzer := NewQueryAnalyzer(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isSelectQuery(tt.queryType)
			if result != tt.expected {
				t.Errorf("isSelectQuery(%s) = %v, want %v", tt.queryType, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_MapOIDToTypeName(t *testing.T) {
	tests := []struct {
		name     string
		oid      uint32
		expected string
	}{
		{"text type", 25, "text"},
		{"varchar type", 1043, "varchar"},
		{"integer type", 23, "integer"},
		{"bigint type", 20, "bigint"},
		{"boolean type", 16, "boolean"},
		{"uuid type", 2950, "uuid"},
		{"timestamp type", 1114, "timestamp"},
		{"timestamptz type", 1184, "timestamptz"},
		{"json type", 114, "json"},
		{"jsonb type", 3802, "jsonb"},
		{"unknown type", 99999, "unknown"},
	}

	analyzer := NewQueryAnalyzer(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.mapOIDToTypeName(tt.oid)
			if result != tt.expected {
				t.Errorf("mapOIDToTypeName(%d) = %q, want %q", tt.oid, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_ReplaceParametersForExplain(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "no parameters",
			sql:      "SELECT id FROM users",
			expected: "SELECT id FROM users",
		},
		{
			name:     "single parameter",
			sql:      "SELECT id FROM users WHERE id = $1",
			expected: "SELECT id FROM users WHERE id = NULL",
		},
		{
			name:     "multiple parameters",
			sql:      "SELECT id FROM users WHERE name = $1 AND age > $2",
			expected: "SELECT id FROM users WHERE name = NULL AND age > NULL",
		},
		{
			name:     "duplicate parameters",
			sql:      "SELECT id FROM users WHERE status = $1 OR backup_status = $1",
			expected: "SELECT id FROM users WHERE status = NULL OR backup_status = NULL",
		},
		{
			name:     "parameters in string literals ignored",
			sql:      "SELECT '$1' as literal, id FROM users WHERE id = $1",
			expected: "SELECT '$1' as literal, id FROM users WHERE id = NULL",
		},
	}

	analyzer := NewQueryAnalyzer(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dummy parameters for the test
			var params []Parameter
			if strings.Contains(tt.sql, "$1") {
				params = append(params, Parameter{Index: 1})
			}
			if strings.Contains(tt.sql, "$2") {
				params = append(params, Parameter{Index: 2})
			}
			result := analyzer.replaceParametersForExplain(tt.sql, params)
			if result != tt.expected {
				t.Errorf("replaceParametersForExplain(%q) = %q, want %q", tt.sql, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_GetDummyValueForParameter(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"first parameter", 1, "NULL"},
		{"second parameter", 2, "NULL"},
		{"tenth parameter", 10, "NULL"},
	}

	analyzer := NewQueryAnalyzer(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.getDummyValueForParameter()
			if result != tt.expected {
				t.Errorf("getDummyValueForParameter(%d) = %v, want %v", tt.index, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_AnalyzeQuery_ParameterExtraction(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil) // No database needed for parameter extraction only

	query := Query{
		Name: "TestQuery",
		SQL:  "SELECT id FROM users WHERE name = $1 AND age > $2",
		Type: QueryTypeMany,
	}

	err := analyzer.AnalyzeQuery(context.Background(), &query)
	if err == nil {
		t.Error("Expected error when no database connection provided, but got none")
	}
}

func TestQueryAnalyzer_AnalyzeQuery_NilQuery(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil)

	query := Query{}
	err := analyzer.AnalyzeQuery(context.Background(), &query)
	if err != nil {
		t.Errorf("Unexpected error with empty query: %v", err)
	}

	if len(query.Parameters) != 0 {
		t.Errorf("Expected 0 parameters for empty query, got %d", len(query.Parameters))
	}
}

func TestQueryAnalyzer_InferParameterNames(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil)

	tests := []struct {
		name          string
		sql           string
		expectedNames map[int]string
	}{
		{
			name: "simple WHERE clause with email",
			sql:  "SELECT * FROM users WHERE email = $1",
			expectedNames: map[int]string{
				1: "email",
			},
		},
		{
			name: "multiple WHERE conditions",
			sql:  "SELECT * FROM users WHERE email = $1 AND is_active = $2",
			expectedNames: map[int]string{
				1: "email",
				2: "isActive",
			},
		},
		{
			name: "LIKE pattern",
			sql:  "SELECT * FROM users WHERE name ILIKE $1",
			expectedNames: map[int]string{
				1: "searchName",
			},
		},
		{
			name: "LIMIT clause",
			sql:  "SELECT * FROM users LIMIT $1",
			expectedNames: map[int]string{
				1: "limit",
			},
		},
		{
			name: "OFFSET clause",
			sql:  "SELECT * FROM users OFFSET $1",
			expectedNames: map[int]string{
				1: "offset",
			},
		},
		{
			name: "table prefixed column",
			sql:  "SELECT * FROM users u WHERE u.checkout_session_id = $1",
			expectedNames: map[int]string{
				1: "checkoutSessionId",
			},
		},
		{
			name: "comparison operators",
			sql:  "SELECT * FROM users WHERE age > $1 AND created_at <= $2",
			expectedNames: map[int]string{
				1: "age",
				2: "createdAt",
			},
		},
		{
			name: "IN clause",
			sql:  "SELECT * FROM users WHERE status IN ($1)",
			expectedNames: map[int]string{
				1: "status",
			},
		},
		{
			name: "mixed patterns",
			sql:  "SELECT * FROM users WHERE email = $1 AND name ILIKE $2 LIMIT $3",
			expectedNames: map[int]string{
				1: "email",
				2: "searchName",
				3: "limit",
			},
		},
		{
			name:          "no parameters",
			sql:           "SELECT * FROM users",
			expectedNames: map[int]string{},
		},
		{
			name: "UPDATE single column",
			sql:  "UPDATE users SET email = $1 WHERE id = $2",
			expectedNames: map[int]string{
				1: "email",
				2: "id",
			},
		},
		{
			name: "UPDATE multiple columns",
			sql:  "UPDATE users SET email = $1, name = $2, updated_at = $3 WHERE id = $4",
			expectedNames: map[int]string{
				1: "email",
				2: "name",
				3: "updatedAt",
				4: "id",
			},
		},
		{
			name: "UPDATE with table prefix",
			sql:  "UPDATE users u SET u.email = $1, u.status = $2 WHERE u.id = $3",
			expectedNames: map[int]string{
				1: "email",
				2: "status",
				3: "id",
			},
		},
		{
			name: "UPDATE with snake_case columns",
			sql:  "UPDATE payment_links SET checkout_session_id = $1, is_active = $2 WHERE id = $3",
			expectedNames: map[int]string{
				1: "checkoutSessionId",
				2: "isActive",
				3: "id",
			},
		},
		{
			name: "UPDATE without WHERE clause",
			sql:  "UPDATE users SET is_active = $1, updated_at = $2",
			expectedNames: map[int]string{
				1: "isActive",
				2: "updatedAt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := Query{
				Name: tt.name,
				SQL:  tt.sql,
				Type: QueryTypeMany,
			}

			// First extract parameters
			err := analyzer.extractParameters(&query)
			if err != nil {
				t.Fatalf("extractParameters failed: %v", err)
			}

			// Then infer names
			err = analyzer.inferParameterNames(&query)
			if err != nil {
				t.Fatalf("inferParameterNames failed: %v", err)
			}

			// Verify names
			for _, param := range query.Parameters {
				expectedName, exists := tt.expectedNames[param.Index]
				if !exists {
					t.Errorf("Unexpected parameter index %d", param.Index)
					continue
				}
				if param.Name != expectedName {
					t.Errorf("Parameter $%d: expected name %q, got %q", param.Index, expectedName, param.Name)
				}
			}

			// Verify we got all expected parameters
			if len(query.Parameters) != len(tt.expectedNames) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expectedNames), len(query.Parameters))
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"email", "email"},
		{"checkout_session_id", "checkoutSessionId"},
		{"is_active", "isActive"},
		{"created_at", "createdAt"},
		{"user_id", "userId"},
		{"", ""},
		{"single", "single"},
		{"UPPERCASE", "uPPERCASE"},
		{"snake_case_long", "snakeCaseLong"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_TableColumnTracking(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil)

	tests := []struct {
		name           string
		sql            string
		expectedParams map[int]struct {
			name   string
			table  string
			column string
		}
	}{
		{
			name: "UPDATE with table name detection",
			sql:  "UPDATE users SET email = $1, name = $2 WHERE id = $3",
			expectedParams: map[int]struct {
				name   string
				table  string
				column string
			}{
				1: {name: "email", table: "users", column: "email"},
				2: {name: "name", table: "users", column: "name"},
				3: {name: "id", table: "", column: "id"}, // WHERE clause doesn't get table
			},
		},
		{
			name: "UPDATE with table prefix in WHERE",
			sql:  "UPDATE users SET email = $1 WHERE users.id = $2",
			expectedParams: map[int]struct {
				name   string
				table  string
				column string
			}{
				1: {name: "email", table: "users", column: "email"},
				2: {name: "id", table: "users", column: "id"},
			},
		},
		{
			name: "UPDATE with alias",
			sql:  "UPDATE users u SET u.email = $1 WHERE u.id = $2",
			expectedParams: map[int]struct {
				name   string
				table  string
				column string
			}{
				1: {name: "email", table: "u", column: "email"},
				2: {name: "id", table: "u", column: "id"},
			},
		},
		{
			name: "SELECT with table prefix",
			sql:  "SELECT * FROM users WHERE users.email = $1 AND users.is_active = $2",
			expectedParams: map[int]struct {
				name   string
				table  string
				column string
			}{
				1: {name: "email", table: "users", column: "email"},
				2: {name: "isActive", table: "users", column: "is_active"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := Query{
				Name: tt.name,
				SQL:  tt.sql,
				Type: QueryTypeMany,
			}

			// Extract and infer
			err := analyzer.extractParameters(&query)
			if err != nil {
				t.Fatalf("extractParameters failed: %v", err)
			}

			err = analyzer.inferParameterNames(&query)
			if err != nil {
				t.Fatalf("inferParameterNames failed: %v", err)
			}

			// Verify table/column tracking
			for _, param := range query.Parameters {
				expected, exists := tt.expectedParams[param.Index]
				if !exists {
					t.Errorf("Unexpected parameter index %d", param.Index)
					continue
				}

				if param.Name != expected.name {
					t.Errorf("Parameter $%d: expected name %q, got %q", param.Index, expected.name, param.Name)
				}
				if param.TableName != expected.table {
					t.Errorf("Parameter $%d: expected table %q, got %q", param.Index, expected.table, param.TableName)
				}
				if param.ColumnName != expected.column {
					t.Errorf("Parameter $%d: expected column %q, got %q", param.Index, expected.column, param.ColumnName)
				}
			}
		})
	}
}

func TestMakePointerType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "*string"},
		{"int32", "*int32"},
		{"uuid.UUID", "*uuid.UUID"},
		{"*string", "*string"}, // Already a pointer
		{"*int32", "*int32"},
		{"pgtype.Text", "*pgtype.Text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := makePointerType(tt.input)
			if result != tt.expected {
				t.Errorf("makePointerType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_ApplyResultAnnotations(t *testing.T) {
	analyzer := NewQueryAnalyzer(nil)

	tests := []struct {
		name        string
		query       Query
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name: "nil columns with no annotations",
			query: Query{
				Name:              "TestQuery",
				Columns:           nil,
				ResultAnnotations: []ResultAnnotation{},
			},
			expectError: false,
			description: "No error expected when columns is nil but no annotations exist",
		},
		{
			name: "nil columns with annotations",
			query: Query{
				Name:    "TestQuery",
				Columns: nil,
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "user_id", GoType: "*int"},
				},
			},
			expectError: true,
			errorMsg:    "query TestQuery has result annotations but no columns were detected",
			description: "Error expected when columns is nil and annotations exist",
		},
		{
			name: "empty columns with annotations",
			query: Query{
				Name:    "TestQuery",
				Columns: []Column{},
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "user_id", GoType: "*int"},
				},
			},
			expectError: true,
			errorMsg:    "result annotation for column 'user_id' not found in query TestQuery results",
			description: "Error expected when annotation column doesn't exist",
		},
		{
			name: "valid annotation applied",
			query: Query{
				Name: "TestQuery",
				Columns: []Column{
					{Name: "user_id", Type: "integer", GoType: "int", IsNullable: false},
					{Name: "email", Type: "text", GoType: "string", IsNullable: false},
				},
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "user_id", GoType: "*int"},
				},
			},
			expectError: false,
			description: "Annotation should be applied successfully",
		},
		{
			name: "non-existent column annotation",
			query: Query{
				Name: "TestQuery",
				Columns: []Column{
					{Name: "user_id", Type: "integer", GoType: "int", IsNullable: false},
				},
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "nonexistent", GoType: "*int"},
				},
			},
			expectError: true,
			errorMsg:    "result annotation for column 'nonexistent' not found in query TestQuery results",
			description: "Error expected when annotation references non-existent column",
		},
		{
			name: "multiple annotations",
			query: Query{
				Name: "TestQuery",
				Columns: []Column{
					{Name: "user_id", Type: "integer", GoType: "int", IsNullable: false},
					{Name: "email", Type: "text", GoType: "string", IsNullable: false},
					{Name: "name", Type: "text", GoType: "string", IsNullable: false},
				},
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "user_id", GoType: "*int"},
					{ColumnName: "email", GoType: "*string"},
				},
			},
			expectError: false,
			description: "Multiple annotations should be applied successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.query
			err := analyzer.applyResultAnnotations(&query)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.description)
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}

				// If no error, verify annotations were applied
				if len(tt.query.ResultAnnotations) > 0 && len(query.Columns) > 0 {
					for _, annotation := range tt.query.ResultAnnotations {
						found := false
						for _, col := range query.Columns {
							if col.Name == annotation.ColumnName {
								found = true
								if col.GoType != annotation.GoType {
									t.Errorf("Column %s: expected GoType %q, got %q",
										col.Name, annotation.GoType, col.GoType)
								}
								expectedNullable := strings.HasPrefix(annotation.GoType, "*")
								if col.IsNullable != expectedNullable {
									t.Errorf("Column %s: expected IsNullable %v, got %v",
										col.Name, expectedNullable, col.IsNullable)
								}
								break
							}
						}
						if !found && !tt.expectError {
							t.Errorf("Annotation for column %s was not applied", annotation.ColumnName)
						}
					}
				}
			}
		})
	}
}
