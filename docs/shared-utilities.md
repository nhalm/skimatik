# Shared Utilities Guide

## Overview

skimatik generates shared utility functions in `database_operations.go` that eliminate code duplication across all repositories. These utilities provide consistent error handling patterns throughout your generated code.

## What Gets Generated

Every generated package includes `/repository/generated/database_operations.go` with these utilities:

### Core Database Operations

```go
// Single-row operations (Create, Get, Update, custom queries returning one row)
func ExecuteQueryRow(ctx, db, operation, entity, query, args...) pgx.Row

// Multi-row operations (List, custom queries returning many rows)
func ExecuteQuery(ctx, db, operation, entity, query, args...) (pgx.Rows, error)

// Non-query operations (Delete, updates without RETURNING)
func ExecuteNonQuery(ctx, db, operation, entity, query, args...) error
func ExecuteNonQueryWithRowsAffected(ctx, db, operation, entity, query, args...) (int64, error)

// Error handling helpers
func HandleQueryRowError(operation, entity, err) error
func HandleRowsResult(entity, rows) error
```

These are **internal helpers** used by generated code. You don't typically call them directly.

## How Generated Code Uses Utilities

### Example: Generated CRUD Operations

```go
// From users_generated.go
// All methods require db pgxkit.Executor as parameter
func (r *UsersRepository) Create(ctx context.Context, db pgxkit.Executor, params CreateUsersParams) (*Users, error) {
    id := r.generateIdFunc()
    query := `INSERT INTO users (id, name, email, bio) VALUES ($1, $2, $3, $4) RETURNING ...`

    var u Users
    // Uses ExecuteQueryRow utility with db parameter
    row := ExecuteQueryRow(ctx, db, "create", "Users", query, id, params.Name, params.Email, params.Bio)
    err := row.Scan(&u.Id, &u.Name, &u.Email, &u.Bio, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)

    // Uses HandleQueryRowError utility
    if err := HandleQueryRowError("create", "Users", err); err != nil {
        return nil, err
    }
    return &u, nil
}

func (r *UsersRepository) List(ctx context.Context, db pgxkit.Executor) ([]Users, error) {
    query := `SELECT id, name, email FROM users ORDER BY id ASC`

    // Uses ExecuteQuery utility with db parameter
    rows, err := ExecuteQuery(ctx, db, "list", "Users", query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []Users
    for rows.Next() {
        var u Users
        if err := rows.Scan(&u.Id, &u.Name, &u.Email); err != nil {
            return nil, HandleDatabaseError("scan", "Users", err)
        }
        results = append(results, u)
    }

    // Uses HandleRowsResult utility
    return results, HandleRowsResult("Users", rows)
}
```

### Example: Generated Custom Queries

```go
// From users_queries_generated.go
// Note: Example shows all fields for accuracy. Real queries may select fewer columns.
// All query methods require db pgxkit.Executor as parameter
func (r *UsersQueries) GetUserByEmail(ctx context.Context, db pgxkit.Executor, email string) (*GetUserByEmailResult, error) {
    query := `SELECT id, name, email, bio, is_active, created_at, updated_at
FROM users WHERE email = $1 AND is_active = true`

    var result GetUserByEmailResult
    row := ExecuteQueryRow(ctx, db, "GetUserByEmail", "GetUserByEmailResult", query, email)
    err := row.Scan(&result.Id, &result.Name, &result.Email, &result.Bio, &result.IsActive, &result.CreatedAt, &result.UpdatedAt)

    if err := HandleQueryRowError("GetUserByEmail", "GetUserByEmailResult", err); err != nil {
        return nil, err
    }
    return &result, nil
}
```

## Custom Repositories Using Generated Code

The recommended pattern is to **embed generated types** and store a `db` field to pass to generated methods:

### Pattern 1: Embed Repository + Queries (Most Common)

```go
// example-app/repository/user_repository.go
type UserRepository struct {
    db *pgxkit.DB               // Store db to pass to generated methods
    *generated.UsersRepository  // Provides CRUD operations
    *generated.UsersQueries     // Provides custom queries from .sql files
}

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        db:              db,
        UsersRepository: generated.NewUsersRepository(nil),  // nil = default UUID v7
        UsersQueries:    generated.NewUsersQueries(),
    }
}

// Add business logic methods that delegate to generated code
func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error) {
    // Uses generated query, passing db
    results, err := r.UsersQueries.GetActiveUsers(ctx, r.db, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to get active users: %w", err)
    }

    // Convert to domain types
    users := make([]domain.UserSummary, len(results))
    for i, result := range results {
        users[i] = domain.UserSummary{
            ID:    result.Id,
            Name:  result.Name,
            Email: result.Email,
        }
    }
    return users, nil
}
```

### Pattern 2: Embed Queries Only

```go
// example-app/repository/post_repository.go
type PostRepository struct {
    db *pgxkit.DB            // Store db to pass to generated methods
    *generated.PostsQueries  // Only custom queries needed
}

func NewPostRepository(db *pgxkit.DB) *PostRepository {
    return &PostRepository{
        db:           db,
        PostsQueries: generated.NewPostsQueries(),
    }
}

// Add custom business logic using generated queries
func (r *PostRepository) GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
    // Use generated query as base, passing db
    posts, err := r.GetPublishedPosts(ctx, r.db, limit*2)
    if err != nil {
        return nil, fmt.Errorf("failed to get posts: %w", err)
    }

    // Apply custom filtering logic
    var featured []domain.PostSummary
    for _, post := range posts {
        if len(post.Title) > 20 && len(featured) < limit {
            featured = append(featured, post)
        }
    }
    return featured, nil
}
```

## Key Principles

### 1. Use Generated Code, Don't Recreate It

**Good - Delegate to generated code:**
```go
func (r *UserRepository) CreateUser(ctx context.Context, name, email string) (*User, error) {
    params := generated.CreateUsersParams{Name: name, Email: email}
    return r.UsersRepository.Create(ctx, r.db, params)  // Pass db to method
}
```

**Bad - Duplicating generated code:**
```go
func (r *UserRepository) CreateUser(ctx context.Context, name, email string) (*User, error) {
    // Don't do this - use generated Create method instead
    query := `INSERT INTO users...`
    row := r.db.QueryRow(ctx, query, name, email)
    // ...
}
```

### 2. Custom Queries Go in .sql Files

**Good - Define in .sql file, use generated query:**
```sql
-- database/queries/get_active_users.sql
-- name: GetActiveUsers :many
SELECT id, name, email FROM users WHERE is_active = true LIMIT $1;
```

```go
// Use generated query, passing db
func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]User, error) {
    return r.UsersQueries.GetActiveUsers(ctx, r.db, limit)
}
```

**Bad - Writing raw SQL in Go:**
```go
func (r *UserRepository) GetActiveUsers(ctx context.Context) ([]User, error) {
    // Avoid this - put queries in .sql files instead
    query := `SELECT id, name, email FROM users WHERE is_active = true`
    rows, err := r.db.Query(ctx, query)
    // ...
}
```

### 3. Store DB Field to Pass to Generated Methods

**v2 API pattern - db field required:**
```go
type UserRepository struct {
    db *pgxkit.DB               // Required to pass to generated methods
    *generated.UsersRepository
    *generated.UsersQueries
}

func NewUserRepository(db *pgxkit.DB) *UserRepository {
    return &UserRepository{
        db:              db,
        UsersRepository: generated.NewUsersRepository(nil),
        UsersQueries:    generated.NewUsersQueries(),
    }
}
```

The `db` field is needed because the v2 API passes the executor to each method, enabling transaction support.

## Benefits of This Approach

### Code Reuse
- **90% reduction** in generated code duplication
- Utilities used across all CRUD operations and custom queries
- Consistent patterns throughout codebase

### Maintainability
- Single source of truth for database operations
- Error handling centralized in utilities
- Easy to update patterns across all repositories

### Type Safety
- All utilities generate concrete code (no reflection)
- Full compile-time type checking
- Zero runtime overhead

## Related Documentation

- **[Embedding Patterns](embedding-patterns)** - Repository composition and extension patterns
- **[Error Handling Guide](error-handling)** - Comprehensive error management strategies
- **[Examples & Tutorials](examples)** - Real-world examples from example-app 