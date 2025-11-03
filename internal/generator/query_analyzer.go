package generator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nhalm/pgxkit"
)

// QueryAnalyzer analyzes SQL queries using PostgreSQL EXPLAIN to determine column types and validate queries
type QueryAnalyzer struct {
	db         *pgxkit.DB
	typeMapper *TypeMapper
	sqlParser  *SQLParser
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(db *pgxkit.DB) *QueryAnalyzer {
	return &QueryAnalyzer{
		db:         db,
		typeMapper: NewTypeMapper(nil),
		sqlParser:  NewSQLParser(),
	}
}

// AnalyzeQuery analyzes a query using PostgreSQL EXPLAIN to determine column types and parameters
func (qa *QueryAnalyzer) AnalyzeQuery(ctx context.Context, query *Query) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}

	// Extract parameters from the query (doesn't require database connection)
	if err := qa.extractParameters(query); err != nil {
		return fmt.Errorf("failed to extract parameters: %w", err)
	}

	// Infer parameter names and track table/column associations (doesn't require database connection)
	if err := qa.inferParameterNames(query); err != nil {
		return fmt.Errorf("failed to infer parameter names: %w", err)
	}

	// If query is empty, no further analysis needed
	if strings.TrimSpace(query.SQL) == "" {
		return nil
	}

	// Database connection is required for further analysis
	if qa.db == nil {
		return fmt.Errorf("database connection required for query analysis")
	}

	// For SELECT queries, analyze columns using EXPLAIN
	if qa.isSelectQuery(query.Type) {
		if err := qa.analyzeSelectQuery(ctx, query); err != nil {
			return fmt.Errorf("failed to analyze SELECT query: %w", err)
		}

		// Apply result annotations after automatic type detection
		if err := qa.applyResultAnnotations(query); err != nil {
			return fmt.Errorf("failed to apply result annotations: %w", err)
		}
	}

	// Infer parameter types using PostgreSQL PREPARE (for all query types)
	if err := qa.inferParameterTypesFromPrepare(ctx, query); err != nil {
		return fmt.Errorf("failed to infer parameter types: %w", err)
	}

	// Infer parameter nullability from schema for parameters with known table/column
	if err := qa.inferParameterNullability(ctx, query); err != nil {
		return fmt.Errorf("failed to infer parameter nullability: %w", err)
	}

	// Validate query syntax by attempting to prepare it
	if err := qa.validateQuerySyntax(ctx, query); err != nil {
		return fmt.Errorf("query syntax validation failed: %w", err)
	}

	return nil
}

// applyResultAnnotations applies result type annotations to override automatically detected types
func (qa *QueryAnalyzer) applyResultAnnotations(query *Query) error {
	if len(query.ResultAnnotations) == 0 {
		return nil
	}

	// Apply each result annotation
	for _, annotation := range query.ResultAnnotations {
		// Find the column in query results
		columnFound := false
		for i := range query.Columns {
			if query.Columns[i].Name == annotation.ColumnName {
				columnFound = true

				// Override GoType with annotation
				query.Columns[i].GoType = annotation.GoType

				// Update IsNullable based on whether GoType is a pointer
				query.Columns[i].IsNullable = strings.HasPrefix(annotation.GoType, "*")

				break
			}
		}

		// Error if annotation references non-existent column
		if !columnFound {
			return fmt.Errorf("result annotation for column '%s' not found in query %s results", annotation.ColumnName, query.Name)
		}
	}

	return nil
}

// extractParameters extracts parameter placeholders from the SQL query
func (qa *QueryAnalyzer) extractParameters(query *Query) error {
	// Use SQLParser to extract parameters
	queryInfo, err := qa.sqlParser.Parse(query.SQL)
	if err != nil {
		// Fall back to basic regex extraction if parsing fails
		slog.Warn("SQL parser failed, falling back to regex extraction",
			"query", query.Name,
			"error", err.Error())
		return qa.extractParametersRegex(query)
	}

	if len(queryInfo.Parameters) == 0 {
		query.Parameters = []Parameter{}
		return nil
	}

	// Create parameter list from parse tree
	paramMap := make(map[int]bool)
	for _, paramInfo := range queryInfo.Parameters {
		paramMap[paramInfo.Position] = true
	}

	// Create parameter list from the parameters found
	var parameters []Parameter
	for paramNum := range paramMap {
		param := Parameter{
			Name:   fmt.Sprintf("param%d", paramNum),
			Type:   "text",
			GoType: "string",
			Index:  paramNum,
		}
		parameters = append(parameters, param)
	}

	// Sort parameters by index to ensure consistent ordering
	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].Index < parameters[j].Index
	})

	query.Parameters = parameters
	return nil
}

// extractParametersRegex is a fallback regex-based parameter extraction
func (qa *QueryAnalyzer) extractParametersRegex(query *Query) error {
	// Remove string literals and quoted identifiers to avoid false positives
	cleanSQL := qa.removeQuotedContent(query.SQL)

	// Find all parameter placeholders ($1, $2, etc.)
	// This is a simple fallback - the parser should handle most cases
	paramMap := make(map[int]bool)
	for i := 0; i < len(cleanSQL); i++ {
		if cleanSQL[i] == '$' && i+1 < len(cleanSQL) && cleanSQL[i+1] >= '0' && cleanSQL[i+1] <= '9' {
			numStr := ""
			j := i + 1
			for j < len(cleanSQL) && cleanSQL[j] >= '0' && cleanSQL[j] <= '9' {
				numStr += string(cleanSQL[j])
				j++
			}
			if num, err := strconv.Atoi(numStr); err == nil {
				paramMap[num] = true
			}
		}
	}

	if len(paramMap) == 0 {
		query.Parameters = []Parameter{}
		return nil
	}

	// Create parameter list
	var parameters []Parameter
	for paramNum := range paramMap {
		param := Parameter{
			Name:   fmt.Sprintf("param%d", paramNum),
			Type:   "text",
			GoType: "string",
			Index:  paramNum,
		}
		parameters = append(parameters, param)
	}

	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].Index < parameters[j].Index
	})

	query.Parameters = parameters
	return nil
}

// inferParameterNames infers semantic parameter names from SQL context using the SQL parser
func (qa *QueryAnalyzer) inferParameterNames(query *Query) error {
	if len(query.Parameters) == 0 {
		return nil
	}

	// Parse the SQL to get parameter context
	queryInfo, err := qa.sqlParser.Parse(query.SQL)
	if err != nil {
		// If parsing fails, leave parameter names as default "paramN"
		return nil
	}

	// Create maps to track inferred data by parameter index
	inferredNames := make(map[int]string)
	inferredColumns := make(map[int]string)
	inferredTables := make(map[int]string)

	// Extract information from parsed parameters
	for _, paramInfo := range queryInfo.Parameters {
		pos := paramInfo.Position

		// Infer parameter name based on context
		if paramInfo.IsInLimit {
			inferredNames[pos] = "limit"
		} else if paramInfo.IsInOffset {
			inferredNames[pos] = "offset"
		} else if paramInfo.ColumnName != "" {
			// Check if this is a LIKE operator with search term
			if paramInfo.Operator == "~~" || paramInfo.Operator == "~~*" ||
			   strings.ToUpper(paramInfo.Operator) == "LIKE" ||
			   strings.ToUpper(paramInfo.Operator) == "ILIKE" {
				// For LIKE patterns, use "search" prefix with column name to avoid collisions
				if _, exists := inferredNames[pos]; !exists {
					inferredNames[pos] = "search" + toPascalCase(paramInfo.ColumnName)
				}
			} else {
				// Use column name converted to camelCase
				inferredNames[pos] = toCamelCase(paramInfo.ColumnName)
			}

			// Track column name for nullability lookup
			inferredColumns[pos] = paramInfo.ColumnName

			// Track table name if available
			if paramInfo.TableName != "" {
				inferredTables[pos] = paramInfo.TableName
			}
		}
	}

	// Apply inferred names and table/column associations to parameters
	for i := range query.Parameters {
		paramIndex := query.Parameters[i].Index
		if name, exists := inferredNames[paramIndex]; exists {
			query.Parameters[i].Name = name
		}
		if column, exists := inferredColumns[paramIndex]; exists {
			query.Parameters[i].ColumnName = column
		}
		if table, exists := inferredTables[paramIndex]; exists {
			query.Parameters[i].TableName = table
		}
	}

	return nil
}

// toCamelCase converts a snake_case or regular identifier to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}

	// Split by underscore
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		// No underscores, just lowercase first letter
		return strings.ToLower(s[:1]) + s[1:]
	}

	// First part is lowercase, rest are Title case
	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return result
}

// inferParameterNullability looks up column nullability from the database schema
func (qa *QueryAnalyzer) inferParameterNullability(ctx context.Context, query *Query) error {
	for i := range query.Parameters {
		param := &query.Parameters[i]

		// Only look up nullability if we have both table and column name
		if param.TableName == "" || param.ColumnName == "" {
			continue
		}

		// Query information_schema to check if column is nullable
		schemaQuery := `SELECT is_nullable FROM information_schema.columns
		                WHERE table_schema = 'public'
		                AND table_name = $1
		                AND column_name = $2`

		var isNullable string
		err := qa.db.QueryRow(ctx, schemaQuery, param.TableName, param.ColumnName).Scan(&isNullable)
		if err != nil {
			// If we can't find the column, skip nullability (might be an alias or expression)
			continue
		}

		// Set nullable flag if column allows NULL
		param.Nullable = (isNullable == "YES")

		// If nullable, update GoType to be a pointer
		if param.Nullable {
			param.GoType = makePointerType(param.GoType)
		}
	}

	return nil
}

// makePointerType converts a Go type to its pointer equivalent
func makePointerType(goType string) string {
	// Already a pointer
	if strings.HasPrefix(goType, "*") {
		return goType
	}
	return "*" + goType
}

// removeQuotedContent removes string literals and quoted identifiers to avoid false parameter detection
func (qa *QueryAnalyzer) removeQuotedContent(sql string) string {
	result := []rune(sql)
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(result); i++ {
		if result[i] == '\'' && (i == 0 || result[i-1] != '\\') {
			if inSingleQuote {
				// Check for escaped quote ''
				if i+1 < len(result) && result[i+1] == '\'' {
					result[i] = ' '
					result[i+1] = ' '
					i++
				} else {
					inSingleQuote = false
				}
			} else if !inDoubleQuote {
				inSingleQuote = true
			}
			if inSingleQuote || (!inSingleQuote && i > 0) {
				result[i] = ' '
			}
		} else if result[i] == '"' && (i == 0 || result[i-1] != '\\') {
			if inDoubleQuote {
				// Check for escaped quote ""
				if i+1 < len(result) && result[i+1] == '"' {
					result[i] = ' '
					result[i+1] = ' '
					i++
				} else {
					inDoubleQuote = false
				}
			} else if !inSingleQuote {
				inDoubleQuote = true
			}
			if inDoubleQuote || (!inDoubleQuote && i > 0) {
				result[i] = ' '
			}
		} else if inSingleQuote || inDoubleQuote {
			result[i] = ' '
		} else if result[i] == '-' && i+1 < len(result) && result[i+1] == '-' {
			// Remove single-line comments
			for i < len(result) && result[i] != '\n' && result[i] != '\r' {
				result[i] = ' '
				i++
			}
		}
	}

	return string(result)
}

// isSelectQuery checks if the query type requires column analysis
func (qa *QueryAnalyzer) isSelectQuery(queryType QueryType) bool {
	return queryType == QueryTypeOne || queryType == QueryTypeMany || queryType == QueryTypePaginated
}

// analyzeSelectQuery uses EXPLAIN to analyze a SELECT query and determine column types
func (qa *QueryAnalyzer) analyzeSelectQuery(ctx context.Context, query *Query) error {
	// Replace parameters with dummy values for EXPLAIN
	analyzableSQL := qa.replaceParametersForExplain(query.SQL, query.Parameters)
	explainSQL := fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", analyzableSQL)

	// Execute EXPLAIN query
	rows, err := qa.db.Query(ctx, explainSQL)
	if err != nil {
		return fmt.Errorf("failed to execute EXPLAIN query: %w", err)
	}
	defer rows.Close()

	// For now, we'll use a simpler approach: try to execute the query with dummy parameters
	// to get the column information from the result set
	return qa.analyzeQueryColumns(ctx, query)
}

// replaceParametersForExplain replaces parameter placeholders with dummy values for EXPLAIN
func (qa *QueryAnalyzer) replaceParametersForExplain(sql string, parameters []Parameter) string {
	result := sql

	// Replace parameters in reverse order to avoid issues with $1 vs $10
	for i := len(parameters); i >= 1; i-- {
		placeholder := fmt.Sprintf("$%d", i)
		dummyValue := qa.getDummyValueForParameter()

		// Use a more sophisticated replacement that avoids string literals
		// For now, we'll use a simple approach but this could be enhanced
		result = qa.replaceParameterOutsideQuotes(result, placeholder, dummyValue)
	}
	return result
}

// replaceParameterOutsideQuotes replaces parameter only when it's not inside quotes
func (qa *QueryAnalyzer) replaceParameterOutsideQuotes(sql, placeholder, replacement string) string {
	result := []rune(sql)
	searchRunes := []rune(placeholder)
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(result); i++ {
		if result[i] == '\'' && (i == 0 || result[i-1] != '\\') {
			if inSingleQuote {
				if i+1 < len(result) && result[i+1] == '\'' {
					i++
				} else {
					inSingleQuote = false
				}
			} else if !inDoubleQuote {
				inSingleQuote = true
			}
		} else if result[i] == '"' && (i == 0 || result[i-1] != '\\') {
			if inDoubleQuote {
				if i+1 < len(result) && result[i+1] == '"' {
					i++
				} else {
					inDoubleQuote = false
				}
			} else if !inSingleQuote {
				inDoubleQuote = true
			}
		} else if !inSingleQuote && !inDoubleQuote {
			// Check if we're at the placeholder position
			match := true
			if i+len(searchRunes) <= len(result) {
				for j := 0; j < len(searchRunes); j++ {
					if result[i+j] != searchRunes[j] {
						match = false
						break
					}
				}
				if match {
					// Replace the placeholder
					replRunes := []rune(replacement)
					newResult := make([]rune, 0, len(result)-len(searchRunes)+len(replRunes))
					newResult = append(newResult, result[:i]...)
					newResult = append(newResult, replRunes...)
					newResult = append(newResult, result[i+len(searchRunes):]...)
					result = newResult
					i += len(replRunes) - 1
				}
			}
		}
	}

	return string(result)
}

// getDummyValueForParameter returns a dummy value for a parameter
func (qa *QueryAnalyzer) getDummyValueForParameter() string {
	// Use NULL which works with all types and avoids type conversion issues
	return "NULL"
}

// analyzeQueryColumns analyzes the columns returned by a SELECT query
func (qa *QueryAnalyzer) analyzeQueryColumns(ctx context.Context, query *Query) error {
	// Remove trailing semicolon if present
	sql := strings.TrimSpace(query.SQL)
	sql = strings.TrimSuffix(sql, ";")

	// Create a modified query that returns column information
	// We'll use a LIMIT 0 query to get column metadata without executing the full query
	limitedSQL := fmt.Sprintf("SELECT * FROM (%s) AS subquery LIMIT 0", sql)

	// Replace parameters with dummy values
	analyzableSQL := qa.replaceParametersForExplain(limitedSQL, query.Parameters)

	// Execute the query to get column information
	rows, err := qa.db.Query(ctx, analyzableSQL)
	if err != nil {
		return fmt.Errorf("failed to analyze query columns: %w", err)
	}
	defer rows.Close()

	// Get column descriptions
	fieldDescriptions := rows.FieldDescriptions()
	var columns []Column

	for _, field := range fieldDescriptions {
		// Map PostgreSQL OID to type name
		pgType := qa.mapOIDToTypeName(field.DataTypeOID)

		// Determine if the column is nullable using intelligent detection
		isNullable, err := qa.isColumnNullable(ctx, field, query.SQL)
		if err != nil {
			return fmt.Errorf("failed to determine nullability for %s: %w", field.Name, err)
		}

		// Map to Go type
		goType, err := qa.mapToIntelligentGoType(pgType, isNullable)
		if err != nil {
			return fmt.Errorf("failed to map column type for %s (pgType=%s, nullable=%v): %w", field.Name, pgType, isNullable, err)
		}

		if goType == "" {
			return fmt.Errorf("empty GoType generated for column %s (pgType=%s, nullable=%v)", field.Name, pgType, isNullable)
		}

		column := Column{
			Name:       field.Name,
			Type:       pgType,
			GoType:     goType,
			IsNullable: isNullable,
			IsArray:    false, // TODO: Detect array types from OID
		}
		columns = append(columns, column)
	}

	query.Columns = columns
	return nil
}

// isColumnNullable determines if a column can be NULL based on FieldDescription
func (qa *QueryAnalyzer) isColumnNullable(ctx context.Context, field pgconn.FieldDescription, sql string) (bool, error) {
	// If TableOID and TableAttributeNumber are both non-zero, this is a table column
	if field.TableOID != 0 && field.TableAttributeNumber != 0 {
		// Query pg_attribute to check if column is NOT NULL
		var notNull bool
		err := qa.db.QueryRow(ctx, `
			SELECT attnotnull
			FROM pg_attribute
			WHERE attrelid = $1 AND attnum = $2
		`, field.TableOID, field.TableAttributeNumber).Scan(&notNull)

		if err != nil {
			return true, fmt.Errorf("failed to query pg_attribute: %w", err)
		}

		return !notNull, nil
	}

	// Computed column (TableOID == 0 or TableAttributeNumber == 0)
	// Check if it's a COUNT aggregate which is never nullable
	if qa.isCountAggregate(field.Name, sql) {
		return false, nil
	}

	// Other computed columns default to nullable
	return true, nil
}

// isCountAggregate checks if a column is a COUNT aggregate
func (qa *QueryAnalyzer) isCountAggregate(columnName, sql string) bool {
	// Parse the SQL to check if this column is a COUNT aggregate
	queryInfo, err := qa.sqlParser.Parse(sql)
	if err != nil {
		return false
	}

	// Check if this column name matches a COUNT target
	for _, target := range queryInfo.SelectTargets {
		if strings.EqualFold(target.Alias, columnName) && target.IsCount {
			return true
		}
	}

	return false
}

// mapToIntelligentGoType maps PostgreSQL type to Go type with intelligent nullability
func (qa *QueryAnalyzer) mapToIntelligentGoType(pgType string, isNullable bool) (string, error) {
	// Get base type
	var baseType string
	switch strings.ToLower(pgType) {
	case "uuid":
		baseType = "uuid.UUID"
	case "text", "varchar", "character varying", "char", "character":
		baseType = "string"
	case "smallint", "int2", "integer", "int", "int4", "bigint", "int8":
		baseType = "int"
	case "real", "float4":
		baseType = "float32"
	case "double precision", "float8", "numeric", "decimal":
		baseType = "float64"
	case "boolean", "bool":
		baseType = "bool"
	case "date", "time", "time without time zone", "timetz", "time with time zone",
		"timestamp", "timestamp without time zone", "timestamptz", "timestamp with time zone":
		baseType = "time.Time"
	case "bytea":
		baseType = "[]byte"
	case "json", "jsonb":
		baseType = "json.RawMessage"
	default:
		return "", fmt.Errorf("unsupported PostgreSQL type: %s", pgType)
	}

	// Apply nullable modifier
	if isNullable {
		return makePointerType(baseType), nil
	}

	return baseType, nil
}

// mapOIDToTypeName maps PostgreSQL OID to type name
func (qa *QueryAnalyzer) mapOIDToTypeName(oid uint32) string {
	// Common PostgreSQL type OIDs
	// This is a simplified mapping - in a production system, you'd want a more comprehensive mapping
	switch oid {
	case 16:
		return "boolean"
	case 20:
		return "bigint"
	case 21:
		return "smallint"
	case 23:
		return "integer"
	case 25:
		return "text"
	case 700:
		return "real"
	case 701:
		return "double precision"
	case 1043:
		return "varchar"
	case 1082:
		return "date"
	case 1114:
		return "timestamp"
	case 1184:
		return "timestamptz"
	case 1700:
		return "numeric"
	case 2950:
		return "uuid"
	case 114:
		return "json"
	case 3802:
		return "jsonb"
	case 17:
		return "bytea"
	default:
		return "unknown" // Return unknown for unrecognized OIDs
	}
}

// validateQuerySyntax validates that the query is syntactically correct
func (qa *QueryAnalyzer) validateQuerySyntax(ctx context.Context, query *Query) error {
	// For SELECT queries, we already validated them in analyzeQueryColumns
	// For EXEC queries, we validate via PREPARE in inferParameterTypesFromPrepare
	return nil
}

// inferParameterTypesFromPrepare uses PostgreSQL PREPARE to infer parameter types for all query types
func (qa *QueryAnalyzer) inferParameterTypesFromPrepare(ctx context.Context, query *Query) error {
	// Skip if no parameters
	if len(query.Parameters) == 0 {
		return nil
	}

	// Try to prepare the statement to infer types
	// We'll use a transaction that we roll back to avoid side effects
	tx, err := qa.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction for type inference: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare the statement with a unique name
	stmtName := fmt.Sprintf("infer_types_%s", query.Name)
	stmt, err := tx.Prepare(ctx, stmtName, query.SQL)
	if err != nil {
		return fmt.Errorf("query preparation failed: %w", err)
	}

	// Check that the parameter count matches
	if len(stmt.ParamOIDs) != len(query.Parameters) {
		return fmt.Errorf("parameter count mismatch: query expects %d parameters, found %d", len(stmt.ParamOIDs), len(query.Parameters))
	}

	// Update parameter types based on the prepared statement
	for i, paramOID := range stmt.ParamOIDs {
		if i < len(query.Parameters) {
			pgType := qa.mapOIDToTypeName(paramOID)
			goType, err := qa.typeMapper.MapType(pgType, false, false)
			if err != nil {
				return fmt.Errorf("failed to map parameter type: %w", err)
			}

			// Use int instead of int64 for limit/offset parameters (Go idiomatic)
			paramName := query.Parameters[i].Name
			if (paramName == "limit" || paramName == "offset") && goType == "int64" {
				goType = "int"
			}

			query.Parameters[i].Type = pgType
			query.Parameters[i].GoType = goType
		}
	}

	return nil
}

// InferParameterTypes attempts to infer parameter types from query context
func (qa *QueryAnalyzer) InferParameterTypes(ctx context.Context, query *Query) error {
	// This is a more advanced feature that could analyze the query context
	// to infer parameter types based on how they're used
	// For now, we'll keep the basic implementation from extractParameters
	return nil
}

// ValidateQueryExecution validates that a query can be executed successfully
func (qa *QueryAnalyzer) ValidateQueryExecution(ctx context.Context, query *Query) error {
	// This could be used to validate that the query executes without errors
	// using test data or in a test transaction
	return nil
}
