package generator

import (
	"testing"
)

func TestQueryParser_ParseAnnotation(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		line     string
		expected *QueryAnnotation
	}{
		{
			name:     "basic annotation",
			line:     "-- name: GetUser :one",
			expected: &QueryAnnotation{Name: "GetUser", Type: QueryTypeOne},
		},
		{
			name:     "annotation with extra spaces",
			line:     "--   name:   GetUser   :one   ",
			expected: &QueryAnnotation{Name: "GetUser", Type: QueryTypeOne},
		},
		{
			name:     "annotation with semicolon",
			line:     "-- name: GetUser :one;",
			expected: &QueryAnnotation{Name: "GetUser", Type: QueryTypeOne},
		},
		{
			name:     "many type",
			line:     "-- name: ListUsers :many",
			expected: &QueryAnnotation{Name: "ListUsers", Type: QueryTypeMany},
		},
		{
			name:     "exec type",
			line:     "-- name: CreateUser :exec",
			expected: &QueryAnnotation{Name: "CreateUser", Type: QueryTypeExec},
		},
		{
			name:     "paginated type",
			line:     "-- name: GetUsersPaginated :paginated",
			expected: &QueryAnnotation{Name: "GetUsersPaginated", Type: QueryTypePaginated},
		},
		{
			name:     "underscore in name",
			line:     "-- name: get_user_by_email :one",
			expected: &QueryAnnotation{Name: "get_user_by_email", Type: QueryTypeOne},
		},
		{
			name:     "invalid format",
			line:     "-- name GetUser :one",
			expected: nil,
		},
		{
			name:     "invalid type",
			line:     "-- name: GetUser :invalid",
			expected: nil,
		},
		{
			name:     "regular comment",
			line:     "-- This is a regular comment",
			expected: nil,
		},
		{
			name:     "empty line",
			line:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseAnnotation(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %+v, got nil", tt.expected)
				return
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Expected name %s, got %s", tt.expected.Name, result.Name)
			}

			if result.Type != tt.expected.Type {
				t.Errorf("Expected type %s, got %s", tt.expected.Type, result.Type)
			}
		})
	}
}

func TestQueryParser_ParseQueryType(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		input    string
		expected QueryType
		hasError bool
	}{
		{"one", "one", QueryTypeOne, false},
		{"many", "many", QueryTypeMany, false},
		{"exec", "exec", QueryTypeExec, false},
		{"paginated", "paginated", QueryTypePaginated, false},
		{"ONE uppercase", "ONE", QueryTypeOne, false},
		{"Many mixed case", "Many", QueryTypeMany, false},
		{"invalid type", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseQueryType(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestQueryParser_ValidateQuery(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		query    Query
		hasError bool
	}{
		{
			name: "valid select one query",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeOne,
				SQL:  "SELECT id, name FROM users WHERE id = $1",
			},
			hasError: false,
		},
		{
			name: "valid select many query",
			query: Query{
				Name: "ListUsers",
				Type: QueryTypeMany,
				SQL:  "SELECT id, name FROM users ORDER BY name",
			},
			hasError: false,
		},
		{
			name: "valid exec query",
			query: Query{
				Name: "CreateUser",
				Type: QueryTypeExec,
				SQL:  "INSERT INTO users (name) VALUES ($1)",
			},
			hasError: false,
		},
		{
			name: "valid paginated query",
			query: Query{
				Name: "GetUsersPaginated",
				Type: QueryTypePaginated,
				SQL:  "SELECT id, name FROM users ORDER BY id LIMIT $1",
			},
			hasError: false,
		},
		{
			name: "valid CTE query",
			query: Query{
				Name: "GetUsersWithCTE",
				Type: QueryTypeMany,
				SQL:  "WITH active_users AS (SELECT id FROM users WHERE active = true) SELECT * FROM active_users",
			},
			hasError: false,
		},
		{
			name: "empty name",
			query: Query{
				Name: "",
				Type: QueryTypeOne,
				SQL:  "SELECT id FROM users",
			},
			hasError: true,
		},
		{
			name: "empty SQL",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeOne,
				SQL:  "",
			},
			hasError: true,
		},
		{
			name: "empty type",
			query: Query{
				Name: "GetUser",
				Type: "",
				SQL:  "SELECT id FROM users",
			},
			hasError: true,
		},
		{
			name: "invalid Go identifier",
			query: Query{
				Name: "123GetUser",
				Type: QueryTypeOne,
				SQL:  "SELECT id FROM users",
			},
			hasError: true,
		},
		{
			name: "select with exec type",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeExec,
				SQL:  "SELECT id FROM users",
			},
			hasError: true,
		},
		{
			name: "CTE with exec type",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeExec,
				SQL:  "WITH active_users AS (SELECT id FROM users WHERE active = true) SELECT * FROM active_users",
			},
			hasError: true,
		},
		{
			name: "insert with one type",
			query: Query{
				Name: "CreateUser",
				Type: QueryTypeOne,
				SQL:  "INSERT INTO users (name) VALUES ($1)",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateQuery(tt.query)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestQueryParser_IsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid identifier", "GetUser", true},
		{"underscore prefix", "_GetUser", true},
		{"with numbers", "GetUser123", true},
		{"with underscores", "get_user_by_email", true},
		{"single letter", "a", true},
		{"single underscore", "_", true},
		{"empty string", "", false},
		{"starts with number", "123GetUser", false},
		{"with spaces", "Get User", false},
		{"with hyphens", "get-user", false},
		{"with dots", "get.user", false},
		{"with special chars", "get@user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGoIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("isValidGoIdentifier(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQueryParser_ParseResultAnnotation(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		line     string
		expected *ResultAnnotation
	}{
		{
			name:     "basic result annotation",
			line:     "-- result: payment_count int",
			expected: &ResultAnnotation{ColumnName: "payment_count", GoType: "int"},
		},
		{
			name:     "result annotation with pointer type",
			line:     "-- result: total_amount *int",
			expected: &ResultAnnotation{ColumnName: "total_amount", GoType: "*int"},
		},
		{
			name:     "result annotation with qualified type",
			line:     "-- result: created_at time.Time",
			expected: &ResultAnnotation{ColumnName: "created_at", GoType: "time.Time"},
		},
		{
			name:     "result annotation with pointer qualified type",
			line:     "-- result: user_id *uuid.UUID",
			expected: &ResultAnnotation{ColumnName: "user_id", GoType: "*uuid.UUID"},
		},
		{
			name:     "result annotation with extra spaces",
			line:     "--   result:   payment_count   int   ",
			expected: &ResultAnnotation{ColumnName: "payment_count", GoType: "int"},
		},
		{
			name:     "result annotation with underscore in name",
			line:     "-- result: payment_method_count int64",
			expected: &ResultAnnotation{ColumnName: "payment_method_count", GoType: "int64"},
		},
		{
			name:     "invalid format - missing colon",
			line:     "-- result payment_count int",
			expected: nil,
		},
		{
			name:     "invalid format - missing type",
			line:     "-- result: payment_count",
			expected: nil,
		},
		{
			name:     "invalid format - missing column name",
			line:     "-- result: int",
			expected: nil,
		},
		{
			name:     "regular comment",
			line:     "-- This is a regular comment",
			expected: nil,
		},
		{
			name:     "empty line",
			line:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseResultAnnotation(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %+v, got nil", tt.expected)
				return
			}

			if result.ColumnName != tt.expected.ColumnName {
				t.Errorf("Expected column name %s, got %s", tt.expected.ColumnName, result.ColumnName)
			}

			if result.GoType != tt.expected.GoType {
				t.Errorf("Expected Go type %s, got %s", tt.expected.GoType, result.GoType)
			}
		})
	}
}

func TestQueryParser_ValidateResultAnnotations(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		query    Query
		hasError bool
		errorMsg string
	}{
		{
			name: "valid result annotations",
			query: Query{
				Name: "GetPaymentSummary",
				Type: QueryTypeOne,
				SQL:  "SELECT payment_count, total_amount FROM payments",
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "payment_count", GoType: "int"},
					{ColumnName: "total_amount", GoType: "*int64"},
				},
			},
			hasError: false,
		},
		{
			name: "valid result annotation with qualified type",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeOne,
				SQL:  "SELECT id, created_at FROM users",
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "id", GoType: "uuid.UUID"},
					{ColumnName: "created_at", GoType: "time.Time"},
				},
			},
			hasError: false,
		},
		{
			name: "duplicate column names",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeOne,
				SQL:  "SELECT id FROM users",
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "payment_count", GoType: "int"},
					{ColumnName: "payment_count", GoType: "int64"},
				},
			},
			hasError: true,
			errorMsg: "duplicate result annotation",
		},
		{
			name: "invalid Go type",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeOne,
				SQL:  "SELECT id FROM users",
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "payment_count", GoType: "invalid type"},
				},
			},
			hasError: true,
			errorMsg: "invalid Go type",
		},
		{
			name: "invalid Go type with special chars",
			query: Query{
				Name: "GetUser",
				Type: QueryTypeOne,
				SQL:  "SELECT id FROM users",
				ResultAnnotations: []ResultAnnotation{
					{ColumnName: "payment_count", GoType: "int@invalid"},
				},
			},
			hasError: true,
			errorMsg: "invalid Go type",
		},
		{
			name: "empty result annotations",
			query: Query{
				Name:              "GetUser",
				Type:              QueryTypeOne,
				SQL:               "SELECT id FROM users",
				ResultAnnotations: []ResultAnnotation{},
			},
			hasError: false,
		},
		{
			name: "nil result annotations",
			query: Query{
				Name:              "GetUser",
				Type:              QueryTypeOne,
				SQL:               "SELECT id FROM users",
				ResultAnnotations: nil,
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateResultAnnotations(&tt.query)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestQueryParser_IsValidGoType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"basic type", "int", true},
		{"pointer type", "*int", true},
		{"qualified type", "uuid.UUID", true},
		{"pointer qualified type", "*time.Time", true},
		{"int64", "int64", true},
		{"string type", "string", true},
		{"custom type", "MyCustomType", true},
		{"qualified custom type", "pkg.MyType", true},
		{"empty string", "", false},
		{"invalid with space", "int 64", false},
		{"invalid with special char", "int@type", false},
		{"too many dots", "pkg.sub.Type", false},
		{"starts with number", "123Type", false},
		{"invalid qualified", ".Type", false},
		{"invalid qualified end", "pkg.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGoType(tt.input)
			if result != tt.expected {
				t.Errorf("isValidGoType(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQueryParser_ParseParameterAnnotation(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		line     string
		expected *ParameterAnnotation
	}{
		{
			name: "basic nullable string",
			line: "-- param: $1 status *string",
			expected: &ParameterAnnotation{
				Position: 1,
				Name:     "status",
				GoType:   "*string",
			},
		},
		{
			name: "non-nullable UUID",
			line: "-- param: $2 tenant_id uuid.UUID",
			expected: &ParameterAnnotation{
				Position: 2,
				Name:     "tenant_id",
				GoType:   "uuid.UUID",
			},
		},
		{
			name: "nullable time",
			line: "-- param: $3 created_after *time.Time",
			expected: &ParameterAnnotation{
				Position: 3,
				Name:     "created_after",
				GoType:   "*time.Time",
			},
		},
		{
			name: "nullable int",
			line: "-- param: $4 limit *int",
			expected: &ParameterAnnotation{
				Position: 4,
				Name:     "limit",
				GoType:   "*int",
			},
		},
		{
			name:     "invalid - no dollar sign",
			line:     "-- param: 1 status string",
			expected: nil,
		},
		{
			name:     "invalid - missing type",
			line:     "-- param: $1 status",
			expected: nil,
		},
		{
			name:     "not a param annotation",
			line:     "-- some other comment",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseParameterAnnotation(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected %+v, got nil", tt.expected)
			}

			if result.Position != tt.expected.Position {
				t.Errorf("Position: expected %d, got %d", tt.expected.Position, result.Position)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name: expected %s, got %s", tt.expected.Name, result.Name)
			}
			if result.GoType != tt.expected.GoType {
				t.Errorf("GoType: expected %s, got %s", tt.expected.GoType, result.GoType)
			}
		})
	}
}

func TestQueryParser_ValidateParameterAnnotations(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name        string
		annotations []ParameterAnnotation
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid sequential annotations",
			annotations: []ParameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "uuid.UUID"},
				{Position: 2, Name: "status", GoType: "*string"},
				{Position: 3, Name: "limit", GoType: "int"},
			},
			shouldError: false,
		},
		{
			name:        "empty annotations is valid",
			annotations: []ParameterAnnotation{},
			shouldError: false,
		},
		{
			name: "duplicate position",
			annotations: []ParameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "uuid.UUID"},
				{Position: 1, Name: "status", GoType: "string"},
			},
			shouldError: true,
			errorMsg:    "duplicate parameter annotation for $1",
		},
		{
			name: "non-sequential (missing $2)",
			annotations: []ParameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "uuid.UUID"},
				{Position: 3, Name: "limit", GoType: "int"},
			},
			shouldError: true,
			errorMsg:    "parameter annotations must be sequential starting at $1, missing $2",
		},
		{
			name: "invalid Go type",
			annotations: []ParameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "invalid type"},
			},
			shouldError: true,
			errorMsg:    "invalid Go type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &Query{
				Name:                 "TestQuery",
				ParameterAnnotations: tt.annotations,
			}

			err := parser.validateParameterAnnotations(query)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestQueryParser_ParseCursorColumnsAnnotation(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "basic cursor_columns with two columns",
			line:     "-- cursor_columns: id, created_at",
			expected: []string{"id", "created_at"},
		},
		{
			name:     "cursor_columns with extra spaces",
			line:     "--   cursor_columns:   id,   created_at,   updated_at   ",
			expected: []string{"id", "created_at", "updated_at"},
		},
		{
			name:     "single column",
			line:     "-- cursor_columns: id",
			expected: []string{"id"},
		},
		{
			name:     "three columns",
			line:     "-- cursor_columns: published_at, id, title",
			expected: []string{"published_at", "id", "title"},
		},
		{
			name:     "columns with underscores",
			line:     "-- cursor_columns: user_id, created_at, updated_at",
			expected: []string{"user_id", "created_at", "updated_at"},
		},
		{
			name:     "invalid - missing colon",
			line:     "-- cursor_columns id, created_at",
			expected: nil,
		},
		{
			name:     "invalid - empty columns",
			line:     "-- cursor_columns:",
			expected: nil,
		},
		{
			name:     "invalid - only spaces after colon",
			line:     "-- cursor_columns:    ",
			expected: nil,
		},
		{
			name:     "regular comment",
			line:     "-- This is a regular comment",
			expected: nil,
		},
		{
			name:     "empty line",
			line:     "",
			expected: nil,
		},
		{
			name:     "annotation with trailing comma",
			line:     "-- cursor_columns: id, created_at,",
			expected: []string{"id", "created_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseCursorColumnsAnnotation(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %+v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d columns, got %d", len(tt.expected), len(result))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Column %d: expected %s, got %s", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestQueryParser_ValidateCursorColumns(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		query    Query
		hasError bool
		errorMsg string
	}{
		{
			name: "valid cursor_columns with :many query",
			query: Query{
				Name:          "GetPublishedPosts",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, title, published_at FROM posts WHERE is_published = true",
				CursorColumns: []string{"published_at", "id"},
			},
			hasError: false,
		},
		{
			name: "valid cursor_columns single column",
			query: Query{
				Name:          "ListUsers",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, name FROM users",
				CursorColumns: []string{"id"},
			},
			hasError: false,
		},
		{
			name: "valid cursor_columns multiple columns",
			query: Query{
				Name:          "SearchPosts",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, title, created_at, updated_at FROM posts",
				CursorColumns: []string{"created_at", "id", "title"},
			},
			hasError: false,
		},
		{
			name: "empty cursor_columns is valid",
			query: Query{
				Name:          "ListUsers",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, name FROM users",
				CursorColumns: []string{},
			},
			hasError: false,
		},
		{
			name: "nil cursor_columns is valid",
			query: Query{
				Name:          "ListUsers",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, name FROM users",
				CursorColumns: nil,
			},
			hasError: false,
		},
		{
			name: "invalid - cursor_columns with :one query",
			query: Query{
				Name:          "GetPost",
				Type:          QueryTypeOne,
				SQL:           "SELECT id, title FROM posts WHERE id = $1",
				CursorColumns: []string{"id", "created_at"},
			},
			hasError: true,
			errorMsg: "cursor_columns annotation only valid for :many queries",
		},
		{
			name: "invalid - cursor_columns with :exec query",
			query: Query{
				Name:          "CreatePost",
				Type:          QueryTypeExec,
				SQL:           "INSERT INTO posts (title) VALUES ($1)",
				CursorColumns: []string{"id"},
			},
			hasError: true,
			errorMsg: "cursor_columns annotation only valid for :many queries",
		},
		{
			name: "invalid - cursor_columns with :paginated query",
			query: Query{
				Name:          "GetPostsPaginated",
				Type:          QueryTypePaginated,
				SQL:           "SELECT id, title FROM posts",
				CursorColumns: []string{"id"},
			},
			hasError: true,
			errorMsg: "cursor_columns annotation only valid for :many queries",
		},
		{
			name: "invalid - cursor column with invalid characters",
			query: Query{
				Name:          "ListUsers",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, name FROM users",
				CursorColumns: []string{"id", "invalid-name"},
			},
			hasError: true,
			errorMsg: "invalid column name",
		},
		{
			name: "invalid - cursor column starting with number",
			query: Query{
				Name:          "ListUsers",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, name FROM users",
				CursorColumns: []string{"123invalid"},
			},
			hasError: true,
			errorMsg: "invalid column name",
		},
		{
			name: "invalid - cursor column with spaces",
			query: Query{
				Name:          "ListUsers",
				Type:          QueryTypeMany,
				SQL:           "SELECT id, name FROM users",
				CursorColumns: []string{"invalid name"},
			},
			hasError: true,
			errorMsg: "invalid column name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateCursorColumns(&tt.query)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
