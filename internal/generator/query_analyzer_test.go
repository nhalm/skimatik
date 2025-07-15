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
			result := analyzer.getDummyValueForParameter(tt.index)
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
