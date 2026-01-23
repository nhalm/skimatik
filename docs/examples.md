# Examples & Tutorials

## Overview

This guide demonstrates **real usage** of skimatik generated repositories with **shared utility patterns** from the duplication reduction implementation. It showcases repository embedding, custom business logic, and practical integration patterns.

## 🎯 Key Features Demonstrated

### 🔧 **Shared Utility Patterns**
- **Database Operations**: `ExecuteQueryRow()`, `ExecuteQuery()`, `HandleQueryRowError()`
- **Retry Operations**: Uses `pgxkit.Retry()` for transient error handling
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
# From project root - database auto-starts with test commands
make test-integration   # Auto-starts PostgreSQL
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
    generateIdFunc func() uuid.UUID
}

func NewUsersRepository(idGen func() uuid.UUID) *UsersRepository {
    if idGen == nil {
        idGen = UUIDv7
    }
    return &UsersRepository{
        generateIdFunc: idGen,
    }
}

// All methods require db pgxkit.Executor as parameter
func (r *UsersRepository) Create(ctx context.Context, db pgxkit.Executor, params CreateUsersParams) (*Users, error) {
    id := r.generateIdFunc()
    query := `INSERT INTO users (id, name, email) VALUES ($1, $2, $3) RETURNING ...`

    var user Users
    row := ExecuteQueryRow(ctx, db, "create", "Users", query, id, params.Name, params.Email)
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    if err := HandleQueryRowError("create", "Users", err); err != nil {
        return nil, err
    }

    return &user, nil
}
```

### Service Layer with Embedding
```go
// Custom repository embeds generated repository
type UserRepository struct {
    db *pgxkit.DB         // Store db to pass to generated methods
    *UsersRepository      // Generated CRUD methods
    *UsersQueries         // Generated custom queries
}

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        db:              db,
        UsersRepository: NewUsersRepository(nil),  // nil = default UUID v7
        UsersQueries:    NewUsersQueries(),
    }
}

// Delegate to generated methods, passing db
func (r *UserRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    return r.UsersRepository.Create(ctx, r.db, params)
}

// Generated query methods also need db passed
func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]Users, error) {
    return r.UsersQueries.GetActiveUsers(ctx, r.db, limit)
}
```

### Retry Operations
```go
// All CRUD operations have retry variants that handle transient errors
// (connection issues, deadlocks, serialization failures)
// Uses pgxkit.Retry for exponential backoff with jitter
// Note: Retry methods also require db parameter

func (r *UsersRepository) CreateWithRetry(ctx context.Context, db pgxkit.Executor, params CreateUsersParams) (*Users, error) {
    pgdb, ok := db.(*pgxkit.DB)
    if !ok {
        return r.Create(ctx, db, params)
    }
    return pgxkit.Retry(ctx, func(ctx context.Context) (*Users, error) {
        return r.Create(ctx, pgdb, params)
    })
}

func (r *UsersRepository) GetWithRetry(ctx context.Context, db pgxkit.Executor, id uuid.UUID) (*Users, error) {
    pgdb, ok := db.(*pgxkit.DB)
    if !ok {
        return r.Get(ctx, db, id)
    }
    return pgxkit.Retry(ctx, func(ctx context.Context) (*Users, error) {
        return r.Get(ctx, pgdb, id)
    })
}

func (r *UsersRepository) UpdateWithRetry(ctx context.Context, db pgxkit.Executor, id uuid.UUID, params UpdateUsersParams) (*Users, error) {
    pgdb, ok := db.(*pgxkit.DB)
    if !ok {
        return r.Update(ctx, db, id, params)
    }
    return pgxkit.Retry(ctx, func(ctx context.Context) (*Users, error) {
        return r.Update(ctx, pgdb, id, params)
    })
}

func (r *UsersRepository) DeleteWithRetry(ctx context.Context, db pgxkit.Executor, id uuid.UUID) error {
    pgdb, ok := db.(*pgxkit.DB)
    if !ok {
        return r.Delete(ctx, db, id)
    }
    _, err := pgxkit.Retry(ctx, func(ctx context.Context) (struct{}, error) {
        return struct{}{}, r.Delete(ctx, pgdb, id)
    })
    return err
}

func (r *UsersRepository) ListWithRetry(ctx context.Context, db pgxkit.Executor) ([]Users, error) {
    pgdb, ok := db.(*pgxkit.DB)
    if !ok {
        return r.List(ctx, db)
    }
    return pgxkit.Retry(ctx, func(ctx context.Context) ([]Users, error) {
        return r.List(ctx, pgdb)
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
// UUID v7 generator set automatically (pass nil)
userRepo := repositories.NewUsersRepository(nil)
// Pass db to methods
user, err := userRepo.Create(ctx, db, params)
```

### 2. **Repository Implementation with Embedding**
```go
type UserRepository struct {
    db *pgxkit.DB                  // Store db to pass to methods
    *repositories.UsersRepository  // Embed for CRUD
}

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        db:              db,
        UsersRepository: repositories.NewUsersRepository(nil),
    }
}

func (r *UserRepository) Create(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error) {
    return r.UsersRepository.Create(ctx, r.db, params)
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
import (
    "context"

    "github.com/google/uuid"
    "your-project/repositories"
)

// Define what your application needs
type UserManager interface {
    CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error)
    GetActiveUsers(ctx context.Context) ([]repositories.Users, error)
    DeactivateUser(ctx context.Context, id uuid.UUID) error
}
```

### Step 2: Implement Using Embedding

```go
// Repository layer embeds generated repositories and queries
// Converts generated types to domain types
type UserRepository struct {
    db *pgxkit.DB               // Store db to pass to generated methods
    *generated.UsersRepository  // Generated CRUD methods
    *generated.UsersQueries     // Generated custom query methods
}

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        db:              db,
        UsersRepository: generated.NewUsersRepository(nil),  // nil = default UUID v7
        UsersQueries:    generated.NewUsersQueries(),
    }
}

// Example domain conversion method
func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error) {
    // Pass r.db to generated query
    results, err := r.UsersQueries.GetActiveUsers(ctx, r.db, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to get active users: %w", err)
    }

    // Convert generated types to domain types
    users := make([]domain.UserSummary, len(results))
    for i, result := range results {
        users[i] = domain.UserSummary{
            ID:       result.Id,
            Name:     result.Name,
            Email:    result.Email,
            IsActive: result.IsActive,
        }
    }
    return users, nil
}

// Service layer adds business logic
type UserService struct {
    userRepo *UserRepository
}

func NewUserService(userRepo *UserRepository) *UserService {
    return &UserService{
        userRepo: userRepo,
    }
}

// Service methods can add validation, logging, or cross-cutting concerns
func (s *UserService) GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error) {
    // Add business logic (validation, logging, etc.)
    users, err := s.userRepo.GetActiveUsers(ctx, limit)
    if err != nil {
        return nil, fmt.Errorf("service: failed to get active users: %w", err)
    }
    return users, nil
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

    // Wire up layers: Repository → Service → API Handler
    userRepo := NewUserRepository(db)
    userService := NewUserService(userRepo)

    // Use through service
    users, err := userService.GetActiveUsers(ctx, 10)
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

    // Wire up layers for testing
    userRepo := NewUserRepository(testDB.DB)
    userService := NewUserService(userRepo)

    // Test with real database
    activeUsers, err := userService.GetActiveUsers(ctx, 10)
    require.NoError(t, err)
    assert.NotNil(t, activeUsers)

    // For unit testing, you can mock the repository interface
    // and test business logic in isolation
}
```

## 🎯 Next Steps

This example demonstrates the foundation patterns. In a real application, you would:

1. **Define Domain Interfaces**: Create interfaces that match your business needs
2. **Implement Services**: Embed repositories and add business logic
3. **Add Tests**: Mock interfaces for unit tests, use real repositories for integration tests
4. **Scale Architecture**: Compose multiple repositories for complex operations

The shared utility patterns ensure consistency across your entire codebase while maintaining the flexibility to implement complex business requirements.

## 📄 Paginated Query Examples

The `:paginated` query type generates cursor-based pagination functions. Here are examples showing SQL queries and their generated Go code.

### Basic Paginated Query (No Parameters)

**SQL:**
```sql
-- name: GetPublishedPosts :paginated
SELECT id, title, content, published_at
FROM posts
WHERE is_published = true
ORDER BY published_at DESC
```

**Generated Function Signatures:**
```go
// Regular function - returns all results (requires db parameter)
func (r *PostsQueries) GetPublishedPosts(ctx context.Context, db pgxkit.Executor) ([]GetPublishedPostsResult, error)

// Paginated function - cursor-based pagination (requires db parameter)
func (r *PostsQueries) GetPublishedPostsPaginated(ctx context.Context, db pgxkit.Executor, params PaginationParams) (*PaginationResult[GetPublishedPostsResult], error)
```

**Usage:**
```go
// Get first page (pass db)
result, err := postQueries.GetPublishedPostsPaginated(ctx, db, generated.PaginationParams{
    Limit: 20,
})

// Get next page
if result.HasMore {
    nextPage, err := postQueries.GetPublishedPostsPaginated(ctx, db, generated.PaginationParams{
        Limit:      20,
        NextCursor: result.NextCursor,
    })
}
```

### Paginated Query with Filter Parameters

**SQL:**
```sql
-- name: GetPostsByAuthor :paginated
-- param: author_id uuid
SELECT id, title, content, published_at
FROM posts
WHERE author_id = $1 AND is_published = true
ORDER BY published_at DESC
```

**Generated Function Signatures:**
```go
// Regular function - db comes after ctx, then filter parameters
func (r *PostsQueries) GetPostsByAuthor(ctx context.Context, db pgxkit.Executor, authorId uuid.UUID) ([]GetPostsByAuthorResult, error)

// Paginated function - db comes after ctx, filter params, then PaginationParams
func (r *PostsQueries) GetPostsByAuthorPaginated(ctx context.Context, db pgxkit.Executor, authorId uuid.UUID, params PaginationParams) (*PaginationResult[GetPostsByAuthorResult], error)
```

**Usage:**
```go
authorID := uuid.MustParse("...")

// Get first page of posts by author (pass db)
result, err := postQueries.GetPostsByAuthorPaginated(ctx, db, authorID, generated.PaginationParams{
    Limit: 10,
})

// Get next page (same author filter)
if result.HasMore {
    nextPage, err := postQueries.GetPostsByAuthorPaginated(ctx, db, authorID, generated.PaginationParams{
        Limit:      10,
        NextCursor: result.NextCursor,
    })
}
```

### ASC vs DESC Ordering

The sort direction in your SQL determines cursor comparison:

**DESC (newest first):**
```sql
-- name: GetRecentPosts :paginated
SELECT id, title, created_at FROM posts
ORDER BY created_at DESC
```
- Uses `<` comparison for cursor
- First page returns newest items
- NextCursor navigates to older items

**ASC (oldest first):**
```sql
-- name: GetOldestPosts :paginated
SELECT id, title, created_at FROM posts
ORDER BY created_at ASC
```
- Uses `>` comparison for cursor
- First page returns oldest items
- NextCursor navigates to newer items

### PaginationResult Structure

```go
type PaginationResult[T any] struct {
    Items        []T    // The items for this page
    HasMore      bool   // True if more pages exist (forward)
    HasPrevious  bool   // True if previous pages exist (backward)
    NextCursor   string // Pass to NextCursor for next page
    BeforeCursor string // Pass to BeforeCursor for previous page
}

type PaginationParams struct {
    Limit        int    // Max items per page (default: 20, max: 100)
    NextCursor   string // Cursor for forward pagination (empty for first page)
    BeforeCursor string // Cursor for backward pagination
}
```

### Bidirectional Navigation Example

```go
// Get page 2, then navigate backward to page 1
page2, _ := postQueries.GetPublishedPostsPaginated(ctx, db, generated.PaginationParams{
    Limit:      10,
    NextCursor: page1.NextCursor,
})

// Go back to page 1
if page2.HasPrevious {
    page1Again, _ := postQueries.GetPublishedPostsPaginated(ctx, db, generated.PaginationParams{
        Limit:        10,
        BeforeCursor: page2.BeforeCursor,
    })
}
```

## 🎯 Next Steps

- **[Complete Blog Application Tutorial](tutorial-blog-app)** - Full-stack example app with HTTP API, database persistence, and production patterns
- **[Shared Utilities Guide](shared-utilities)** - Database operations, retry logic, and error handling utilities
- **[Embedding Patterns](embedding-patterns)** - Repository composition and extension patterns
- **[Error Handling Guide](error-handling)** - Comprehensive error management strategies
- **[Quick Start Guide](quick-start)** - Installation and basic usage 