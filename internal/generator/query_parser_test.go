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
