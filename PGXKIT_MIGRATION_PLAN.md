# pgxkit Migration Plan

**Project**: skimatik - Database-first code generator for PostgreSQL  
**Goal**: Integrate [pgxkit](https://github.com/nhalm/pgxkit) into both the CLI application and generated repositories  
**Status**: Phase 2 Complete  
**Created**: 2025-01-15  
**Last Updated**: 2025-01-15  

## Overview

This document outlines the comprehensive plan for migrating skimatik from direct pgxpool usage to pgxkit integration. The migration will enhance both the CLI application and generated repositories with production-ready features including structured error handling, retry logic, connection management, and testing utilities.

## Current State Analysis

### CLI Application
- **Database Connections**: ✅ **COMPLETED** - Now uses `pgxkit.NewDB()` and `Connect()` in `internal/generator/generator.go`
- **Schema Introspection**: ✅ **COMPLETED** - `internal/generator/introspect.go` uses `*pgxkit.DB`
- **Query Analysis**: ✅ **COMPLETED** - `internal/generator/query_analyzer.go` uses `*pgxkit.DB`
- **Testing**: ✅ **COMPLETED** - `internal/generator/test_helpers.go` uses `pgxkit.RequireDB()`

### Generated Code
- **Repository Structure**: All templates generate repositories with `*pgxpool.Pool` fields
- **Constructor Pattern**: Repository constructors accept `*pgxpool.Pool` parameters
- **Database Operations**: CRUD operations use pgxpool methods directly (`conn.Query`, `conn.QueryRow`, `conn.Exec`)
- **Error Handling**: Basic pgx error handling without structured error types

## Migration Phases

### Phase 1: Add pgxkit Dependencies ✅ COMPLETED
**Goal**: Add pgxkit to the project without breaking existing functionality

**Status**: ✅ **COMPLETED**  
**Completed**: 2025-01-15  
**Duration**: 1 hour  
**Risk Level**: Low  

#### What Was Accomplished
- ✅ **Updated to pgxkit v1.1.0** - Successfully upgraded from old sqlc-focused API to new simple API
- ✅ **Resolved version conflicts** - Worked through Go module proxy caching issues and release management
- ✅ **Verified pgxkit imports work** - Confirmed new API with `NewDB()`, `Connect()`, `RequireDB()` functions
- ✅ **Project compiles successfully** - All builds pass with pgxkit dependency

#### Success Criteria Met
- ✅ pgxkit v1.1.0 dependency added to go.mod
- ✅ No import conflicts or version issues
- ✅ Project compiles successfully with pgxkit dependency

---

### Phase 2: Migrate CLI App to pgxkit ✅ COMPLETED
**Goal**: Update the CLI application to use pgxkit for database connections

**Status**: ✅ **COMPLETED**  
**Completed**: 2025-01-15  
**Duration**: 3 hours  
**Risk Level**: Medium  

#### What Was Accomplished

**1. Updated `internal/generator/generator.go`**
- ✅ **Changed connection logic** - Uses `pgxkit.NewDB()` and `Connect()` instead of `pgxpool.New()`
- ✅ **Updated Generator struct** - Changed `db` field from `*pgxpool.Pool` to `*pgxkit.DB`
- ✅ **Fixed shutdown logic** - Uses `Shutdown(context.Background())` instead of `Close()`
- ✅ **Maintained compatibility** - All existing functionality preserved

**2. Updated `internal/generator/introspect.go`**
- ✅ **Changed interface** - Updated `Introspector` struct to use `*pgxkit.DB`
- ✅ **Updated constructor** - `NewIntrospector()` now accepts `*pgxkit.DB`
- ✅ **Maintained functionality** - All database introspection operations work unchanged

**3. Updated `internal/generator/query_analyzer.go`**
- ✅ **Changed interface** - Updated `QueryAnalyzer` struct to use `*pgxkit.DB`
- ✅ **Fixed transaction calls** - Changed `Begin()` to `BeginTx()` with proper pgx imports
- ✅ **Updated constructor** - `NewQueryAnalyzer()` now accepts `*pgxkit.DB`

**4. Updated `internal/generator/test_helpers.go`**
- ✅ **Leveraged pgxkit test infrastructure** - Uses `pgxkit.RequireDB()` for test database connections
- ✅ **Simplified test setup** - pgxkit handles test database management and skipping automatically
- ✅ **Improved reliability** - Better test database connection handling

**5. Fixed all test files**
- ✅ **Updated integration tests** - Fixed `Close()` calls to use `Shutdown()` in `integration_test.go`
- ✅ **Fixed UUID validation tests** - Updated `uuid_validation_test.go` with proper shutdown calls
- ✅ **All tests passing** - Both unit and integration tests work with pgxkit

#### Technical Implementation Details

**Connection Management**:
```go
// Before (pgxpool)
pool, err := pgxpool.New(context.Background(), dsn)
defer pool.Close()

// After (pgxkit)
db := pgxkit.NewDB()
err := db.Connect(ctx, dsn)
defer db.Shutdown(ctx)
```

**Test Infrastructure**:
```go
// Before (custom test helper)
func getTestDB(t *testing.T) *pgxpool.Pool {
    pool, err := pgxpool.New(context.Background(), dbURL)
    return pool
}

// After (pgxkit test infrastructure)
func getTestDB(t *testing.T) *pgxkit.DB {
    testDB := pgxkit.RequireDB(t)
    return testDB.DB  // TestDB embeds *DB
}
```

#### Success Criteria Met
- ✅ CLI app successfully connects to database using pgxkit
- ✅ Schema introspection works with pgxkit
- ✅ Query analysis works with pgxkit
- ✅ All existing tests pass (100% test suite success)
- ✅ No breaking changes to CLI interface
- ✅ Production-ready connection management

#### Benefits Achieved
- **Better connection management** - pgxkit provides production-ready connection handling
- **Improved test infrastructure** - Built-in test database management with `RequireDB()`
- **Future-ready architecture** - Foundation for pgxkit's advanced features (retry, hooks, health checks)
- **Cleaner API** - Consistent interface with `Connect()`, `Shutdown()`, and proper error handling

---

### Phase 3: Update Code Generation Templates ⏳
**Goal**: Generate repositories that use pgxkit instead of pgxpool

**Status**: Not Started  
**Estimated Time**: 6-8 hours  
**Risk Level**: Medium  

#### Templates to Update
- `internal/generator/templates/repository/repository_struct.tmpl`
- `internal/generator/templates/queries/repository.tmpl`
- All CRUD operation templates (`crud/` directory)
- All query templates (`queries/` directory)
- Pagination templates (`pagination/` directory)

#### Changes Required

**1. Update repository struct template**
```go
// Current: internal/generator/templates/repository/repository_struct.tmpl
type {{.RepositoryName}} struct {
    conn *pgxpool.Pool  // Change this
}

func New{{.RepositoryName}}(conn *pgxpool.Pool) *{{.RepositoryName}} {
    return &{{.RepositoryName}}{
        conn: conn,
    }
}

// New: Use pgxkit.DB
type {{.RepositoryName}} struct {
    db *pgxkit.DB        // Use pgxkit interface
}

func New{{.RepositoryName}}(db *pgxkit.DB) *{{.RepositoryName}} {
    return &{{.RepositoryName}}{
        db: db,
    }
}
```

**2. Update import templates**
```go
// Add new template: internal/generator/templates/shared/imports.tmpl
import (
    "context"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/nhalm/pgxkit"
)
```

**3. Update CRUD operation templates**
```go
// Current: r.conn.Query(ctx, query, args...)
// New: r.db.Query(ctx, query, args...)

// Current: r.conn.QueryRow(ctx, query, args...)
// New: r.db.QueryRow(ctx, query, args...)

// Current: r.conn.Exec(ctx, query, args...)
// New: r.db.Exec(ctx, query, args...)
```

**4. Add pgxkit error handling templates**
```go
// New: internal/generator/templates/shared/error_handling.tmpl

// NotFound error handling
if err != nil {
    if err == pgx.ErrNoRows {
        return nil, pgxkit.NewNotFoundError("{{.StructName}}", id.String())
    }
    return nil, pgxkit.NewDatabaseError("{{.StructName}}", "query", err)
}

// Validation error handling
if err != nil {
    return nil, pgxkit.NewValidationError("{{.StructName}}", "create", "validation", err.Error(), nil)
}
```

#### Tasks
- [ ] Update repository struct template to use pgxkit.DB
- [ ] Update all CRUD templates to use pgxkit methods
- [ ] Update all query templates to use pgxkit methods
- [ ] Update pagination templates to use pgxkit methods
- [ ] Add pgxkit import statements to generated files
- [ ] Add structured error handling templates
- [ ] Test generated code compiles with pgxkit
- [ ] Verify generated repositories work correctly

#### Success Criteria
- ✅ All templates generate pgxkit-compatible code
- ✅ Generated repositories use pgxkit.DB interface
- ✅ Generated code compiles without errors
- ✅ All existing functionality works with pgxkit
- ✅ Structured error handling is implemented

---

### Phase 4: Update Code Generation Logic ✅ COMPLETED
**Goal**: Update the code generator to use pgxkit-compatible imports and patterns

**Status**: ✅ **COMPLETED**  
**Completed**: 2025-01-28  
**Duration**: 1 hour (already implemented)  
**Risk Level**: Low  

#### Files to Modify
- `internal/generator/codegen.go` - Import generation logic
- `internal/generator/templates.go` - Template rendering
- Template data structures

#### Changes Required

**1. Update import generation**
```go
// Add pgxkit imports to generated files
func (cg *CodeGenerator) generateImports() []string {
    imports := []string{
        "context",
        "fmt",
        "github.com/google/uuid",
        "github.com/nhalm/pgxkit",  // Add pgxkit import
    }
    return imports
}
```

**2. Update template data structures**
```go
// Ensure template data includes pgxkit-specific information
type TemplateData struct {
    // ... existing fields ...
    UsePgxkit bool  // Flag to enable pgxkit features
}
```

#### What Was Accomplished

**1. Import Generation Updated**
- ✅ **Table generation** - `generateTableCode()` includes `"github.com/nhalm/pgxkit"` in standardImports
- ✅ **Query generation** - `generateQueryCode()` includes `"github.com/nhalm/pgxkit"` in standardImports
- ✅ **Proper deduplication** - `combineImports()` ensures no duplicate imports

**2. Template Data Structures**
- ✅ **Consistent data flow** - All template data structures already support pgxkit patterns
- ✅ **Repository patterns** - Templates receive correct repository names and database field names
- ✅ **No breaking changes** - Existing template data structure maintained compatibility

**3. Code Generation Logic**
- ✅ **Consistent pgxkit patterns** - All generated code uses `*pgxkit.DB` interface
- ✅ **Proper imports** - Generated files include correct pgxkit imports automatically
- ✅ **Template rendering** - All templates render correctly with pgxkit data

#### Technical Implementation Details

**Import Generation**:
```go
// generateTableCode() - includes pgxkit imports
standardImports := []string{
    "context",
    "fmt",
    "github.com/nhalm/pgxkit",  // ✅ pgxkit import included
    "github.com/google/uuid",
}

// generateQueryCode() - includes pgxkit imports  
standardImports := []string{
    "context",
    "github.com/nhalm/pgxkit",  // ✅ pgxkit import included
    "github.com/google/uuid",
}
```

**Generated Code Structure**:
```go
// Repository struct uses pgxkit.DB
type UserRepository struct {
    db *pgxkit.DB  // ✅ Uses pgxkit interface
}

// Constructor accepts pgxkit.DB
func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{db: db}
}

// Methods use pgxkit database operations
func (r *UserRepository) Create(ctx context.Context, params CreateUserParams) (*User, error) {
    err := r.db.QueryRow(ctx, query, args...).Scan(...)  // ✅ Uses pgxkit methods
    // ...
}
```

#### Success Criteria Met
- ✅ Generated files include correct pgxkit imports (`"github.com/nhalm/pgxkit"`)
- ✅ Template data supports pgxkit features (all templates work correctly)
- ✅ Code generation logic is consistent (both table and query generation)
- ✅ All generated code compiles successfully (verified by tests)
- ✅ Integration tests pass with pgxkit code generation

#### Benefits Achieved
- **Automatic pgxkit imports** - No manual import management needed
- **Consistent interface** - All generated repositories use `*pgxkit.DB`
- **Template compatibility** - Existing templates work seamlessly with pgxkit
- **Test coverage** - Full test suite validates pgxkit code generation

---

### Phase 5: Add Enhanced pgxkit Features ⏳
**Goal**: Leverage pgxkit's enhanced features in generated code

**Status**: Not Started  
**Estimated Time**: 4-6 hours  
**Risk Level**: Low  

#### Enhanced Features to Add
1. **Structured error handling** - pgxkit.NotFoundError, pgxkit.DatabaseError, etc.
2. **Built-in retry logic** - Retry methods for all operations
3. **Connection health checks** - Health check methods
4. **Testing utilities** - pgxkit test helpers

#### New Template Features

**1. Add retry methods to repositories**
```go
// New: internal/generator/templates/repository/retry_methods.tmpl
func (r *{{.RepositoryName}}) CreateWithRetry(ctx context.Context, params Create{{.StructName}}Params) (*{{.StructName}}, error) {
    config := pgxkit.DefaultRetryConfig()
    return pgxkit.RetryOperation(ctx, config, func(ctx context.Context) (*{{.StructName}}, error) {
        return r.Create(ctx, params)
    })
}
```

**2. Add health check methods**
```go
// New: internal/generator/templates/repository/health_methods.tmpl
func (r *{{.RepositoryName}}) HealthCheck(ctx context.Context) error {
    return r.db.HealthCheck(ctx)
}
```

**3. Add test utilities**
```go
// New: internal/generator/templates/tests/pgxkit_test.tmpl
func Test{{.StructName}}Repository(t *testing.T) {
    testDB := pgxkit.RequireDB(t)
    
    repo := New{{.RepositoryName}}(testDB.DB)
    // ... test code ...
}
```

#### Tasks
- [ ] Create retry method templates
- [ ] Create health check method templates
- [ ] Create test utility templates
- [ ] Add structured error handling throughout
- [ ] Test enhanced features work correctly
- [ ] Document new features in generated code

#### Success Criteria
- ✅ Retry methods available for all operations
- ✅ Health check methods implemented
- ✅ Test utilities integrated
- ✅ Structured error handling works
- ✅ Enhanced features are documented

---

### Phase 6: Update Configuration ⏳
**Goal**: Add pgxkit configuration options

**Status**: Not Started  
**Estimated Time**: 2-3 hours  
**Risk Level**: Low  

#### Files to Modify
- `internal/generator/config.go` - Add pgxkit config options
- CLI flag parsing - Add pgxkit-specific flags

#### New Configuration Options
```yaml
# skimatik.yaml
database:
  dsn: "postgres://..."
  schema: "public"
  pgxkit:
    enabled: true
    retry:
      enabled: true
      max_retries: 3
    health_checks: true
    testing: true
```

#### Tasks
- [ ] Add pgxkit configuration struct
- [ ] Add CLI flags for pgxkit options
- [ ] Update configuration validation
- [ ] Test configuration parsing
- [ ] Document configuration options

#### Success Criteria
- ✅ pgxkit configuration options available
- ✅ CLI flags work correctly
- ✅ Configuration validation includes pgxkit options
- ✅ Documentation updated

---

### Phase 7: Update Documentation and Examples ⏳
**Goal**: Update all documentation to reflect pgxkit integration

**Status**: Not Started  
**Estimated Time**: 3-4 hours  
**Risk Level**: Low  

#### Files to Update
- `README.md` - Add pgxkit integration section
- `examples/` - Update example application (when ready)
- `docs/` - Update all documentation

#### Documentation Updates
1. **Installation instructions** - Include pgxkit dependency
2. **Usage examples** - Show pgxkit connection patterns
3. **Configuration reference** - Document pgxkit options
4. **Migration guide** - Help users migrate from pgxpool to pgxkit

#### Tasks
- [ ] Update README with pgxkit integration
- [ ] Update installation instructions
- [ ] Add usage examples with pgxkit
- [ ] Document configuration options
- [ ] Create migration guide
- [ ] Update API documentation

#### Success Criteria
- ✅ README includes pgxkit integration
- ✅ Installation instructions updated
- ✅ Usage examples show pgxkit patterns
- ✅ Configuration documented
- ✅ Migration guide available

---

### Phase 8: Testing and Validation ⏳
**Goal**: Ensure pgxkit integration works correctly

**Status**: Not Started  
**Estimated Time**: 4-6 hours  
**Risk Level**: Medium  

#### Test Updates
1. **Update existing tests** - Use pgxkit connections
2. **Add pgxkit-specific tests** - Test retry logic, error handling
3. **Integration tests** - Test full workflow with pgxkit
4. **Performance tests** - Ensure no regressions

#### Tasks
- [ ] Update all existing tests to use pgxkit
- [ ] Add tests for pgxkit-specific features
- [ ] Add integration tests for full workflow
- [ ] Add performance regression tests
- [ ] Test error handling scenarios
- [ ] Test retry logic
- [ ] Test health checks

#### Success Criteria
- ✅ All existing tests pass with pgxkit
- ✅ pgxkit-specific features are tested
- ✅ Integration tests pass
- ✅ No performance regressions
- ✅ Error handling works correctly

---

## Implementation Strategy

### Implementation Order
1. **Phase 1**: Dependencies ✅ **COMPLETED** (foundational, low risk)
2. **Phase 2**: CLI app migration ✅ **COMPLETED** (core functionality, medium risk)
3. **Phase 3**: Template updates (affects generated code, medium risk)
4. **Phase 4**: Code generation logic (supporting changes, low risk)
5. **Phase 5**: Enhanced features (additive, low risk)
6. **Phase 6**: Configuration (optional, low risk)
7. **Phase 7**: Documentation (user-facing, low risk)
8. **Phase 8**: Testing (validation, ongoing)

### Backward Compatibility Strategy
- Consider adding a configuration flag to enable/disable pgxkit
- Maintain pgxpool support during transition period
- Provide migration tools/scripts for existing users
- Ensure generated code remains compatible with existing applications

### Risk Mitigation
- **Medium Risk Phases**: Create feature branches and thorough testing
- **Database Connection Changes**: Test with multiple PostgreSQL versions
- **Template Changes**: Validate generated code compiles and works
- **Breaking Changes**: Document all breaking changes and provide migration paths

## Success Criteria

### Overall Success Metrics
- ✅ CLI app successfully uses pgxkit for database connections
- ✅ Generated repositories use pgxkit.DB interface (Phase 3)
- ✅ All existing functionality works with pgxkit
- ✅ New pgxkit features are available in generated code (Phase 5)
- ✅ Performance is maintained or improved
- ✅ All tests pass with pgxkit integration
- ✅ Documentation is updated and accurate (Phase 7)
- ✅ Backward compatibility is maintained where possible

### Quality Gates
- **Code Quality**: All generated code passes linting and formatting
- **Test Coverage**: Maintain or improve test coverage
- **Performance**: No significant performance regressions
- **Documentation**: All features are documented with examples
- **User Experience**: Migration is smooth for existing users

## Progress Tracking

### Completed Phases
- ✅ **Phase 1: Add pgxkit Dependencies** - Successfully upgraded to pgxkit v1.1.0
- ✅ **Phase 2: Migrate CLI App to pgxkit** - All CLI components now use pgxkit
- ✅ **Phase 3: Update Code Generation Templates** - All templates use pgxkit.DB interface
- ✅ **Phase 4: Update Code Generation Logic** - Import generation and template data structures complete

### Current Phase
**Phase 5: Add Enhanced pgxkit Features**

### Next Actions
1. Create retry method templates for all operations
2. Create health check method templates  
3. Add structured error handling throughout generated code
4. Create test utility templates with pgxkit.RequireDB()

### Notes and Decisions
- **2025-01-15**: Initial migration plan created
- **2025-01-15**: Phase 1 completed - pgxkit v1.1.0 successfully integrated
- **2025-01-15**: Phase 2 completed - CLI app fully migrated to pgxkit
- **Decision**: Use pgxkit.DB interface throughout for flexibility
- **Decision**: Maintain backward compatibility during transition
- **Decision**: Add enhanced features as optional/additive
- **Discovery**: pgxkit v1.1.0 has much simpler API than initially documented
- **Success**: All tests passing with pgxkit integration

---

*This document will be updated as phases are completed and new requirements are discovered.* 