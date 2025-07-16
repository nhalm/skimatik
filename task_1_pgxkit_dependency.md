# Task 1: Add pgxkit Dependency and Update Project Structure

**Parent Issue**: #4 - Integrate pgxkit for Enhanced Database Operations and Testing

## 🎯 Objective

Add pgxkit as a dependency to the skimatik project. No configuration changes needed - pgxkit will be the default database interface for all generated code.

## 📋 Tasks

### 1. Add pgxkit Dependency
- [ ] Add `github.com/nhalm/pgxkit` to `go.mod`
- [ ] Run `go mod tidy` to ensure clean dependencies
- [ ] Verify pgxkit imports work correctly

### 2. Update Import Generation
- [ ] Update `internal/generator/codegen.go` to include pgxkit imports in generated files
- [ ] Remove pgxpool imports from generated code
- [ ] Add necessary error handling imports (`errors` package)

### 3. Update Example Projects
- [ ] Update `examples/main.go` to use pgxkit.Connection
- [ ] Update example documentation to show pgxkit usage
- [ ] Ensure examples compile and run with pgxkit

## 🧪 Testing Requirements

### Unit Tests
- [ ] Test that pgxkit imports are correctly added to generated code
- [ ] Test that pgxpool imports are removed from generated code
- [ ] Test that generated code compiles with pgxkit

### Integration Tests
- [ ] Test that pgxkit dependency is properly imported
- [ ] Test that examples work with pgxkit
- [ ] Test that generated repositories work with pgxkit.Connection

## 📝 Acceptance Criteria

- [ ] pgxkit dependency is added to go.mod
- [ ] Generated code uses pgxkit.Connection instead of pgxpool.Pool
- [ ] All imports are correctly updated
- [ ] Examples are updated to use pgxkit
- [ ] All tests pass
- [ ] Generated code compiles without errors

## 🔧 Implementation Notes

### Configuration Stays Simple
```yaml
# No changes needed - configuration stays the same
database:
  dsn: "postgres://user:pass@localhost/mydb"
  schema: "public"
```

### Generated Code Changes
```go
// OLD: Generated with pgxpool
func NewUsersRepository(conn *pgxpool.Pool) *UsersRepository

// NEW: Generated with pgxkit
func NewUsersRepository(conn *pgxkit.Connection) *UsersRepository
```

## 🔗 Related Files

- `go.mod` - Add pgxkit dependency
- `internal/generator/codegen.go` - Update import generation
- `examples/main.go` - Update example usage
- `internal/generator/templates/` - Template files (updated in Task 2)

## 📚 Resources

- [pgxkit Repository](https://github.com/nhalm/pgxkit)
- [pgxkit Documentation](https://github.com/nhalm/pgxkit/blob/main/README.md)
- [pgxkit Examples](https://github.com/nhalm/pgxkit/blob/main/examples.md)

---

**Priority**: High
**Effort**: Small
**Dependencies**: None
**Next Task**: Task 2 - Update Code Generation Templates 