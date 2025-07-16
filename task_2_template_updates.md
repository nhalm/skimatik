# Task 2: Update Code Generation Templates for pgxkit Integration

**Parent Issue**: #4 - Integrate pgxkit for Enhanced Database Operations and Testing
**Depends On**: #5 - Task 1: Add pgxkit Dependency and Update Project Structure

## 🎯 Objective

Update the code generation templates to generate repositories that use pgxkit.Connection instead of pgxpool.Pool and include pgxkit's enhanced features.

## 📋 Tasks

### 1. Update Repository Template
- [ ] Modify `internal/generator/templates/repository/complete_repository.tmpl` to:
  - Use `*pgxkit.Connection` instead of `*pgxpool.Pool`
  - Update constructor to accept pgxkit.Connection
  - Add pgxkit import statements

### 2. Update Constructor Generation
- [ ] Update repository constructor template:
  ```go
  func New{{.TableName}}Repository(conn *pgxkit.Connection) *{{.TableName}}Repository {
      return &{{.TableName}}Repository{conn: conn}
  }
  ```

### 3. Update Error Handling Templates
- [ ] Create new error handling template `internal/generator/templates/errors/pgxkit_errors.tmpl`
- [ ] Update CRUD operation templates to use pgxkit structured errors:
  ```go
  if err != nil {
      if errors.Is(err, pgx.ErrNoRows) {
          return nil, pgxkit.NewNotFoundError("{{.TableName}}", id.String())
      }
      return nil, pgxkit.NewDatabaseError("{{.TableName}}", "query", err)
  }
  ```

### 4. Add Transaction Support Templates
- [ ] Create transaction template `internal/generator/templates/transactions/pgxkit_transactions.tmpl`
- [ ] Generate transaction helper methods:
  ```go
  func (r *{{.TableName}}Repository) WithTransaction(ctx context.Context, fn func(context.Context, *pgxkit.Connection) error) error {
      return r.conn.WithTransaction(ctx, fn)
  }
  ```

### 5. Update Import Generation
- [ ] Update import generation in `internal/generator/codegen.go` to include:
  - `github.com/nhalm/pgxkit`
  - `errors` package for error handling
  - Remove unnecessary pgxpool imports when pgxkit is enabled

### 6. Update All Templates
- [ ] Ensure all templates consistently use pgxkit.Connection
- [ ] Add template helper functions for pgxkit-specific features
- [ ] Remove any pgxpool references from templates

## 🧪 Testing Requirements

### Unit Tests
- [ ] Test template rendering with pgxkit
- [ ] Test generated code compilation with pgxkit
- [ ] Test error handling template generation
- [ ] Test transaction template generation

### Integration Tests
- [ ] Test generated repositories work with pgxkit.Connection
- [ ] Test error handling in generated code
- [ ] Test transaction support in generated code
- [ ] Test all generated code compiles and runs

## 📝 Acceptance Criteria

- [ ] All templates generate pgxkit-compatible code
- [ ] Generated repositories use pgxkit.Connection
- [ ] Error handling uses pgxkit structured errors
- [ ] Transaction support is included in generated code
- [ ] All tests pass
- [ ] Generated code compiles without errors

## 🔧 Implementation Notes

### Template Structure Updates
```
internal/generator/templates/
├── repository/
│   └── complete_repository.tmpl      # Updated for pgxkit
├── errors/
│   └── pgxkit_errors.tmpl            # New error handling
├── transactions/
│   └── pgxkit_transactions.tmpl      # New transaction support
└── imports/
    └── pgxkit_imports.tmpl           # New import management
```

### Generated Code Example
```go
// Generated repository with pgxkit
type UsersRepository struct {
    conn *pgxkit.Connection
}

func NewUsersRepository(conn *pgxkit.Connection) *UsersRepository {
    return &UsersRepository{conn: conn}
}

func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*Users, error) {
    user := &Users{}
    err := r.conn.QueryRow(ctx, getUserByIDQuery, id).Scan(...)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, pgxkit.NewNotFoundError("User", id.String())
        }
        return nil, pgxkit.NewDatabaseError("User", "query", err)
    }
    return user, nil
}
```

### Simplified Template Usage
```go
// In codegen.go - no conditional logic needed
func (g *Generator) generateRepository(table Table) error {
    // Always use pgxkit templates
    return g.renderTemplate("repository/complete_repository.tmpl", table)
}
```

## 🔗 Related Files

- `internal/generator/templates/repository/complete_repository.tmpl`
- `internal/generator/codegen.go`
- `internal/generator/generator.go`
- `internal/generator/templates/` (various template files)

## 📚 Resources

- [pgxkit Connection API](https://github.com/nhalm/pgxkit/blob/main/connection.go)
- [pgxkit Error Types](https://github.com/nhalm/pgxkit/blob/main/errors.go)
- [pgxkit Transaction Support](https://github.com/nhalm/pgxkit/blob/main/examples.md#transaction-support)

---

**Priority**: High
**Effort**: Medium
**Dependencies**: Task 1 (pgxkit dependency)
**Next Task**: Task 3 - Generate Test Files with pgxkit Testing Utilities 