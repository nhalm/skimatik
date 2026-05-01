package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test data for code generation tests
// getTestTable and getTestConfig are now in test_helpers.go

func TestNewCodeGenerator(t *testing.T) {
	config := getTestConfig()
	cg := NewCodeGenerator(config, "test")

	if cg.config != config {
		t.Error("Config not set correctly")
	}
}

func TestCodeGenerator_prepareCRUDTemplateData(t *testing.T) {
	cg := NewCodeGenerator(getTestConfig(), "test")
	table := getTestTable()

	data := cg.prepareCRUDTemplateData(table)

	// Test essential template data
	tests := []struct {
		key      string
		expected any
	}{
		{"StructName", "Users"},
		{"RepositoryName", "UsersRepository"},
		{"TableName", "users"},
		{"IDColumn", "id"},
	}

	for _, tt := range tests {
		if data[tt.key] != tt.expected {
			t.Errorf("Expected %s '%v', got %v", tt.key, tt.expected, data[tt.key])
		}
	}

	// Check that select columns contain expected fields
	selectColumns := data["SelectColumns"].(string)
	expectedColumns := []string{"id", "name", "email", "is_active", "created_at", "metadata"}
	for _, col := range expectedColumns {
		if !strings.Contains(selectColumns, col) {
			t.Errorf("SelectColumns missing column: %s", col)
		}
	}

	// Check create fields (should exclude ID and columns with defaults)
	createFields := data["CreateFields"].([]map[string]string)
	if len(createFields) != 3 { // name, email, metadata
		t.Errorf("Expected 3 create fields, got %d", len(createFields))
	}

	// Check update fields (should include all non-ID columns)
	updateFields := data["UpdateFields"].([]map[string]string)
	if len(updateFields) != 5 { // name, email, is_active, created_at, metadata
		t.Errorf("Expected 5 update fields, got %d", len(updateFields))
	}
}

func TestCodeGenerator_combineImports(t *testing.T) {
	cg := NewCodeGenerator(getTestConfig(), "test")

	list1 := []string{"context", "fmt"}
	list2 := []string{"fmt", "time", "context"}
	list3 := []string{"github.com/google/uuid"}

	combined := cg.combineImports(list1, list2, list3)

	// Check that duplicates are removed and all imports are present
	expected := []string{"context", "fmt", "time", "github.com/google/uuid"}
	if len(combined) != len(expected) {
		t.Errorf("Expected %d imports, got %d", len(expected), len(combined))
	}

	// Check that all expected imports are present
	for _, exp := range expected {
		found := false
		for _, imp := range combined {
			if imp == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected import: %s", exp)
		}
	}
}

func TestCodeGenerator_GenerateTableRepository_Integration(t *testing.T) {
	config := getTestConfigWithTempDir(t)

	cg := NewCodeGenerator(config, "test")
	table := getTestTable()

	// Generate the repository
	err := cg.GenerateTableRepository(table)
	if err != nil {
		t.Fatalf("GenerateTableRepository failed: %v", err)
	}

	// Check that file was created and contains basic structure
	expectedFilename := filepath.Join(config.OutputDir, "users_generated.go")
	if _, err := os.Stat(expectedFilename); os.IsNotExist(err) {
		t.Fatal("Generated file does not exist")
	}

	content, err := os.ReadFile(expectedFilename)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package repositories") {
		t.Error("Generated file missing package declaration")
	}

	if len(contentStr) < 100 {
		t.Error("Generated file seems too short")
	}
}

func TestContainsOrderBy(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "simple ORDER BY",
			sql:      "SELECT id, name FROM users ORDER BY name",
			expected: true,
		},
		{
			name:     "ORDER BY at end with DESC",
			sql:      "SELECT id, name FROM users ORDER BY created_at DESC",
			expected: true,
		},
		{
			name:     "ORDER BY with multiple columns",
			sql:      "SELECT id, name FROM users ORDER BY name ASC, created_at DESC",
			expected: true,
		},
		{
			name:     "uppercase ORDER BY",
			sql:      "SELECT id, name FROM users ORDER BY name",
			expected: true,
		},
		{
			name:     "mixed case ORDER BY",
			sql:      "SELECT id, name FROM users OrDeR bY name",
			expected: true,
		},
		{
			name:     "ORDER BY in CTE",
			sql:      "WITH sorted AS (SELECT id FROM users ORDER BY name) SELECT * FROM sorted",
			expected: true,
		},
		{
			name:     "ORDER BY in subquery",
			sql:      "SELECT * FROM (SELECT id FROM users ORDER BY name) AS sub",
			expected: true,
		},
		{
			name:     "no ORDER BY",
			sql:      "SELECT id, name FROM users WHERE is_active = true",
			expected: false,
		},
		{
			name:     "ORDER BY in string literal should be ignored",
			sql:      "SELECT id, name, 'ORDER BY' as label FROM users",
			expected: false,
		},
		{
			name:     "ORDER BY in single-quoted string",
			sql:      "SELECT id, comment FROM posts WHERE comment = 'sorted ORDER BY date'",
			expected: false,
		},
		{
			name:     "ORDER BY with escaped quotes in string",
			sql:      "SELECT id, text FROM messages WHERE text = 'Let''s ORDER BY name'",
			expected: false,
		},
		{
			name:     "ORDER BY after string literal",
			sql:      "SELECT id, 'test' as label FROM users ORDER BY name",
			expected: true,
		},
		{
			name:     "complex query without ORDER BY",
			sql:      "SELECT u.id, u.name FROM users u JOIN posts p ON u.id = p.user_id WHERE p.published = true",
			expected: false,
		},
		{
			name:     "empty SQL",
			sql:      "",
			expected: false,
		},
		{
			name:     "ORDER BY with window function OVER",
			sql:      "SELECT id, ROW_NUMBER() OVER (ORDER BY created_at) as rn FROM users",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsOrderBy(tt.sql)
			if result != tt.expected {
				t.Errorf("containsOrderBy(%q) = %v, expected %v", tt.sql, result, tt.expected)
			}
		})
	}
}

func TestRemoveStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no string literals",
			input:    "SELECT id FROM users",
			expected: "SELECT id FROM users",
		},
		{
			name:     "simple string literal",
			input:    "SELECT 'hello' as greeting",
			expected: "SELECT         as greeting",
		},
		{
			name:     "multiple string literals",
			input:    "SELECT 'foo', 'bar' FROM users",
			expected: "SELECT      ,       FROM users",
		},
		{
			name:     "string with escaped quote",
			input:    "SELECT 'it''s' as text",
			expected: "SELECT         as text",
		},
		{
			name:     "string with ORDER BY text",
			input:    "SELECT 'ORDER BY name' as label",
			expected: "SELECT                 as label",
		},
		{
			name:     "mixed content",
			input:    "SELECT id, 'test' as label, name FROM users WHERE name = 'john'",
			expected: "SELECT id,        as label, name FROM users WHERE name =       ",
		},
		{
			name:     "consecutive string literals",
			input:    "SELECT 'a''b''c' as text",
			expected: "SELECT           as text",
		},
		{
			name:     "empty string literal",
			input:    "SELECT '' as empty",
			expected: "SELECT    as empty",
		},
		{
			name:     "string at start",
			input:    "'hello' FROM users",
			expected: "        FROM users",
		},
		{
			name:     "string at end",
			input:    "SELECT id WHERE name = 'test'",
			expected: "SELECT id WHERE name =       ",
		},
		{
			name:     "complex nested escapes",
			input:    "WHERE text = 'it''s a ''test'''",
			expected: "WHERE text =                   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeStringLiterals(tt.input)
			if result != tt.expected {
				t.Errorf("removeStringLiterals(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
