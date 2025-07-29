# Generated Code Issues - TODO

This file documents issues found in the generated code that need to be addressed in future iterations.

## Current Status
- Date: 2025-01-29
- Location: `example-app/repository/generated/`
- Generated with: skimatik from main branch

## Issues Found

### 1. pgxkit API Compatibility Issues

**Files Affected:**
- `comments_generated.go`
- `posts_generated.go` 
- `users_generated.go`

**Problems:**
- `r.db.Ping undefined (type *pgxkit.DB has no field or method Ping)`
- Method signature mismatches in Update methods

**Example Error:**
```
repository/generated/users_generated.go:269:17: r.db.Ping undefined (type *pgxkit.DB has no field or method Ping)
repository/generated/users_generated.go:295:17: r.db.Ping undefined (type *pgxkit.DB has no field or method Ping)
```

### 2. Update Method Parameter Mismatch

**Files Affected:**
- All generated repository files

**Problem:**
```
not enough arguments in call to r.Update
    have (context.Context, UpdateUsersParams)
    want (context.Context, uuid.UUID, UpdateUsersParams)
```

**Root Cause:** 
The generated `UpdateWithRetry` method is calling `r.Update(ctx, params)` but the actual `Update` method expects `(ctx, id, params)`.

### 3. Unused Import

**Files Affected:**
- `comments_generated.go` (and likely others)

**Problem:**
```
"errors" imported and not used
```

## Impact

### What Works
- Interface compliance tests pass
- Custom repository implementations (like `user_repository.go`) compile correctly
- Type conversions between generated and domain types work
- Basic repository pattern is sound

### What Doesn't Work
- Generated code doesn't compile due to pgxkit API mismatches
- Full application build fails
- Integration testing blocked

## Next Steps

### High Priority
1. **Fix pgxkit.DB API compatibility**
   - Investigate correct method names for pgxkit (likely `Ping(ctx)` instead of `Ping()`)
   - Update generator templates to use correct pgxkit API

2. **Fix Update method parameter passing**
   - Update `UpdateWithRetry` template to pass ID parameter correctly
   - Verify all retry method templates have correct signatures

3. **Clean up unused imports**
   - Update generator to only import packages that are actually used
   - Add import optimization to template processing

### Medium Priority
4. **Add integration tests for generated code**
   - Test against real database once compilation issues are resolved
   - Verify all generated methods work end-to-end

5. **Improve error handling in generated code**
   - Ensure all database errors are properly wrapped
   - Add consistent error context

## Workarounds Used

1. **Stub Implementations**: Created `*_repository_stub.go` files that implement service interfaces with "not implemented" errors
2. **Custom Repositories**: Built `user_repository.go` that embeds generated code (when it compiles) and handles type conversions
3. **Interface Testing**: Verified interface compliance separately from full compilation

## Testing Status

- ✅ Interface compliance: `go build -tags=test -o /dev/null ./repository/interface_test.go`
- ❌ Full compilation: Multiple errors due to pgxkit API issues
- ❌ Integration tests: Blocked by compilation issues

## Files to Investigate

### Generator Templates
- `internal/generator/templates/repository/health_methods.tmpl` (for Ping method)
- `internal/generator/templates/repository/retry_methods.tmpl` (for Update parameter passing)
- Template import handling logic

### Generated Code Analysis
- Compare generated API calls with actual pgxkit documentation
- Test minimal pgxkit example to understand correct API usage
- Verify UUID parameter handling in all CRUD operations

---

**Note**: The architecture and patterns are correct - these are implementation details in the code generation that need refinement. The example-app demonstrates the proper way to build on skimatik-generated code once these issues are resolved. 