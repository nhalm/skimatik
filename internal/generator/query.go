package generator

import (
	"strings"
)

// query represents a parsed SQL query with metadata
type query struct {
	Name                 string                `json:"name"`
	SQL                  string                `json:"sql"`
	Type                 queryType             `json:"type"` // :one, :many, :exec, :paginated
	Parameters           []parameter           `json:"parameters"`
	Columns              []column              `json:"columns"` // Result columns (for SELECT queries)
	SourceFile           string                `json:"source_file"`
	ParameterAnnotations []parameterAnnotation `json:"parameter_annotations"` // Optional type annotations
	ResultAnnotations    []resultAnnotation    `json:"result_annotations"`    // Optional result column type annotations
	OrderByColumns       []orderByColumn       `json:"order_by_columns"`      // Columns from ORDER BY clause (for :paginated queries)
}

// queryType represents the type of query operation
type queryType string

const (
	queryTypeOne       queryType = "one"       // Returns single row
	queryTypeMany      queryType = "many"      // Returns multiple rows
	queryTypeExec      queryType = "exec"      // Executes without returning rows
	queryTypePaginated queryType = "paginated" // Returns paginated results
)

// parameter represents a query parameter
type parameter struct {
	Name       string `json:"name"`
	Type       string `json:"type"`        // PostgreSQL type
	GoType     string `json:"go_type"`     // Go type
	Index      int    `json:"index"`       // Parameter position (1-based)
	Nullable   bool   `json:"nullable"`    // Whether the parameter should be nullable (pointer type)
	TableName  string `json:"table_name"`  // Table name if detected (for nullability lookup)
	ColumnName string `json:"column_name"` // Column name if detected (for nullability lookup)
}

// parameterAnnotation represents an explicit parameter type annotation from SQL comments
type parameterAnnotation struct {
	Position int    `json:"position"` // 1-based parameter position ($1, $2, etc)
	Name     string `json:"name"`     // Parameter name in snake_case
	GoType   string `json:"go_type"`  // Go type (supports pointers like "*string")
}

// resultAnnotation represents an explicit result column type annotation from SQL comments
type resultAnnotation struct {
	ColumnName string `json:"column_name"` // Column name in snake_case
	GoType     string `json:"go_type"`     // Go type (supports pointers like "*string")
}

// orderByColumn represents a column in an ORDER BY clause with its sort direction
type orderByColumn struct {
	Name      string `json:"name"`      // Column name (from SELECT alias or column name)
	Direction string `json:"direction"` // "ASC" or "DESC"
	GoType    string `json:"go_type"`   // Resolved Go type from Query.Columns
}

// goFunctionName returns the Go function name for this query
func (q *query) goFunctionName() string {
	return toPascalCase(q.Name)
}

// goFileName returns the Go file name for queries from the same source file
func (q *query) goFileName() string {
	// Extract base name from source file path
	parts := strings.Split(q.SourceFile, "/")
	filename := parts[len(parts)-1]

	// Remove .sql extension and convert to snake_case
	name := strings.TrimSuffix(filename, ".sql")
	return toSnakeCase(name) + "_queries_generated.go"
}
