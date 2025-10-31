package generator

import (
	"testing"
)

func TestParseParameterAnnotation(t *testing.T) {
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

func TestValidateParameterAnnotations(t *testing.T) {
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

func TestParseResultAnnotation(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name     string
		line     string
		expected *ResultAnnotation
	}{
		{
			name: "basic non-nullable int",
			line: "-- result: payment_count int",
			expected: &ResultAnnotation{
				ColumnName: "payment_count",
				GoType:     "int",
			},
		},
		{
			name: "nullable int pointer",
			line: "-- result: total_amount *int",
			expected: &ResultAnnotation{
				ColumnName: "total_amount",
				GoType:     "*int",
			},
		},
		{
			name: "UUID type",
			line: "-- result: user_id uuid.UUID",
			expected: &ResultAnnotation{
				ColumnName: "user_id",
				GoType:     "uuid.UUID",
			},
		},
		{
			name: "nullable time pointer",
			line: "-- result: created_at *time.Time",
			expected: &ResultAnnotation{
				ColumnName: "created_at",
				GoType:     "*time.Time",
			},
		},
		{
			name: "string type",
			line: "-- result: name string",
			expected: &ResultAnnotation{
				ColumnName: "name",
				GoType:     "string",
			},
		},
		{
			name: "nullable string pointer",
			line: "-- result: description *string",
			expected: &ResultAnnotation{
				ColumnName: "description",
				GoType:     "*string",
			},
		},
		{
			name: "extra whitespace",
			line: "-- result:   amount   *int  ",
			expected: &ResultAnnotation{
				ColumnName: "amount",
				GoType:     "*int",
			},
		},
		{
			name: "json.RawMessage type",
			line: "-- result: metadata json.RawMessage",
			expected: &ResultAnnotation{
				ColumnName: "metadata",
				GoType:     "json.RawMessage",
			},
		},
		{
			name:     "invalid - missing type",
			line:     "-- result: column_name",
			expected: nil,
		},
		{
			name:     "invalid - missing column name",
			line:     "-- result: int",
			expected: nil,
		},
		{
			name:     "invalid - empty annotation",
			line:     "-- result:",
			expected: nil,
		},
		{
			name:     "not a result annotation",
			line:     "-- some other comment",
			expected: nil,
		},
		{
			name:     "param annotation not result",
			line:     "-- param: $1 status string",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseResultAnnotation(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected %+v, got nil", tt.expected)
			}

			if result.ColumnName != tt.expected.ColumnName {
				t.Errorf("ColumnName: expected %s, got %s", tt.expected.ColumnName, result.ColumnName)
			}
			if result.GoType != tt.expected.GoType {
				t.Errorf("GoType: expected %s, got %s", tt.expected.GoType, result.GoType)
			}
		})
	}
}

func TestValidateResultAnnotations(t *testing.T) {
	parser := NewQueryParser("")

	tests := []struct {
		name        string
		annotations []ResultAnnotation
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid multiple annotations",
			annotations: []ResultAnnotation{
				{ColumnName: "payment_count", GoType: "int"},
				{ColumnName: "total_amount", GoType: "*int"},
				{ColumnName: "user_id", GoType: "uuid.UUID"},
			},
			shouldError: false,
		},
		{
			name: "valid single annotation",
			annotations: []ResultAnnotation{
				{ColumnName: "amount", GoType: "*int"},
			},
			shouldError: false,
		},
		{
			name:        "empty annotations is valid",
			annotations: []ResultAnnotation{},
			shouldError: false,
		},
		{
			name: "duplicate column name",
			annotations: []ResultAnnotation{
				{ColumnName: "payment_count", GoType: "int"},
				{ColumnName: "payment_count", GoType: "*int"},
			},
			shouldError: true,
			errorMsg:    "duplicate result annotation for column",
		},
		{
			name: "invalid Go type",
			annotations: []ResultAnnotation{
				{ColumnName: "amount", GoType: "invalid type"},
			},
			shouldError: true,
			errorMsg:    "invalid Go type",
		},
		{
			name: "invalid Go type with spaces",
			annotations: []ResultAnnotation{
				{ColumnName: "name", GoType: "string pointer"},
			},
			shouldError: true,
			errorMsg:    "invalid Go type",
		},
		{
			name: "empty Go type",
			annotations: []ResultAnnotation{
				{ColumnName: "amount", GoType: ""},
			},
			shouldError: true,
			errorMsg:    "invalid Go type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &Query{
				Name:              "TestQuery",
				ResultAnnotations: tt.annotations,
			}

			err := parser.validateResultAnnotations(query)

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

func TestIsValidGoType(t *testing.T) {
	tests := []struct {
		goType   string
		expected bool
	}{
		{"string", true},
		{"*string", true},
		{"int", true},
		{"*int", true},
		{"uuid.UUID", true},
		{"*uuid.UUID", true},
		{"time.Time", true},
		{"*time.Time", true},
		{"json.RawMessage", true},
		{"*json.RawMessage", true},
		{"", false},
		{"invalid type", false},
		{"too.many.parts", false},
		{"*", false},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			result := isValidGoType(tt.goType)
			if result != tt.expected {
				t.Errorf("isValidGoType(%q) = %v, expected %v", tt.goType, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
