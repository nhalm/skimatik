package generator

import (
	"fmt"
	"strings"
)

// TypeMapper handles mapping PostgreSQL types to Go types
type TypeMapper struct {
	customMappings map[string]string
}

// NewTypeMapper creates a new type mapper with optional custom mappings
func NewTypeMapper(customMappings map[string]string) *TypeMapper {
	return &TypeMapper{
		customMappings: customMappings,
	}
}

// MapType converts a PostgreSQL type to the appropriate Go type
func (tm *TypeMapper) MapType(pgType string, isNullable bool, isArray bool) (string, error) {
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
	switch strings.ToLower(pgType) {
	// UUID types
	case "uuid":
		return "uuid.UUID", nil

	// String types
	case "text", "varchar", "character varying", "char", "character":
		return "string", nil

	// Integer types
	case "smallint", "int2":
		return "int16", nil
	case "integer", "int", "int4":
		return "int32", nil
	case "bigint", "int8":
		return "int64", nil

	// Floating point types
	case "real", "float4":
		return "float32", nil
	case "double precision", "float8":
		return "float64", nil
	case "numeric", "decimal":
		return "float64", nil // Could also use shopspring/decimal for precision

	// Boolean type
	case "boolean", "bool":
		return "bool", nil

	// Date/time types
	case "date":
		return "time.Time", nil
	case "time", "time without time zone":
		return "time.Time", nil
	case "timetz", "time with time zone":
		return "time.Time", nil
	case "timestamp", "timestamp without time zone":
		return "time.Time", nil
	case "timestamptz", "timestamp with time zone":
		return "time.Time", nil

	// Binary types
	case "bytea":
		return "[]byte", nil

	// JSON types - use json.RawMessage for pgx v5
	case "json", "jsonb":
		return "json.RawMessage", nil

	// Network types
	case "inet", "cidr":
		return "string", nil // Could use net.IP for more type safety
	case "macaddr":
		return "string", nil

	// Geometric types (simplified to strings for now)
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		return "string", nil

	// Range types (simplified to strings for now)
	case "int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange":
		return "string", nil

	// Interval type
	case "interval":
		return "string", nil //TODO: Could use time.Duration for more type safety

	// XML type
	case "xml":
		return "string", nil

	// Array types are handled by the isArray parameter
	default:
		return "", fmt.Errorf("unsupported PostgreSQL type: %s", pgType)
	}
}

// applyNullableAndArray applies nullable and array modifiers to a base type
func (tm *TypeMapper) applyNullableAndArray(baseType string, isNullable bool, isArray bool) string {
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

// makeNullable converts a Go type to its nullable equivalent using pgtype
func (tm *TypeMapper) makeNullable(goType string) string {
	// Handle special cases first
	switch goType {
	case "[]byte":
		// In pgx v5, there's no pgtype.Bytea, use pointer to []byte
		return "*[]byte"
	case "string":
		return "pgtype.Text"
	case "int16":
		return "pgtype.Int2"
	case "int32":
		return "pgtype.Int4"
	case "int64":
		return "pgtype.Int8"
	case "float32":
		return "pgtype.Float4"
	case "float64":
		return "pgtype.Float8"
	case "bool":
		return "pgtype.Bool"
	case "time.Time":
		return "pgtype.Timestamptz"
	case "uuid.UUID":
		return "pgtype.UUID"
	case "json.RawMessage":
		// In pgx v5, there's no pgtype.JSON, use pointer to json.RawMessage
		return "*json.RawMessage"
	}

	// Handle array types
	if strings.HasPrefix(goType, "[]") {
		elementType := goType[2:]
		return "[]" + tm.makeNullable(elementType)
	}

	// For custom types or types we don't have pgtype equivalents for,
	// use a pointer to the type
	return "*" + goType
}

// GetRequiredImports returns the imports needed for the generated Go types
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
	var result []string
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
	case strings.Contains(goType, "pgtype."):
		imports["github.com/jackc/pgx/v5/pgtype"] = true
	}
}

// MapTableColumns maps all columns in a table and sets their GoType field
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

// MapQueryColumns maps all columns in a query and sets their GoType field
func (tm *TypeMapper) MapQueryColumns(query *Query) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}

	for i := range query.Columns {
		goType, err := tm.MapType(query.Columns[i].Type, query.Columns[i].IsNullable, query.Columns[i].IsArray)
		if err != nil {
			return fmt.Errorf("failed to map type for column %s in query %s: %w", query.Columns[i].Name, query.Name, err)
		}
		query.Columns[i].GoType = goType
	}

	// Also map parameter types
	for i := range query.Parameters {
		goType, err := tm.MapType(query.Parameters[i].Type, false, false) // Parameters are typically not nullable
		if err != nil {
			return fmt.Errorf("failed to map type for parameter %d in query %s: %w", query.Parameters[i].Index, query.Name, err)
		}
		query.Parameters[i].GoType = goType
	}

	return nil
}

// ValidateUUIDPrimaryKey ensures a column is a valid UUID type for primary keys
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
