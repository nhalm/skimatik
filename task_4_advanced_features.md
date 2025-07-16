# Task 4: Add Advanced pgxkit Features (Retry Logic, Read/Write Splitting, Health Checks)

**Parent Issue**: #4 - Integrate pgxkit for Enhanced Database Operations and Testing
**Depends On**: #7 - Task 3: Generate Test Files with pgxkit Testing Utilities

## 🎯 Objective

Add advanced pgxkit features to generated repositories including retry logic, read/write splitting, health checks, and metrics collection. These features will be available as optional methods in generated repositories - applications can choose which features to use.

## 📋 Tasks

### 1. Add Retry Logic Support
- [ ] Generate retry-enabled repository methods (optional usage)
- [ ] Add retry logic to CRUD operations using pgxkit's retry features
- [ ] Generate examples showing retry usage

### 2. Implement Read/Write Splitting
- [ ] Generate repositories that can accept separate read/write connections
- [ ] Route read operations to read connections
- [ ] Route write operations to write connections
- [ ] Generate examples showing read/write splitting usage

### 3. Add Health Check Support
- [ ] Generate health check methods for repositories
- [ ] Add database connectivity validation
- [ ] Generate health check endpoints
- [ ] Add connection pool monitoring

### 4. Implement Metrics Collection
- [ ] Add metrics configuration options
- [ ] Generate metrics collection hooks
- [ ] Add query performance tracking
- [ ] Generate metrics export functionality

### 5. Add Connection Management Features
- [ ] Generate connection pool configuration
- [ ] Add connection lifecycle management
- [ ] Generate connection monitoring utilities
- [ ] Add graceful shutdown support

## 🧪 Testing Requirements

### Unit Tests
- [ ] Test retry logic configuration
- [ ] Test read/write splitting logic
- [ ] Test health check functionality
- [ ] Test metrics collection

### Integration Tests
- [ ] Test retry behavior with database failures
- [ ] Test read/write splitting with real connections
- [ ] Test health checks against real database
- [ ] Test metrics collection accuracy

## 📝 Acceptance Criteria

- [ ] Retry logic is configurable and works correctly
- [ ] Read/write splitting routes operations appropriately
- [ ] Health checks accurately reflect database status
- [ ] Metrics collection provides useful performance data
- [ ] All features are optional and configurable
- [ ] Generated code includes proper error handling

## 🔧 Implementation Notes

### Application Usage (No Config Changes)
```yaml
# Configuration stays simple
database:
  dsn: "postgres://user:pass@localhost/mydb"
  schema: "public"
```

```go
// Applications choose which features to use
func main() {
    // Basic usage
    conn, err := pgxkit.NewConnection(ctx, dsn, nil)
    repo := repositories.NewUsersRepository(conn)
    
    // OR with retry
    retryConn := conn.WithRetry(&pgxkit.RetryConfig{MaxRetries: 3})
    repo := repositories.NewUsersRepository(retryConn)
    
    // OR with read/write splitting
    readConn, _ := pgxkit.NewConnection(ctx, readDSN, nil)
    writeConn, _ := pgxkit.NewConnection(ctx, writeDSN, nil)
    repo := repositories.NewUsersRepositoryWithReadWrite(readConn, writeConn)
}
```

### Generated Repository with Retry Logic
```go
// Generated repository with retry support
func (r *UsersRepository) GetByIDWithRetry(ctx context.Context, id uuid.UUID) (*Users, error) {
    retryableConn := r.conn.WithRetry(nil) // Use default retry config
    
    user := &Users{}
    err := retryableConn.QueryRow(ctx, getUserByIDQuery, id).Scan(...)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, pgxkit.NewNotFoundError("User", id.String())
        }
        return nil, pgxkit.NewDatabaseError("User", "query", err)
    }
    return user, nil
}
```

### Read/Write Splitting Example
```go
// Generated repository with read/write splitting
type UsersRepository struct {
    readConn  *pgxkit.Connection
    writeConn *pgxkit.Connection
}

func NewUsersRepository(readConn, writeConn *pgxkit.Connection) *UsersRepository {
    return &UsersRepository{
        readConn:  readConn,
        writeConn: writeConn,
    }
}

func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*Users, error) {
    // Use read connection for SELECT operations
    user := &Users{}
    err := r.readConn.QueryRow(ctx, getUserByIDQuery, id).Scan(...)
    // ... error handling
    return user, nil
}

func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    // Use write connection for INSERT operations
    user := &Users{}
    err := r.writeConn.QueryRow(ctx, createUserQuery, params.Name, params.Email).Scan(...)
    // ... error handling
    return user, nil
}
```

### Health Check Implementation
```go
// Generated health check methods
func (r *UsersRepository) HealthCheck(ctx context.Context) error {
    if !r.conn.IsReady(ctx) {
        return pgxkit.NewDatabaseError("User", "health_check", errors.New("database not ready"))
    }
    return nil
}

func (r *UsersRepository) GetConnectionStats() pgxpool.Stat {
    return r.conn.Stats()
}
```

### Metrics Collection
```go
// Generated metrics hooks
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    start := time.Now()
    defer func() {
        r.metrics.RecordQuery("users", "create", time.Since(start))
    }()
    
    // ... actual implementation
}
```

### Template Structure Updates
```
internal/generator/templates/
├── advanced/
│   ├── retry_logic.tmpl          # Retry functionality
│   ├── read_write_split.tmpl     # Read/write splitting
│   ├── health_checks.tmpl        # Health check methods
│   └── metrics.tmpl              # Metrics collection
├── repository/
│   ├── advanced_repository.tmpl  # Repository with all features
│   └── basic_repository.tmpl     # Fallback basic repository
└── config/
    └── pgxkit_config.tmpl        # Configuration helpers
```

### Feature Flags
```go
// In generator configuration
type PgxkitConfig struct {
    Enabled     bool                `yaml:"enabled"`
    Retry       RetryConfig         `yaml:"retry"`
    ReadWrite   ReadWriteConfig     `yaml:"read_write_splitting"`
    HealthCheck HealthCheckConfig   `yaml:"health_checks"`
    Metrics     MetricsConfig       `yaml:"metrics"`
}

// Generate features based on configuration
func (g *Generator) generateAdvancedFeatures() {
    if g.config.Database.Pgxkit.Retry.Enabled {
        g.generateRetryLogic()
    }
    if g.config.Database.Pgxkit.ReadWrite.Enabled {
        g.generateReadWriteSplitting()
    }
    // ... other features
}
```

## 🔗 Related Files

- `internal/config/config.go` - Configuration extensions
- `internal/generator/templates/advanced/` - New template directory
- `internal/generator/generator.go` - Feature generation logic
- `internal/generator/codegen.go` - Code generation updates

## 📚 Resources

- [pgxkit Retry Logic](https://github.com/nhalm/pgxkit/blob/main/retry.go)
- [pgxkit Read/Write Splitting](https://github.com/nhalm/pgxkit/blob/main/readwrite.go)
- [pgxkit Health Checks](https://github.com/nhalm/pgxkit/blob/main/connection.go)
- [pgxkit Metrics](https://github.com/nhalm/pgxkit/blob/main/hooks.go)

---

**Priority**: Medium
**Effort**: Large
**Dependencies**: Task 3 (test generation)
**Next Task**: Task 5 - Documentation and Examples 