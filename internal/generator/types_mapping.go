package generator

import (
	"fmt"
	"strings"
)

// TypeMapper maps PostgreSQL types to Go types using intelligent type conventions.
// It generates consistent, idiomatic Go types with pointer-based nullability:
// - All integer types (smallint, integer, bigint) map to 'int'
// - Nullable columns use pointers (*int, *string, etc.)
// - NOT NULL columns use native Go types (int, string, uuid.UUID, etc.)
type TypeMapper struct {
	customMappings map[string]string
}

// NewTypeMapper creates a new type mapper with optional custom mappings.
// Custom mappings allow overriding default PostgreSQL to Go type conversions.
func NewTypeMapper(customMappings map[string]string) *TypeMapper {
	return &TypeMapper{
		customMappings: customMappings,
	}
}

// MapType converts a PostgreSQL type to the appropriate Go type.
// It applies custom mappings first, then defaults to intelligent type detection.
// The isNullable flag determines pointer vs value types, and isArray wraps in slices.
func (tm *TypeMapper) MapType(pgType string, isNullable, isArray bool) (string, error) {
	// Check custom mappings first
	if customType, exists := tm.customMappings[pgType]; exists {
		result := tm.applyNullableAndArray(customType, isNullable, isArray)
		return result, nil
	}

	// Get the base Go type
	baseType, err := tm.getBaseGoType(pgType)
	if err != nil {
		return "", err
	}

	result := tm.applyNullableAndArray(baseType, isNullable, isArray)
	return result, nil
}

// getBaseGoType returns the base Go type for a PostgreSQL type
func (tm *TypeMapper) getBaseGoType(pgType string) (string, error) {
	normalized := strings.ToLower(pgType)
	if goType, ok := mapNumericPgType(normalized); ok {
		return goType, nil
	}
	if goType, ok := mapStringPgType(normalized); ok {
		return goType, nil
	}
	if goType, ok := mapTimePgType(normalized); ok {
		return goType, nil
	}
	if goType, ok := mapMiscPgType(normalized); ok {
		return goType, nil
	}
	return "", fmt.Errorf("unsupported PostgreSQL type: %s", pgType)
}

// mapNumericPgType maps numeric/boolean PostgreSQL types to their Go types.
func mapNumericPgType(pgType string) (string, bool) {
	switch pgType {
	// Integer types - all map to int for ergonomic, idiomatic Go
	case "smallint", "int2", "integer", "int", "int4", "bigint", "int8", "serial", "bigserial", "smallserial":
		return "int", true
	// Floating point types
	case "real", "float4":
		return "float32", true
	case "double precision", "float8":
		return "float64", true
	case "numeric", "decimal":
		return "float64", true // Could also use shopspring/decimal for precision
	// Boolean type
	case "boolean", "bool":
		return "bool", true
	}
	return "", false
}

// mapStringPgType maps string-like PostgreSQL types to their Go types.
// This includes textual types as well as network, geometric, range and other
// types that we currently represent as strings.
func mapStringPgType(pgType string) (string, bool) {
	switch pgType {
	// String types
	case "text", "varchar", "character varying", "char", "character":
		return "string", true
	// Network types
	case "inet", "cidr":
		return "string", true // Could use net.IP for more type safety
	case "macaddr":
		return "string", true
	// Geometric types (simplified to strings for now)
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		return "string", true
	// Range types (simplified to strings for now)
	case "int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange":
		return "string", true
	// Interval type
	case "interval":
		return "string", true //TODO: Could use time.Duration for more type safety
	// XML type
	case "xml":
		return "string", true
	}
	return "", false
}

// mapTimePgType maps date/time PostgreSQL types to their Go types.
func mapTimePgType(pgType string) (string, bool) {
	switch pgType {
	case "date":
		return "time.Time", true
	case "time", "time without time zone":
		return "time.Time", true
	case "timetz", "time with time zone":
		return "time.Time", true
	case "timestamp", "timestamp without time zone":
		return "time.Time", true
	case "timestamptz", "timestamp with time zone":
		return "time.Time", true
	}
	return "", false
}

// mapMiscPgType maps miscellaneous PostgreSQL types (uuid, bytea, json) to their Go types.
func mapMiscPgType(pgType string) (string, bool) {
	switch pgType {
	// UUID types
	case "uuid":
		return "uuid.UUID", true
	// Binary types
	case "bytea":
		return "[]byte", true
	// JSON types - use json.RawMessage for pgx v5
	case "json", "jsonb":
		return "json.RawMessage", true
	}
	return "", false
}

// applyNullableAndArray applies nullable and array modifiers to a base type
func (tm *TypeMapper) applyNullableAndArray(baseType string, isNullable, isArray bool) string {
	result := baseType

	// Handle arrays first
	if isArray {
		result = "[]" + result
	}

	// Handle nullable types
	if isNullable {
		result = tm.makeNullable(result)
	}

	return result
}

// makeNullable converts a Go type to its nullable equivalent using pointers
func (tm *TypeMapper) makeNullable(goType string) string {
	// Already a pointer
	if strings.HasPrefix(goType, "*") {
		return goType
	}

	// Special case: []byte represents binary data, nullable should be *[]byte
	if goType == "[]byte" {
		return "*[]byte"
	}

	// Handle array types - make elements nullable
	if strings.HasPrefix(goType, "[]") {
		elementType := goType[2:]
		return "[]" + tm.makeNullable(elementType)
	}

	// For all types, use pointer for nullable
	return "*" + goType
}

// GetRequiredImports returns the imports needed for the generated Go types.
// It scans all column types and collects necessary package imports (uuid, time, json, etc.).
func (tm *TypeMapper) GetRequiredImports(columns []Column) []string {
	imports := make(map[string]bool)

	for _, col := range columns {
		goType, err := tm.MapType(col.Type, col.IsNullable, col.IsArray)
		if err != nil {
			continue // Skip unsupported types
		}

		// Check what imports are needed based on the Go type
		tm.addImportsForType(goType, imports)
	}

	// Convert map to slice
	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}

	// Ensure we return an empty slice instead of nil
	if result == nil {
		result = []string{}
	}

	return result
}

// addImportsForType adds necessary imports for a Go type
func (tm *TypeMapper) addImportsForType(goType string, imports map[string]bool) {
	// Handle array types
	if strings.HasPrefix(goType, "[]") {
		tm.addImportsForType(goType[2:], imports)
		return
	}

	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		tm.addImportsForType(goType[1:], imports)
		return
	}

	// Check for specific types that need imports
	switch {
	case strings.Contains(goType, "uuid.UUID"):
		imports["github.com/google/uuid"] = true
	case strings.Contains(goType, "time.Time"):
		imports["time"] = true
	case strings.Contains(goType, "json.RawMessage"):
		imports["encoding/json"] = true
	}
}

// MapTableColumns maps all columns in a table and sets their GoType field.
// This is used during table struct generation from database introspection.
func (tm *TypeMapper) MapTableColumns(table *Table) error {
	if table == nil {
		return fmt.Errorf("table cannot be nil")
	}

	for i := range table.Columns {
		goType, err := tm.MapType(table.Columns[i].Type, table.Columns[i].IsNullable, table.Columns[i].IsArray)
		if err != nil {
			return fmt.Errorf("failed to map type for column %s: %w", table.Columns[i].Name, err)
		}
		table.Columns[i].GoType = goType
	}
	return nil
}

// ValidateUUIDPrimaryKey ensures a column is a valid UUID type for primary keys.
// It checks that the column is UUID type, non-nullable, and not an array.
func (tm *TypeMapper) ValidateUUIDPrimaryKey(column *Column) error {
	if column == nil {
		return fmt.Errorf("column cannot be nil")
	}

	if !column.IsUUID() {
		return fmt.Errorf("primary key column %s must be UUID type, got %s", column.Name, column.Type)
	}

	if column.IsNullable {
		return fmt.Errorf("primary key column %s cannot be nullable", column.Name)
	}

	if column.IsArray {
		return fmt.Errorf("primary key column %s cannot be an array", column.Name)
	}

	return nil
}
