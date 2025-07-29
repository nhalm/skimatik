# skimatik Example Application

## Overview

This example demonstrates **real usage** of skimatik generated repositories with **shared utility patterns** from the duplication reduction implementation. It showcases repository embedding, custom business logic, and practical integration patterns.

## Key Features Demonstrated

### 🔧 **Shared Utility Patterns**
- **Database Operations**: `ExecuteQueryRow()`, `ExecuteQuery()`, `HandleQueryRowError()`
- **Retry Operations**: `RetryOperation()`, `RetryOperationSlice()`, `ShouldRetryError()`
- **Error Handling**: Consistent patterns across generated and custom code
- **Zero Duplication**: Shared utilities eliminate code repetition

### 🏗️ **Repository Embedding**
- **Generated Repository**: Standard CRUD operations with shared utilities
- **Service Layer**: Repository embedding with custom business logic
- **Interface Design**: Teams define domain-specific interfaces
- **Type Safety**: Full compile-time checking maintained

### 📊 **Real Database Integration**
- **Actual Queries**: No mock responses - real database operations
- **Error Handling**: Production-ready error patterns
- **Logging**: Comprehensive operation logging
- **Health Checks**: Database connectivity verification

## Quick Start

### 1. Setup Database
```bash
# From project root
make dev-setup      # Start PostgreSQL with test data
```

### 2. Generate Repositories (if needed)
```bash
# Build the skimatik tool
make build

# Generate repositories using test configuration
./bin/skimatik --config=configs/test-config.yaml
```

### 3. Run Example
```bash
cd examples
go run main.go
```

## API Endpoints

### **Standard CRUD with Shared Utilities**
```bash
# List users (shared database utilities)
curl http://localhost:8080/users

# Get user by ID (shared error handling)
curl http://localhost:8080/users/{id}

# Create user (retry operation utilities)
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# Update user (shared database patterns)
curl -X PUT http://localhost:8080/users/{id} \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name"}'

# Delete user (shared error handling)
curl -X DELETE http://localhost:8080/users/{id}
```

### **Custom Business Logic**
```bash
# Get active users (custom query with shared utilities)
curl http://localhost:8080/users/active
```

### **Health Check**
```bash
# Verify database connectivity and features
curl http://localhost:8080/health
```

## Code Structure

### Generated Repository Pattern
```go
// Generated repository with shared utilities
type UsersRepository struct {
    db *pgxpool.Pool
}

func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    query := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING ...`
    
    // Using shared database utilities
    row := ExecuteQueryRow(ctx, r.db, "create", "Users", query, params.Name, params.Email)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    return &user, HandleQueryRowError("create", "Users", err)
}
```

### Service Layer with Embedding
```go
// Service embeds generated repository
type UserService struct {
    *UsersRepository  // All CRUD methods available
}

// Custom business logic using shared utilities
func (s *UserService) GetActiveUsers(ctx context.Context) ([]Users, error) {
    query := `SELECT ... FROM users WHERE is_active = true`
    
    // Using shared database utilities
    rows, err := ExecuteQuery(ctx, s.db, "get_active_users", "Users", query)
    // ... handle results with shared patterns
}
```

### Retry Operations
```go
// Retry with shared utilities
func (r *UsersRepository) CreateWithRetry(ctx context.Context, params CreateUsersParams) (*Users, error) {
    return RetryOperation(ctx, DefaultRetryConfig, "create", func(ctx context.Context) (*Users, error) {
        return r.Create(ctx, params)
    })
}
```

## Benefits Demonstrated

### 🚀 **For Development**
- **90% Less Duplication**: Shared utilities eliminate repetitive code
- **Consistent Patterns**: Same patterns in generated and custom code
- **Type Safety**: Full compile-time checking maintained
- **Zero Runtime Overhead**: All utilities generate concrete code

### 🏗️ **For Architecture**
- **Clean Embedding**: Generated repositories work perfectly with composition
- **Interface Freedom**: Teams define interfaces based on domain needs
- **Easy Testing**: Mock interfaces, not repositories
- **Maintainable**: Regeneration doesn't affect custom code

### 📊 **For Production**
- **Error Resilience**: Built-in retry logic for transient failures
- **Observability**: Comprehensive logging and error context
- **Performance**: No reflection, direct database operations
- **Reliability**: Battle-tested error handling patterns

## Real vs Mock Comparison

### Before (Mock Response)
```go
func handleListUsers(w http.ResponseWriter, r *http.Request) {
    // Mock data - not real
    mockResponse := map[string]interface{}{
        "items": []map[string]interface{}{
            {"id": "mock-id", "name": "Mock User"},
        },
    }
    json.NewEncoder(w).Encode(mockResponse)
}
```

### After (Real Repository)
```go
func (s *APIServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
    // Real database operation with shared utilities
    users, err := s.userService.List(ctx)
    if err != nil {
        log.Printf("Failed to list users: %v", err)
        http.Error(w, "Failed to list users", http.StatusInternalServerError)
        return
    }
    
    response := map[string]interface{}{
        "items": users,
        "count": len(users),
    }
    json.NewEncoder(w).Encode(response)
}
```

## Integration Patterns

### 1. **Direct Repository Usage**
```go
userRepo := repositories.NewUsersRepository(conn)
user, err := userRepo.Create(ctx, params)
```

### 2. **Repository Implementation with Embedding**
```go
type UserRepository struct {
    *repositories.UsersRepository  // Embed for CRUD
}

func NewUserRepository(conn *pgxpool.Pool) *UserRepository {
    return &UserRepository{
        UsersRepository: repositories.NewUsersRepository(conn),
    }
}

func (r *UserRepository) CustomMethod() {
    // Add business logic using shared utilities
}
```

### 3. **Interface-Driven Design**
```go
type UserManager interface {
    CreateUser(context.Context, CreateUsersParams) (*Users, error)
    GetActiveUsers(context.Context) ([]Users, error)
}

// Service implements interface via embedding + extensions
```

## Testing the Example

### Manual Testing
```bash
# Start the application
go run main.go

# In another terminal, test endpoints
curl http://localhost:8080/health
curl http://localhost:8080/users
curl -X POST http://localhost:8080/users -d '{"name":"Test","email":"test@example.com"}' -H "Content-Type: application/json"
```

### Expected Output
- Real database operations (not mocks)
- Comprehensive error handling
- Retry logic for creation operations
- Custom business logic for active users
- Consistent logging patterns

## Next Steps

This example demonstrates the foundation patterns. In a real application, you would:

1. **Define Domain Interfaces**: Create interfaces that match your business needs
2. **Implement Services**: Embed repositories and add business logic
3. **Add Tests**: Mock interfaces for unit tests, use real repositories for integration tests
4. **Scale Architecture**: Compose multiple repositories for complex operations

The shared utility patterns ensure consistency across your entire codebase while maintaining the flexibility to implement complex business requirements. 