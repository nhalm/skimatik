package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDefaultFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    any
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
			input:    []any{"create", "get", "update"},
			expected: []string{"create", "get", "update"},
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    []any{},
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "array with non-string",
			input:    []any{"create", 123},
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

			err := os.WriteFile(configPath, []byte(tt.yamlContent), 0o644)
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

func TestConfigurationFormats(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		checks      []struct {
			table    string
			expected []string
		}
	}{
		{
			name: "backward_compatibility",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  users:
    functions: ["create", "get", "update"]
  posts:
    functions: ["create", "list"]
`,
			checks: []struct {
				table    string
				expected []string
			}{
				{"users", []string{"create", "get", "update"}},
				{"posts", []string{"create", "list"}},
				{"comments", []string{"create", "get", "update", "delete", "list", "paginate"}},
			},
		},
		{
			name: "new_configuration_format",
			yamlContent: `
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
`,
			checks: []struct {
				table    string
				expected []string
			}{
				{"users", []string{"create", "get", "update", "delete", "list", "paginate"}},
				{"posts", []string{"create", "get", "update", "delete", "list", "paginate"}},
				{"audit_logs", []string{"create", "list"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.yamlContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			config, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() failed: %v", err)
			}

			for _, check := range tt.checks {
				result := config.GetTableFunctions(check.table)
				if !stringSlicesEqual(result, check.expected) {
					t.Errorf("GetTableFunctions(%q) = %v, want %v", check.table, result, check.expected)
				}
			}
		})
	}
}

func TestLoadConfig_TableAudit(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		tableName   string
		expected    bool
		description string
	}{
		{
			name: "audit true",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  posts:
    audit: true
`,
			tableName:   "posts",
			expected:    true,
			description: "audit: true should parse as true",
		},
		{
			name: "audit omitted defaults to false",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  posts:
`,
			tableName:   "posts",
			expected:    false,
			description: "audit omitted should default to false",
		},
		{
			name: "audit false explicit",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  posts:
    audit: false
`,
			tableName:   "posts",
			expected:    false,
			description: "audit: false should parse as false",
		},
		{
			name: "audit alongside functions",
			yamlContent: `
database:
  dsn: "postgres://test"
output:
  directory: "./test"
tables:
  posts:
    audit: true
    functions: ["create", "get", "update"]
`,
			tableName:   "posts",
			expected:    true,
			description: "audit should coexist with functions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.yamlContent), 0o644); err != nil {
				t.Fatalf("failed to write test config file: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatalf("LoadConfig() failed: %v", err)
			}

			tc, ok := cfg.TableConfigs[tt.tableName]
			if !ok {
				t.Fatalf("expected TableConfigs[%q] to exist", tt.tableName)
			}
			if tc.Audit != tt.expected {
				t.Errorf("TableConfigs[%q].Audit = %v, want %v\nDescription: %s",
					tt.tableName, tc.Audit, tt.expected, tt.description)
			}

			if got := cfg.IsTableAudited(tt.tableName); got != tt.expected {
				t.Errorf("IsTableAudited(%q) = %v, want %v",
					tt.tableName, got, tt.expected)
			}
		})
	}
}

func TestIsTableAudited_UnknownTable(t *testing.T) {
	cfg := &Config{TableConfigs: map[string]TableConfig{}}
	if cfg.IsTableAudited("unknown") {
		t.Errorf("IsTableAudited for unknown table = true, want false")
	}
}

func TestTableAuditPropagation(t *testing.T) {
	// Drives Generator.resolveTables — the same code path generateTables uses —
	// to confirm that introspected Table values pick up their Audit flag from
	// TableConfigs and that non-included tables are dropped.
	cfg := &Config{
		Include: []string{"posts", "comments"},
		TableConfigs: map[string]TableConfig{
			"posts":    {Audit: true},
			"comments": {Audit: false},
		},
	}
	g := &Generator{config: cfg}

	introspected := []Table{
		{Name: "posts", Schema: "public"},
		{Name: "comments", Schema: "public"},
		{Name: "tags", Schema: "public"}, // not configured at all
	}

	resolvedSlice := g.resolveTables(introspected)
	resolved := make(map[string]Table, len(resolvedSlice))
	for _, tbl := range resolvedSlice {
		resolved[tbl.Name] = tbl
	}

	if got := resolved["posts"].Audit; got != true {
		t.Errorf("posts.Audit = %v, want true", got)
	}
	if got, ok := resolved["comments"]; !ok || got.Audit != false {
		t.Errorf("comments.Audit = %v (present=%v), want false (present=true)", got.Audit, ok)
	}
	if _, ok := resolved["tags"]; ok {
		t.Errorf("tags should be filtered out by Include")
	}

	// resolveTables must not mutate the caller's slice — the introspector's
	// returned values should keep Audit at its zero default.
	for _, tbl := range introspected {
		if tbl.Audit {
			t.Errorf("resolveTables mutated input slice: %s.Audit=true", tbl.Name)
		}
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
