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
	// ForeignKeys lists every FK column on this table. Composite FKs surface
	// as multiple rows sharing ConstraintName. Cross-schema references are
	// not surfaced.
	ForeignKeys []ForeignKey `json:"foreign_keys"`
	// Audit indicates the table is configured to maintain an SCD Type 2
	// audit history via a parallel <table>_audit table.
	Audit bool `json:"audit"`
}

// ForeignKey represents one column of a foreign-key constraint. Composite
// (multi-column) foreign keys surface as multiple ForeignKey entries sharing
// the same ConstraintName. The referenced table is always in the same schema.
type ForeignKey struct {
	ConstraintName   string `json:"constraint_name"`
	ColumnName       string `json:"column_name"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
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
	Name                 string                `json:"name"`
	SQL                  string                `json:"sql"`
	Type                 QueryType             `json:"type"` // :one, :many, :exec, :paginated
	Parameters           []Parameter           `json:"parameters"`
	Columns              []Column              `json:"columns"` // Result columns (for SELECT queries)
	SourceFile           string                `json:"source_file"`
	ParameterAnnotations []ParameterAnnotation `json:"parameter_annotations"` // Optional type annotations
	ResultAnnotations    []ResultAnnotation    `json:"result_annotations"`    // Optional result column type annotations
	OrderByColumns       []OrderByColumn       `json:"order_by_columns"`      // Columns from ORDER BY clause (for :paginated queries)
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
	Name       string `json:"name"`
	Type       string `json:"type"`        // PostgreSQL type
	GoType     string `json:"go_type"`     // Go type
	Index      int    `json:"index"`       // Parameter position (1-based)
	Nullable   bool   `json:"nullable"`    // Whether the parameter should be nullable (pointer type)
	TableName  string `json:"table_name"`  // Table name if detected (for nullability lookup)
	ColumnName string `json:"column_name"` // Column name if detected (for nullability lookup)
}

// ParameterAnnotation represents an explicit parameter type annotation from SQL comments
type ParameterAnnotation struct {
	Position int    `json:"position"` // 1-based parameter position ($1, $2, etc)
	Name     string `json:"name"`     // Parameter name in snake_case
	GoType   string `json:"go_type"`  // Go type (supports pointers like "*string")
}

// ResultAnnotation represents an explicit result column type annotation from SQL comments
type ResultAnnotation struct {
	ColumnName string `json:"column_name"` // Column name in snake_case
	GoType     string `json:"go_type"`     // Go type (supports pointers like "*string")
}

// OrderByColumn represents a column in an ORDER BY clause with its sort direction
type OrderByColumn struct {
	Name      string `json:"name"`      // Column name (from SELECT alias or column name)
	Direction string `json:"direction"` // "ASC" or "DESC"
	GoType    string `json:"go_type"`   // Resolved Go type from Query.Columns
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

// HasIndexLeadingWith reports whether the table has an index whose first key
// column is `column`. An empty argument never matches (expression indexes
// have an empty leading column slot).
func (t *Table) HasIndexLeadingWith(column string) bool {
	if column == "" {
		return false
	}
	for i := range t.Indexes {
		cols := t.Indexes[i].Columns
		if len(cols) > 0 && cols[0] == column {
			return true
		}
	}
	return false
}

// HasForeignKeyTo reports whether the table has a foreign key from
// `childColumn` referencing `referencedTable`.`referencedColumn`.
func (t *Table) HasForeignKeyTo(childColumn, referencedTable, referencedColumn string) bool {
	for i := range t.ForeignKeys {
		fk := &t.ForeignKeys[i]
		if fk.ColumnName == childColumn &&
			fk.ReferencedTable == referencedTable &&
			fk.ReferencedColumn == referencedColumn {
			return true
		}
	}
	return false
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
	return strings.EqualFold(c.Type, "uuid")
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
			if part != "" {
				result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
			}
		}
		return result
	}

	// If it's already PascalCase or camelCase, just ensure first letter is uppercase
	if s != "" {
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
