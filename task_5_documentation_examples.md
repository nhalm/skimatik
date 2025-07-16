# Task 5: Documentation and Examples for pgxkit Integration

**Parent Issue**: #4 - Integrate pgxkit for Enhanced Database Operations and Testing
**Depends On**: #8 - Task 4: Add Advanced pgxkit Features

## 🎯 Objective

Update documentation and create comprehensive examples showing how to use skimatik with pgxkit integration, including migration guides and best practices.

## 📋 Tasks

### 1. Update README.md
- [ ] Add pgxkit integration section to main README
- [ ] Update installation instructions to include pgxkit
- [ ] Add configuration examples with pgxkit options
- [ ] Update usage examples to show pgxkit features
- [ ] Add performance benefits section

### 2. Create Migration Guide
- [ ] Create `docs/PGXKIT_MIGRATION.md` with step-by-step migration instructions
- [ ] Document configuration changes needed
- [ ] Provide before/after code examples
- [ ] Include troubleshooting section
- [ ] Add performance comparison data

### 3. Create Example Projects
- [ ] Create `examples/pgxkit-basic/` - Basic pgxkit usage
- [ ] Create `examples/pgxkit-advanced/` - Advanced features demo
- [ ] Create `examples/pgxkit-testing/` - Testing utilities demo
- [ ] Create `examples/pgxkit-production/` - Production-ready setup
- [ ] Include working code and documentation for each example

### 4. Update Configuration Documentation
- [ ] Document all pgxkit configuration options
- [ ] Add configuration validation rules
- [ ] Provide configuration templates
- [ ] Include environment variable documentation
- [ ] Add troubleshooting for common configuration issues

### 5. Create API Documentation
- [ ] Document generated repository methods with pgxkit
- [ ] Create API reference for pgxkit features
- [ ] Document error handling patterns
- [ ] Add code examples for each feature
- [ ] Include best practices guide

### 6. Update CLI Documentation
- [ ] Document new CLI flags for pgxkit
- [ ] Update help text and usage examples
- [ ] Add command-line configuration examples
- [ ] Document flag precedence and overrides

## 🧪 Testing Requirements

### Documentation Tests
- [ ] Test all code examples in documentation
- [ ] Verify example projects compile and run
- [ ] Test configuration examples are valid
- [ ] Validate links and references

### Example Project Tests
- [ ] Add CI tests for all example projects
- [ ] Test examples against real databases
- [ ] Verify performance claims in documentation
- [ ] Test migration guide steps

## 📝 Acceptance Criteria

- [ ] README.md is updated with pgxkit integration information
- [ ] Migration guide is complete and tested
- [ ] Example projects demonstrate all key features
- [ ] Configuration documentation is comprehensive
- [ ] API documentation covers all generated methods
- [ ] CLI documentation is updated
- [ ] All code examples work correctly

## 🔧 Implementation Notes

### Updated README Structure
```markdown
# skimatik

## Features
- Database-First Code Generation
- Type-Safe Go Repositories
- Built-in Pagination
- **pgxkit Integration** (NEW)
  - Production-ready connection pooling
  - Advanced error handling
  - Testing utilities
  - Retry logic and health checks

## Installation
```bash
go install github.com/nhalm/skimatik/cmd/skimatic@latest
```

## Quick Start with pgxkit
```go
// With pgxkit integration
conn, err := pgxkit.NewConnection(ctx, dsn, nil)
if err != nil {
    log.Fatal(err)
}

userRepo := repositories.NewUsersRepository(conn)
```

## Configuration
```yaml
database:
  dsn: "postgres://user:pass@localhost/mydb"
  pgxkit:
    enabled: true
    max_conns: 20
    enable_retry: true
```
```

### Migration Guide Structure
```markdown
# Migrating to pgxkit Integration

## Overview
This guide helps you migrate from basic pgx repositories to pgxkit-enhanced repositories.

## Step 1: Update Configuration
Before:
```yaml
database:
  dsn: "postgres://user:pass@localhost/mydb"
```

After:
```yaml
database:
  dsn: "postgres://user:pass@localhost/mydb"
  pgxkit:
    enabled: true
```

## Step 2: Update Dependencies
```bash
go get github.com/nhalm/pgxkit
go mod tidy
```

## Step 3: Regenerate Code
```bash
skimatik --config=skimatik.yaml
```

## Step 4: Update Application Code
Before:
```go
conn, err := pgxpool.New(ctx, dsn)
repo := repositories.NewUsersRepository(conn)
```

After:
```go
conn, err := pgxkit.NewConnection(ctx, dsn, nil)
repo := repositories.NewUsersRepository(conn)
```
```

### Example Project Structure
```
examples/
├── pgxkit-basic/
│   ├── main.go
│   ├── skimatik.yaml
│   ├── go.mod
│   └── README.md
├── pgxkit-advanced/
│   ├── main.go
│   ├── skimatik.yaml
│   ├── go.mod
│   └── README.md
├── pgxkit-testing/
│   ├── main_test.go
│   ├── skimatik.yaml
│   ├── go.mod
│   └── README.md
└── pgxkit-production/
    ├── main.go
    ├── config/
    │   └── production.yaml
    ├── docker-compose.yml
    ├── go.mod
    └── README.md
```

### Configuration Documentation Template
```markdown
# pgxkit Configuration Reference

## Database Configuration
```yaml
database:
  pgxkit:
    enabled: true              # Enable pgxkit integration
    max_conns: 20             # Maximum connections (default: 10)
    min_conns: 5              # Minimum connections (default: 2)
    max_conn_lifetime: "1h"   # Connection lifetime (default: 1h)
    enable_metrics: true      # Enable metrics collection
    enable_retry: true        # Enable retry logic
    retry_attempts: 3         # Number of retry attempts
```

## Testing Configuration
```yaml
testing:
  pgxkit:
    shared_connections: true  # Use shared test connections
    cleanup_queries:          # Cleanup queries between tests
      - "DELETE FROM users WHERE email LIKE 'test_%'"
```

## Validation Rules
- `max_conns` must be greater than `min_conns`
- `retry_attempts` must be between 1 and 10
- `max_conn_lifetime` must be a valid duration string
```

### API Documentation Example
```markdown
# Generated Repository API with pgxkit

## UsersRepository

### Constructor
```go
func NewUsersRepository(conn *pgxkit.Connection) *UsersRepository
```

### Methods

#### GetByID
```go
func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*Users, error)
```
Returns a user by ID with pgxkit error handling.

**Errors:**
- `*pgxkit.NotFoundError` - User not found
- `*pgxkit.DatabaseError` - Database operation failed

#### Create
```go
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error)
```
Creates a new user with validation and error handling.

**Errors:**
- `*pgxkit.ValidationError` - Invalid input parameters
- `*pgxkit.DatabaseError` - Database operation failed
```

## 🔗 Related Files

- `README.md` - Main project documentation
- `docs/PGXKIT_MIGRATION.md` - Migration guide
- `examples/` - Example projects
- `docs/` - Additional documentation
- `cmd/skimatic/main.go` - CLI help text updates

## 📚 Resources

- [pgxkit Documentation](https://github.com/nhalm/pgxkit/blob/main/README.md)
- [pgxkit Examples](https://github.com/nhalm/pgxkit/blob/main/examples.md)
- [PostgreSQL Best Practices](https://wiki.postgresql.org/wiki/Performance_Optimization)

---

**Priority**: Medium
**Effort**: Medium
**Dependencies**: Task 4 (advanced features)
**Next Task**: Final testing and release preparation 