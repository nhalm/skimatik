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
			expected: &QueryAnnotation{Name: "GetUser", Type: queryTypeOne},
		},
		{
			name:     "annotation with extra spaces",
			line:     "--   name:   GetUser   :one   ",
			expected: &QueryAnnotation{Name: "GetUser", Type: queryTypeOne},
		},
		{
			name:     "annotation with semicolon",
			line:     "-- name: GetUser :one;",
			expected: &QueryAnnotation{Name: "GetUser", Type: queryTypeOne},
		},
		{
			name:     "many type",
			line:     "-- name: ListUsers :many",
			expected: &QueryAnnotation{Name: "ListUsers", Type: queryTypeMany},
		},
		{
			name:     "exec type",
			line:     "-- name: CreateUser :exec",
			expected: &QueryAnnotation{Name: "CreateUser", Type: queryTypeExec},
		},
		{
			name:     "paginated type",
			line:     "-- name: GetUsersPaginated :paginated",
			expected: &QueryAnnotation{Name: "GetUsersPaginated", Type: queryTypePaginated},
		},
		{
			name:     "underscore in name",
			line:     "-- name: get_user_by_email :one",
			expected: &QueryAnnotation{Name: "get_user_by_email", Type: queryTypeOne},
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
		expected queryType
		hasError bool
	}{
		{"one", "one", queryTypeOne, false},
		{"many", "many", queryTypeMany, false},
		{"exec", "exec", queryTypeExec, false},
		{"paginated", "paginated", queryTypePaginated, false},
		{"ONE uppercase", "ONE", queryTypeOne, false},
		{"Many mixed case", "Many", queryTypeMany, false},
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

func TestQueryParser_ParseResultAnnotation(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		line     string
		expected *resultAnnotation
	}{
		{
			name:     "basic result annotation",
			line:     "-- result: payment_count int",
			expected: &resultAnnotation{ColumnName: "payment_count", GoType: "int"},
		},
		{
			name:     "result annotation with pointer type",
			line:     "-- result: total_amount *int",
			expected: &resultAnnotation{ColumnName: "total_amount", GoType: "*int"},
		},
		{
			name:     "result annotation with qualified type",
			line:     "-- result: created_at time.Time",
			expected: &resultAnnotation{ColumnName: "created_at", GoType: "time.Time"},
		},
		{
			name:     "result annotation with pointer qualified type",
			line:     "-- result: user_id *uuid.UUID",
			expected: &resultAnnotation{ColumnName: "user_id", GoType: "*uuid.UUID"},
		},
		{
			name:     "result annotation with extra spaces",
			line:     "--   result:   payment_count   int   ",
			expected: &resultAnnotation{ColumnName: "payment_count", GoType: "int"},
		},
		{
			name:     "result annotation with underscore in name",
			line:     "-- result: payment_method_count int64",
			expected: &resultAnnotation{ColumnName: "payment_method_count", GoType: "int64"},
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
		query    query
		hasError bool
		errorMsg string
	}{
		{
			name: "valid result annotations",
			query: query{
				Name: "GetPaymentSummary",
				Type: queryTypeOne,
				SQL:  "SELECT payment_count, total_amount FROM payments",
				ResultAnnotations: []resultAnnotation{
					{ColumnName: "payment_count", GoType: "int"},
					{ColumnName: "total_amount", GoType: "*int64"},
				},
			},
			hasError: false,
		},
		{
			name: "valid result annotation with qualified type",
			query: query{
				Name: "GetUser",
				Type: queryTypeOne,
				SQL:  "SELECT id, created_at FROM users",
				ResultAnnotations: []resultAnnotation{
					{ColumnName: "id", GoType: "uuid.UUID"},
					{ColumnName: "created_at", GoType: "time.Time"},
				},
			},
			hasError: false,
		},
		{
			name: "duplicate column names",
			query: query{
				Name: "GetUser",
				Type: queryTypeOne,
				SQL:  "SELECT id FROM users",
				ResultAnnotations: []resultAnnotation{
					{ColumnName: "payment_count", GoType: "int"},
					{ColumnName: "payment_count", GoType: "int64"},
				},
			},
			hasError: true,
			errorMsg: "duplicate result annotation",
		},
		{
			name: "empty result annotations",
			query: query{
				Name:              "GetUser",
				Type:              queryTypeOne,
				SQL:               "SELECT id FROM users",
				ResultAnnotations: []resultAnnotation{},
			},
			hasError: false,
		},
		{
			name: "nil result annotations",
			query: query{
				Name:              "GetUser",
				Type:              queryTypeOne,
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
		{"slice of bytes", "[]byte", true},
		{"pointer to slice", "*[]byte", true},
		{"json.RawMessage", "json.RawMessage", true},
		{"pointer json.RawMessage", "*json.RawMessage", true},
		{"slice of strings", "[]string", true},
		{"map type", "map[string]any", true},
		{"deeply qualified", "pkg.sub.Type", true},
		{"empty string", "", false},
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
		expected *parameterAnnotation
	}{
		{
			name: "basic nullable string",
			line: "-- param: $1 status *string",
			expected: &parameterAnnotation{
				Position: 1,
				Name:     "status",
				GoType:   "*string",
			},
		},
		{
			name: "non-nullable UUID",
			line: "-- param: $2 tenant_id uuid.UUID",
			expected: &parameterAnnotation{
				Position: 2,
				Name:     "tenant_id",
				GoType:   "uuid.UUID",
			},
		},
		{
			name: "nullable time",
			line: "-- param: $3 created_after *time.Time",
			expected: &parameterAnnotation{
				Position: 3,
				Name:     "created_after",
				GoType:   "*time.Time",
			},
		},
		{
			name: "nullable int",
			line: "-- param: $4 limit *int",
			expected: &parameterAnnotation{
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
		annotations []parameterAnnotation
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid sequential annotations",
			annotations: []parameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "uuid.UUID"},
				{Position: 2, Name: "status", GoType: "*string"},
				{Position: 3, Name: "limit", GoType: "int"},
			},
			shouldError: false,
		},
		{
			name:        "empty annotations is valid",
			annotations: []parameterAnnotation{},
			shouldError: false,
		},
		{
			name: "duplicate position",
			annotations: []parameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "uuid.UUID"},
				{Position: 1, Name: "status", GoType: "string"},
			},
			shouldError: true,
			errorMsg:    "duplicate parameter annotation for $1",
		},
		{
			name: "non-sequential (missing $2)",
			annotations: []parameterAnnotation{
				{Position: 1, Name: "tenant_id", GoType: "uuid.UUID"},
				{Position: 3, Name: "limit", GoType: "int"},
			},
			shouldError: true,
			errorMsg:    "parameter annotations must be sequential starting at $1, missing $2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &query{
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
