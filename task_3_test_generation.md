# Task 3: Generate Test Files with pgxkit Testing Utilities

**Parent Issue**: #4 - Integrate pgxkit for Enhanced Database Operations and Testing
**Depends On**: #6 - Task 2: Update Code Generation Templates for pgxkit Integration

## 🎯 Objective

Generate test files that use pgxkit's testing utilities for faster, more reliable integration tests with shared connections and automatic cleanup.

## 📋 Tasks

### 1. Create Test Generation Templates
- [ ] Create `internal/generator/templates/tests/pgxkit_test.tmpl` for generating test files
- [ ] Create `internal/generator/templates/tests/test_helpers.tmpl` for test utility functions
- [ ] Create `internal/generator/templates/tests/test_setup.tmpl` for test setup/teardown

### 2. Generate Repository Test Files
- [ ] Generate `*_generated_test.go` files for each repository
- [ ] Include pgxkit.RequireTestDB() for shared test connections
- [ ] Add pgxkit.CleanupTestData() for test isolation
- [ ] Generate comprehensive CRUD operation tests

### 3. Test Infrastructure Generation
- [ ] Generate test configuration helpers
- [ ] Create test database connection utilities
- [ ] Add test data factory functions
- [ ] Generate benchmark tests for performance validation

### 4. Update Test Template Logic
- [ ] Modify `internal/generator/generator.go` to generate test files with pgxkit
- [ ] Include test file generation in the main generation pipeline
- [ ] Ensure test files are always generated with pgxkit utilities

### 5. Integration Test Generation
- [ ] Generate integration tests that use real PostgreSQL database
- [ ] Include tests for pagination functionality
- [ ] Add tests for error handling scenarios
- [ ] Generate tests for transaction support

## 🧪 Testing Requirements

### Unit Tests
- [ ] Test template rendering for test files
- [ ] Test test helper generation
- [ ] Test conditional test generation logic
- [ ] Test generated test code compilation

### Integration Tests
- [ ] Test generated tests run successfully
- [ ] Test pgxkit testing utilities work correctly
- [ ] Test shared connection functionality
- [ ] Test cleanup between tests

## 📝 Acceptance Criteria

- [ ] Test files are always generated with pgxkit utilities
- [ ] Generated tests use pgxkit testing utilities
- [ ] Tests include proper setup and cleanup
- [ ] All generated tests pass
- [ ] Test execution is faster with shared connections
- [ ] Tests are isolated and don't interfere with each other

## 🔧 Implementation Notes

### Generated Test File Example
```go
// users_generated_test.go
package repositories

import (
    "context"
    "testing"
    "github.com/nhalm/pgxkit"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUsersRepository_GetByID(t *testing.T) {
    conn := pgxkit.RequireTestDB(t, nil)
    defer pgxkit.CleanupTestData(conn, "DELETE FROM users WHERE email LIKE 'test_%'")
    
    repo := NewUsersRepository(conn)
    
    // Test user creation
    user, err := repo.Create(context.Background(), CreateUsersParams{
        Name:  "Test User",
        Email: "test@example.com",
    })
    require.NoError(t, err)
    require.NotNil(t, user)
    
    // Test user retrieval
    retrieved, err := repo.GetByID(context.Background(), user.Id)
    require.NoError(t, err)
    assert.Equal(t, user.Id, retrieved.Id)
    assert.Equal(t, user.Name, retrieved.Name)
    assert.Equal(t, user.Email, retrieved.Email)
}

func TestUsersRepository_GetByID_NotFound(t *testing.T) {
    conn := pgxkit.RequireTestDB(t, nil)
    repo := NewUsersRepository(conn)
    
    nonExistentID := uuid.New()
    user, err := repo.GetByID(context.Background(), nonExistentID)
    
    assert.Nil(t, user)
    var notFoundErr *pgxkit.NotFoundError
    assert.ErrorAs(t, err, &notFoundErr)
    assert.Equal(t, "User", notFoundErr.ResourceType)
}
```

### Test Helper Functions
```go
// test_helpers_generated.go
package repositories

import (
    "context"
    "testing"
    "github.com/nhalm/pgxkit"
    "github.com/google/uuid"
)

func setupTestConnection(t *testing.T) *pgxkit.Connection {
    return pgxkit.RequireTestDB(t, nil)
}

func cleanupTestData(conn *pgxkit.Connection, queries ...string) {
    pgxkit.CleanupTestData(conn, queries...)
}

func createTestUser(t *testing.T, repo *UsersRepository) *Users {
    user, err := repo.Create(context.Background(), CreateUsersParams{
        Name:  "Test User " + uuid.New().String()[:8],
        Email: "test_" + uuid.New().String()[:8] + "@example.com",
    })
    require.NoError(t, err)
    return user
}
```

### Template Structure
```
internal/generator/templates/tests/
├── pgxkit_test.tmpl              # Main test file template
├── test_helpers.tmpl             # Test helper functions
├── test_setup.tmpl               # Test setup/teardown
├── integration_test.tmpl         # Integration test template
└── benchmark_test.tmpl           # Benchmark test template
```

### Simple Test Generation
```yaml
# No special configuration needed - tests always use pgxkit
database:
  dsn: "postgres://user:pass@localhost/mydb"
  schema: "public"
```

## 🔗 Related Files

- `internal/generator/templates/tests/` (new directory)
- `internal/generator/generator.go`
- `internal/generator/codegen.go`
- Generated `*_generated_test.go` files

## 📚 Resources

- [pgxkit Testing Utilities](https://github.com/nhalm/pgxkit/blob/main/test_connection.go)
- [pgxkit Testing Examples](https://github.com/nhalm/pgxkit/blob/main/examples.md#testing)
- [pgxkit RequireTestDB Documentation](https://github.com/nhalm/pgxkit/blob/main/README.md#testing)

---

**Priority**: High
**Effort**: Medium
**Dependencies**: Task 2 (template updates)
**Next Task**: Task 4 - Add Advanced pgxkit Features 