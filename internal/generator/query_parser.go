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
			sqlLines = []string{} // Reset SQL lines
			continue
		}

		// Skip empty lines and comments (except annotations)
		if trimmedLine == "" || (strings.HasPrefix(trimmedLine, "--") && !strings.Contains(trimmedLine, "name:")) {
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
