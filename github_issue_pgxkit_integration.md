# Integrate pgxkit for Enhanced Database Operations and Testing

## 🎯 Overview

Integrate [pgxkit](https://github.com/nhalm/pgxkit) into skimatik's code generation to produce production-ready repositories with enhanced database operations, testing infrastructure, and error handling.

## 🚀 Motivation

Currently, skimatik generates basic pgx-based repositories. By integrating pgxkit, we can generate repositories that include:

- **Production-ready connection pooling** with health checks and monitoring
- **Optimized testing infrastructure** with shared connections for faster tests
- **Structured error handling** with consistent error types
- **Transaction support** with automatic rollback
- **Read/write splitting** capabilities
- **Retry logic** for resilient database operations
- **Type-safe PostgreSQL operations** with comprehensive helpers

## 📋 Current State vs. Desired State

### Current Generated Code
```go
// Basic pgx repository
func NewUsersRepository(conn *pgxpool.Pool) *UsersRepository {
    return &UsersRepository{conn: conn}
}

func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*Users, error) {
    user := &Users{}
    err := r.conn.QueryRow(ctx, getUserByIDQuery, id).Scan(...)
    if err != nil {
        return nil, err  // Basic error handling
    }
    return user, nil
}
```

### Desired Generated Code with pgxkit
```go
// Enhanced pgxkit repository
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

// Transaction support
func (r *UsersRepository) CreateUserWithProfile(ctx context.Context, userParams CreateUsersParams, profileParams CreateProfileParams) (*Users, error) {
    var user *Users
    err := r.conn.WithTransaction(ctx, func(ctx context.Context, tx *sqlc.Queries) error {
        var err error
        user, err = r.Create(ctx, userParams)
        if err != nil {
            return err
        }
        
        profileParams.UserID = user.Id
        _, err = r.profileRepo.Create(ctx, profileParams)
        return err
    })
    return user, err
}
```

## 🛠️ Implementation Plan

### Phase 1: Core Integration
- [ ] Add pgxkit as a dependency to generated projects
- [ ] Update code generation templates to use `pgxkit.Connection` instead of `pgxpool.Pool`
- [ ] Generate repository constructors that accept pgxkit connections
- [ ] Update examples to use pgxkit

### Phase 2: Enhanced Error Handling
- [ ] Generate repositories with pgxkit's structured error types
- [ ] Add error handling templates for common database operations
- [ ] Include proper error context and type checking
- [ ] Generate error handling examples in documentation

### Phase 3: Testing Infrastructure
- [ ] Generate test files that use pgxkit's testing utilities
- [ ] Include `pgxkit.RequireTestDB()` in generated test setup
- [ ] Add `pgxkit.CleanupTestData()` for test isolation
- [ ] Generate integration test examples

### Phase 4: Advanced Features
- [ ] Generate optional methods for read/write splitting
- [ ] Generate optional methods for retry logic
- [ ] Generate optional methods for health checks
- [ ] Generate optional methods for metrics collection

### Phase 5: Documentation & Examples
- [ ] Update README with pgxkit integration examples
- [ ] Generate comprehensive usage documentation
- [ ] Create example projects showing pgxkit features
- [ ] Add migration guide for existing skimatik users

## 🔧 Technical Details

### Configuration Changes
No configuration changes needed - skimatik.yaml stays the same:

```yaml
database:
  dsn: "postgres://user:pass@localhost/mydb"
  schema: "public"

output:
  directory: "./repositories"
  package: "repositories"
```

### Template Updates
Update existing templates in `internal/generator/templates/`:

1. **Repository Template**: Use pgxkit.Connection
2. **Error Handling Template**: Include structured errors
3. **Test Template**: Use pgxkit testing utilities
4. **Constructor Template**: Accept pgxkit connections

### Generated File Structure
```
repositories/
├── pagination.go              # Shared pagination types
├── pgxkit_config.go          # pgxkit configuration helpers (new)
├── users_generated.go         # Enhanced Users repository
├── users_generated_test.go    # pgxkit-based tests (new)
├── posts_generated.go         # Enhanced Posts repository
├── posts_generated_test.go    # pgxkit-based tests (new)
└── ...
```

## 🧪 Testing Strategy

### Unit Tests
- [ ] Test pgxkit integration in code generation
- [ ] Verify generated code compiles with pgxkit
- [ ] Test error handling generation
- [ ] Validate configuration parsing

### Integration Tests
- [ ] Test generated repositories against real database
- [ ] Verify pgxkit testing utilities work correctly
- [ ] Test transaction handling
- [ ] Validate error scenarios

### Performance Tests
- [ ] Compare performance with/without pgxkit
- [ ] Test connection pooling efficiency
- [ ] Measure test execution speed improvements

## 📊 Benefits

### For Generated Code
- **Better Error Handling**: Structured, typed errors instead of generic database errors
- **Production Ready**: Built-in connection pooling, health checks, and monitoring
- **Transaction Support**: Automatic rollback and proper transaction handling
- **Retry Logic**: Resilient operations with configurable retry policies

### For Testing
- **Faster Tests**: Shared connections reduce test setup time
- **Better Isolation**: Automatic cleanup between tests
- **Realistic Testing**: Integration tests against real PostgreSQL

### For Developers
- **Consistent Patterns**: Standardized error handling and connection management
- **Less Boilerplate**: pgxkit handles common database operation patterns
- **Better Debugging**: Enhanced logging and error context

## 🔄 Migration Path

### For Existing Users
1. **Simple Migration**: Just update application code to use pgxkit.Connection
2. **No Config Changes**: Existing skimatik.yaml files work unchanged
3. **Migration Guide**: Provide step-by-step migration instructions
4. **Examples**: Show before/after code comparisons

### Migration Example
```go
// Before: Application code
conn, err := pgxpool.New(ctx, dsn)
repo := repositories.NewUsersRepository(conn)

// After: Application code  
conn, err := pgxkit.NewConnection(ctx, dsn, nil)
repo := repositories.NewUsersRepository(conn)
```

## 📝 Acceptance Criteria

- [ ] Generated repositories use pgxkit.Connection
- [ ] Structured error handling in all generated methods
- [ ] pgxkit testing utilities in generated test files
- [ ] Configuration options for pgxkit features
- [ ] Documentation and examples updated
- [ ] All existing tests pass
- [ ] Performance benchmarks show improvement
- [ ] Migration guide for existing users

## 🔗 Related Issues

- Relates to testing infrastructure improvements
- Enhances error handling capabilities
- Supports production-ready code generation goals

## 💡 Future Enhancements

- **Metrics Integration**: Generate Prometheus metrics for repositories
- **Tracing Support**: Add OpenTelemetry tracing to generated code
- **Advanced Pooling**: Support for multiple connection pools
- **Schema Migrations**: Integration with golang-migrate using pgxkit

---

**Priority**: High
**Effort**: Medium-Large
**Impact**: High

This integration would significantly enhance the quality and production-readiness of generated code while maintaining backward compatibility. 