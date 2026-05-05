package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

func (qp *QueryParser) parseQueries() ([]query, error) {
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
	var allQueries []query
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

// parseFileState holds the in-progress state while parseFile scans a SQL
// file line by line. It is intentionally unexported and used only by
// parseFile and its helpers.
type parseFileState struct {
	queries           []query
	currentQuery      *query
	sqlLines          []string
	paramAnnotations  []parameterAnnotation
	resultAnnotations []resultAnnotation
}

// parseFile parses a single SQL file and extracts queries with annotations
func (qp *QueryParser) parseFile(filename string) ([]query, error) {
	file, err := os.Open(filename) // #nosec G304 -- user-supplied query file path by design
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	state := &parseFileState{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		handled, err := qp.handleAnnotationLine(state, line, trimmedLine, filename, lineNum)
		if err != nil {
			return nil, err
		}
		if handled {
			continue
		}

		// Skip empty lines and comments (except annotations)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "--") {
			continue
		}

		// Collect SQL lines for current query
		if state.currentQuery != nil {
			state.sqlLines = append(state.sqlLines, line)
		}
	}

	// Save the last query
	if state.currentQuery != nil {
		if err := qp.finalizeQuery(state, filename, 0); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return state.queries, nil
}

// handleAnnotationLine attempts to interpret a line as a skimatik annotation
// (-- name:, -- param:, -- result:). It returns handled=true when the line
// was consumed, in which case the caller should skip its remaining work for
// that line. An error is returned only when finalizing a previous query
// fails (empty SQL or invalid annotations).
func (qp *QueryParser) handleAnnotationLine(state *parseFileState, line, trimmedLine, filename string, lineNum int) (bool, error) {
	// -- name: <Name> :<type>
	if annotation := qp.parseAnnotation(trimmedLine); annotation != nil {
		if state.currentQuery != nil {
			if err := qp.finalizeQuery(state, filename, lineNum); err != nil {
				return false, err
			}
		}
		qp.startNewQuery(state, annotation, filename)
		return true, nil
	}

	// -- param: $N name type
	if paramAnnotation := qp.parseParameterAnnotation(trimmedLine); paramAnnotation != nil {
		if state.currentQuery != nil {
			state.paramAnnotations = append(state.paramAnnotations, *paramAnnotation)
		}
		return true, nil
	}

	// -- result: column type
	if resultAnnotation := qp.parseResultAnnotation(trimmedLine); resultAnnotation != nil {
		if state.currentQuery != nil {
			state.resultAnnotations = append(state.resultAnnotations, *resultAnnotation)
		}
		return true, nil
	}

	_ = line // line is unused once we've classified the annotation; kept for parity with caller
	return false, nil
}

// finalizeQuery completes the in-progress query in state: it joins the
// collected SQL lines, validates annotations, and appends the resulting
// Query to state.queries. lineNum, when non-zero, is used in the
// "empty query" error message; pass 0 when finalizing the trailing query
// at end-of-file. After this call state.currentQuery is left as-is — the
// caller (startNewQuery / parseFile) overwrites or stops using it.
func (qp *QueryParser) finalizeQuery(state *parseFileState, filename string, lineNum int) error {
	state.currentQuery.SQL = strings.TrimSpace(strings.Join(state.sqlLines, "\n"))
	if state.currentQuery.SQL == "" {
		if lineNum > 0 {
			return fmt.Errorf("empty query for %s at line %d in %s", state.currentQuery.Name, lineNum, filename)
		}
		return fmt.Errorf("empty query for %s in %s", state.currentQuery.Name, filename)
	}
	state.currentQuery.ParameterAnnotations = state.paramAnnotations
	state.currentQuery.ResultAnnotations = state.resultAnnotations
	if err := qp.validateParameterAnnotations(state.currentQuery); err != nil {
		return fmt.Errorf("error in query %s: %w", state.currentQuery.Name, err)
	}
	if err := qp.validateResultAnnotations(state.currentQuery); err != nil {
		return fmt.Errorf("error in query %s: %w", state.currentQuery.Name, err)
	}
	state.queries = append(state.queries, *state.currentQuery)
	return nil
}

// startNewQuery resets the per-query buffers in state and installs a fresh
// currentQuery seeded from the annotation. Existing queries already saved in
// state.queries are preserved.
func (qp *QueryParser) startNewQuery(state *parseFileState, annotation *QueryAnnotation, filename string) {
	state.currentQuery = &query{
		Name:       annotation.Name,
		Type:       annotation.Type,
		SourceFile: filename,
		Parameters: []parameter{}, // Will be populated by analyzer
		Columns:    []column{},    // Will be populated by analyzer
	}
	state.sqlLines = []string{}
	state.paramAnnotations = []parameterAnnotation{}
	state.resultAnnotations = []resultAnnotation{}
}

// QueryAnnotation represents a parsed sqlc-style annotation
type QueryAnnotation struct {
	Name string
	Type queryType
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
func (qp *QueryParser) parseQueryType(typeStr string) (queryType, error) {
	switch strings.ToLower(typeStr) {
	case "one":
		return queryTypeOne, nil
	case "many":
		return queryTypeMany, nil
	case "exec":
		return queryTypeExec, nil
	case "paginated":
		return queryTypePaginated, nil
	default:
		return "", fmt.Errorf("invalid query type: %s (supported: one, many, exec, paginated)", typeStr)
	}
}

// parseParameterAnnotation parses a parameter type annotation
// Expected format: -- param: $N parameter_name go_type
func (qp *QueryParser) parseParameterAnnotation(line string) *parameterAnnotation {
	// Regex to match: -- param: $N parameter_name go_type
	// Type pattern allows: pointers (*), slices ([]), maps (map[]), and qualified names (pkg.Type)
	paramRegex := regexp.MustCompile(`^--\s*param:\s*\$(\d+)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+(.+?)\s*$`)

	matches := paramRegex.FindStringSubmatch(line)
	if len(matches) != 4 {
		return nil
	}

	position, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}
	paramName := strings.TrimSpace(matches[2])
	goType := strings.TrimSpace(matches[3])

	if position < 1 || paramName == "" || goType == "" {
		return nil
	}

	return &parameterAnnotation{
		Position: position,
		Name:     paramName,
		GoType:   goType,
	}
}

// parseResultAnnotation parses a result column type annotation
// Expected format: -- result: column_name go_type
func (qp *QueryParser) parseResultAnnotation(line string) *resultAnnotation {
	// Regex to match: -- result: column_name go_type
	// Type pattern allows: pointers (*), slices ([]), maps (map[]), and qualified names (pkg.Type)
	resultRegex := regexp.MustCompile(`^--\s*result:\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+(.+?)\s*$`)

	matches := resultRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil
	}

	columnName := strings.TrimSpace(matches[1])
	goType := strings.TrimSpace(matches[2])

	if columnName == "" || goType == "" {
		return nil
	}

	return &resultAnnotation{
		ColumnName: columnName,
		GoType:     goType,
	}
}

// validateParameterAnnotations validates parameter annotations for a query
func (qp *QueryParser) validateParameterAnnotations(query *query) error {
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
func (qp *QueryParser) validateResultAnnotations(query *query) error {
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

// isValidGoType performs basic validation of Go type syntax
// This is intentionally permissive - the Go compiler will catch actual type errors
func isValidGoType(goType string) bool {
	if goType == "" {
		return false
	}

	// Basic sanity check: must contain at least one letter
	hasLetter := false
	for _, r := range goType {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
			break
		}
	}

	return hasLetter
}
