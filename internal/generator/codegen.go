// Package generator implements skimatik's database-first code generation:
// PostgreSQL schema introspection, SQL file parsing, query analysis, and
// emission of type-safe Go repositories with cursor-based pagination.
package generator

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

// CodeGenerator handles generating Go code from database schema
type CodeGenerator struct {
	config         *Config
	typeMapper     *TypeMapper
	templateMgr    *TemplateManager
	version        string
	generatedFiles []string
}

// GeneratedFiles returns the list of file paths written so far.
func (cg *CodeGenerator) GeneratedFiles() []string {
	return cg.generatedFiles
}

// fileSpec describes the inputs to the wrapper template that assembles a
// generated source file. Exactly one of Source or Description should be set:
// Source is used for table/query files (e.g. "table users"), Description for
// shared files (e.g. "Shared pagination types and utilities").
type fileSpec struct {
	Version     string
	Source      string
	Description string
	Package     string
	Imports     []string
	Body        string
}

// renderFile assembles a complete generated source file by rendering the
// shared wrapper template against the given spec.
func (cg *CodeGenerator) renderFile(spec fileSpec) (string, error) {
	return cg.templateMgr.ExecuteTemplate(TemplateFile, spec)
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(config *Config, version string) *CodeGenerator {
	return &CodeGenerator{
		config:      config,
		typeMapper:  NewTypeMapper(config.TypeMappings),
		templateMgr: NewTemplateManager(templateFS),
		version:     version,
	}
}

// GenerateTableRepository generates a complete repository file for a table
func (cg *CodeGenerator) GenerateTableRepository(table Table) error {
	// Map column types
	if err := cg.typeMapper.MapTableColumns(&table); err != nil {
		return fmt.Errorf("failed to map column types: %w", err)
	}

	// Generate the code
	code, err := cg.generateTableCode(table)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Write to file
	filename := cg.config.GetOutputPath(table.GoFileName())
	if err := cg.writeCodeToFile(filename, code); err != nil {
		return fmt.Errorf("failed to write code to file: %w", err)
	}

	return nil
}

// generateTableCode generates the complete Go code for a table
func (cg *CodeGenerator) generateTableCode(table Table) (string, error) {
	// Get required imports from column types
	typeImports := cg.typeMapper.GetRequiredImports(table.Columns)

	// Add minimal imports required for table-specific operations
	coreImports := []string{
		"context",
		"fmt",
		"github.com/nhalm/pgxkit/v2",
		"github.com/google/uuid",
	}

	// Combine and deduplicate imports
	allImports := cg.combineImports(coreImports, typeImports)

	// Generate struct
	structCode, err := cg.generateStruct(table)
	if err != nil {
		return "", fmt.Errorf("failed to generate struct: %w", err)
	}

	// Generate repository
	repositoryCode, err := cg.generateRepository(table)
	if err != nil {
		return "", fmt.Errorf("failed to generate repository: %w", err)
	}

	// Generate CRUD operations
	crudCode, err := cg.generateCRUDOperations(table)
	if err != nil {
		return "", fmt.Errorf("failed to generate CRUD operations: %w", err)
	}

	// Generate enhanced features
	enhancedCode, err := cg.generateEnhancedFeatures(table)
	if err != nil {
		return "", fmt.Errorf("failed to generate enhanced features: %w", err)
	}

	// Build body: struct + repository + CRUD + (optional) enhanced features.
	body := structCode + "\n\n" + repositoryCode + "\n\n" + crudCode
	if enhancedCode != "" {
		body += "\n\n" + enhancedCode
	}

	return cg.renderFile(fileSpec{
		Version: cg.version,
		Source:  "table " + table.Name,
		Package: cg.config.PackageName,
		Imports: allImports,
		Body:    body,
	})
}

// combineImports combines and deduplicates import lists
func (cg *CodeGenerator) combineImports(lists ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, list := range lists {
		for _, imp := range list {
			if !seen[imp] {
				seen[imp] = true
				result = append(result, imp)
			}
		}
	}

	return result
}

// getQueryImports returns the imports needed for all queries
func (cg *CodeGenerator) getQueryImports(queries []Query) []string {
	importSet := make(map[string]bool)

	hasPaginatedQueries := false
	for i := range queries {
		query := &queries[i]
		// Get imports for result columns
		queryImports := cg.typeMapper.GetRequiredImports(query.Columns)
		for _, imp := range queryImports {
			importSet[imp] = true
		}

		// Get imports for parameters
		paramImports := cg.typeMapper.GetRequiredImports(convertParametersToColumns(query.Parameters))
		for _, imp := range paramImports {
			importSet[imp] = true
		}

		// Check if query uses OrderByColumns (for pagination)
		if len(query.OrderByColumns) > 0 {
			hasPaginatedQueries = true
		}
	}

	// Add reflect package if any query uses OrderByColumns
	if hasPaginatedQueries {
		importSet["reflect"] = true
	}

	// Convert map to slice
	result := make([]string, 0, len(importSet))
	for imp := range importSet {
		result = append(result, imp)
	}

	return result
}

// convertParametersToColumns converts parameters to columns for import calculation
func convertParametersToColumns(params []Parameter) []Column {
	columns := make([]Column, 0, len(params))
	for _, param := range params {
		columns = append(columns, Column{
			Name:   param.Name,
			Type:   param.Type,
			GoType: param.GoType,
		})
	}
	return columns
}

// generateEnhancedFeatures generates enhanced pgxkit features (retry methods)
func (cg *CodeGenerator) generateEnhancedFeatures(table Table) (string, error) {
	var code strings.Builder

	// Prepare template data
	data := cg.prepareCRUDTemplateData(table)

	// Generate retry methods
	retryCode, err := cg.templateMgr.ExecuteTemplate(TemplateRepositoryRetry, data)
	if err != nil {
		return "", fmt.Errorf("failed to generate retry methods: %w", err)
	}
	code.WriteString(retryCode)

	// Health check methods are not generated - left to implementor

	return code.String(), nil
}

// generateStruct generates the Go struct for a table
func (cg *CodeGenerator) generateStruct(table Table) (string, error) {
	// Prepare template data
	idColumn := table.GetPrimaryKeyColumn()
	data := struct {
		StructName   string
		TableName    string
		ReceiverName string
		IDField      string
		IDType       string
		Fields       []struct {
			Name string
			Type string
			Tag  string
		}
	}{
		StructName:   table.GoStructName(),
		TableName:    table.Name,
		ReceiverName: strings.ToLower(table.GoStructName()[:1]),
		IDField:      idColumn.GoFieldName(),
		IDType:       idColumn.GoType,
	}

	// Add fields
	for _, col := range table.Columns {
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
	}

	// Execute template using template manager
	return cg.templateMgr.ExecuteTemplate(TemplateStruct, data)
}

// generateRepository generates the repository struct and constructor
func (cg *CodeGenerator) generateRepository(table Table) (string, error) {
	idColumn := table.GetPrimaryKeyColumn()
	if idColumn == nil {
		return "", fmt.Errorf("table %s has no primary key", table.Name)
	}

	// Prepare template data
	data := struct {
		RepositoryName string
		TableName      string
		IDType         string
		IsUUIDType     bool
	}{
		RepositoryName: table.GoStructName() + "Repository",
		TableName:      table.Name,
		IDType:         idColumn.GoType,
		IsUUIDType:     idColumn.IsUUID(),
	}

	// Execute template using template manager
	return cg.templateMgr.ExecuteTemplate(TemplateRepositoryStruct, data)
}

// generateCRUDOperations generates specified CRUD operations for a table
func (cg *CodeGenerator) generateCRUDOperations(table Table) (string, error) {
	var code strings.Builder

	// Generate template data
	data := cg.prepareCRUDTemplateData(table)

	// Get the functions to generate for this table
	functions := cg.config.GetTableFunctions(table.Name)

	// Map function names to templates (using template manager)
	operationTemplates := map[string]string{
		"get":      TemplateGetByID,
		"create":   TemplateCreate,
		"update":   TemplateUpdate,
		"delete":   TemplateDelete,
		"list":     TemplateList,
		"paginate": TemplatePaginationSharedListPaginated,
	}

	// Generate each requested CRUD operation
	first := true
	for _, function := range functions {
		templateStr, exists := operationTemplates[function]
		if !exists {
			return "", fmt.Errorf("unknown function type: %s", function)
		}

		if !first {
			code.WriteString("\n\n")
		}
		first = false

		var result string
		var err error

		// Check if this is a template manager template (starts with "templates/")
		if strings.HasPrefix(templateStr, "templates/") {
			// Use template manager
			result, err = cg.templateMgr.ExecuteTemplate(templateStr, data)
			if err != nil {
				return "", fmt.Errorf("failed to execute template for %s: %w", function, err)
			}
		} else {
			// Use old inline template parsing
			tmpl, parseErr := template.New("crud").Parse(templateStr)
			if parseErr != nil {
				return "", fmt.Errorf("failed to parse template for %s: %w", function, parseErr)
			}

			var resultBuilder strings.Builder
			if err := tmpl.Execute(&resultBuilder, data); err != nil {
				return "", fmt.Errorf("failed to execute template for %s: %w", function, err)
			}
			result = resultBuilder.String()
		}

		code.WriteString(result)
	}

	return code.String(), nil
}

// prepareCRUDTemplateData prepares the data structure for CRUD templates.
// All inputs come from already-validated schema introspection, so no error
// path is required.
func (cg *CodeGenerator) prepareCRUDTemplateData(table Table) map[string]any {
	structName := table.GoStructName()
	repositoryName := structName + "Repository"
	receiverName := strings.ToLower(structName[:1])
	idColumn := table.GetPrimaryKeyColumn()
	createParamIndex := 1
	updateParamIndex := 1

	// Build column lists
	var selectColumns []string
	var scanArgs []string
	var createFields []map[string]string
	var updateFields []map[string]string
	var insertColumns []string
	var insertPlaceholders []string
	var insertArgs []string
	var updateAssignments []string
	var updateArgs []string

	// First, add ID column to insert lists
	insertColumns = append(insertColumns, idColumn.Name)
	insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", createParamIndex))
	insertArgs = append(insertArgs, "id")
	createParamIndex++

	for _, col := range table.Columns {
		// Select columns and scan args (for all operations)
		selectColumns = append(selectColumns, col.Name)
		scanArgs = append(scanArgs, "&"+receiverName+"."+col.GoFieldName())

		// Skip ID column for create/update params (it's auto-generated in code)
		if col.Name == idColumn.Name {
			continue
		}

		// Create fields (exclude ID and columns with defaults)
		if col.DefaultValue == "" {
			createFields = append(createFields, map[string]string{
				"Name": col.GoFieldName(),
				"Type": col.GoType,
				"Tag":  col.GoStructTag(),
			})

			insertColumns = append(insertColumns, col.Name)
			insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("$%d", createParamIndex))
			insertArgs = append(insertArgs, "params."+col.GoFieldName())
			createParamIndex++
		}

		// Update fields (all non-ID columns)
		updateFields = append(updateFields, map[string]string{
			"Name": col.GoFieldName(),
			"Type": col.GoType,
			"Tag":  col.GoStructTag(),
		})

		updateAssignments = append(updateAssignments, fmt.Sprintf("%s = $%d", col.Name, updateParamIndex))
		updateArgs = append(updateArgs, "params."+col.GoFieldName())
		updateParamIndex++
	}

	// ID parameter comes last in update
	updateArgs = append(updateArgs, "id")
	idParamIndex := updateParamIndex

	return map[string]any{
		"StructName":         structName,
		"RepositoryName":     repositoryName,
		"ReceiverName":       receiverName,
		"TableName":          table.Name,
		"IDColumn":           idColumn.Name,
		"IDColumnType":       idColumn.Type,
		"IDType":             idColumn.GoType,
		"IDParamIndex":       idParamIndex,
		"SelectColumns":      strings.Join(selectColumns, ", "),
		"ScanArgs":           strings.Join(scanArgs, ", "),
		"CreateFields":       createFields,
		"UpdateFields":       updateFields,
		"InsertColumns":      strings.Join(insertColumns, ", "),
		"InsertPlaceholders": strings.Join(insertPlaceholders, ", "),
		"InsertArgs":         strings.Join(insertArgs, ", "),
		"UpdateColumns":      strings.Join(updateAssignments, ", "),
		"UpdateArgs":         strings.Join(updateArgs, ", "),
	}
}

// GenerateSharedPaginationTypes generates the shared pagination types file
func (cg *CodeGenerator) GenerateSharedPaginationTypes() error {
	body, err := cg.templateMgr.ExecuteTemplate(TemplatePaginationSharedTypes, nil)
	if err != nil {
		return fmt.Errorf("failed to execute pagination template: %w", err)
	}

	code, err := cg.renderFile(fileSpec{
		Version:     cg.version,
		Description: "Shared pagination types and utilities",
		Package:     cg.config.PackageName,
		Imports: []string{
			"encoding/base64",
			"encoding/json",
			"fmt",
			"strings",
			"time",
			"github.com/google/uuid",
		},
		Body: body,
	})
	if err != nil {
		return fmt.Errorf("failed to render pagination file: %w", err)
	}

	filename := cg.config.GetOutputPath("pagination.go")
	if err := cg.writeCodeToFile(filename, code); err != nil {
		return fmt.Errorf("failed to write pagination file: %w", err)
	}

	return nil
}

// GenerateSharedErrors generates the shared error handling utilities file
func (cg *CodeGenerator) GenerateSharedErrors() error {
	body, err := cg.templateMgr.ExecuteTemplate(TemplateSharedErrors, nil)
	if err != nil {
		return fmt.Errorf("failed to execute shared errors template: %w", err)
	}

	code, err := cg.renderFile(fileSpec{
		Version:     cg.version,
		Description: "This file provides shared error handling utilities for all repositories",
		Package:     cg.config.PackageName,
		Imports: []string{
			"context",
			"errors",
			"fmt",
			"strings",
			"github.com/jackc/pgx/v5",
			"github.com/jackc/pgx/v5/pgconn",
		},
		Body: body,
	})
	if err != nil {
		return fmt.Errorf("failed to render errors file: %w", err)
	}

	filename := cg.config.GetOutputPath("errors.go")
	if err := cg.writeCodeToFile(filename, code); err != nil {
		return fmt.Errorf("failed to write errors file: %w", err)
	}

	return nil
}

func (cg *CodeGenerator) GenerateSharedDatabaseOperations() error {
	body, err := cg.templateMgr.ExecuteTemplate(TemplateDatabaseOperations, nil)
	if err != nil {
		return fmt.Errorf("failed to execute database operations template: %w", err)
	}

	code, err := cg.renderFile(fileSpec{
		Version:     cg.version,
		Description: "This file provides shared database operation utilities for all repositories",
		Package:     cg.config.PackageName,
		Imports: []string{
			"context",
			"github.com/jackc/pgx/v5",
			"github.com/nhalm/pgxkit/v2",
		},
		Body: body,
	})
	if err != nil {
		return fmt.Errorf("failed to render database operations file: %w", err)
	}

	filename := cg.config.GetOutputPath("database_operations.go")
	if err := cg.writeCodeToFile(filename, code); err != nil {
		return fmt.Errorf("failed to write database operations file: %w", err)
	}

	return nil
}

// GenerateSharedIDGenerators generates the shared ID generator utilities file
func (cg *CodeGenerator) GenerateSharedIDGenerators() error {
	body, err := cg.templateMgr.ExecuteTemplate(TemplateIDGenerators, nil)
	if err != nil {
		return fmt.Errorf("failed to execute ID generators template: %w", err)
	}

	code, err := cg.renderFile(fileSpec{
		Version:     cg.version,
		Description: "This file provides shared ID generator utilities for all repositories",
		Package:     cg.config.PackageName,
		Imports:     []string{"github.com/google/uuid"},
		Body:        body,
	})
	if err != nil {
		return fmt.Errorf("failed to render ID generators file: %w", err)
	}

	filename := cg.config.GetOutputPath("id_generators.go")
	if err := cg.writeCodeToFile(filename, code); err != nil {
		return fmt.Errorf("failed to write ID generators file: %w", err)
	}

	return nil
}

// writeCodeToFile writes generated code to a file with proper formatting
// and records the path in cg.generatedFiles for the caller to surface.
func (cg *CodeGenerator) writeCodeToFile(filename, code string) error {
	// Format the code
	formatted, err := imports.Process("", []byte(code), nil)
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, formatted, 0o644); err != nil { // #nosec G306 -- generated source is intentionally world-readable for editors/CI/containers
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	cg.generatedFiles = append(cg.generatedFiles, filename)
	return nil
}

func (cg *CodeGenerator) GenerateQueries(queries []Query) error {
	if len(queries) == 0 {
		return nil
	}

	// Group queries by source file
	queryGroups := cg.groupQueriesByFile(queries)

	// Generate code for each file group
	for sourceFile, fileQueries := range queryGroups {
		if err := cg.generateQueryFile(sourceFile, fileQueries); err != nil {
			return fmt.Errorf("failed to generate queries for file %s: %w", sourceFile, err)
		}
	}

	return nil
}

// groupQueriesByFile groups queries by their source file
func (cg *CodeGenerator) groupQueriesByFile(queries []Query) map[string][]Query {
	groups := make(map[string][]Query)
	for i := range queries {
		groups[queries[i].SourceFile] = append(groups[queries[i].SourceFile], queries[i])
	}
	return groups
}

// generateQueryFile generates a complete Go file for a group of queries from the same source file
func (cg *CodeGenerator) generateQueryFile(sourceFile string, queries []Query) error {
	if len(queries) == 0 {
		return nil
	}

	// QueryAnalyzer has already set intelligent types for result columns.

	// Generate the code
	code, err := cg.generateQueryCode(sourceFile, queries)
	if err != nil {
		return fmt.Errorf("failed to generate query code: %w", err)
	}

	// Get output filename from first query (they all have the same source file)
	filename := cg.config.GetOutputPath(queries[0].GoFileName())

	// Write to file
	if err := cg.writeCodeToFile(filename, code); err != nil {
		return fmt.Errorf("failed to write query code to file: %w", err)
	}

	return nil
}

// generateQueryCode generates the complete Go code for a group of queries from the same source file
func (cg *CodeGenerator) generateQueryCode(sourceFile string, queries []Query) (string, error) {
	// Get required imports from all queries
	allImports := cg.getQueryImports(queries)

	// Add standard imports
	standardImports := []string{
		"context",
		"github.com/nhalm/pgxkit/v2",
		"github.com/google/uuid",
	}

	// Check if any queries are paginated
	hasPaginatedQueries := false
	for i := range queries {
		if queries[i].Type == QueryTypePaginated {
			hasPaginatedQueries = true
			break
		}
	}

	if hasPaginatedQueries {
		standardImports = append(standardImports, "fmt", "encoding/base64")
	}

	// Combine and deduplicate imports
	allImports = cg.combineImports(standardImports, allImports)

	// Build body: result structs + repository + per-query functions.
	var body strings.Builder

	// Generate result structs for queries that need them
	structsGenerated := make(map[string]bool)
	for i := range queries {
		query := &queries[i]
		if cg.needsResultStruct(*query) {
			structName := cg.getQueryResultStructName(*query)
			if !structsGenerated[structName] {
				structCode, err := cg.generateQueryResultStruct(*query)
				if err != nil {
					return "", fmt.Errorf("failed to generate result struct for query %s: %w", query.Name, err)
				}
				body.WriteString(structCode)
				body.WriteString("\n\n")
				structsGenerated[structName] = true
			}
		}
	}

	// Pagination types are in the shared pagination.go file

	// Generate repository struct and constructor
	repoCode, err := cg.generateQueryRepository(sourceFile, queries)
	if err != nil {
		return "", fmt.Errorf("failed to generate query repository: %w", err)
	}
	body.WriteString(repoCode)
	body.WriteString("\n\n")

	// Generate functions for each query
	for i := range queries {
		query := &queries[i]
		if i > 0 {
			body.WriteString("\n\n")
		}

		functionCode, err := cg.generateQueryFunction(*query)
		if err != nil {
			return "", fmt.Errorf("failed to generate function for query %s: %w", query.Name, err)
		}
		body.WriteString(functionCode)
	}

	return cg.renderFile(fileSpec{
		Version: cg.version,
		Source:  sourceFile,
		Package: cg.config.PackageName,
		Imports: allImports,
		Body:    body.String(),
	})
}

// Query generation helper methods moved from query_templates.go

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

	// Prepare template data
	data := struct {
		StructName      string
		QueryName       string
		IDField         string
		IDType          string
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

		// Use the first field named "id" or ending with "_id" as the ID field for pagination
		if data.IDField == "" && (col.Name == "id" || strings.HasSuffix(col.Name, "_id")) {
			data.IDField = col.GoFieldName()
			data.IDType = col.GoType
			data.IDFieldIsPgtype = col.GoType == "pgtype.UUID"
		}
	}

	// Execute template using template manager
	return cg.templateMgr.ExecuteTemplate(TemplateQueryResultStruct, data)
}

// generateQueryRepository generates the repository struct and constructor for queries
func (cg *CodeGenerator) generateQueryRepository(sourceFile string, _ []Query) (string, error) {
	// Extract base name from source file path for repository name
	parts := strings.Split(sourceFile, "/")
	filename := parts[len(parts)-1]
	baseName := strings.TrimSuffix(filename, ".sql")
	repositoryName := toPascalCase(baseName) + "Queries"

	// Prepare template data
	data := struct {
		RepositoryName string
		SourceFile     string
	}{
		RepositoryName: repositoryName,
		SourceFile:     sourceFile,
	}

	// Execute template using template manager
	return cg.templateMgr.ExecuteTemplate(TemplateQueryRepository, data)
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
	data := cg.prepareQueryTemplateData(query)
	return cg.templateMgr.ExecuteTemplate(TemplateQueryOne, data)
}

// generateManyQueryFunction generates a function that returns multiple rows
func (cg *CodeGenerator) generateManyQueryFunction(query Query) (string, error) {
	data := cg.prepareQueryTemplateData(query)
	return cg.templateMgr.ExecuteTemplate(TemplateQueryMany, data)
}

// generateExecQueryFunction generates a function that executes without returning rows
func (cg *CodeGenerator) generateExecQueryFunction(query Query) (string, error) {
	data := cg.prepareQueryTemplateData(query)
	return cg.templateMgr.ExecuteTemplate(TemplateQueryExec, data)
}

// generatePaginatedQueryFunction generates a function that returns paginated results
func (cg *CodeGenerator) generatePaginatedQueryFunction(query Query) (string, error) {
	data := cg.prepareQueryTemplateData(query)
	return cg.templateMgr.ExecuteTemplate(TemplateQueryPaginated, data)
}

// prepareQueryTemplateData prepares common template data for query functions.
// All inputs come from already-validated query analysis, so no error path
// is required.
func (cg *CodeGenerator) prepareQueryTemplateData(query Query) map[string]any {
	// Extract base name from source file for repository name
	parts := strings.Split(query.SourceFile, "/")
	filename := parts[len(parts)-1]
	baseName := strings.TrimSuffix(filename, ".sql")
	repositoryName := toPascalCase(baseName) + "Queries"

	// Build parameter declarations and arguments. Consecutive parameters that
	// share the same Go type are combined into a single declaration (e.g.
	// "a, b int" instead of "a int, b int") to satisfy the gocritic
	// paramTypeCombine check.
	type paramInfo struct {
		Name   string
		GoType string
	}
	var params []paramInfo
	var paramArgs []string

	if len(query.ParameterAnnotations) > 0 {
		for _, annotation := range query.ParameterAnnotations {
			paramName := toCamelCase(annotation.Name)
			params = append(params, paramInfo{Name: paramName, GoType: annotation.GoType})
			paramArgs = append(paramArgs, paramName)
		}
	} else {
		for _, param := range query.Parameters {
			params = append(params, paramInfo{Name: param.Name, GoType: param.GoType})
			paramArgs = append(paramArgs, param.Name)
		}
	}

	var paramDeclarations []string
	for i := 0; i < len(params); {
		j := i + 1
		for j < len(params) && params[j].GoType == params[i].GoType {
			j++
		}
		// params[i:j] all share the same type
		names := make([]string, 0, j-i)
		for k := i; k < j; k++ {
			names = append(names, params[k].Name)
		}
		paramDeclarations = append(paramDeclarations, fmt.Sprintf("%s %s", strings.Join(names, ", "), params[i].GoType))
		i = j
	}

	// Build scan arguments for result columns
	scanArgs := make([]string, 0, len(query.Columns))
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
	paramArgsOnly := ""
	if len(paramArgs) > 0 {
		paramArgsOnly = strings.Join(paramArgs, ", ")
		paramArgStr = ", " + paramArgsOnly
	}

	return map[string]any{
		"FunctionName":          query.GoFunctionName(),
		"QueryName":             query.Name,
		"RepositoryName":        repositoryName,
		"SQL":                   query.SQL,
		"ResultType":            resultType,
		"ParameterDeclarations": paramDeclStr,
		"ParameterArgs":         paramArgStr,
		"ParameterArgsOnly":     paramArgsOnly,
		"ScanArgs":              strings.Join(scanArgs, ", "),
		"OrderByColumns":        query.OrderByColumns,
	}
}
