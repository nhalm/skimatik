package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// QueryParser handles parsing SQL files with sqlc-style annotations
type QueryParser struct {
	dir string
}

// NewQueryParser creates a new query parser for the given directory
func NewQueryParser(dir string) *QueryParser {
	return &QueryParser{dir: dir}
}

// ParseQueries parses all SQL files in the directory and returns Query objects
func (qp *QueryParser) ParseQueries() ([]Query, error) {
	if qp.dir == "" {
		return nil, fmt.Errorf("queries directory not specified")
	}

	// Check if directory exists
	if _, err := os.Stat(qp.dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("queries directory does not exist: %s", qp.dir)
	}

	// Find all SQL files
	sqlFiles, err := qp.findSQLFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find SQL files: %w", err)
	}

	if len(sqlFiles) == 0 {
		return nil, fmt.Errorf("no SQL files found in directory: %s", qp.dir)
	}

	// Parse each SQL file
	var allQueries []Query
	for _, sqlFile := range sqlFiles {
		queries, err := qp.parseFile(sqlFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", sqlFile, err)
		}
		allQueries = append(allQueries, queries...)
	}

	return allQueries, nil
}

// findSQLFiles finds all .sql files in the directory
func (qp *QueryParser) findSQLFiles() ([]string, error) {
	var sqlFiles []string

	err := filepath.Walk(qp.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".sql") {
			sqlFiles = append(sqlFiles, path)
		}

		return nil
	})

	return sqlFiles, err
}

// parseFile parses a single SQL file and extracts queries with annotations
func (qp *QueryParser) parseFile(filename string) ([]Query, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var queries []Query
	var currentQuery *Query
	var sqlLines []string
	var paramAnnotations []ParameterAnnotation
	var resultAnnotations []ResultAnnotation
	var cursorColumns []string

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check for query annotation
		if annotation := qp.parseAnnotation(trimmedLine); annotation != nil {
			// Save previous query if exists
			if currentQuery != nil {
				currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, "\n"))
				if currentQuery.SQL == "" {
					return nil, fmt.Errorf("empty query for %s at line %d in %s", currentQuery.Name, lineNum, filename)
				}
				currentQuery.ParameterAnnotations = paramAnnotations
				currentQuery.ResultAnnotations = resultAnnotations
				currentQuery.CursorColumns = cursorColumns
				if err := qp.validateParameterAnnotations(currentQuery); err != nil {
					return nil, fmt.Errorf("error in query %s: %w", currentQuery.Name, err)
				}
				if err := qp.validateResultAnnotations(currentQuery); err != nil {
					return nil, fmt.Errorf("error in query %s: %w", currentQuery.Name, err)
				}
				if err := qp.validateCursorColumns(currentQuery); err != nil {
					return nil, fmt.Errorf("error in query %s: %w", currentQuery.Name, err)
				}
				queries = append(queries, *currentQuery)
			}

			// Start new query
			currentQuery = &Query{
				Name:       annotation.Name,
				Type:       annotation.Type,
				SourceFile: filename,
				Parameters: []Parameter{}, // Will be populated by analyzer
				Columns:    []Column{},    // Will be populated by analyzer
			}
			sqlLines = []string{}
			paramAnnotations = []ParameterAnnotation{}
			resultAnnotations = []ResultAnnotation{}
			cursorColumns = []string{}
			continue
		}

		// Check for parameter annotation
		if paramAnnotation := qp.parseParameterAnnotation(trimmedLine); paramAnnotation != nil {
			if currentQuery != nil {
			cursorColumns = []string{}
				paramAnnotations = append(paramAnnotations, *paramAnnotation)
			}
			continue
		}

		// Check for result annotation
		if resultAnnotation := qp.parseResultAnnotation(trimmedLine); resultAnnotation != nil {
			if currentQuery != nil {
				resultAnnotations = append(resultAnnotations, *resultAnnotation)
			}
			continue
		}

		// Check for cursor_columns annotation
		if cursorCols := qp.parseCursorColumnsAnnotation(trimmedLine); cursorCols != nil {
			if currentQuery != nil {
				cursorColumns = cursorCols
			}
			continue
		}

		// Skip empty lines and comments (except annotations)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "--") {
			continue
		}

		// Collect SQL lines for current query
		if currentQuery != nil {
			sqlLines = append(sqlLines, line)
		}
	}

	// Save the last query
	if currentQuery != nil {
		currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, "\n"))
		if currentQuery.SQL == "" {
			return nil, fmt.Errorf("empty query for %s in %s", currentQuery.Name, filename)
		}
		currentQuery.ParameterAnnotations = paramAnnotations
		currentQuery.ResultAnnotations = resultAnnotations
		currentQuery.CursorColumns = cursorColumns
		if err := qp.validateParameterAnnotations(currentQuery); err != nil {
			return nil, fmt.Errorf("error in query %s: %w", currentQuery.Name, err)
		}
		if err := qp.validateResultAnnotations(currentQuery); err != nil {
			return nil, fmt.Errorf("error in query %s: %w", currentQuery.Name, err)
		}
		if err := qp.validateCursorColumns(currentQuery); err != nil {
			return nil, fmt.Errorf("error in query %s: %w", currentQuery.Name, err)
		}
		queries = append(queries, *currentQuery)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return queries, nil
}

// QueryAnnotation represents a parsed sqlc-style annotation
type QueryAnnotation struct {
	Name string
	Type QueryType
}

// parseAnnotation parses a sqlc-style annotation line
// Expected format: -- name: QueryName :type
func (qp *QueryParser) parseAnnotation(line string) *QueryAnnotation {
	// Regex to match: -- name: QueryName :type
	// Allow for flexible whitespace and optional semicolon
	annotationRegex := regexp.MustCompile(`^--\s*name:\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:([a-zA-Z]+)\s*;?\s*$`)

	matches := annotationRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil
	}

	queryName := strings.TrimSpace(matches[1])
	queryTypeStr := strings.TrimSpace(matches[2])

	// Parse query type
	queryType, err := qp.parseQueryType(queryTypeStr)
	if err != nil {
		return nil // Invalid query type, skip this annotation
	}

	return &QueryAnnotation{
		Name: queryName,
		Type: queryType,
	}
}

// parseQueryType converts string to QueryType enum
func (qp *QueryParser) parseQueryType(typeStr string) (QueryType, error) {
	switch strings.ToLower(typeStr) {
	case "one":
		return QueryTypeOne, nil
	case "many":
		return QueryTypeMany, nil
	case "exec":
		return QueryTypeExec, nil
	case "paginated":
		return QueryTypePaginated, nil
	default:
		return "", fmt.Errorf("invalid query type: %s (supported: one, many, exec, paginated)", typeStr)
	}
}

// ValidateQuery performs basic validation on a parsed query
func (qp *QueryParser) ValidateQuery(query Query) error {
	if query.Name == "" {
		return fmt.Errorf("query name cannot be empty")
	}

	if query.SQL == "" {
		return fmt.Errorf("query SQL cannot be empty")
	}

	if query.Type == "" {
		return fmt.Errorf("query type cannot be empty")
	}

	// Validate query name format (must be valid Go identifier)
	if !isValidGoIdentifier(query.Name) {
		return fmt.Errorf("query name '%s' is not a valid Go identifier", query.Name)
	}

	// Basic SQL validation
	sqlLower := strings.ToLower(strings.TrimSpace(query.SQL))

	// Check query type matches SQL statement
	switch query.Type {
	case QueryTypeOne, QueryTypeMany, QueryTypePaginated:
		// Allow SELECT statements and CTEs (Common Table Expressions)
		if !strings.HasPrefix(sqlLower, "select") && !strings.HasPrefix(sqlLower, "with") {
			sqlSnippet := query.SQL
			if len(sqlSnippet) > 50 {
				sqlSnippet = sqlSnippet[:50] + "..."
			}
			return fmt.Errorf("query type %s requires SELECT statement or CTE, got: %s", query.Type, sqlSnippet)
		}
	case QueryTypeExec:
		// Exec queries should not be SELECT or CTE
		if strings.HasPrefix(sqlLower, "select") || strings.HasPrefix(sqlLower, "with") {
			sqlSnippet := query.SQL
			if len(sqlSnippet) > 50 {
				sqlSnippet = sqlSnippet[:50] + "..."
			}
			return fmt.Errorf("query type %s cannot use SELECT statement or CTE, got: %s", query.Type, sqlSnippet)
		}
	}

	return nil
}

// isValidGoIdentifier checks if a string is a valid Go identifier
func isValidGoIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Must start with letter or underscore
	if (name[0] < 'a' || name[0] > 'z') && (name[0] < 'A' || name[0] > 'Z') && name[0] != '_' {
		return false
	}

	// Rest must be letters, digits, or underscores
	for i := 1; i < len(name); i++ {
		if (name[i] < 'a' || name[i] > 'z') && (name[i] < 'A' || name[i] > 'Z') && (name[i] < '0' || name[i] > '9') && name[i] != '_' {
			return false
		}
	}

	return true
}

// parseParameterAnnotation parses a parameter type annotation
// Expected format: -- param: $N parameter_name go_type
func (qp *QueryParser) parseParameterAnnotation(line string) *ParameterAnnotation {
	// Regex to match: -- param: $N parameter_name go_type
	paramRegex := regexp.MustCompile(`^--\s*param:\s*\$(\d+)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+(\*?[a-zA-Z_][a-zA-Z0-9_.]*)\s*$`)

	matches := paramRegex.FindStringSubmatch(line)
	if len(matches) != 4 {
		return nil
	}

	position := 0
	fmt.Sscanf(matches[1], "%d", &position)
	paramName := strings.TrimSpace(matches[2])
	goType := strings.TrimSpace(matches[3])

	if position < 1 || paramName == "" || goType == "" {
		return nil
	}

	return &ParameterAnnotation{
		Position: position,
		Name:     paramName,
		GoType:   goType,
	}
}

// parseResultAnnotation parses a result column type annotation
// Expected format: -- result: column_name go_type
func (qp *QueryParser) parseResultAnnotation(line string) *ResultAnnotation {
	// Regex to match: -- result: column_name go_type
	resultRegex := regexp.MustCompile(`^--\s*result:\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+(\*?[a-zA-Z_][a-zA-Z0-9_.]*)\s*$`)

	matches := resultRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil
	}

	columnName := strings.TrimSpace(matches[1])
	goType := strings.TrimSpace(matches[2])

	if columnName == "" || goType == "" {
		return nil
	}

	return &ResultAnnotation{
		ColumnName: columnName,
		GoType:     goType,
	}
}

// parseCursorColumnsAnnotation parses cursor_columns annotation
// Expected format: -- cursor_columns: col1, col2, col3
func (qp *QueryParser) parseCursorColumnsAnnotation(line string) []string {
	// Regex to match: -- cursor_columns: col1, col2, col3
	cursorRegex := regexp.MustCompile(`^--\s*cursor_columns:\s*(.+)$`)

	matches := cursorRegex.FindStringSubmatch(line)
	if len(matches) != 2 {
		return nil
	}

	columnsList := strings.TrimSpace(matches[1])
	if columnsList == "" {
		return nil
	}

	// Split by comma and trim whitespace
	var columns []string
	for _, col := range strings.Split(columnsList, ",") {
		col = strings.TrimSpace(col)
		if col != "" {
			columns = append(columns, col)
		}
	}

	return columns
}

// validateParameterAnnotations validates parameter annotations for a query
func (qp *QueryParser) validateParameterAnnotations(query *Query) error {
	if len(query.ParameterAnnotations) == 0 {
		return nil
	}

	// Check for duplicate positions
	positionsSeen := make(map[int]bool)
	for _, pa := range query.ParameterAnnotations {
		if positionsSeen[pa.Position] {
			return fmt.Errorf("duplicate parameter annotation for $%d", pa.Position)
		}
		positionsSeen[pa.Position] = true
	}

	// Check that positions are sequential starting at $1
	for i := 1; i <= len(query.ParameterAnnotations); i++ {
		if !positionsSeen[i] {
			return fmt.Errorf("parameter annotations must be sequential starting at $1, missing $%d", i)
		}
	}

	// Validate Go type syntax (basic check)
	for _, pa := range query.ParameterAnnotations {
		if !isValidGoType(pa.GoType) {
			return fmt.Errorf("invalid Go type %q for parameter $%d", pa.GoType, pa.Position)
		}
	}

	return nil
}

// validateResultAnnotations validates result annotations for a query
func (qp *QueryParser) validateResultAnnotations(query *Query) error {
	if len(query.ResultAnnotations) == 0 {
		return nil
	}

	// Check for duplicate column names
	columnsSeen := make(map[string]bool)
	for _, ra := range query.ResultAnnotations {
		if columnsSeen[ra.ColumnName] {
			return fmt.Errorf("duplicate result annotation for column %q", ra.ColumnName)
		}
		columnsSeen[ra.ColumnName] = true
	}

	// Validate Go type syntax (basic check)
	for _, ra := range query.ResultAnnotations {
		if !isValidGoType(ra.GoType) {
			return fmt.Errorf("invalid Go type %q for result column %q", ra.GoType, ra.ColumnName)
		}
	}

	return nil
}

// validateCursorColumns validates cursor_columns annotation
func (qp *QueryParser) validateCursorColumns(query *Query) error {
	if len(query.CursorColumns) == 0 {
		return nil
	}

	// cursor_columns only valid for :many queries
	if query.Type != QueryTypeMany {
		return fmt.Errorf("cursor_columns annotation only valid for :many queries, got :%s", query.Type)
	}

	// Validate column names
	for _, col := range query.CursorColumns {
		if !isValidColumnName(col) {
			return fmt.Errorf("invalid column name in cursor_columns: %q", col)
		}
	}

	return nil
}

// isValidColumnName checks if a string is a valid PostgreSQL column name
func isValidColumnName(name string) bool {
	if name == "" {
		return false
	}

	// Must start with letter or underscore
	if (name[0] < 'a' || name[0] > 'z') && (name[0] < 'A' || name[0] > 'Z') && name[0] != '_' {
		return false
	}

	// Rest must be letters, digits, or underscores
	for i := 1; i < len(name); i++ {
		if (name[i] < 'a' || name[i] > 'z') && (name[i] < 'A' || name[i] > 'Z') && (name[i] < '0' || name[i] > '9') && name[i] != '_' {
			return false
		}
	}

	return true
}

// isValidGoType performs basic validation of Go type syntax
func isValidGoType(goType string) bool {
	if goType == "" {
		return false
	}

	// Handle pointer types
	goType = strings.TrimPrefix(goType, "*")

	// Must be a valid Go identifier or qualified identifier (e.g., uuid.UUID, time.Time)
	parts := strings.Split(goType, ".")
	for _, part := range parts {
		if !isValidGoIdentifier(part) {
			return false
		}
	}

	return len(parts) <= 2
}

// parseCursorColumnsAnnotation parses a cursor_columns annotation
// Expected format: -- cursor_columns: col1, col2, col3
func (qp *QueryParser) parseCursorColumnsAnnotation(line string) []string {
	// Regex to match: -- cursor_columns: col1, col2, col3
	cursorRegex := regexp.MustCompile(`^--\s*cursor_columns:\s*(.+)\s*$`)

	matches := cursorRegex.FindStringSubmatch(line)
	if len(matches) != 2 {
		return nil
	}

	// Parse comma-separated column names
	columnsStr := strings.TrimSpace(matches[1])
	if columnsStr == "" {
		return nil
	}

	var columns []string
	for _, col := range strings.Split(columnsStr, ",") {
		col = strings.TrimSpace(col)
		if col != "" && isValidGoIdentifier(col) {
			columns = append(columns, col)
		}
	}

	return columns
}
