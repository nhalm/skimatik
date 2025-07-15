# Template Migration Guide: Inline Strings → Embedded .tmpl Files

## Overview

This guide explains how to migrate the skimatic code generator from inline Go string templates to embedded `.tmpl` files using Go's `embed` package. This migration will improve template maintainability, readability, and developer experience.

## Why Migrate?

### Current Pain Points
- **Escaping Hell**: Complex backtick escaping and string concatenation
- **Poor Readability**: Large templates (300+ lines) are hard to read in Go strings
- **No Syntax Highlighting**: No proper template syntax highlighting in editors
- **Maintenance Burden**: Adding template features requires string manipulation

### Benefits of .tmpl Files
- ✅ **Better Syntax Highlighting**: Proper Go template syntax highlighting
- ✅ **No Escaping**: Clean template syntax without Go string escaping
- ✅ **Easier Maintenance**: Large templates become manageable
- ✅ **Better Tooling**: Template-specific linting and formatting
- ✅ **Zero Runtime Dependencies**: Templates embedded at build time

## Migration Strategy

### Phase 1: Setup Infrastructure
1. Create template directory structure
2. Implement template loading system
3. Create template manager utilities

### Phase 2: Migrate Templates Systematically
Work through each template file in order:
1. `shared_types_templates.go` → `templates/repository/complete_repository.tmpl`
2. `query_templates.go` → `templates/queries/*.tmpl`
3. `shared_pagination_templates.go` → `templates/pagination/shared_types.tmpl`
4. `inline_pagination_templates.go` → `templates/pagination/*.tmpl`
5. `crud_templates.go` → `templates/crud/*.tmpl`
6. Inline templates in `codegen.go`

### Phase 3: Cleanup
1. Remove old template constant files
2. Run full test suite to verify no regressions

## Implementation Plan

### Step 1: Create Template Directory Structure

```
internal/generator/
├── templates/
│   ├── crud/
│   │   ├── get_by_id.tmpl
│   │   ├── create.tmpl
│   │   ├── update.tmpl
│   │   ├── delete.tmpl
│   │   └── list.tmpl
│   ├── pagination/
│   │   ├── shared_types.tmpl
│   │   ├── inline_paginated.tmpl
│   │   └── pagination_utils.tmpl
│   ├── queries/
│   │   ├── one_query.tmpl
│   │   ├── many_query.tmpl
│   │   ├── exec_query.tmpl
│   │   └── paginated_query.tmpl
│   ├── repository/
│   │   ├── repository_struct.tmpl
│   │   └── complete_repository.tmpl
│   └── shared/
│       ├── struct.tmpl
│       └── header.tmpl
├── template_manager.go      # New file
├── templates.go            # New file (embed declarations)
└── ...existing files...
```

### Step 2: Create Template Manager

Create `internal/generator/template_manager.go`:

```go
package generator

import (
	"embed"
	"fmt"
	"text/template"
)

// TemplateManager handles loading and executing embedded templates
type TemplateManager struct {
	templates map[string]*template.Template
	fs        embed.FS
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(fs embed.FS) *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*template.Template),
		fs:        fs,
	}
}

// LoadTemplate loads and parses a template from the embedded filesystem
func (tm *TemplateManager) LoadTemplate(name string) (*template.Template, error) {
	// Check cache first
	if tmpl, exists := tm.templates[name]; exists {
		return tmpl, nil
	}

	// Read template file
	content, err := tm.fs.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", name, err)
	}

	// Parse template
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Cache template
	tm.templates[name] = tmpl
	return tmpl, nil
}

// ExecuteTemplate executes a template with given data
func (tm *TemplateManager) ExecuteTemplate(name string, data interface{}) (string, error) {
	tmpl, err := tm.LoadTemplate(name)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return result.String(), nil
}
```

### Step 3: Create Template Embeddings

Create `internal/generator/templates.go`:

```go
package generator

import "embed"

// Embed all template files at build time
//go:embed templates/*
var templateFS embed.FS

// Template file paths (constants for type safety)
const (
	// CRUD templates
	TemplateGetByID = "templates/crud/get_by_id.tmpl"
	TemplateCreate  = "templates/crud/create.tmpl"
	TemplateUpdate  = "templates/crud/update.tmpl"
	TemplateDelete  = "templates/crud/delete.tmpl"
	TemplateList    = "templates/crud/list.tmpl"

	// Pagination templates
	TemplatePaginationShared = "templates/pagination/shared_types.tmpl"
	TemplatePaginationInline = "templates/pagination/inline_paginated.tmpl"
	TemplatePaginationUtils  = "templates/pagination/pagination_utils.tmpl"

	// Query templates
	TemplateQueryOne        = "templates/queries/one_query.tmpl"
	TemplateQueryMany       = "templates/queries/many_query.tmpl"
	TemplateQueryExec       = "templates/queries/exec_query.tmpl"
	TemplateQueryPaginated  = "templates/queries/paginated_query.tmpl"

	// Repository templates
	TemplateRepositoryStruct    = "templates/repository/repository_struct.tmpl"
	TemplateRepositoryComplete  = "templates/repository/complete_repository.tmpl"

	// Shared templates
	TemplateStruct = "templates/shared/struct.tmpl"
	TemplateHeader = "templates/shared/header.tmpl"
)
```

### Step 4: Update CodeGenerator

Modify `internal/generator/codegen.go` to use template manager:

```go
// Add to CodeGenerator struct
type CodeGenerator struct {
	config       *Config
	typeMapper   *TypeMapper
	templateMgr  *TemplateManager  // Add this field
}

// Update constructor
func NewCodeGenerator(config *Config) *CodeGenerator {
	return &CodeGenerator{
		config:      config,
		typeMapper:  NewTypeMapper(),
		templateMgr: NewTemplateManager(templateFS),  // Initialize template manager
	}
}

// Update template usage (example)
func (cg *CodeGenerator) generateStruct(table Table) (string, error) {
	// Prepare template data (same as before)
	data := struct {
		StructName   string
		TableName    string
		ReceiverName string
		IDField      string
		Fields       []struct {
			Name string
			Type string
			Tag  string
		}
	}{
		// ... populate data ...
	}

	// Use template manager instead of inline string
	return cg.templateMgr.ExecuteTemplate(TemplateStruct, data)
}
```

## Migration Order

Work through template files systematically:

1. **`shared_types_templates.go`** (304 lines) → `templates/repository/complete_repository.tmpl`
2. **`query_templates.go`** (431 lines) → `templates/queries/*.tmpl`
3. **`shared_pagination_templates.go`** (169 lines) → `templates/pagination/shared_types.tmpl`
4. **`inline_pagination_templates.go`** (181 lines) → `templates/pagination/*.tmpl`
5. **`crud_templates.go`** (110 lines) → `templates/crud/*.tmpl`
6. **Inline templates in `codegen.go`** → `templates/shared/*.tmpl`

## Template Conversion Examples

### Before (Inline String)
```go
// crud_templates.go
const getByIDTemplate = `// GetByID retrieves a {{.StructName}} by its ID
func (r *{{.RepositoryName}}) GetByID(ctx context.Context, id uuid.UUID) (*{{.StructName}}, error) {
	query := ` + "`" + `
		SELECT {{.SelectColumns}}
		FROM {{.TableName}}
		WHERE {{.IDColumn}} = $1
	` + "`" + `
	
	var {{.ReceiverName}} {{.StructName}}
	err := r.conn.QueryRow(ctx, query, id).Scan({{.ScanArgs}})
	if err != nil {
		return nil, err
	}
	
	return &{{.ReceiverName}}, nil
}`
```

### After (.tmpl file)
```go
// templates/crud/get_by_id.tmpl
// GetByID retrieves a {{.StructName}} by its ID
func (r *{{.RepositoryName}}) GetByID(ctx context.Context, id uuid.UUID) (*{{.StructName}}, error) {
	query := `
		SELECT {{.SelectColumns}}
		FROM {{.TableName}}
		WHERE {{.IDColumn}} = $1
	`
	
	var {{.ReceiverName}} {{.StructName}}
	err := r.conn.QueryRow(ctx, query, id).Scan({{.ScanArgs}})
	if err != nil {
		return nil, err
	}
	
	return &{{.ReceiverName}}, nil
}
```

## Testing Strategy

**Simple approach**: Use existing comprehensive test suite

1. **After each template migration**: Run `go test ./internal/generator`
2. **After completing a file**: Run `go test ./internal/generator -v`
3. **Before cleanup**: Run full test suite `make test && make integration-test`

The existing tests will catch any regressions - no need for additional testing infrastructure.

## Migration Checklist

### Phase 1: Infrastructure
- [x] Create `internal/generator/templates/` directory structure
- [x] Implement `TemplateManager` in `template_manager.go`
- [x] Create `templates.go` with embed declarations
- [x] Add template path constants
- [x] Update `CodeGenerator` to use `TemplateManager`

### Phase 2: Template Migration
- [x] Migrate `shared_types_templates.go` → `templates/repository/complete_repository.tmpl`
- [x] Migrate `query_templates.go` → `templates/queries/*.tmpl`
- [ ] Migrate `shared_pagination_templates.go` → `templates/pagination/shared_types.tmpl`
- [x] Migrate `inline_pagination_templates.go` → `templates/pagination/*.tmpl`
- [ ] Migrate `crud_templates.go` → `templates/crud/*.tmpl`
- [ ] Migrate remaining inline templates

### Phase 3: Cleanup
- [ ] Remove old template constant files
- [ ] Update all template references
- [ ] Add template tests
- [ ] Update documentation
- [ ] Run full test suite

### Phase 4: Verification
- [ ] Compare generated output (old vs new)
- [ ] Run integration tests
- [ ] Test with real database
- [ ] Performance testing
- [ ] Documentation updates

## Rollback Plan

Simple git-based rollback:

1. **Work on feature branch** for template migration
2. **Commit after each template file** migration
3. **If issues arise**: `git revert` the problematic commit
4. **Tests will catch problems** immediately

## Common Pitfalls

### 1. Template Path Issues
- Use `embed.FS` correctly with proper paths
- Test template loading in unit tests
- Use constants for template paths

### 2. Template Syntax Errors
- Validate templates during build
- Add template syntax tests
- Use proper escaping for special characters

### 3. Performance Concerns
- Cache parsed templates
- Avoid reloading templates unnecessarily
- Profile template execution

### 4. Build Issues
- Ensure templates are included in build
- Test with `go build` from different directories
- Verify embed paths are correct

## Success Metrics

- [ ] **Maintainability**: Templates are easier to read and modify
- [ ] **Performance**: No significant performance regression
- [ ] **Functionality**: Generated code is identical to previous version
- [ ] **Developer Experience**: Template editing is improved
- [ ] **Build Process**: Clean builds with no template errors

## Resources

- [Go embed package documentation](https://pkg.go.dev/embed)
- [Go text/template package](https://pkg.go.dev/text/template)
- [Template syntax reference](https://pkg.go.dev/text/template#hdr-Text_and_spaces)

## Completed Work

### Phase 1: Infrastructure Setup (✅ COMPLETED)
**Agent**: Template Migration Agent  
**Date**: Current session  

**Completed Tasks:**
1. ✅ Created `internal/generator/templates/` directory structure:
   - `templates/crud/` - CRUD operation templates
   - `templates/pagination/` - Pagination-related templates  
   - `templates/queries/` - Query generation templates
   - `templates/repository/` - Repository structure templates
   - `templates/shared/` - Shared/common templates

2. ✅ Implemented `TemplateManager` in `template_manager.go`:
   - Template loading and caching system
   - Error handling for template parsing
   - Template execution with data binding
   - Uses Go's `embed.FS` for embedded template files

3. ✅ Created `templates.go` with embed declarations:
   - Embedded template files using `//go:embed` directive
   - Template path constants for type safety
   - Currently embeds `templates/crud/*` and `templates/pagination/*`

4. ✅ Updated `CodeGenerator` to use `TemplateManager`:
   - Added `templateMgr` field to `CodeGenerator` struct
   - Updated constructor to initialize template manager
   - Ready for template usage in code generation

### Phase 2: Template Migration (✅ PARTIALLY COMPLETED)
**Agent**: Template Migration Agent  
**Date**: Current session  

**Completed Migrations:**

1. ✅ **`inline_pagination_templates.go`** → `templates/pagination/*.tmpl`
   - Migrated `inlinePaginationTypesTemplate` → `templates/pagination/pagination_utils.tmpl`
   - Migrated `inlineListTemplate` → `templates/crud/list.tmpl`
   - Migrated `inlineListPaginatedTemplate` → `templates/pagination/inline_paginated.tmpl`
   - **Benefits**: Removed complex string escaping, improved readability, proper template syntax

2. ✅ **`shared_types_templates.go`** → `templates/repository/*.tmpl` + `templates/pagination/*.tmpl`
   - Migrated `repositoryFileTemplate` → `templates/repository/complete_repository.tmpl`
   - Migrated `paginationFileTemplate` → `templates/pagination/shared_types.tmpl`
   - **Benefits**: Eliminated 304 lines of complex inline template strings with extensive backtick escaping
   - **Clean Templates**: No more string concatenation or Go escaping hell

3. ✅ **`query_templates.go`** → `templates/queries/*.tmpl` (431 lines) - **COMPLETED**
   - Migrated `generateQueryResultStruct` template → `templates/queries/result_struct.tmpl`
   - Migrated `generateQueryRepository` template → `templates/queries/repository.tmpl`
   - Migrated `generateOneQueryFunction` template → `templates/queries/one_query.tmpl`
   - Migrated `generateManyQueryFunction` template → `templates/queries/many_query.tmpl`
   - Migrated `generateExecQueryFunction` template → `templates/queries/exec_query.tmpl`
   - Migrated `generatePaginatedQueryFunction` template → `templates/queries/paginated_query.tmpl`
   - **Benefits**: Eliminated 431 lines of complex inline template strings with extensive backtick escaping
   - **Clean Templates**: No more string concatenation or Go escaping hell
   - **Template Manager Integration**: All query functions now use `templateMgr.ExecuteTemplate()`

**Files Created:**
- `internal/generator/templates/pagination/pagination_utils.tmpl` - Pagination types and utility functions
- `internal/generator/templates/crud/list.tmpl` - Simple list operation template
- `internal/generator/templates/pagination/inline_paginated.tmpl` - Paginated list template
- `internal/generator/templates/repository/complete_repository.tmpl` - Complete repository template with all CRUD operations
- `internal/generator/templates/pagination/shared_types.tmpl` - Shared pagination types and utilities
- `internal/generator/templates/queries/result_struct.tmpl` - Query result struct template
- `internal/generator/templates/queries/repository.tmpl` - Query repository struct template
- `internal/generator/templates/queries/one_query.tmpl` - Single row query function template
- `internal/generator/templates/queries/many_query.tmpl` - Multiple row query function template
- `internal/generator/templates/queries/exec_query.tmpl` - Exec query function template
- `internal/generator/templates/queries/paginated_query.tmpl` - Paginated query function template
- `internal/generator/templates/shared/.gitkeep` - Placeholder for future shared templates

**Infrastructure Updates:**
- ✅ Updated `templates.go` embed directive to include `templates/repository/*` and `templates/queries/*`
- ✅ Template path constants ready for new templates
- ✅ Added `TemplateQueryResultStruct`, `TemplateQueryRepository`, `TemplateQueryOne`, `TemplateQueryMany`, `TemplateQueryExec`, `TemplateQueryPaginated` constants
- ✅ All tests passing, build successful

**Important Notes:**
- ⚠️ **Pagination Template Differences**: The migrated `shared_types_templates.go` content differs from `shared_pagination_templates.go`:
  - `shared_types_templates.go` uses public functions (`EncodeCursor`, `DecodeCursor`, `ValidatePaginationParams`)
  - `shared_pagination_templates.go` uses private functions (`encodeCursor`, `decodeCursor`, `validatePaginationParams`)
  - Both files contain similar but not identical pagination logic
  - Next agent should reconcile these differences or determine which approach to use

**Build Status**: ✅ All code compiles successfully

**Testing Status**: ✅ All tests passing including:
- End-to-end system tests
- Query generation tests
- Template generation tests
- Integration tests

**Query Templates Migration Summary:**
- ✅ **6 Templates Migrated**: Successfully migrated all 6 query templates from inline strings to `.tmpl` files
- ✅ **431 Lines Eliminated**: Removed 431 lines of complex inline template strings with extensive backtick escaping
- ✅ **Template Manager Integration**: All query generation functions now use `templateMgr.ExecuteTemplate()`
- ✅ **Clean Template Syntax**: Templates now have proper Go template syntax without string escaping
- ✅ **Zero Regression**: All existing functionality preserved, tests passing

**Technical Implementation:**
- ✅ Created 6 new template files in `templates/queries/`
- ✅ Updated `templates.go` embed directive to include `templates/queries/*`
- ✅ Added 6 new template path constants for type safety
- ✅ Modified all query generation functions to use template manager
- ✅ Removed unused `text/template` import
- ✅ Maintained existing template data structures and logic

## Next Steps

### For Next Agent:
1. **Priority**: Migrate `shared_pagination_templates.go` (169 lines) - **NOTE**: Different from migrated templates - uses private functions and shared approach
2. **Reconcile pagination differences** between migrated `shared_types_templates.go` and `shared_pagination_templates.go`
3. **Test template usage** by updating code generation functions to use `templateMgr.ExecuteTemplate()`
4. **Verify output** matches existing generated code exactly

### Remaining Template Migrations:
1. `shared_pagination_templates.go` → `templates/pagination/shared_*.tmpl` (169 lines) - **NEXT PRIORITY** - **NOTE**: Different from migrated templates - uses private functions and shared approach
2. `crud_templates.go` → `templates/crud/*.tmpl` (110 lines)
3. Remaining inline templates in `codegen.go` and other files

### Original Next Steps:
1. **Review this guide** with the team
2. **Create feature branch** for template migration
3. **Start with Phase 1** (infrastructure setup) ✅ DONE
4. **Migrate one template** as proof of concept ✅ DONE
5. **Iterate and improve** based on learnings 