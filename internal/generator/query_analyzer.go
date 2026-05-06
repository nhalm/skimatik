package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nhalm/pgxkit/v2"
)

// QueryAnalyzer introspects SQL queries using PostgreSQL's PREPARE and EXPLAIN features
// to determine result column types, parameter types, and nullability.
// It generates intelligent types that match TypeMapper conventions (int for all integers, pointers for nullability).
type QueryAnalyzer struct {
	db         *pgxkit.DB
	schema     string
	typeMapper *TypeMapper
	sqlParser  *SQLParser
}

// NewQueryAnalyzer creates a new query analyzer with the given database connection and schema.
// The database is used to introspect query structure and validate syntax.
func NewQueryAnalyzer(db *pgxkit.DB, schema string) *QueryAnalyzer {
	return &QueryAnalyzer{
		db:         db,
		schema:     schema,
		typeMapper: NewTypeMapper(nil),
		sqlParser:  NewSQLParser(),
	}
}

// AnalyzeQuery performs complete analysis of a SQL query including:
// - Parameter extraction and naming
// - Result column type detection (for SELECT queries)
// - Parameter type inference via PREPARE
// - Nullability detection from schema and query structure
// - Query syntax validation
func (qa *QueryAnalyzer) AnalyzeQuery(ctx context.Context, query *query) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}

	// Extract parameters from the query (doesn't require database connection)
	if err := qa.extractParameters(query); err != nil {
		return fmt.Errorf("failed to extract parameters: %w", err)
	}

	// Infer parameter names and track table/column associations (doesn't require database connection)
	qa.inferParameterNames(query)

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
	qa.inferParameterNullability(ctx, query)

	// Validate query syntax by attempting to prepare it
	if err := qa.validateQuerySyntax(ctx, query); err != nil {
		return fmt.Errorf("query syntax validation failed: %w", err)
	}

	// For :paginated queries, extract ORDER BY columns
	if query.Type == queryTypePaginated {
		if err := qa.extractOrderByColumns(query); err != nil {
			return fmt.Errorf("failed to extract ORDER BY columns: %w", err)
		}
	}

	return nil
}

// extractOrderByColumns extracts ORDER BY columns and resolves their types
func (qa *QueryAnalyzer) extractOrderByColumns(query *query) error {
	// Extract ORDER BY from SQL
	orderByCols, err := qa.sqlParser.extractOrderBy(query.SQL)
	if err != nil {
		return fmt.Errorf("failed to parse ORDER BY: %w", err)
	}

	// No ORDER BY is valid - user takes responsibility for pagination behavior
	if len(orderByCols) == 0 {
		query.OrderByColumns = nil
		return nil
	}

	// Resolve types from Query.Columns
	for i := range orderByCols {
		colName := orderByCols[i].Name
		found := false

		for _, qCol := range query.Columns {
			if strings.EqualFold(qCol.Name, colName) {
				orderByCols[i].GoType = qCol.GoType
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("ORDER BY column %q not found in SELECT list for query %s", colName, query.Name)
		}
	}

	query.OrderByColumns = orderByCols
	return nil
}

// applyResultAnnotations applies result type annotations to override automatically detected types
func (qa *QueryAnalyzer) applyResultAnnotations(query *query) error {
	if len(query.ResultAnnotations) == 0 {
		return nil
	}

	// Check if query.Columns is nil to avoid panic
	if query.Columns == nil {
		return fmt.Errorf("query %s has result annotations but no columns were detected", query.Name)
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
func (qa *QueryAnalyzer) extractParameters(query *query) error {
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
		query.Parameters = []parameter{}
		return nil
	}

	// Create parameter list from parse tree
	paramMap := make(map[int]bool)
	for _, paramInfo := range queryInfo.Parameters {
		paramMap[paramInfo.Position] = true
	}

	// Create parameter list from the parameters found
	var parameters []parameter
	for paramNum := range paramMap {
		param := parameter{
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
func (qa *QueryAnalyzer) extractParametersRegex(query *query) error {
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
		query.Parameters = []parameter{}
		return nil
	}

	// Create parameter list
	var parameters []parameter
	for paramNum := range paramMap {
		param := parameter{
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

// inferParameterNames infers semantic parameter names from SQL context using the SQL parser.
// SQL parser failures are tolerated — parameters keep their default "paramN" names — so the
// function never returns an error.
func (qa *QueryAnalyzer) inferParameterNames(query *query) {
	if len(query.Parameters) == 0 {
		return
	}

	// Parse the SQL to get parameter context
	queryInfo, err := qa.sqlParser.Parse(query.SQL)
	if err != nil {
		// If parsing fails, leave parameter names as default "paramN"
		return
	}

	// Create maps to track inferred data by parameter index
	inferredNames := make(map[int]string)
	inferredColumns := make(map[int]string)
	inferredTables := make(map[int]string)

	// Extract information from parsed parameters
	for _, paramInfo := range queryInfo.Parameters {
		qa.collectParameterInference(paramInfo, inferredNames, inferredColumns, inferredTables)
	}

	// Apply inferred names and table/column associations to parameters
	qa.applyInferredParameterInfo(query, inferredNames, inferredColumns, inferredTables)
}

// collectParameterInference inspects a single ParameterInfo and records the
// inferred name / column / table for its position in the supplied maps.
func (qa *QueryAnalyzer) collectParameterInference(
	paramInfo ParameterInfo,
	inferredNames, inferredColumns, inferredTables map[int]string,
) {
	pos := paramInfo.Position

	switch {
	case paramInfo.IsInLimit:
		inferredNames[pos] = "limit"
	case paramInfo.IsInOffset:
		inferredNames[pos] = "offset"
	case paramInfo.ColumnName != "":
		qa.recordColumnParameter(paramInfo, inferredNames, inferredColumns, inferredTables)
	}
}

// recordColumnParameter records inferred name/table/column for a parameter
// that references a column. The naming rule depends on the operator (LIKE
// patterns get a "search" prefix to avoid collisions with equality params).
func (qa *QueryAnalyzer) recordColumnParameter(
	paramInfo ParameterInfo,
	inferredNames, inferredColumns, inferredTables map[int]string,
) {
	pos := paramInfo.Position

	if isLikeOperator(paramInfo.Operator) {
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

// isLikeOperator reports whether op is one of the LIKE / ILIKE operators.
// Both the textual ("LIKE", "ILIKE") and parser-internal ("~~", "~~*") forms
// are recognized.
func isLikeOperator(op string) bool {
	if op == "~~" || op == "~~*" {
		return true
	}
	return strings.EqualFold(op, "LIKE") || strings.EqualFold(op, "ILIKE")
}

// applyInferredParameterInfo writes inferred names, columns and tables onto
// query.Parameters, keying by Parameter.Index.
func (qa *QueryAnalyzer) applyInferredParameterInfo(
	query *query,
	inferredNames, inferredColumns, inferredTables map[int]string,
) {
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
		if parts[i] != "" {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return result
}

// inferParameterNullability looks up column nullability from the database schema.
// Failures to look up an individual column are tolerated (the parameter may be
// an alias or an expression with no schema column) — the function never returns
// an error.
func (qa *QueryAnalyzer) inferParameterNullability(ctx context.Context, query *query) {
	for i := range query.Parameters {
		param := &query.Parameters[i]

		// Only look up nullability if we have both table and column name
		if param.TableName == "" || param.ColumnName == "" {
			continue
		}

		// Query information_schema to check if column is nullable
		schemaQuery := `SELECT is_nullable FROM information_schema.columns
		                WHERE table_schema = $1
		                AND table_name = $2
		                AND column_name = $3`

		var isNullable string
		err := qa.db.QueryRow(ctx, schemaQuery, qa.schema, param.TableName, param.ColumnName).Scan(&isNullable)
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
		switch {
		case result[i] == '\'' && (i == 0 || result[i-1] != '\\'):
			i = qa.handleSingleQuote(result, i, &inSingleQuote, inDoubleQuote)
		case result[i] == '"' && (i == 0 || result[i-1] != '\\'):
			i = qa.handleDoubleQuote(result, i, &inDoubleQuote, inSingleQuote)
		case inSingleQuote || inDoubleQuote:
			result[i] = ' '
		case result[i] == '-' && i+1 < len(result) && result[i+1] == '-':
			i = qa.blankLineComment(result, i)
		}
	}

	return string(result)
}

// handleSingleQuote processes a single-quote character at position i in result.
// It updates *inSingleQuote, optionally consumes a paired escape quote, blanks
// the appropriate position, and returns the (possibly advanced) index.
//
// The blanking rule preserves the original behavior:
//   - When entering a string (inSingleQuote becomes true), the quote is blanked.
//   - When leaving a string (inSingleQuote becomes false), the quote is only
//     blanked if i > 0. (The original code always set the closing quote to a
//     space when i > 0; when i == 0 nothing is blanked because there's no
//     string content to remove.)
func (qa *QueryAnalyzer) handleSingleQuote(result []rune, i int, inSingleQuote *bool, inDoubleQuote bool) int {
	if *inSingleQuote {
		// Check for escaped quote ''
		if i+1 < len(result) && result[i+1] == '\'' {
			result[i] = ' '
			result[i+1] = ' '
			i++
		} else {
			*inSingleQuote = false
		}
	} else if !inDoubleQuote {
		*inSingleQuote = true
	}
	if *inSingleQuote || (!*inSingleQuote && i > 0) {
		result[i] = ' '
	}
	return i
}

// handleDoubleQuote is the double-quote analogue of handleSingleQuote.
func (qa *QueryAnalyzer) handleDoubleQuote(result []rune, i int, inDoubleQuote *bool, inSingleQuote bool) int {
	if *inDoubleQuote {
		// Check for escaped quote ""
		if i+1 < len(result) && result[i+1] == '"' {
			result[i] = ' '
			result[i+1] = ' '
			i++
		} else {
			*inDoubleQuote = false
		}
	} else if !inSingleQuote {
		*inDoubleQuote = true
	}
	if *inDoubleQuote || (!*inDoubleQuote && i > 0) {
		result[i] = ' '
	}
	return i
}

// blankLineComment blanks out a `-- ...` single-line SQL comment starting at
// index i. It returns the index of the line terminator (or len(result)-1 if
// there isn't one); the caller's loop will advance past it on the next iteration.
func (qa *QueryAnalyzer) blankLineComment(result []rune, i int) int {
	for i < len(result) && result[i] != '\n' && result[i] != '\r' {
		result[i] = ' '
		i++
	}
	return i
}

// isSelectQuery checks if the query type requires column analysis
func (qa *QueryAnalyzer) isSelectQuery(queryType queryType) bool {
	return queryType == queryTypeOne || queryType == queryTypeMany || queryType == queryTypePaginated
}

// analyzeSelectQuery analyzes a SELECT query and determines column types
func (qa *QueryAnalyzer) analyzeSelectQuery(ctx context.Context, query *query) error {
	// Analyze query columns by executing with NULL parameters
	// This approach works for all queries including those with parameters in HAVING, WHERE, etc.
	return qa.analyzeQueryColumns(ctx, query)
}

// analyzeQueryColumns analyzes the columns returned by a SELECT query.
//
// The metadata is sourced from a PREPARE inside a rolled-back transaction
// rather than from executing the query with NULL placeholder args, because
// pgx v5.9+ surfaces bind/execute errors at rows.Next() rather than at
// Query() — leaving FieldDescriptions empty for any query whose dummy NULL
// args would violate NOT NULL or other constraints (notably DML CTEs that
// INSERT/UPDATE on real columns). Prepare returns the result-column
// descriptions without executing, so it works for SELECT and DML-CTE shapes
// alike and is independent of pgx error-timing changes.
func (qa *QueryAnalyzer) analyzeQueryColumns(ctx context.Context, query *query) error {
	// Remove trailing semicolon if present
	sql := strings.TrimSpace(query.SQL)
	sql = strings.TrimSuffix(sql, ";")

	tx, err := qa.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction for column analysis: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("failed to rollback transaction during column analysis",
				"query", query.Name,
				"error", err)
		}
	}()

	stmtName := fmt.Sprintf("analyze_columns_%s", query.Name)
	stmt, err := tx.Tx().Prepare(ctx, stmtName, sql)
	if err != nil {
		return fmt.Errorf("failed to prepare query for column analysis: %w", err)
	}

	var columns []column

	for _, field := range stmt.Fields {
		// Detect if this is an array type
		isArray := qa.isArrayOID(field.DataTypeOID)

		// Map PostgreSQL OID to base type name
		pgType := qa.mapOIDToTypeName(field.DataTypeOID)

		// Determine if the column is nullable using intelligent detection
		isNullable, err := qa.isColumnNullable(ctx, field, query.SQL)
		if err != nil {
			return fmt.Errorf("failed to determine nullability for %s: %w", field.Name, err)
		}

		// Map to Go type using TypeMapper which handles arrays and nullability
		goType, err := qa.typeMapper.MapType(pgType, isNullable, isArray)
		if err != nil {
			return fmt.Errorf("failed to map column type for %s (pgType=%s, nullable=%v, array=%v): %w", field.Name, pgType, isNullable, isArray, err)
		}

		if goType == "" {
			return fmt.Errorf("empty GoType generated for column %s (pgType=%s, nullable=%v, array=%v)", field.Name, pgType, isNullable, isArray)
		}

		column := column{
			Name:       field.Name,
			Type:       pgType,
			GoType:     goType,
			IsNullable: isNullable,
			IsArray:    isArray,
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
		return qa.isTableColumnNullable(ctx, field, sql)
	}
	return qa.isComputedColumnNullable(field, sql)
}

// isTableColumnNullable determines whether a column backed by a real table
// (FieldDescription.TableOID and TableAttributeNumber are both non-zero)
// is nullable. It consults pg_attribute for the NOT NULL constraint and then
// checks whether an outer JOIN promotes the column to nullable.
func (qa *QueryAnalyzer) isTableColumnNullable(ctx context.Context, field pgconn.FieldDescription, sql string) (bool, error) {
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

	// Get the table name from OID to check JOIN rules
	var tableName string
	err = qa.db.QueryRow(ctx, `
		SELECT relname FROM pg_class WHERE oid = $1
	`, field.TableOID).Scan(&tableName)
	if err != nil {
		return true, fmt.Errorf("failed to query pg_class: %w", err)
	}

	// Check if this table is on the nullable side of an outer join
	if qa.isTableNullableFromJoin(tableName, sql) {
		return true, nil // Outer join makes column nullable
	}

	return !notNull, nil
}

// isComputedColumnNullable determines whether a computed column (one whose
// FieldDescription has no TableOID/TableAttributeNumber) is nullable. It
// inspects the parsed SELECT for non-null-guaranteeing expressions
// (COUNT, ranking functions, COALESCE/CASE with non-null literals) and
// recursively checks subqueries and CTEs.
func (qa *QueryAnalyzer) isComputedColumnNullable(field pgconn.FieldDescription, sql string) (bool, error) {
	queryInfo, err := qa.sqlParser.Parse(sql)
	if err != nil {
		// If parsing fails, default to nullable
		return true, nil
	}

	// Find the target matching this column name in the top-level SELECT list.
	for _, target := range queryInfo.SelectTargets {
		if !strings.EqualFold(target.Alias, field.Name) {
			continue
		}
		if guaranteedNonNull(target) {
			return false, nil
		}
	}

	// Check if this column comes from a subquery or CTE
	columnName := field.Name

	if nullable, found := qa.findColumnInSubqueries(queryInfo.Tables, columnName); found {
		return nullable, nil
	}

	if nullable, found := qa.findColumnInCTEs(queryInfo.CTEs, columnName); found {
		return nullable, nil
	}

	// Other computed columns default to nullable
	return true, nil
}

// guaranteedNonNull reports whether a SelectTarget's expression form
// guarantees a non-null result.
func guaranteedNonNull(target SelectTarget) bool {
	if target.IsCount {
		return true // COUNT never returns NULL
	}
	if target.IsRowNumber || target.IsRank || target.IsDenseRank {
		return true // Window ranking functions never return NULL
	}
	if target.IsCoalesce && target.HasNonNullLiteral {
		return true // COALESCE with non-null literal guarantees non-null
	}
	if target.IsCaseWithElse && target.HasNonNullLiteral {
		return true // CASE with non-null ELSE literal guarantees non-null
	}
	return false
}

// findColumnInSubqueries searches subquery tables for a SelectTarget matching
// columnName. It returns (nullable, true) on the first match, or (_, false)
// when none of the subqueries expose that column.
func (qa *QueryAnalyzer) findColumnInSubqueries(tables []TableRef, columnName string) (nullable, found bool) {
	for _, table := range tables {
		if !table.IsSubquery || table.SubqueryInfo == nil {
			continue
		}
		if n, ok := qa.checkTargetNullability(table.SubqueryInfo.SelectTargets, columnName); ok {
			return n, true
		}
	}
	return true, false
}

// findColumnInCTEs searches CTE definitions for a SelectTarget matching
// columnName. It returns (nullable, true) on the first match, or (_, false)
// when none of the CTEs expose that column.
func (qa *QueryAnalyzer) findColumnInCTEs(ctes []CTEInfo, columnName string) (nullable, found bool) {
	for _, cte := range ctes {
		if cte.Query == nil {
			continue
		}
		if n, ok := qa.checkTargetNullability(cte.Query.SelectTargets, columnName); ok {
			return n, true
		}
	}
	return true, false
}

// checkTargetNullability checks if a SelectTarget guarantees non-null based on expression type
func (qa *QueryAnalyzer) checkTargetNullability(targets []SelectTarget, columnName string) (nullable, found bool) {
	for _, target := range targets {
		if strings.EqualFold(target.Alias, columnName) {
			if target.IsCount || target.IsRowNumber || target.IsRank || target.IsDenseRank {
				return false, true // These functions never return NULL
			}
			if (target.IsCoalesce || target.IsCaseWithElse) && target.HasNonNullLiteral {
				return false, true // Non-null literal guarantees non-null
			}
			// Found the target but it's nullable
			return true, true
		}
	}
	return true, false // Not found
}

// isTableNullableFromJoin checks if a table is on the nullable side of an outer join
func (qa *QueryAnalyzer) isTableNullableFromJoin(tableName, sql string) bool {
	queryInfo, err := qa.sqlParser.Parse(sql)
	if err != nil {
		return false
	}

	// JOINs use aliases, so resolve the table's alias (if any) up front.
	tableIdentifier := resolveTableAlias(queryInfo.Tables, tableName)

	// Check each JOIN to see if this table is on the nullable side.
	for _, join := range queryInfo.Joins {
		if joinMakesTableNullable(join, tableName, tableIdentifier) {
			return true
		}
	}

	return false
}

// resolveTableAlias returns the alias used for tableName in the given table
// list, or tableName itself when no alias is defined. JOIN clauses reference
// the alias when present, so this is what we compare against join.LeftTable /
// join.RightTable.
func resolveTableAlias(tables []TableRef, tableName string) string {
	for _, table := range tables {
		if table.Name == tableName && table.Alias != "" {
			return table.Alias
		}
	}
	return tableName
}

// joinMakesTableNullable reports whether the given join promotes the named
// table (matched by either alias or raw name) to nullable. The rules:
//   - LEFT JOIN: right table becomes nullable.
//   - RIGHT JOIN: left table becomes nullable.
//   - FULL OUTER JOIN: both tables become nullable.
//   - INNER JOIN: neither table becomes nullable.
func joinMakesTableNullable(join JoinInfo, tableName, tableIdentifier string) bool {
	matchesLeft := join.LeftTable == tableIdentifier || join.LeftTable == tableName
	matchesRight := join.RightTable == tableIdentifier || join.RightTable == tableName

	switch join.Type {
	case JoinTypeLeft:
		return matchesRight
	case JoinTypeRight:
		return matchesLeft
	case JoinTypeFull:
		return matchesLeft || matchesRight
	}
	return false
}

// mapOIDToTypeName maps PostgreSQL OID to type name
// Handles both base types and array types, returning the base type name
func (qa *QueryAnalyzer) mapOIDToTypeName(oid uint32) string {
	if name := mapNumericOID(oid); name != "" {
		return name
	}
	if name := mapStringOID(oid); name != "" {
		return name
	}
	if name := mapTimeOID(oid); name != "" {
		return name
	}
	if name := mapMiscOID(oid); name != "" {
		return name
	}
	return "unknown"
}

// mapNumericOID maps numeric / boolean / bytea PostgreSQL OIDs to type names.
func mapNumericOID(oid uint32) string {
	switch oid {
	case 16, 1000: // boolean, boolean[]
		return "boolean"
	case 17, 1001: // bytea, bytea[]
		return "bytea"
	case 21, 1005: // smallint, smallint[]
		return "smallint"
	case 23, 1007: // integer, integer[]
		return "integer"
	case 20, 1016: // bigint, bigint[]
		return "bigint"
	case 700, 1021: // real, real[]
		return "real"
	case 701, 1022: // double precision, double precision[]
		return "double precision"
	case 1700, 1231: // numeric, numeric[]
		return "numeric"
	}
	return ""
}

// mapStringOID maps string-like PostgreSQL OIDs to type names.
func mapStringOID(oid uint32) string {
	switch oid {
	case 25, 1009: // text, text[]
		return "text"
	case 1042, 1014: // char, char[]
		return "char"
	case 1043, 1015: // varchar, varchar[]
		return "varchar"
	}
	return ""
}

// mapTimeOID maps date/time PostgreSQL OIDs to type names.
func mapTimeOID(oid uint32) string {
	switch oid {
	case 1082, 1182: // date, date[]
		return "date"
	case 1114, 1183: // timestamp, timestamp[]
		return "timestamp"
	case 1184, 1185: // timestamptz, timestamptz[]
		return "timestamptz"
	}
	return ""
}

// mapMiscOID maps miscellaneous PostgreSQL OIDs (uuid, json, jsonb) to type names.
func mapMiscOID(oid uint32) string {
	switch oid {
	case 2950, 2951: // uuid, uuid[]
		return "uuid"
	case 114, 199: // json, json[]
		return "json"
	case 3802, 3807: // jsonb, jsonb[]
		return "jsonb"
	}
	return ""
}

// isArrayOID checks if a PostgreSQL OID represents an array type
func (qa *QueryAnalyzer) isArrayOID(oid uint32) bool {
	switch oid {
	case 199, 1000, 1001, 1005, 1007, 1009, 1014, 1015, 1016, 1021, 1022,
		1182, 1183, 1185, 1231, 2951, 3807:
		return true
	default:
		return false
	}
}

// validateQuerySyntax validates that the query is syntactically correct
func (qa *QueryAnalyzer) validateQuerySyntax(ctx context.Context, query *query) error {
	// For SELECT queries, we already validated them in analyzeQueryColumns
	// For EXEC queries, we validate via PREPARE in inferParameterTypesFromPrepare
	return nil
}

// inferParameterTypesFromPrepare uses PostgreSQL PREPARE to infer parameter types for all query types.
// It creates a temporary prepared statement and extracts parameter OIDs, then maps them to Go types.
func (qa *QueryAnalyzer) inferParameterTypesFromPrepare(ctx context.Context, query *query) error {
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
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("failed to rollback transaction during type inference",
				"query", query.Name,
				"error", err)
		}
	}()

	// Prepare the statement with a unique name
	stmtName := fmt.Sprintf("infer_types_%s", query.Name)
	stmt, err := tx.Tx().Prepare(ctx, stmtName, query.SQL)
	if err != nil {
		return fmt.Errorf("query preparation failed: %w", err)
	}

	// Check that the parameter count matches
	if len(stmt.ParamOIDs) != len(query.Parameters) {
		return fmt.Errorf("parameter count mismatch: query expects %d parameters, found %d", len(stmt.ParamOIDs), len(query.Parameters))
	}

	// Update parameter types based on the prepared statement
	for i, paramOID := range stmt.ParamOIDs {
		if i >= len(query.Parameters) {
			continue
		}
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

	return nil
}
