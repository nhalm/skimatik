package generator

import (
	"fmt"
	"strings"
	"text/template"
)

// Query generation helper methods for CodeGenerator

// getQueryImports returns the imports needed for all queries
func (cg *CodeGenerator) getQueryImports(queries []Query) []string {
	imports := make(map[string]bool)

	for _, query := range queries {
		// Get imports for result columns
		queryImports := cg.typeMapper.GetRequiredImports(query.Columns)
		for _, imp := range queryImports {
			imports[imp] = true
		}

		// Get imports for parameters
		paramImports := cg.typeMapper.GetRequiredImports(convertParametersToColumns(query.Parameters))
		for _, imp := range paramImports {
			imports[imp] = true
		}
	}

	// Convert map to slice
	var result []string
	for imp := range imports {
		result = append(result, imp)
	}

	return result
}

// convertParametersToColumns converts parameters to columns for import calculation
func convertParametersToColumns(params []Parameter) []Column {
	var columns []Column
	for _, param := range params {
		columns = append(columns, Column{
			Name:   param.Name,
			Type:   param.Type,
			GoType: param.GoType,
		})
	}
	return columns
}

// needsResultStruct determines if a query needs a custom result struct
func (cg *CodeGenerator) needsResultStruct(query Query) bool {
	// Only SELECT queries (:one, :many, :paginated) need result structs
	return query.Type == QueryTypeOne || query.Type == QueryTypeMany || query.Type == QueryTypePaginated
}

// getQueryResultStructName returns the struct name for a query's result
func (cg *CodeGenerator) getQueryResultStructName(query Query) string {
	return query.GoFunctionName() + "Result"
}

// generateQueryResultStruct generates a result struct for a query
func (cg *CodeGenerator) generateQueryResultStruct(query Query) (string, error) {
	if len(query.Columns) == 0 {
		return "", fmt.Errorf("query %s has no columns for result struct", query.Name)
	}

	tmpl := `// {{.StructName}} represents the result of the {{.QueryName}} query
type {{.StructName}} struct {
{{range .Fields}}	{{.Name}} {{.Type}} ` + "`{{.Tag}}`" + `
{{end}}}

// GetID returns the ID field for pagination (assumes first UUID field is the ID)
func (r {{.StructName}}) GetID() uuid.UUID {
{{if .IDField}}{{if .IDFieldIsPgtype}}	return uuid.UUID(r.{{.IDField}}.Bytes)
{{else}}	return r.{{.IDField}}
{{end}}{{else}}	// No UUID field found, return zero UUID
	return uuid.UUID{}
{{end}}}`

	// Prepare template data
	data := struct {
		StructName      string
		QueryName       string
		IDField         string
		IDFieldIsPgtype bool
		Fields          []struct {
			Name string
			Type string
			Tag  string
		}
	}{
		StructName: cg.getQueryResultStructName(query),
		QueryName:  query.Name,
	}

	// Add fields from query columns and find ID field
	for _, col := range query.Columns {
		field := struct {
			Name string
			Type string
			Tag  string
		}{
			Name: col.GoFieldName(),
			Type: col.GoType,
			Tag:  col.GoStructTag(),
		}
		data.Fields = append(data.Fields, field)

		// Use the first UUID field as the ID field for pagination
		if data.IDField == "" && col.IsUUID() {
			data.IDField = col.GoFieldName()
			data.IDFieldIsPgtype = col.GoType == "pgtype.UUID"
		}
	}

	// Execute template
	t, err := template.New("resultStruct").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// generateQueryRepository generates the repository struct and constructor for queries
func (cg *CodeGenerator) generateQueryRepository(sourceFile string, queries []Query) (string, error) {
	// Extract base name from source file path for repository name
	parts := strings.Split(sourceFile, "/")
	filename := parts[len(parts)-1]
	baseName := strings.TrimSuffix(filename, ".sql")
	repositoryName := toPascalCase(baseName) + "Queries"

	tmpl := `// {{.RepositoryName}} provides database operations for queries in {{.SourceFile}}
type {{.RepositoryName}} struct {
	conn *pgxpool.Pool
}

// New{{.RepositoryName}} creates a new {{.RepositoryName}}
func New{{.RepositoryName}}(conn *pgxpool.Pool) *{{.RepositoryName}} {
	return &{{.RepositoryName}}{
		conn: conn,
	}
}`

	// Prepare template data
	data := struct {
		RepositoryName string
		SourceFile     string
	}{
		RepositoryName: repositoryName,
		SourceFile:     sourceFile,
	}

	// Execute template
	t, err := template.New("queryRepository").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// generateQueryFunction generates a Go function for a specific query
func (cg *CodeGenerator) generateQueryFunction(query Query) (string, error) {
	switch query.Type {
	case QueryTypeOne:
		return cg.generateOneQueryFunction(query)
	case QueryTypeMany:
		return cg.generateManyQueryFunction(query)
	case QueryTypeExec:
		return cg.generateExecQueryFunction(query)
	case QueryTypePaginated:
		return cg.generatePaginatedQueryFunction(query)
	default:
		return "", fmt.Errorf("unsupported query type: %s", query.Type)
	}
}

// generateOneQueryFunction generates a function that returns a single row
func (cg *CodeGenerator) generateOneQueryFunction(query Query) (string, error) {
	tmpl := `// {{.FunctionName}} executes the {{.QueryName}} query and returns a single result
func (r *{{.RepositoryName}}) {{.FunctionName}}(ctx context.Context{{.ParameterDeclarations}}) (*{{.ResultType}}, error) {
	query := ` + "`" + `{{.SQL}}` + "`" + `
	
	var result {{.ResultType}}
	err := r.conn.QueryRow(ctx, query{{.ParameterArgs}}).Scan({{.ScanArgs}})
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}`

	data, err := cg.prepareQueryTemplateData(query)
	if err != nil {
		return "", err
	}

	t, err := template.New("oneQuery").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// generateManyQueryFunction generates a function that returns multiple rows
func (cg *CodeGenerator) generateManyQueryFunction(query Query) (string, error) {
	tmpl := `// {{.FunctionName}} executes the {{.QueryName}} query and returns multiple results
func (r *{{.RepositoryName}}) {{.FunctionName}}(ctx context.Context{{.ParameterDeclarations}}) ([]{{.ResultType}}, error) {
	query := ` + "`" + `{{.SQL}}` + "`" + `
	
	rows, err := r.conn.Query(ctx, query{{.ParameterArgs}})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []{{.ResultType}}
	for rows.Next() {
		var result {{.ResultType}}
		err := rows.Scan({{.ScanArgs}})
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	
	return results, rows.Err()
}`

	data, err := cg.prepareQueryTemplateData(query)
	if err != nil {
		return "", err
	}

	t, err := template.New("manyQuery").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// generateExecQueryFunction generates a function that executes without returning rows
func (cg *CodeGenerator) generateExecQueryFunction(query Query) (string, error) {
	tmpl := `// {{.FunctionName}} executes the {{.QueryName}} query
func (r *{{.RepositoryName}}) {{.FunctionName}}(ctx context.Context{{.ParameterDeclarations}}) error {
	query := ` + "`" + `{{.SQL}}` + "`" + `
	
	_, err := r.conn.Exec(ctx, query{{.ParameterArgs}})
	return err
}`

	data, err := cg.prepareQueryTemplateData(query)
	if err != nil {
		return "", err
	}

	t, err := template.New("execQuery").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// generatePaginatedQueryFunction generates a function that returns paginated results
func (cg *CodeGenerator) generatePaginatedQueryFunction(query Query) (string, error) {
	tmpl := `// {{.FunctionName}} executes the {{.QueryName}} query with pagination
func (r *{{.RepositoryName}}) {{.FunctionName}}(ctx context.Context, params PaginationParams{{.ParameterDeclarations}}) (*PaginationResult[{{.ResultType}}], error) {
	// Validate pagination parameters
	if err := validatePaginationParams(params); err != nil {
		return nil, err
	}

	// Build query with pagination
	query := ` + "`" + `{{.SQL}}` + "`" + `
	args := []interface{}{}
	
	// Add cursor condition if provided
	if params.Cursor != "" {
		cursorID, err := decodeCursor(params.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		args = append(args, cursorID)
	}
	
	// Add limit (request one extra to determine hasMore)
	args = append(args, params.Limit+1)
	
	// Add user parameters{{.ParameterArgs}}
	
	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []{{.ResultType}}
	for rows.Next() {
		var result {{.ResultType}}
		err := rows.Scan({{.ScanArgs}})
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}
	
	// Calculate pagination metadata
	hasMore := len(results) > int(params.Limit)
	if hasMore {
		// Remove the extra item
		results = results[:params.Limit]
	}
	
	var nextCursor string
	if hasMore && len(results) > 0 {
		// Use the last item's ID as the next cursor
		lastItem := results[len(results)-1]
		nextCursor = encodeCursor(lastItem.GetID())
	}
	
	return &PaginationResult[{{.ResultType}}]{
		Items:      results,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}`

	data, err := cg.prepareQueryTemplateData(query)
	if err != nil {
		return "", err
	}

	t, err := template.New("paginatedQuery").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// prepareQueryTemplateData prepares common template data for query functions
func (cg *CodeGenerator) prepareQueryTemplateData(query Query) (map[string]interface{}, error) {
	// Extract base name from source file for repository name
	parts := strings.Split(query.SourceFile, "/")
	filename := parts[len(parts)-1]
	baseName := strings.TrimSuffix(filename, ".sql")
	repositoryName := toPascalCase(baseName) + "Queries"

	// Build parameter declarations and arguments
	var paramDeclarations []string
	var paramArgs []string

	for _, param := range query.Parameters {
		paramDeclarations = append(paramDeclarations, fmt.Sprintf("%s %s", param.Name, param.GoType))
		paramArgs = append(paramArgs, param.Name)
	}

	// Build scan arguments for result columns
	var scanArgs []string
	for _, col := range query.Columns {
		scanArgs = append(scanArgs, "&result."+col.GoFieldName())
	}

	// Determine result type
	resultType := cg.getQueryResultStructName(query)
	if query.Type == QueryTypeExec {
		resultType = "" // Exec queries don't return data
	}

	// Format parameter declarations and arguments
	paramDeclStr := ""
	if len(paramDeclarations) > 0 {
		paramDeclStr = ", " + strings.Join(paramDeclarations, ", ")
	}

	paramArgStr := ""
	if len(paramArgs) > 0 {
		paramArgStr = ", " + strings.Join(paramArgs, ", ")
	}

	return map[string]interface{}{
		"FunctionName":          query.GoFunctionName(),
		"QueryName":             query.Name,
		"RepositoryName":        repositoryName,
		"SQL":                   query.SQL,
		"ResultType":            resultType,
		"ParameterDeclarations": paramDeclStr,
		"ParameterArgs":         paramArgStr,
		"ScanArgs":              strings.Join(scanArgs, ", "),
	}, nil
}
