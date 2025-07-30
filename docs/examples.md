# Examples & Tutorials

## Overview

This guide demonstrates **real usage** of skimatik generated repositories with **shared utility patterns** from the duplication reduction implementation. It showcases repository embedding, custom business logic, and practical integration patterns.

## 🎯 Key Features Demonstrated

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

## 🚀 Quick Start

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

## 🌐 API Endpoints

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

## 💻 Code Structure

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

## 🎯 Benefits Demonstrated

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

## 🔄 Real vs Mock Comparison

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

## 🔗 Integration Patterns

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

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        UsersRepository: repositories.NewUsersRepository(db),
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

## 🧪 Testing the Example

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

## 📚 Tutorial: Building Your First Integration

### Step 1: Define Your Domain Interface

```go
// Define what your application needs
type UserManager interface {
    CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error)
    GetActiveUsers(ctx context.Context) ([]repositories.Users, error)
    DeactivateUser(ctx context.Context, id uuid.UUID) error
}
```

### Step 2: Implement Using Embedding

```go
type UserService struct {
    *repositories.UsersRepository  // Gets all generated CRUD methods
}

func NewUserService(db *pgxkit.DB) UserManager {
    return &UserService{
        UsersRepository: repositories.NewUsersRepository(db),
    }
}

// CreateUser automatically satisfied by embedding

// Add custom business methods
func (s *UserService) GetActiveUsers(ctx context.Context) ([]repositories.Users, error) {
    query := `SELECT id, name, email, created_at FROM users WHERE is_active = true`
    
    rows, err := repositories.ExecuteQuery(ctx, s.db, "get_active_users", "Users", query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var users []repositories.Users
    for rows.Next() {
        var user repositories.Users
        err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
        if err != nil {
            return nil, repositories.HandleDatabaseError("scan", "Users", err)
        }
        users = append(users, user)
    }
    
    return users, repositories.HandleRowsResult("Users", rows.Err())
}
```

### Step 3: Use in Your Application

```go
func main() {
    db := pgxkit.NewDB()
    err := db.Connect(ctx, "postgres://...")
    if err != nil {
        log.Fatal(err)
    }
    
    userService := NewUserService(db)
    
    // Use through interface
    users, err := userService.GetActiveUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d active users\n", len(users))
}
```

### Step 4: Add Testing

```go
func TestUserService(t *testing.T) {
    testDB := pgxkit.RequireDB(t)
    userService := NewUserService(testDB.DB)
    
    // Test with real database
    user, err := userService.CreateUser(ctx, repositories.CreateUsersParams{
        Name:  "Test User",
        Email: "test@example.com",
    })
    
    require.NoError(t, err)
    assert.Equal(t, "Test User", user.Name)
    
    // Test custom method
    activeUsers, err := userService.GetActiveUsers(ctx)
    require.NoError(t, err)
    assert.Len(t, activeUsers, 1)
}
```

## 🎯 Next Steps

This example demonstrates the foundation patterns. In a real application, you would:

1. **Define Domain Interfaces**: Create interfaces that match your business needs
2. **Implement Services**: Embed repositories and add business logic
3. **Add Tests**: Mock interfaces for unit tests, use real repositories for integration tests
4. **Scale Architecture**: Compose multiple repositories for complex operations

The shared utility patterns ensure consistency across your entire codebase while maintaining the flexibility to implement complex business requirements.

## 📝 Complete Blog Application Example

The **`example-app/`** directory contains a **complete blog application** that demonstrates skimatik in a real-world scenario. This is a full-stack Go application with HTTP API, service layer, and database persistence using generated repositories.

### 🏗️ Application Architecture

```
example-app/
├── api/           # HTTP handlers and request/response types  
├── service/       # Business logic layer
├── repository/    # Repository layer with generated code
│   ├── generated/ # Generated repositories and queries (DO NOT EDIT)
│   ├── user_repository.go    # Custom repository extensions
│   └── post_repository.go    # Custom repository extensions
├── domain/        # Domain types and business objects
├── database/      # Database schema and SQL queries
│   ├── schema.sql        # PostgreSQL schema
│   └── queries/          # SQL query files for generation
└── main.go        # Application entry point
```

### 🚀 Quick Start

```bash
# Navigate to example app
cd example-app

# Start database and apply schema  
make setup

# Generate repositories from database schema + SQL files
make generate

# Run integration tests
make test

# Start the application
make run
```

The application will start on `http://localhost:8080` with a REST API for managing blog posts and users.

### 📊 Database Schema

The example uses a realistic blog schema with proper relationships:

```sql
-- Users table with UUID primary key (required for skimatik)
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    email       TEXT NOT NULL UNIQUE,
    bio         TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Posts table with foreign key relationships
CREATE TABLE posts (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    author_id   UUID NOT NULL REFERENCES users(id),
    is_published BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Additional tables: comments, tags, post_tags...
```

### ⚡ Generated Code

Skimatik generates comprehensive repositories for each table:

#### **Repository Files Generated:**
- `users_generated.go` - Complete CRUD operations for users
- `posts_generated.go` - Complete CRUD operations for posts  
- `comments_generated.go` - Complete CRUD operations for comments
- `users_queries_generated.go` - Custom queries from SQL files
- `posts_queries_generated.go` - Custom queries from SQL files
- `pagination.go` - Shared pagination utilities

#### **Features Generated:**
- **CRUD Operations**: Create, Get, Update, Delete, List
- **Pagination**: Cursor-based pagination with `ListPaginated()`
- **Retry Logic**: `CreateWithRetry()`, `GetWithRetry()`, etc.
- **Custom Queries**: Generated from `database/queries/*.sql` files
- **Type Safety**: Full PostgreSQL type mapping to Go types

### 🎯 HTTP API Endpoints

The application exposes a complete REST API:

#### **User Endpoints:**
```http
GET    /api/users              # List active users
GET    /api/users/{id}         # Get user by ID  
GET    /api/users/email/{email} # Get user by email
GET    /api/users/search       # Search users by name/email
GET    /api/users/{id}/stats   # Get user statistics
DELETE /api/users/{id}         # Deactivate user
```

#### **Post Endpoints:**
```http
GET    /api/posts              # List published posts
GET    /api/posts/{id}         # Get post by ID
GET    /api/posts/with-stats   # Posts with comment counts
PUT    /api/posts/{id}/publish # Publish a post
```

#### **Health Check:**
```http
GET    /api/health             # Application health status
```

### 🔧 Repository Pattern Example

The example demonstrates the **recommended pattern** for extending generated repositories:

```go
// Custom repository that embeds generated code
type UserRepository struct {
    *generated.UsersRepository  // Generated CRUD operations
    *generated.UsersQueries     // Generated custom queries
}

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        UsersRepository: generated.NewUsersRepository(db),
        UsersQueries:    generated.NewUsersQueries(db),
    }
}

// Add custom business logic methods
func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int32) ([]domain.UserSummary, error) {
    // Use generated query and convert to domain types
    results, err := r.UsersQueries.GetActiveUsers(ctx, fmt.Sprintf("%d", limit))
    if err != nil {
        return nil, fmt.Errorf("failed to get active users: %w", err)
    }
    
    // Convert generated types to domain types
    users := make([]domain.UserSummary, len(results))
    for i, result := range results {
        users[i] = domain.UserSummary{
            ID:       uuid.UUID(result.Id.Bytes),
            Name:     result.Name.String,
            Email:    result.Email.String,
            IsActive: result.IsActive.Bool,
        }
    }
    return users, nil
}
```

### 🧪 Integration Testing

The example includes comprehensive integration testing that validates the **complete pipeline**:

```bash
# Run complete integration test  
make example-app-test

# This validates:
# 1. Database setup and schema migration
# 2. Code generation from tables + SQL queries  
# 3. Generated code compilation
# 4. Application startup with database connectivity
# 5. HTTP endpoints responding correctly
```

### 📁 Configuration

The example uses a complete skimatik configuration:

```yaml
# example-app/skimatik.yaml
database:
  dsn: "postgres://postgres:password@localhost:5432/blog?sslmode=disable"
  schema: "public"

output:
  directory: "./repository/generated"
  package: "generated"

# Generate all functions by default
default_functions: "all"

# Generate from SQL query files
queries:
  directory: "./database/queries"

# Also generate from tables
tables:
  users:
  posts:
  comments:

verbose: true
```

### 🎯 Key Learnings

This example demonstrates:

1. **Complete Application**: Real HTTP API with database persistence
2. **Schema Design**: Proper UUID primary keys and foreign key relationships  
3. **Repository Embedding**: How to extend generated code with custom logic
4. **Service Layer**: Business logic separation with domain type conversion
5. **Integration Testing**: End-to-end validation of the complete pipeline
6. **SQL Query Generation**: Custom queries alongside table-based generation
7. **Production Patterns**: Error handling, retry logic, and pagination

### 🔗 Try It Yourself

1. **Clone the repository**: `git clone https://github.com/nhalm/skimatik.git`
2. **Navigate to example**: `cd skimatik/example-app`  
3. **Follow the Quick Start** above
4. **Explore the generated code** in `repository/generated/`
5. **Make changes** to the schema and regenerate
6. **Add custom queries** in `database/queries/`

This example serves as a **comprehensive reference** for building production applications with skimatik.

## Related Documentation

- **[Shared Utilities Guide](Shared-Utilities-Guide)** - Database operations, retry logic, and error handling utilities
- **[Embedding Patterns](Embedding-Patterns)** - Repository composition and extension patterns
- **[Error Handling Guide](Error-Handling-Guide)** - Comprehensive error management strategies
- **[Quick Start Guide](Quick-Start-Guide)** - Installation and basic usage 