package generator

import (
	"os"
	"strings"
	"testing"
)

func TestInlinePagination_TemplateGeneration(t *testing.T) {
	config := getTestConfigWithTempDir(t)
	config.TableConfigs = map[string]TableConfig{
		"users": {
			Functions: []string{"create", "get", "update", "delete", "paginate"},
		},
	}

	cg := NewCodeGenerator(config)
	table := getTestTable()

	// Test shared pagination types generation
	err := cg.GenerateSharedPaginationTypes()
	if err != nil {
		t.Fatalf("GenerateSharedPaginationTypes failed: %v", err)
	}

	// Read the generated pagination file
	paginationFile := cg.config.GetOutputPath("pagination.go")
	paginationContent, err := os.ReadFile(paginationFile)
	if err != nil {
		t.Fatalf("Failed to read pagination file: %v", err)
	}
	paginationTypes := string(paginationContent)

	// Check that all required components are present in shared pagination file
	expectedComponents := []string{
		"type PaginationParams struct",
		"type PaginationResult[T any] struct",
		"func encodeCursor(id uuid.UUID) string",
		"func decodeCursor(cursor string) (uuid.UUID, error)",
		"func validatePaginationParams(params PaginationParams) error",
		"Items []T `json:\"items\"`",
		"HasMore bool `json:\"has_more\"`",
		"NextCursor string `json:\"next_cursor,omitempty\"`",
		"base64.URLEncoding.EncodeToString(id[:])",
		"base64.URLEncoding.DecodeString(cursor)",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(paginationTypes, component) {
			t.Errorf("Pagination types missing component: %s", component)
		}
	}

	// Test repository generation with the new CRUD operations system
	repositoryCode, err := cg.generateTableCode(table)
	if err != nil {
		t.Fatalf("generateTableCode failed: %v", err)
	}

	expectedListComponents := []string{
		"func (r *UsersRepository) ListPaginated(ctx context.Context, params PaginationParams) (*PaginationResult[Users], error)",
		"validatePaginationParams(params)",
		"decodeCursor(params.Cursor)",
		"encodeCursor(lastItem.GetID())",
		"WHERE ($1::uuid IS NULL OR id > $1)",
		"ORDER BY id ASC",
		"LIMIT $2",
		"hasMore := len(items) > limit",
		"items = items[:limit]",
	}

	for _, component := range expectedListComponents {
		if !strings.Contains(repositoryCode, component) {
			t.Errorf("Repository code missing component: %s", component)
		}
	}
}

func TestInlinePagination_CursorLogic(t *testing.T) {
	// Create temporary directory for test output
	tempDir := t.TempDir()

	config := &Config{
		OutputDir:   tempDir,
		PackageName: "repositories",
		Verbose:     false,
	}

	cg := NewCodeGenerator(config)

	// Generate shared pagination types
	err := cg.GenerateSharedPaginationTypes()
	if err != nil {
		t.Fatalf("GenerateSharedPaginationTypes failed: %v", err)
	}

	// Read the generated pagination file
	paginationFile := cg.config.GetOutputPath("pagination.go")
	paginationContent, err := os.ReadFile(paginationFile)
	if err != nil {
		t.Fatalf("Failed to read pagination file: %v", err)
	}
	paginationTypes := string(paginationContent)

	// Test cursor encoding logic
	if !strings.Contains(paginationTypes, "base64.URLEncoding.EncodeToString(id[:])") {
		t.Error("Missing cursor encoding logic")
	}

	// Test cursor decoding logic
	expectedDecodingComponents := []string{
		"base64.URLEncoding.DecodeString(cursor)",
		"if len(cursorBytes) != 16",
		"copy(id[:], cursorBytes)",
		"return uuid.Nil, fmt.Errorf(\"empty cursor\")",
		"return uuid.Nil, fmt.Errorf(\"invalid cursor format: %w\", err)",
		"return uuid.Nil, fmt.Errorf(\"invalid cursor length: expected 16 bytes, got %d\", len(cursorBytes))",
	}

	for _, component := range expectedDecodingComponents {
		if !strings.Contains(paginationTypes, component) {
			t.Errorf("Missing cursor decoding component: %s", component)
		}
	}

	// Test parameter validation logic
	expectedValidationComponents := []string{
		"if params.Limit < 0",
		"if params.Limit > 100",
		"if params.Cursor != \"\"",
		"decodeCursor(params.Cursor)",
		"return fmt.Errorf(\"limit cannot be negative\")",
		"return fmt.Errorf(\"limit cannot exceed 100\")",
		"return fmt.Errorf(\"invalid cursor: %w\", err)",
	}

	for _, component := range expectedValidationComponents {
		if !strings.Contains(paginationTypes, component) {
			t.Errorf("Missing parameter validation component: %s", component)
		}
	}
}

func TestInlinePagination_GetIDMethod(t *testing.T) {
	cg := NewCodeGenerator(getTestConfig())
	table := getTestTable()

	// Generate struct code
	structCode, err := cg.generateStruct(table)
	if err != nil {
		t.Fatalf("generateStruct failed: %v", err)
	}

	// Test that GetID method uses value receiver, not pointer receiver
	expectedGetIDSignature := "func (u Users) GetID() uuid.UUID"
	if !strings.Contains(structCode, expectedGetIDSignature) {
		t.Errorf("GetID method should use value receiver, not pointer receiver")
	}

	// Test that GetID method returns the correct field
	if !strings.Contains(structCode, "return u.Id") {
		t.Error("GetID method should return u.Id")
	}

	// Ensure we don't have the old pointer receiver version
	oldPointerSignature := "func (u *Users) GetID() uuid.UUID"
	if strings.Contains(structCode, oldPointerSignature) {
		t.Error("GetID method should not use pointer receiver")
	}
}
