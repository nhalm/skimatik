package generator

import (
	"strings"
)

// Table represents a database table with its columns and metadata
type Table struct {
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	Columns    []Column `json:"columns"`
	PrimaryKey []string `json:"primary_key"`
	Indexes    []Index  `json:"indexes"`
}

// Column represents a database column with its type and constraints
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`    // PostgreSQL type (e.g., "uuid", "text", "integer")
	GoType       string `json:"go_type"` // Go type (e.g., "uuid.UUID", "string", "int32")
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value"`
	IsArray      bool   `json:"is_array"`
	MaxLength    int    `json:"max_length"`
}

// Index represents a database index
type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"is_unique"`
}

// Query represents a parsed SQL query with metadata
type Query struct {
	Name       string      `json:"name"`
	SQL        string      `json:"sql"`
	Type       QueryType   `json:"type"` // :one, :many, :exec, :paginated
	Parameters []Parameter `json:"parameters"`
	Columns    []Column    `json:"columns"` // Result columns (for SELECT queries)
	SourceFile string      `json:"source_file"`
}

// QueryType represents the type of query operation
type QueryType string

const (
	QueryTypeOne       QueryType = "one"       // Returns single row
	QueryTypeMany      QueryType = "many"      // Returns multiple rows
	QueryTypeExec      QueryType = "exec"      // Executes without returning rows
	QueryTypePaginated QueryType = "paginated" // Returns paginated results
)

// Parameter represents a query parameter
type Parameter struct {
	Name   string `json:"name"`
	Type   string `json:"type"`    // PostgreSQL type
	GoType string `json:"go_type"` // Go type
	Index  int    `json:"index"`   // Parameter position (1-based)
}

// GetColumn returns a column by name, or nil if not found
func (t *Table) GetColumn(name string) *Column {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i]
		}
	}
	return nil
}

// GetPrimaryKeyColumn returns the primary key column (assumes single column PK)
func (t *Table) GetPrimaryKeyColumn() *Column {
	if len(t.PrimaryKey) != 1 {
		return nil
	}
	return t.GetColumn(t.PrimaryKey[0])
}

// GoStructName returns the Go struct name for this table
func (t *Table) GoStructName() string {
	return toPascalCase(t.Name)
}

// GoFileName returns the Go file name for this table's repository
func (t *Table) GoFileName() string {
	return toSnakeCase(t.Name) + "_generated.go"
}

// IsUUID checks if the column is a UUID type
func (c *Column) IsUUID() bool {
	return strings.ToLower(c.Type) == "uuid"
}

// IsString checks if the column is a string type
func (c *Column) IsString() bool {
	switch strings.ToLower(c.Type) {
	case "text", "varchar", "character varying", "char", "character":
		return true
	default:
		return false
	}
}

// IsInteger checks if the column is an integer type
func (c *Column) IsInteger() bool {
	switch strings.ToLower(c.Type) {
	case "integer", "int", "int4", "bigint", "int8", "smallint", "int2":
		return true
	default:
		return false
	}
}

// IsBoolean checks if the column is a boolean type
func (c *Column) IsBoolean() bool {
	return strings.ToLower(c.Type) == "boolean" || strings.ToLower(c.Type) == "bool"
}

// IsTimestamp checks if the column is a timestamp type
func (c *Column) IsTimestamp() bool {
	//TODO:  change this to a map or use a constant slice.  The switch is not a good solution.
	switch strings.ToLower(c.Type) {
	case "timestamp", "timestamptz", "timestamp with time zone", "timestamp without time zone", "date", "time", "time without time zone", "time with time zone":
		return true
	default:
		return false
	}
}

// GoFieldName returns the Go field name for this column
func (c *Column) GoFieldName() string {
	return toPascalCase(c.Name)
}

// GoStructTag returns the Go struct tag for this column
func (c *Column) GoStructTag() string {
	return `json:"` + c.Name + `" db:"` + c.Name + `"`
}

// GoFunctionName returns the Go function name for this query
func (q *Query) GoFunctionName() string {
	return toPascalCase(q.Name)
}

// GoFileName returns the Go file name for queries from the same source file
func (q *Query) GoFileName() string {
	// Extract base name from source file path
	parts := strings.Split(q.SourceFile, "/")
	filename := parts[len(parts)-1]

	// Remove .sql extension and convert to snake_case
	name := strings.TrimSuffix(filename, ".sql")
	return toSnakeCase(name) + "_queries_generated.go"
}

// Utility functions for naming conventions

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// If it contains underscores, split on them
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
			}
		}
		return result
	}

	// If it's already PascalCase or camelCase, just ensure first letter is uppercase
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}

	return s
}

// toSnakeCase converts PascalCase or camelCase to snake_case
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
