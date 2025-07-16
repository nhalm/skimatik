package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDefaultFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
		wantErr  bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "string 'all'",
			input:    "all",
			expected: []string{"create", "get", "update", "delete", "list", "paginate"},
			wantErr:  false,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "array of strings",
			input:    []interface{}{"create", "get", "update"},
			expected: []string{"create", "get", "update"},
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "array with non-string",
			input:    []interface{}{"create", 123},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid type",
			input:    123,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDefaultFunctions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDefaultFunctions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !stringSlicesEqual(result, tt.expected) {
				t.Errorf("parseDefaultFunctions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetTableFunctions(t *testing.T) {
	tests := []struct {
		name             string
		tableName        string
		tableConfigs     map[string]TableConfig
		defaultFunctions []string
		expected         []string
		description      string
	}{
		{
			name:             "table not in config, no default_functions",
			tableName:        "users",
			tableConfigs:     map[string]TableConfig{},
			defaultFunctions: nil,
			expected:         []string{"create", "get", "update", "delete", "list", "paginate"},
			description:      "Should return all functions when table not configured and no defaults",
		},
		{
			name:             "table not in config, with default_functions",
			tableName:        "users",
			tableConfigs:     map[string]TableConfig{},
			defaultFunctions: []string{"create", "get"},
			expected:         []string{"create", "get"},
			description:      "Should return default_functions when table not configured",
		},
		{
			name:      "table in config with explicit functions",
			tableName: "users",
			tableConfigs: map[string]TableConfig{
				"users": {Functions: []string{"create", "update", "delete"}},
			},
			defaultFunctions: []string{"create", "get"},
			expected:         []string{"create", "update", "delete"},
			description:      "Should return table-specific functions when explicitly configured",
		},
		{
			name:      "table in config with empty functions array",
			tableName: "users",
			tableConfigs: map[string]TableConfig{
				"users": {Functions: []string{}},
			},
			defaultFunctions: []string{"create", "get"},
			expected:         []string{"create", "get"},
			description:      "Should return default_functions when table has empty functions array",
		},
		{
			name:      "table in config with empty functions array, no defaults",
			tableName: "users",
			tableConfigs: map[string]TableConfig{
				"users": {Functions: []string{}},
			},
			defaultFunctions: nil,
			expected:         []string{"create", "get", "update", "delete", "list", "paginate"},
			description:      "Should return all functions when table has empty functions array and no defaults",
		},
		{
			name:             "default_functions set to all",
			tableName:        "posts",
			tableConfigs:     map[string]TableConfig{},
			defaultFunctions: []string{"create", "get", "update", "delete", "list", "paginate"},
			expected:         []string{"create", "get", "update", "delete", "list", "paginate"},
			description:      "Should return all functions when default_functions is set to all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				TableConfigs:     tt.tableConfigs,
				DefaultFunctions: tt.defaultFunctions,
			}
			result := config.GetTableFunctions(tt.tableName)
			if !stringSlicesEqual(result, tt.expected) {
				t.Errorf("GetTableFunctions() = %v, want %v\nDescription: %s", result, tt.expected, tt.description)
			}
		})
	}
}

func TestLoadConfig_DefaultFunctions(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		expectedFunc []string
		wantErr      bool
		description  string
	}{
		{
			name: "default_functions as string 'all'",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
default_functions: "all"
tables:
  users:
`,
			expectedFunc: []string{"create", "get", "update", "delete", "list", "paginate"},
			wantErr:      false,
			description:  "Should parse 'all' string correctly",
		},
		{
			name: "default_functions as array",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
default_functions: ["create", "get", "update"]
tables:
  users:
`,
			expectedFunc: []string{"create", "get", "update"},
			wantErr:      false,
			description:  "Should parse array correctly",
		},
		{
			name: "no default_functions",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  users:
`,
			expectedFunc: nil,
			wantErr:      false,
			description:  "Should handle missing default_functions",
		},
		{
			name: "invalid default_functions string",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
default_functions: "invalid"
tables:
  users:
`,
			expectedFunc: nil,
			wantErr:      true,
			description:  "Should error on invalid string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			// Load config
			config, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v\nDescription: %s", err, tt.wantErr, tt.description)
				return
			}

			if !tt.wantErr {
				if !stringSlicesEqual(config.DefaultFunctions, tt.expectedFunc) {
					t.Errorf("LoadConfig() DefaultFunctions = %v, want %v\nDescription: %s",
						config.DefaultFunctions, tt.expectedFunc, tt.description)
				}
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	yamlContent := `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  users:
    functions: ["create", "get", "update"]
  posts:
    functions: ["create", "list"]
`

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Test that existing table configurations still work
	usersFunctions := config.GetTableFunctions("users")
	expectedUsers := []string{"create", "get", "update"}
	if !stringSlicesEqual(usersFunctions, expectedUsers) {
		t.Errorf("GetTableFunctions('users') = %v, want %v", usersFunctions, expectedUsers)
	}

	postsFunctions := config.GetTableFunctions("posts")
	expectedPosts := []string{"create", "list"}
	if !stringSlicesEqual(postsFunctions, expectedPosts) {
		t.Errorf("GetTableFunctions('posts') = %v, want %v", postsFunctions, expectedPosts)
	}

	// Test that unconfigured tables get default behavior
	commentsFunctions := config.GetTableFunctions("comments")
	expectedComments := []string{"create", "get", "update", "delete", "list", "paginate"}
	if !stringSlicesEqual(commentsFunctions, expectedComments) {
		t.Errorf("GetTableFunctions('comments') = %v, want %v", commentsFunctions, expectedComments)
	}
}

func TestNewConfigurationFormat(t *testing.T) {
	yamlContent := `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
default_functions: "all"
tables:
  users:
  posts:
  audit_logs:
    functions: ["create", "list"]
`

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Test that tables without functions get default_functions
	expectedAll := []string{"create", "get", "update", "delete", "list", "paginate"}

	usersFunctions := config.GetTableFunctions("users")
	if !stringSlicesEqual(usersFunctions, expectedAll) {
		t.Errorf("GetTableFunctions('users') = %v, want %v", usersFunctions, expectedAll)
	}

	postsFunctions := config.GetTableFunctions("posts")
	if !stringSlicesEqual(postsFunctions, expectedAll) {
		t.Errorf("GetTableFunctions('posts') = %v, want %v", postsFunctions, expectedAll)
	}

	// Test that explicit functions override default_functions
	auditLogsFunctions := config.GetTableFunctions("audit_logs")
	expectedAuditLogs := []string{"create", "list"}
	if !stringSlicesEqual(auditLogsFunctions, expectedAuditLogs) {
		t.Errorf("GetTableFunctions('audit_logs') = %v, want %v", auditLogsFunctions, expectedAuditLogs)
	}
}

// Helper function to compare string slices
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
