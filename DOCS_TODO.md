# Documentation TODO List

Generated: 2025-01-07

## Priority 0 - CRITICAL (Blocks Users)

### 1. Fix UPDATE Query Bug - CODE BUG
**Location**: `internal/generator/codegen.go` (generated code)
**Issue**: Generated UPDATE queries have `SET <no value>` which causes SQL syntax errors
**Files affected**:
- `users_generated.go:101`
- `posts_generated.go:104`
**Action**: Fix the template or code generation logic that creates UPDATE statements

### 2. Fix Type Mapping Documentation
**Location**: Multiple docs
**Issue**: Documentation claims skimatik generates native Go types (`*string` for nullable), but actual code uses `pgtype.Text`, `pgtype.Int4`, etc.

**Files to fix**:
- `/Users/nhalm/dev/skimatik/docs/type-mapping.md` - Lines 21, 24 claim native types
- `/Users/nhalm/dev/skimatik/docs/quick-start.md` - Lines 149-206 show examples with wrong types
- `/Users/nhalm/dev/skimatik/docs/examples.md` - Lines 88-104 and others

**Decision needed**:
- Option A: Update docs to show `pgtype` types
- Option B: Change generator to produce native types with custom scanner/valuer

### 3. Add Missing Imports to All Go Examples
**Issue**: All code examples use `uuid.UUID` but don't import `"github.com/google/uuid"`
**Files to fix**:
- `/Users/nhalm/dev/skimatik/docs/quick-start.md`
- `/Users/nhalm/dev/skimatik/docs/examples.md`
- `/Users/nhalm/dev/skimatik/docs/error-handling.md`
- `/Users/nhalm/dev/skimatik/docs/embedding-patterns.md`
- `/Users/nhalm/dev/skimatik/docs/shared-utilities.md`

Add to all examples:
```go
import (
    "github.com/google/uuid"
    // ... other imports
)
```

## Priority 1 - HIGH (Code Won't Work)

### 4. Fix Private Field Access in Examples
**Location**: Multiple docs
**Issue**: Examples show `s.db` but `db` field in repositories is private (lowercase)
**Files**:
- `/Users/nhalm/dev/skimatik/docs/quick-start.md:280`
- `/Users/nhalm/dev/skimatik/docs/shared-utilities.md:135`

**Fix**: Use public methods or show correct embedding patterns

### 5. Fix ExecuteQueryRow Signature in Examples
**Location**: `/Users/nhalm/dev/skimatik/docs/shared-utilities.md:135`
**Issue**: Missing `operation` and `entity` string parameters
**Actual signature**:
```go
func ExecuteQueryRow(ctx context.Context, db *pgxkit.DB, operation, entity, query string, args ...interface{}) pgx.Row
```

### 6. Fix Error Constructor Exports
**Location**: `/Users/nhalm/dev/skimatik/docs/error-handling.md:18-47`
**Issue**: Shows `NewNotFoundError()`, `NewAlreadyExistsError()` but these aren't exported
**Actual**: Only `DatabaseError` struct and `HandleDatabaseError` function are exported

### 7. Fix Type Conversion Errors
**Location**: `/Users/nhalm/dev/skimatik/docs/examples.md:499-503`
**Issue**: Shows `uuid.UUID(result.Id.Bytes)` but `result.Id` is already `uuid.UUID`

## Priority 2 - MEDIUM (UX/Organization)

### 8. Extract Blog App Tutorial
**Action**: Create new file `/Users/nhalm/dev/skimatik/docs/tutorial-blog-app.md`
**Content**: Extract lines 350-577 from `examples.md`
**Update examples.md**: Keep lines 1-348 (API testing patterns only)

### 9. Remove Duplicate Content
**Files**:
- `home.md` lines 40-79 duplicate quick-start.md
- `home.md` lines 24-39 duplicate sidebar navigation

**Action**:
- Remove Quick Start section from home.md (lines 40-79)
- Remove "Documentation Navigation" section (lines 24-39)
- Keep home.md as overview only

### 10. Create Troubleshooting Guide
**Action**: Create `/Users/nhalm/dev/skimatik/docs/troubleshooting.md`

**Content to include**:
```markdown
# Troubleshooting

## Configuration Errors

### "database.dsn is required"
**Cause**: Missing DSN in config or environment
**Fix**: Add database.dsn to skimatik.yaml or set DATABASE_URL

### "YAML parse error"
**Cause**: Invalid YAML syntax or wrong field names
**Fix**: Check field names match FileConfig struct

## Schema Errors

### "Table does not have UUID primary key"
**Cause**: Table primary key is not UUID type
**Fix**: Alter table to use UUID v7 primary key

### "Table 'X' not found in schema 'public'"
**Cause**: Table doesn't exist or wrong schema
**Fix**: Check schema configuration

## Generation Errors

### "Failed to analyze query"
**Cause**: SQL syntax error in query file
**Fix**: Validate SQL syntax

### "SET <no value>" in generated UPDATE
**Cause**: Known bug (see issue #XXX)
**Fix**: [workaround or fix coming]

## Compilation Errors

### "undefined: uuid"
**Cause**: Missing import
**Fix**: Add `import "github.com/google/uuid"`

### "cannot use string as pgtype.Text"
**Cause**: Type mismatch with nullable fields
**Fix**: Use pgtype types for nullable fields
```

### 11. Merge Repository Patterns Documentation
**Action**: Merge `embedding-patterns.md` and `shared-utilities.md` into `repository-patterns.md`

**Structure**:
```markdown
# Repository Patterns

## Why Composition Over Inheritance (5 min read)
[from embedding-patterns.md]

## Generated Repository Features (5 min read)
- What's in the generated code
- CRUD operations
- Pagination
- Error handling

## Available Utilities (10 min read)
[from shared-utilities.md - utility reference]

## Composition Patterns (15 min read)
[from embedding-patterns.md - examples]

## Complete Examples (20 min read)
[combine best examples from both]
```

### 12. Move Migration Setup in Quick Start
**Location**: `/Users/nhalm/dev/skimatik/docs/quick-start.md` lines 46-93
**Action**: Move to end of document as "Optional: Database Migrations"
**Rationale**: Don't block first-time setup with migration complexity

## Priority 3 - LOW (Polish)

### 13. Standardize Error Handling Patterns
**Issue**: Three different patterns shown for same operation
**Action**: Pick one pattern and use consistently

### 14. Fix Context Variable Shadowing
**Location**: `/Users/nhalm/dev/skimatik/docs/examples.md:212-235`
**Issue**: Inner `ctx` parameter shadows outer `ctx`

### 15. Add Compilation Tests for Docs
**Action**: Create CI test that validates all Go code examples compile
**Files**: Create `docs/examples_test.go` that includes all code snippets

## Completed ✅

1. ✅ Fix configuration-reference.md YAML structure
2. ✅ Update all YAML examples in quick-start.md
3. ✅ Fix YAML examples in other docs
4. ✅ Fix CLI flags documentation (only 4 exist)
5. ✅ Fix environment variables (only DATABASE_URL and POSTGRES_*)
6. ✅ Remove fake validation commands
7. ✅ Fix broken documentation links

## Notes

- Configuration docs are now 100% accurate to actual FileConfig struct
- All YAML examples use correct field names
- CLI flags match actual implementation (4 flags only)
- Environment variables match actual code

## Agent Analysis Reports

Full reports available from agent runs:
- Configuration accuracy analysis
- Go code example validation
- UX/organization review
- Type mapping discrepancy report

These contain detailed file:line references for all issues.
