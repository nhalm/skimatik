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
	cg := NewCodeGenerator(config)

	if cg.config != config {
		t.Error("Config not set correctly")
	}
}

func TestCodeGenerator_prepareCRUDTemplateData(t *testing.T) {
	cg := NewCodeGenerator(getTestConfig())
	table := getTestTable()

	data, err := cg.prepareCRUDTemplateData(table)
	if err != nil {
		t.Fatalf("prepareCRUDTemplateData failed: %v", err)
	}

	// Test essential template data
	tests := []struct {
		key      string
		expected interface{}
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
	cg := NewCodeGenerator(getTestConfig())

	list1 := []string{"context", "fmt"}
	list2 := []string{"fmt", "github.com/jackc/pgx/v5/pgtype", "context"}
	list3 := []string{"github.com/google/uuid"}

	combined := cg.combineImports(list1, list2, list3)

	// Check that duplicates are removed and all imports are present
	expected := []string{"context", "fmt", "github.com/jackc/pgx/v5/pgtype", "github.com/google/uuid"}
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

	cg := NewCodeGenerator(config)
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
