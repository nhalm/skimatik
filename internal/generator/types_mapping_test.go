package generator

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Helper function to test all combinations of nullable/array for a given type mapping
func testTypeMapping(t *testing.T, tm *TypeMapper, pgType, baseType, nullableType string) {
	t.Helper()

	tests := []struct {
		name       string
		isNullable bool
		isArray    bool
		want       string
	}{
		{"base", false, false, baseType},
		{"nullable", true, false, nullableType},
		{"array", false, true, "[]" + baseType},
		{"nullable_array", true, true, "[]" + nullableType},
	}

	for _, tt := range tests {
		t.Run(pgType+"_"+tt.name, func(t *testing.T) {
			got, err := tm.MapType(pgType, tt.isNullable, tt.isArray)
			if err != nil {
				t.Errorf("MapType() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("MapType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeMapper_MapType(t *testing.T) {
	tm := NewTypeMapper(nil)

	// Test core type mappings with all combinations
	testTypeMapping(t, tm, "uuid", "uuid.UUID", "pgtype.UUID")
	testTypeMapping(t, tm, "text", "string", "pgtype.Text")
	testTypeMapping(t, tm, "varchar", "string", "pgtype.Text")
	testTypeMapping(t, tm, "integer", "int32", "pgtype.Int4")
	testTypeMapping(t, tm, "bigint", "int64", "pgtype.Int8")
	testTypeMapping(t, tm, "boolean", "bool", "pgtype.Bool")
	testTypeMapping(t, tm, "timestamptz", "time.Time", "pgtype.Timestamptz")
	testTypeMapping(t, tm, "jsonb", "json.RawMessage", "*json.RawMessage")

	// Test type aliases
	aliasTests := []struct {
		pgType   string
		expected string
	}{
		{"character varying", "string"},
		{"int", "int32"},
		{"int4", "int32"},
		{"int8", "int64"},
		{"smallint", "int16"},
		{"int2", "int16"},
		{"real", "float32"},
		{"float4", "float32"},
		{"double precision", "float64"},
		{"float8", "float64"},
		{"bool", "bool"},
		{"timestamp", "time.Time"},
		{"json", "json.RawMessage"},
		{"bytea", "[]byte"},
	}

	for _, tt := range aliasTests {
		t.Run(tt.pgType, func(t *testing.T) {
			got, err := tm.MapType(tt.pgType, false, false)
			if err != nil {
				t.Errorf("MapType() error = %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("MapType() = %v, want %v", got, tt.expected)
			}
		})
	}

	// Test unsupported types
	unsupportedTypes := []string{"unsupported_type", "custom_enum", "pg_lsn"}
	for _, pgType := range unsupportedTypes {
		t.Run(pgType+"_unsupported", func(t *testing.T) {
			_, err := tm.MapType(pgType, false, false)
			if err == nil {
				t.Errorf("MapType() should return error for unsupported type %s", pgType)
			}
		})
	}
}

// TestTypeMapper_MapType_NullableArrays - test nullable array type combinations
func TestTypeMapper_MapType_NullableArrays(t *testing.T) {
	typeMapper := NewTypeMapper(nil)

	testCases := []struct {
		name         string
		pgType       string
		isNullable   bool
		isArray      bool
		expectedType string
		expectError  bool
	}{
		{
			name:         "nullable_text_array",
			pgType:       "text",
			isNullable:   true,
			isArray:      true,
			expectedType: "[]pgtype.Text",
			expectError:  false,
		},
		{
			name:         "nullable_uuid_array",
			pgType:       "uuid",
			isNullable:   true,
			isArray:      true,
			expectedType: "[]pgtype.UUID",
			expectError:  false,
		},
		{
			name:         "non_nullable_text_array",
			pgType:       "text",
			isNullable:   false,
			isArray:      true,
			expectedType: "[]string",
			expectError:  false,
		},
		{
			name:         "nullable_non_array_text",
			pgType:       "text",
			isNullable:   true,
			isArray:      false,
			expectedType: "pgtype.Text",
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goType, err := typeMapper.MapType(tc.pgType, tc.isNullable, tc.isArray)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.name, err)
				}
				if goType != tc.expectedType {
					t.Errorf("MapType(%s, %v, %v) = %s, want %s",
						tc.pgType, tc.isNullable, tc.isArray, goType, tc.expectedType)
				}
			}
		})
	}
}

func TestTypeMapper_MapType_WithCustomMappings(t *testing.T) {
	customMappings := map[string]string{
		"custom_type": "MyCustomType",
		"uuid":        "MyUUID", // Override built-in mapping
	}
	tm := NewTypeMapper(customMappings)

	tests := []struct {
		name       string
		pgType     string
		isNullable bool
		isArray    bool
		want       string
	}{
		{"custom_type_mapping", "custom_type", false, false, "MyCustomType"},
		{"override_built-in_mapping", "uuid", false, false, "MyUUID"},
		{"nullable_custom_type", "custom_type", true, false, "*MyCustomType"},
		{"array_custom_type", "custom_type", false, true, "[]MyCustomType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tm.MapType(tt.pgType, tt.isNullable, tt.isArray)
			if err != nil {
				t.Errorf("MapType() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("MapType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTypeMapper_MapType_CustomMappingsEdgeCases - test custom type mapping edge cases
func TestTypeMapper_MapType_CustomMappingsEdgeCases(t *testing.T) {
	customMappings := map[string]string{
		"custom_type": "CustomStruct",
		"enum_type":   "EnumType",
	}

	typeMapper := NewTypeMapper(customMappings)

	testCases := []struct {
		name         string
		pgType       string
		isNullable   bool
		isArray      bool
		expectedType string
		expectError  bool
	}{
		{
			name:         "custom_type",
			pgType:       "custom_type",
			isNullable:   false,
			isArray:      false,
			expectedType: "CustomStruct",
			expectError:  false,
		},
		{
			name:         "nullable_custom_type",
			pgType:       "custom_type",
			isNullable:   true,
			isArray:      false,
			expectedType: "*CustomStruct",
			expectError:  false,
		},
		{
			name:         "custom_type_array",
			pgType:       "custom_type",
			isNullable:   false,
			isArray:      true,
			expectedType: "[]CustomStruct",
			expectError:  false,
		},
		{
			name:         "nullable_custom_type_array",
			pgType:       "custom_type",
			isNullable:   true,
			isArray:      true,
			expectedType: "[]*CustomStruct",
			expectError:  false,
		},
		{
			name:         "override_builtin_type",
			pgType:       "text",
			isNullable:   false,
			isArray:      false,
			expectedType: "string", // Should still use built-in mapping
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goType, err := typeMapper.MapType(tc.pgType, tc.isNullable, tc.isArray)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.name, err)
				}
				if goType != tc.expectedType {
					t.Errorf("MapType(%s, %v, %v) = %s, want %s",
						tc.pgType, tc.isNullable, tc.isArray, goType, tc.expectedType)
				}
			}
		})
	}
}

func TestTypeMapper_GetRequiredImports(t *testing.T) {
	tm := NewTypeMapper(nil)

	tests := []struct {
		name     string
		columns  []Column
		expected []string
	}{
		{
			name: "basic_types_with_imports",
			columns: []Column{
				{Type: "uuid", IsNullable: false, IsArray: false},
				{Type: "timestamp", IsNullable: false, IsArray: false},
				{Type: "uuid", IsNullable: true, IsArray: false},
				{Type: "json", IsNullable: false, IsArray: false},
			},
			expected: []string{
				"encoding/json",
				"github.com/google/uuid",
				"github.com/jackc/pgx/v5/pgtype",
				"time",
			},
		},
		{
			name: "only_basic_types",
			columns: []Column{
				{Type: "text", IsNullable: false, IsArray: false},
				{Type: "integer", IsNullable: false, IsArray: false},
				{Type: "boolean", IsNullable: false, IsArray: false},
			},
			expected: []string{},
		},
		{
			name: "array_types",
			columns: []Column{
				{Type: "text", IsNullable: false, IsArray: true},
				{Type: "uuid", IsNullable: false, IsArray: true},
				{Type: "uuid", IsNullable: true, IsArray: true},
			},
			expected: []string{
				"github.com/google/uuid",
				"github.com/jackc/pgx/v5/pgtype",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.GetRequiredImports(tt.columns)
			sort.Strings(got)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetRequiredImports() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestTypeMapper_GetRequiredImports_EdgeCases - test import generation edge cases
func TestTypeMapper_GetRequiredImports_EdgeCases(t *testing.T) {
	typeMapper := NewTypeMapper(nil)

	testCases := []struct {
		name            string
		columns         []Column
		expectedImports []string
	}{
		{
			name:            "no_columns",
			columns:         []Column{},
			expectedImports: []string{},
		},
		{
			name: "only_basic_types",
			columns: []Column{
				{Type: "text", IsNullable: false, IsArray: false},
				{Type: "integer", IsNullable: false, IsArray: false},
				{Type: "boolean", IsNullable: false, IsArray: false},
			},
			expectedImports: []string{},
		},
		{
			name: "mixed_imports",
			columns: []Column{
				{Type: "uuid", IsNullable: false, IsArray: false},
				{Type: "text", IsNullable: true, IsArray: false},
				{Type: "timestamp", IsNullable: false, IsArray: false},
				{Type: "json", IsNullable: false, IsArray: false},
			},
			expectedImports: []string{
				"encoding/json",
				"github.com/google/uuid",
				"github.com/jackc/pgx/v5/pgtype",
				"time",
			},
		},
		{
			name: "duplicate_imports",
			columns: []Column{
				{Type: "uuid", IsNullable: false, IsArray: false},
				{Type: "uuid", IsNullable: true, IsArray: false},
				{Type: "uuid", IsNullable: false, IsArray: true},
			},
			expectedImports: []string{
				"github.com/google/uuid",
				"github.com/jackc/pgx/v5/pgtype",
			},
		},
		{
			name:            "nil_columns",
			columns:         nil,
			expectedImports: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imports := typeMapper.GetRequiredImports(tc.columns)

			if imports == nil {
				t.Error("GetRequiredImports should return empty slice, not nil")
			}

			if len(imports) != len(tc.expectedImports) {
				t.Errorf("Expected %d imports, got %d: %v",
					len(tc.expectedImports), len(imports), imports)
			}

			// Convert to maps for easier comparison
			importMap := make(map[string]bool)
			for _, imp := range imports {
				importMap[imp] = true
			}

			expectedMap := make(map[string]bool)
			for _, imp := range tc.expectedImports {
				expectedMap[imp] = true
			}

			for expectedImport := range expectedMap {
				if !importMap[expectedImport] {
					t.Errorf("Missing expected import: %s", expectedImport)
				}
			}

			for actualImport := range importMap {
				if !expectedMap[actualImport] {
					t.Errorf("Unexpected import: %s", actualImport)
				}
			}
		})
	}
}

func TestTypeMapper_MapTableColumns(t *testing.T) {
	tm := NewTypeMapper(nil)

	table := Table{
		Name:   "test_table",
		Schema: "public",
		Columns: []Column{
			{Name: "id", Type: "uuid", IsNullable: false},
			{Name: "name", Type: "text", IsNullable: false},
			{Name: "email", Type: "text", IsNullable: true},
		},
	}

	err := tm.MapTableColumns(&table)
	if err != nil {
		t.Fatalf("MapTableColumns() error = %v", err)
	}

	expected := []string{"uuid.UUID", "string", "pgtype.Text"}
	for i, col := range table.Columns {
		if col.GoType != expected[i] {
			t.Errorf("Column %d GoType = %v, want %v", i, col.GoType, expected[i])
		}
	}
}

func TestTypeMapper_MapTableColumns_WithError(t *testing.T) {
	tm := NewTypeMapper(nil)

	table := Table{
		Name:   "test_table",
		Schema: "public",
		Columns: []Column{
			{Name: "id", Type: "unsupported_type", IsNullable: false},
		},
	}

	err := tm.MapTableColumns(&table)
	if err == nil {
		t.Error("MapTableColumns() should return error for unsupported type")
	}
}

func TestTypeMapper_ValidateUUIDPrimaryKey(t *testing.T) {
	tm := NewTypeMapper(nil)

	tests := []struct {
		name           string
		column         Column
		wantErr        bool
		errorSubstring string
	}{
		{
			name:    "valid_UUID_primary_key",
			column:  Column{Name: "id", Type: "uuid", IsNullable: false, IsArray: false},
			wantErr: false,
		},
		{
			name:    "UUID_uppercase",
			column:  Column{Name: "id", Type: "UUID", IsNullable: false, IsArray: false},
			wantErr: false,
		},
		{
			name:           "non-UUID_type",
			column:         Column{Name: "id", Type: "integer", IsNullable: false, IsArray: false},
			wantErr:        true,
			errorSubstring: "must be UUID type",
		},
		{
			name:           "nullable_UUID",
			column:         Column{Name: "id", Type: "uuid", IsNullable: true, IsArray: false},
			wantErr:        true,
			errorSubstring: "cannot be nullable",
		},
		{
			name:           "UUID_array",
			column:         Column{Name: "id", Type: "uuid", IsNullable: false, IsArray: true},
			wantErr:        true,
			errorSubstring: "cannot be an array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tm.ValidateUUIDPrimaryKey(&tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUUIDPrimaryKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check error message content for error cases
			if tt.wantErr && err != nil && tt.errorSubstring != "" {
				if !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("Error message should contain '%s', got: %s", tt.errorSubstring, err.Error())
				}
			}
		})
	}
}

func TestTypeMapper_makeNullable(t *testing.T) {
	tm := NewTypeMapper(nil)

	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{"string_type", "string", "pgtype.Text"},
		{"int32_type", "int32", "pgtype.Int4"},
		{"int64_type", "int64", "pgtype.Int8"},
		{"bool_type", "bool", "pgtype.Bool"},
		{"time.Time_type", "time.Time", "pgtype.Timestamptz"},
		{"uuid.UUID_type", "uuid.UUID", "pgtype.UUID"},
		{"json.RawMessage_type", "json.RawMessage", "*json.RawMessage"},
		{"[]byte_type", "[]byte", "*[]byte"},
		{"array_of_strings", "[]string", "[]pgtype.Text"},
		{"custom_type", "CustomType", "*CustomType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.makeNullable(tt.goType)
			if got != tt.expected {
				t.Errorf("makeNullable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewTypeMapper(t *testing.T) {
	tests := []struct {
		name           string
		customMappings map[string]string
		wantNil        bool
	}{
		{"nil_custom_mappings", nil, false},
		{"empty_custom_mappings", map[string]string{}, false},
		{"with_custom_mappings", map[string]string{"custom": "Custom"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTypeMapper(tt.customMappings)
			if (got == nil) != tt.wantNil {
				t.Errorf("NewTypeMapper() = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}
