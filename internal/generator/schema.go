package generator

import (
	"strings"
)

// table represents a database table with its columns and metadata
type table struct {
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	Columns    []column `json:"columns"`
	PrimaryKey []string `json:"primary_key"`
	Indexes    []index  `json:"indexes"`
	// ForeignKeys lists every FK column on this table. Composite FKs surface
	// as multiple rows sharing ConstraintName. Cross-schema references are
	// not surfaced.
	ForeignKeys []foreignKey `json:"foreign_keys"`
	// Audit indicates the table is configured to maintain an SCD Type 2
	// audit history via a parallel <table>_audit table.
	Audit bool `json:"audit"`
}

// foreignKey represents one column of a foreign-key constraint. Composite
// (multi-column) foreign keys surface as multiple foreignKey entries sharing
// the same ConstraintName. The referenced table is always in the same schema.
type foreignKey struct {
	ConstraintName   string `json:"constraint_name"`
	ColumnName       string `json:"column_name"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
}

// column represents a database column with its type and constraints
type column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`    // PostgreSQL type (e.g., "uuid", "text", "integer")
	GoType       string `json:"go_type"` // Go type (e.g., "uuid.UUID", "string", "int32")
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value"`
	IsArray      bool   `json:"is_array"`
	MaxLength    int    `json:"max_length"`
}

// index represents a database index
type index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"is_unique"`
}

// getColumn returns a column by name, or nil if not found
func (t *table) getColumn(name string) *column {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i]
		}
	}
	return nil
}

// getPrimaryKeyColumn returns the primary key column (assumes single column PK)
func (t *table) getPrimaryKeyColumn() *column {
	if len(t.PrimaryKey) != 1 {
		return nil
	}
	return t.getColumn(t.PrimaryKey[0])
}

// hasIndexLeadingWith reports whether the table has an index whose first key
// column is `column`. An empty argument never matches (expression indexes
// have an empty leading column slot).
func (t *table) hasIndexLeadingWith(column string) bool {
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

// hasForeignKeyTo reports whether the table has a foreign key from
// `childColumn` referencing `referencedTable`.`referencedColumn`.
func (t *table) hasForeignKeyTo(childColumn, referencedTable, referencedColumn string) bool {
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

// goStructName returns the Go struct name for this table
func (t *table) goStructName() string {
	return toPascalCase(t.Name)
}

// goFileName returns the Go file name for this table's repository
func (t *table) goFileName() string {
	return toSnakeCase(t.Name) + "_generated.go"
}

// isUUID checks if the column is a UUID type
func (c *column) isUUID() bool {
	return strings.EqualFold(c.Type, "uuid")
}

// goFieldName returns the Go field name for this column
func (c *column) goFieldName() string {
	return toPascalCase(c.Name)
}

// goStructTag returns the Go struct tag for this column
func (c *column) goStructTag() string {
	return `json:"` + c.Name + `" db:"` + c.Name + `"`
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
