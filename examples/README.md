# skimatik Shared Utilities & Repository Embedding Example

## Overview

This example demonstrates **real usage** of skimatik's shared utility patterns and repository embedding architecture. It focuses on the core value proposition - eliminating code duplication while enabling clean repository composition.

## Key Features Demonstrated

### 🔧 **Shared Utility Patterns**
- **Database Operations**: `ExecuteQueryRow()`, `ExecuteQuery()`, `HandleQueryRowError()`
- **Retry Operations**: `RetryOperation()`, `RetryOperationSlice()`, `ShouldRetryError()`
- **Error Handling**: Consistent patterns across generated and custom code
- **Zero Duplication**: Shared utilities eliminate code repetition

### 🏗️ **Repository Embedding Architecture**
- **Interface Definition**: Teams define domain-specific interfaces
- **Repository Implementation**: Embeds generated repository and implements interface
- **Service Layer**: Uses interface properties, fulfilled by repository implementations
- **Type Safety**: Full compile-time checking maintained

### 📊 **Real Database Integration**
- **Actual Operations**: No mock responses - real database operations
- **Error Handling**: Production-ready error patterns
- **Logging**: Comprehensive operation logging
- **Health Checks**: Database connectivity verification

## Quick Start

### 1. Setup Database
```bash
# From project root
make dev-setup      # Start PostgreSQL with test data
```

### 2. Run Example
```bash
cd examples
go run main.go
```

## What You'll See

### **Architecture Demonstration**
```bash
✅ Connected to database successfully

🔧 Demonstrating Repository Embedding Pattern:
✅ Created UserRepository (embeds generated repository)
✅ Created UserService (uses interface property)

🚀 Demonstrating Shared Utility Patterns:
✅ Registered user: John Doe (ID: 01234567-89ab-cdef-0123-456789abcdef)
✅ Listed 5 users using shared database utilities
✅ Retrieved 3 active users using custom business logic
✅ Created user with retry utilities: Jane Doe (ID: 01234567-89ab-cdef-0123-456789abcde0)

🎉 Example completed - demonstrated:
   • Repository embedding patterns
   • Shared database operation utilities
   • Retry operation utilities
   • Interface-driven design
   • Service layer with interface properties
```

## Code Structure

### **1. Interface Definition (by team)**
```go
type UserManager interface {
    CreateUser(ctx context.Context, params CreateUsersParams) (*Users, error)
    GetUser(ctx context.Context, id uuid.UUID) (*Users, error)
    GetActiveUsers(ctx context.Context) ([]Users, error)
    // ... other domain-specific methods
}
```

### **2. Repository Implementation (embeds generated repository)**
```go
type UserRepository struct {
    *UsersRepository  // Embed generated repository
}

func NewUserRepository(db *pgxkit.DB) UserManager {
    return &UserRepository{
        UsersRepository: NewUsersRepository(db),
    }
}

// Interface methods automatically satisfied by embedding
func (r *UserRepository) CreateUser(ctx context.Context, params CreateUsersParams) (*Users, error) {
    return r.UsersRepository.Create(ctx, params)
}

// Custom business logic using shared utilities
func (r *UserRepository) GetActiveUsers(ctx context.Context) ([]Users, error) {
    query := `SELECT ... FROM users WHERE is_active = true`
    
    // Using shared database utilities
    rows, err := ExecuteQuery(ctx, r.db, "get_active_users", "Users", query)
    // ... handle results with shared patterns
}
```

### **3. Service Layer (uses interface property)**
```go
type UserService struct {
    userRepo UserManager  // Property of interface type
}

func NewUserService(userRepo UserManager) *UserService {
    return &UserService{
        userRepo: userRepo,
    }
}

// Service methods delegate to repository through interface
func (s *UserService) RegisterUser(ctx context.Context, name, email string) (*Users, error) {
    params := CreateUsersParams{Name: name, Email: email}
    return s.userRepo.CreateUser(ctx, params)
}
```

### **4. Application Usage**
```go
func main() {
    db, _ := pgxkit.New(ctx, "postgres://...")
    
    // Create repository that implements interface
    userRepo := NewUserRepository(db)
    
    // Service has property of interface type, fulfilled by repository
    userService := NewUserService(userRepo)
    
    // Use service for business operations
    user, err := userService.RegisterUser(ctx, "John", "john@example.com")
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

## Architecture Patterns

### **Pattern 1: Direct Repository Usage**
```go
userRepo := NewUserRepository(db)
user, err := userRepo.CreateUser(ctx, params)
```

### **Pattern 2: Repository Implementation with Embedding**
```go
type UserRepository struct {
    *repositories.UsersRepository  // Embed for CRUD
}

func (r *UserRepository) CustomMethod() {
    // Add business logic using shared utilities
}
```

### **Pattern 3: Interface-Driven Design**
```go
type UserManager interface {
    CreateUser(context.Context, CreateUsersParams) (*Users, error)
    GetActiveUsers(context.Context) ([]Users, error)
}

// Service implements interface via embedding + extensions
```

## Key Learnings

### **What This Example Shows**
- ✅ **Repository Embedding**: How to embed generated repositories correctly
- ✅ **Interface Implementation**: How repositories implement domain interfaces
- ✅ **Service Architecture**: How services use interface properties
- ✅ **Shared Utilities**: How to leverage database and retry operation utilities
- ✅ **Custom Business Logic**: How to extend repositories with domain-specific methods

### **What This Example Doesn't Show**
- ❌ **HTTP APIs**: This focuses on data layer patterns, not web frameworks
- ❌ **Complex Domain Logic**: Simplified for demonstration purposes
- ❌ **Multiple Aggregates**: Single entity focus for clarity

## Next Steps

In a real application, you would:

1. **Define Domain Interfaces**: Create interfaces that match your business needs
2. **Implement Repository Layer**: Embed generated repositories and implement interfaces
3. **Build Service Layer**: Use interface properties for business logic
4. **Add Testing**: Mock interfaces for unit tests, use real repositories for integration tests
5. **Scale Architecture**: Compose multiple repositories for complex operations

The shared utility patterns ensure consistency across your entire codebase while maintaining the flexibility to implement complex business requirements - **without the overhead of HTTP routing, JSON marshaling, or web framework concerns**. 