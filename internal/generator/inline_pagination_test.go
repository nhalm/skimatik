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

	cg := NewCodeGenerator(config, "test")
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
		"func encodeJSONCursor(column string, value interface{}) (string, error)",
		"func decodeJSONCursor(cursor string) (column string, value interface{}, err error)",
		"func validatePaginationParams(params PaginationParams) error",
		"Items []T `json:\"items\"`",
		"HasMore bool `json:\"has_more\"`",
		"HasPrevious bool `json:\"has_previous\"`",
		"NextCursor string `json:\"next_cursor,omitempty\"`",
		"BeforeCursor string `json:\"before_cursor,omitempty\"`",
		"base64.URLEncoding.EncodeToString(jsonBytes)",
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
		"decodeJSONCursor",
		"encodeJSONCursor",
		"ORDER BY",
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

	cg := NewCodeGenerator(config, "test")

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

	// Test JSON cursor encoding logic
	if !strings.Contains(paginationTypes, "json.Marshal(data)") {
		t.Error("Missing JSON cursor encoding logic")
	}
	if !strings.Contains(paginationTypes, "base64.URLEncoding.EncodeToString(jsonBytes)") {
		t.Error("Missing base64 cursor encoding logic")
	}

	// Test cursor decoding logic
	expectedDecodingComponents := []string{
		"base64.URLEncoding.DecodeString(cursor)",
		"json.Unmarshal(cursorBytes, &data)",
		"if data.Column == \"\"",
		"return \"\", nil, fmt.Errorf(\"empty cursor\")",
		"return \"\", nil, fmt.Errorf(\"invalid cursor format: %w\", err)",
		"return \"\", nil, fmt.Errorf(\"cursor missing column name\")",
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
		"if params.NextCursor != \"\" && params.BeforeCursor != \"\"",
		"if params.NextCursor != \"\"",
		"if params.BeforeCursor != \"\"",
		"return fmt.Errorf(\"limit cannot be negative\")",
		"return fmt.Errorf(\"limit cannot exceed 100\")",
		"return fmt.Errorf(\"cannot use both next_cursor and before_cursor\")",
	}

	for _, component := range expectedValidationComponents {
		if !strings.Contains(paginationTypes, component) {
			t.Errorf("Missing parameter validation component: %s", component)
		}
	}

	// Test bidirectional logic presence
	if !strings.Contains(paginationTypes, "NextCursor string") {
		t.Error("Missing NextCursor field")
	}
	if !strings.Contains(paginationTypes, "BeforeCursor string") {
		t.Error("Missing BeforeCursor field")
	}
	if !strings.Contains(paginationTypes, "OrderBy string") {
		t.Error("Missing OrderBy field")
	}
	if !strings.Contains(paginationTypes, "HasPrevious bool") {
		t.Error("Missing HasPrevious field")
	}
}

func TestInlinePagination_GetIDMethod(t *testing.T) {
	cg := NewCodeGenerator(getTestConfig(), "test")
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
