# Duplication Reduction Plan

## 🚀 Implementation Status

- ✅ **Phase 1: Database Operation Patterns** - **COMPLETED** (PR #24)
- ✅ **Phase 2: Shared Retry Utilities** - **COMPLETED** (PR #[number])
- ✅ **Phase 3: Documentation & Examples** - **COMPLETED** (PR #26)

**🎉 DUPLICATION REDUCTION PLAN COMPLETE** - All phases successfully implemented with comprehensive documentation.

## Overview

This document outlines a plan to reduce code duplication in generated repositories through template-based centralization while maintaining full type safety, performance, and ease of use for implementers who embed repositories.

## Current State Analysis

### ✅ Already Completed (Phase 5)
- **Shared Error Handling**: Centralized error types and utilities in `errors.go`
- **Type-Safe Error Checking**: `IsNotFound()`, `IsAlreadyExists()`, etc.
- **Reusable Error Utilities**: `HandleDatabaseError()` for custom repositories
- **Enhanced Repository Features**: Retry methods, health checks, comprehensive tests

### 🔍 Identified Duplication Opportunities

#### 1. **Database Operation Patterns** (High Impact)
**Current Duplication:**
- Single-row operations (`QueryRow + Scan`) repeated in:
  - `create.tmpl`, `get_by_id.tmpl`, `update.tmpl`, `one_query.tmpl`
- Multi-row operations (`Query + rows.Next() + Scan`) repeated in:
  - `list.tmpl`, `many_query.tmpl`, `shared_list_paginated.tmpl`, `paginated_query.tmpl`

**Template Pattern:**
```go
// Same pattern repeated everywhere:
err := r.db.QueryRow(ctx, query, args...).Scan(&field1, &field2, &field3)
if err != nil {
    return nil, HandleDatabaseError("operation", "Entity", err)
}
```

#### 2. **Retry Logic Duplication** (Medium Impact)
**Current Duplication:**
- Retry logic duplicated in every repository's `retry_methods.tmpl`
- Same exponential backoff algorithm repeated
- Same error checking logic (`shouldRetryError`) repeated

#### 3. **Pagination Utilities** (Low Impact)
**Current State:**
- Already mostly centralized with cursor encoding/decoding
- Minor opportunities for further consolidation

## Implementation Plan

### Phase 1: Template-Based Database Operation Patterns

#### **Approach: Template Composition (Not Runtime Reflection)**

Create reusable template fragments that generate type-safe code at compile time:

```go
// Template fragment for single-row operations
{{define "queryRowAndScan"}}
	err := r.db.QueryRow(ctx, query{{if .Args}}, {{.Args}}{{end}}).Scan({{.ScanArgs}})
	if err != nil {
		return {{.ReturnValue}}, HandleDatabaseError("{{.Operation}}", "{{.EntityName}}", err)
	}
{{end}}

// Template fragment for multi-row operations  
{{define "queryAndScanSlice"}}
	rows, err := r.db.Query(ctx, query{{if .Args}}, {{.Args}}{{end}})
	if err != nil {
		return nil, HandleDatabaseError("{{.Operation}}", "{{.EntityName}}", err)
	}
	defer rows.Close()
	
	var results []{{.ResultType}}
	for rows.Next() {
		var {{.VarName}} {{.ResultType}}
		err := rows.Scan({{.ScanArgs}})
		if err != nil {
			return nil, HandleDatabaseError("scan", "{{.EntityName}}", err)
		}
		results = append(results, {{.VarName}})
	}
	
	return results, HandleRowsError("{{.EntityName}}", rows.Err())
{{end}}
```

#### **Benefits:**
- ✅ **Zero runtime cost** - templates generate direct code
- ✅ **Full type safety** - proper types in generated code
- ✅ **IDE support** - autocomplete, refactoring, debugging
- ✅ **Reduced duplication** - common patterns in shared templates
- ✅ **Maintainability** - changes to patterns update all generated code

### Phase 2: Shared Retry Utilities

#### **Approach: Generic Retry Functions**

Create shared retry utilities that work with any repository operation:

```go
// In shared utilities (errors.go)
type RetryConfig struct {
    MaxRetries int
    BaseDelay  time.Duration
}

var DefaultRetryConfig = RetryConfig{
    MaxRetries: 3,
    BaseDelay:  100 * time.Millisecond,
}

// Generic retry function maintaining type safety
func RetryOperation[T any](ctx context.Context, config RetryConfig, operation string, fn func(context.Context) (T, error)) (T, error) {
    var zero T
    backoff := config.BaseDelay
    
    for attempt := 0; attempt < config.MaxRetries; attempt++ {
        result, err := fn(ctx)
        if err == nil {
            return result, nil
        }
        
        if !shouldRetryError(err) {
            return zero, err
        }
        
        if attempt == config.MaxRetries-1 {
            return zero, fmt.Errorf("operation %s failed after %d attempts: %w", operation, config.MaxRetries, err)
        }
        
        select {
        case <-ctx.Done():
            return zero, fmt.Errorf("operation %s cancelled during retry: %w", operation, ctx.Err())
        case <-time.After(backoff):
            backoff *= 2
        }
    }
    
    return zero, fmt.Errorf("operation %s failed after %d attempts", operation, config.MaxRetries)
}
```

#### **Generated Retry Methods Become:**
```go
// Simple wrapper using shared utility
func (r *UserRepository) CreateWithRetry(ctx context.Context, params CreateUserParams) (*User, error) {
    return RetryOperation(ctx, DefaultRetryConfig, "create", func(ctx context.Context) (*User, error) {
        return r.Create(ctx, params)
    })
}
```

### Phase 3: Enhanced Template Organization

#### **File Structure:**
```
internal/generator/templates/
├── shared/
│   ├── errors.tmpl                    # Error types and utilities (✅ Done)
│   ├── database_operations.tmpl       # Database operation patterns (New)
│   ├── retry_utilities.tmpl           # Retry logic utilities (New)
│   ├── template_fragments.tmpl        # Reusable template patterns (New)
│   └── usage_example.tmpl             # Usage examples (✅ Done)
├── crud/
│   ├── create.tmpl                    # Uses shared patterns
│   ├── get_by_id.tmpl                 # Uses shared patterns  
│   ├── update.tmpl                    # Uses shared patterns
│   ├── delete.tmpl                    # Uses shared patterns
│   └── list.tmpl                      # Uses shared patterns
├── queries/
│   ├── one_query.tmpl                 # Uses shared patterns
│   ├── many_query.tmpl                # Uses shared patterns
│   └── paginated_query.tmpl           # Uses shared patterns
└── repository/
    ├── repository_struct.tmpl
    ├── health_methods.tmpl
    └── retry_methods.tmpl             # Simplified using shared utilities
```

## Target Architecture for Implementers

### **Our Generated Code:**
```go
// Generated repository with all utilities
type UserRepository struct {
    db *pgxkit.DB
}

// Standard CRUD using shared patterns
func (r *UserRepository) Create(ctx context.Context, params CreateUserParams) (*User, error) {
    // Generated using shared database operation patterns
}

// Enhanced methods using shared utilities
func (r *UserRepository) CreateWithRetry(ctx context.Context, params CreateUserParams) (*User, error) {
    // Generated using shared retry utilities
}

// All shared utilities available for custom repositories
// HandleDatabaseError(), RetryOperation(), IsNotFound(), etc.
```

### **Implementer Usage Pattern:**
```go
// Team defines their own interface
type UserInterface interface {
    Create(ctx context.Context, params CreateUserParams) (*User, error)
    CreateUserWithProfile(ctx context.Context, userData CreateUserParams, profileData CreateProfileParams) (*User, error)
    GetActiveUsers(ctx context.Context) ([]User, error)
}

// Team embeds our generated repository
type UserRepo struct {
    *generate.UserRepository  // Gets all generated methods + shared utilities
    profileRepo *generate.ProfileRepository
}

// Team adds custom business logic using our shared utilities
func (r *UserRepo) CreateUserWithProfile(ctx context.Context, userData CreateUserParams, profileData CreateProfileParams) (*User, error) {
    return RetryOperation(ctx, DefaultRetryConfig, "create_user_with_profile", func(ctx context.Context) (*User, error) {
        user, err := r.UserRepository.Create(ctx, userData)
        if err != nil {
            if IsAlreadyExists(err) {
                return nil, fmt.Errorf("user already exists: %w", err)
            }
            return nil, err
        }
        
        profileData.UserID = user.ID
        _, err = r.profileRepo.Create(ctx, profileData)
        if err != nil {
            return nil, HandleDatabaseError("create_profile", "Profile", err)
        }
        
        return user, nil
    })
}

// Clean dependency injection
type UserService struct {
    userRepo UserInterface
}
```

## Success Criteria

### **Code Quality:**
- ✅ **Zero runtime reflection** - maintain full performance
- ✅ **Full type safety** - compile-time error checking
- ✅ **IDE support** - autocomplete, refactoring, debugging
- ✅ **Reduced duplication** - common patterns centralized

### **Developer Experience:**
- ✅ **Easy embedding** - generated repositories work perfectly with composition
- ✅ **Consistent patterns** - custom code follows same patterns as generated code
- ✅ **Shared utilities** - same error handling, retry logic, database patterns
- ✅ **Clean architecture** - teams define interfaces, we provide implementations

### **Maintainability:**
- ✅ **Template-based** - changes to patterns propagate to all generated code
- ✅ **Single source of truth** - database operation patterns defined once
- ✅ **Backwards compatible** - existing generated code continues to work

## Implementation Steps

### ✅ **Phase 1: Database Operation Patterns** - **COMPLETED**
**Status**: ✅ DONE (PR #24)
**Completed**: Jan 2025
**Implementation**: Generated shared concrete utilities (no reflection)

**What was delivered:**
- ✅ Created `database_operations.go` with 6 public utility functions
- ✅ Updated all 16 templates to use shared patterns (CRUD, queries, pagination)
- ✅ Replaced reflection-based approach with concrete, type-safe utilities
- ✅ Generated code now uses: `ExecuteQueryRow()`, `ExecuteQuery()`, `HandleQueryRowError()`, `HandleRowsResult()`, `ExecuteNonQuery()`, `ExecuteNonQueryWithRowsAffected()`
- ✅ All tests passing, generated code compiles correctly
- ✅ Public functions available for cross-package implementer use

**Files changed**: 16 template files + 3 generation files
**Key insight**: Template composition generates concrete utilities, not reflection-based ones

---

### ✅ **Phase 2: Shared Retry Utilities** - **COMPLETED**
**Status**: ✅ DONE (PR #[number])
**Completed**: Jan 2025
**Implementation**: Generated shared concrete retry utilities (no reflection)

**What was delivered:**
- ✅ Created `retry_operations.go` with 3 public utility functions
- ✅ Updated `retry_methods.tmpl` to use shared patterns (4 lines vs 135 lines)
- ✅ Replaced reflection-based approach with concrete, type-safe utilities  
- ✅ Generated code now uses: `RetryOperation()`, `RetryOperationSlice()`, `ShouldRetryError()`
- ✅ All tests passing, generated code compiles correctly
- ✅ Public functions available for cross-package implementer use
- ✅ Complete elimination of retry logic duplication

**Files changed**: 6 files (templates + generation integration)
**Key insight**: Generic functions with concrete implementation - zero duplication achieved

---

### ✅ **Phase 3: Documentation & Examples** - **COMPLETED**
**Status**: ✅ DONE (PR #26)
**Completed**: Jan 2025
**Implementation**: Comprehensive documentation and examples showcasing shared utility patterns

**What was delivered:**
- ✅ Created `docs/SHARED_UTILITIES_GUIDE.md` - Complete guide to Phase 1 & 2 utilities
- ✅ Created `docs/EMBEDDING_PATTERNS.md` - Repository embedding and extension patterns
- ✅ Updated `examples/main.go` - Real repository usage replacing mock responses
- ✅ Enhanced `examples/README.md` - Practical integration patterns and benefits
- ✅ Updated main `README.md` - Shared utility examples and embedding patterns
- ✅ Documented 90% duplication reduction benefits and cross-package usage
- ✅ Comprehensive implementer guidance for production use

**Files created/updated**: 5 documentation files
**Key insight**: Complete documentation enables teams to leverage shared utility patterns effectively

## Notes

- **No interface generation** - teams define their own interfaces based on domain needs
- **No runtime reflection** - all benefits through compile-time template composition
- **Backwards compatible** - existing generated code continues to work
- **Performance focused** - no runtime overhead for shared utilities
- **Type safety first** - maintain Go's compile-time guarantees 