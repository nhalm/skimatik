package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the code generator
type Config struct {
	// Database connection
	DSN    string `yaml:"dsn"`
	Schema string `yaml:"schema"`

	// Output configuration
	OutputDir   string `yaml:"output_dir"`
	PackageName string `yaml:"package_name"`

	// Generation modes
	Tables     bool   `yaml:"tables"`
	QueriesDir string `yaml:"queries_dir"`

	// Table filtering
	Include []string `yaml:"include"`

	// Table configurations (functions to generate per table)
	TableConfigs map[string]TableConfig `yaml:"table_configs"`

	// Default functions to generate when not specified per table
	DefaultFunctions []string `yaml:"default_functions"`

	// Options
	Verbose bool `yaml:"verbose"`

	// Type mappings (future extension)
	TypeMappings map[string]string `yaml:"type_mappings"`
}

// DatabaseConfig represents database-specific configuration
type DatabaseConfig struct {
	DSN    string `yaml:"dsn"`
	Schema string `yaml:"schema"`
}

// OutputConfig represents output-specific configuration
type OutputConfig struct {
	Directory string `yaml:"directory"`
	Package   string `yaml:"package"`
}

// TableConfig represents configuration for a specific table
type TableConfig struct {
	Functions []string `yaml:"functions"`
}

// TablesConfig represents table generation configuration
type TablesConfig map[string]TableConfig

// QueriesConfig represents query generation configuration
type QueriesConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Directory string   `yaml:"directory"`
	Files     []string `yaml:"files"`
}

// TypesConfig represents type mapping configuration
type TypesConfig struct {
	Mappings map[string]string `yaml:"mappings"`
}

// FileConfig represents the structure of a configuration file
type FileConfig struct {
	Database         DatabaseConfig `yaml:"database"`
	Output           OutputConfig   `yaml:"output"`
	Tables           TablesConfig   `yaml:"tables"`
	Queries          QueriesConfig  `yaml:"queries"`
	Types            TypesConfig    `yaml:"types"`
	DefaultFunctions interface{}    `yaml:"default_functions"` // "all" or []string
	Verbose          bool           `yaml:"verbose"`
}

// parseDefaultFunctions parses the default_functions field from YAML
// It can be either "all" (string) or an array of function names
func parseDefaultFunctions(value interface{}) ([]string, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string:
		if v == "all" {
			return []string{"create", "get", "update", "delete", "list", "paginate"}, nil
		}
		return nil, fmt.Errorf("invalid string value for default_functions: %q (only 'all' is supported)", v)
	case []interface{}:
		var functions []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				functions = append(functions, str)
			} else {
				return nil, fmt.Errorf("default_functions array must contain only strings")
			}
		}
		return functions, nil
	default:
		return nil, fmt.Errorf("default_functions must be a string ('all') or array of strings")
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var fileConfig FileConfig
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Extract table names from the new map structure
	var tableNames []string
	for tableName := range fileConfig.Tables {
		tableNames = append(tableNames, tableName)
	}

	// Parse default_functions field
	defaultFunctions, err := parseDefaultFunctions(fileConfig.DefaultFunctions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse default_functions: %w", err)
	}

	// Convert FileConfig to Config
	cfg := &Config{
		DSN:              fileConfig.Database.DSN,
		Schema:           fileConfig.Database.Schema,
		OutputDir:        fileConfig.Output.Directory,
		PackageName:      fileConfig.Output.Package,
		Tables:           len(fileConfig.Tables) > 0,
		QueriesDir:       fileConfig.Queries.Directory,
		Include:          tableNames,
		TableConfigs:     fileConfig.Tables,
		DefaultFunctions: defaultFunctions,
		TypeMappings:     fileConfig.Types.Mappings,
		Verbose:          fileConfig.Verbose,
	}

	// Set defaults
	if cfg.Schema == "" {
		cfg.Schema = "public"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./repositories"
	}
	if cfg.PackageName == "" {
		cfg.PackageName = "repositories"
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DSN == "" {
		// Check for TEST_DATABASE_URL environment variable for integration tests
		if testURL := os.Getenv("TEST_DATABASE_URL"); testURL != "" {
			c.DSN = testURL
		} else {
			return fmt.Errorf("database connection string (DSN) is required")
		}
	}

	if !c.Tables && c.QueriesDir == "" {
		return fmt.Errorf("must enable either table generation (--tables) or query generation (--queries)")
	}

	if c.QueriesDir != "" {
		if _, err := os.Stat(c.QueriesDir); os.IsNotExist(err) {
			return fmt.Errorf("queries directory does not exist: %s", c.QueriesDir)
		}
	}

	// Ensure output directory exists or can be created
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return nil
}

// GetOutputPath returns the full path for a generated file
func (c *Config) GetOutputPath(filename string) string {
	return filepath.Join(c.OutputDir, filename)
}

// ShouldIncludeTable checks if a table should be included based on include patterns
func (c *Config) ShouldIncludeTable(tableName string) bool {
	// No include patterns means no tables are included
	if len(c.Include) == 0 {
		return false
	}

	// Check include patterns
	for _, pattern := range c.Include {
		if matched, _ := filepath.Match(pattern, tableName); matched {
			return true
		}
	}

	return false
}

// GetTableFunctions returns the list of functions to generate for a specific table
func (c *Config) GetTableFunctions(tableName string) []string {
	// Check for table-specific override first
	if config, exists := c.TableConfigs[tableName]; exists {
		if len(config.Functions) > 0 {
			return config.Functions
		}
		// If table exists but functions are empty, use default
	}

	// Use global default_functions if specified
	if len(c.DefaultFunctions) > 0 {
		return c.DefaultFunctions
	}

	// Final fallback to all functions
	return []string{"create", "get", "update", "delete", "list", "paginate"}
}
