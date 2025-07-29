package generator

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/nhalm/pgxkit"
)

// QueryAnalyzer analyzes SQL queries using PostgreSQL EXPLAIN to determine column types and validate queries
type QueryAnalyzer struct {
	db         *pgxkit.DB
	typeMapper *TypeMapper
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(db *pgxkit.DB) *QueryAnalyzer {
	return &QueryAnalyzer{
		db:         db,
		typeMapper: NewTypeMapper(nil),
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
	}

	// Validate query syntax by attempting to prepare it
	if err := qa.validateQuerySyntax(ctx, query); err != nil {
		return fmt.Errorf("query syntax validation failed: %w", err)
	}

	return nil
}

// extractParameters extracts parameter placeholders from the SQL query
func (qa *QueryAnalyzer) extractParameters(query *Query) error {
	// Remove string literals and quoted identifiers to avoid false positives
	cleanSQL := qa.removeQuotedContent(query.SQL)

	// Find all parameter placeholders ($1, $2, etc.)
	// Match $digits followed by non-digit or end of string
	paramRegex := regexp.MustCompile(`\$(\d+)(?:\D|$)`)
	matches := paramRegex.FindAllStringSubmatch(cleanSQL, -1)

	if len(matches) == 0 {
		query.Parameters = []Parameter{}
		return nil
	}

	// Create a map to track unique parameter indices
	paramMap := make(map[int]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			paramNum, err := strconv.Atoi(match[1])
			if err != nil {
				return fmt.Errorf("invalid parameter number: %s", match[1])
			}
			paramMap[paramNum] = true
		}
	}

	// Create parameter list from the parameters found
	var parameters []Parameter
	for paramNum := range paramMap {
		// For now, we'll use a generic parameter type
		// In a more advanced implementation, we could try to infer types from context
		param := Parameter{
			Name:   fmt.Sprintf("param%d", paramNum),
			Type:   "text", // Default to text, can be overridden by type inference
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

// removeQuotedContent removes string literals and quoted identifiers to avoid false parameter detection
func (qa *QueryAnalyzer) removeQuotedContent(sql string) string {
	// Remove single-quoted string literals
	singleQuoteRegex := regexp.MustCompile(`'(?:[^']|'')*'`)
	result := singleQuoteRegex.ReplaceAllString(sql, "''")

	// Remove double-quoted identifiers
	doubleQuoteRegex := regexp.MustCompile(`"(?:[^"]|"")*"`)
	result = doubleQuoteRegex.ReplaceAllString(result, `""`)

	// Remove single-line comments (-- comments)
	commentRegex := regexp.MustCompile(`--[^\r\n]*`)
	result = commentRegex.ReplaceAllString(result, "")

	return result
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
	// Use regex to find parameter placeholders that are not inside single quotes
	// This is a simplified approach - a full SQL parser would be more robust

	// Pattern to match the placeholder when not inside single quotes
	// This uses negative lookbehind and lookahead to avoid quoted content
	pattern := fmt.Sprintf(`(?:'[^']*'|%s)`, regexp.QuoteMeta(placeholder))

	re := regexp.MustCompile(pattern)
	result := re.ReplaceAllStringFunc(sql, func(match string) string {
		if match == placeholder {
			return replacement
		}
		return match // Keep quoted content unchanged
	})

	return result
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

		// Determine if the column is nullable (this is a simplified approach)
		isNullable := true // Default to nullable for query results

		// Map to Go type
		goType, err := qa.typeMapper.MapType(pgType, isNullable, false)
		if err != nil {
			return fmt.Errorf("failed to map column type for %s: %w", field.Name, err)
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
	// For exec queries, we can't use LIMIT 0, so we'll use a different approach
	if query.Type == QueryTypeExec {
		return qa.validateExecQuery(ctx, query)
	}

	// For SELECT queries, we already validated them in analyzeQueryColumns
	return nil
}

// validateExecQuery validates an EXEC query by preparing it
func (qa *QueryAnalyzer) validateExecQuery(ctx context.Context, query *Query) error {
	// Try to prepare the statement to validate syntax
	// We'll use a transaction that we roll back to avoid side effects
	tx, err := qa.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction for validation: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare the statement with a unique name
	stmtName := fmt.Sprintf("validate_query_%s", query.Name)
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
