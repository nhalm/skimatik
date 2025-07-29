# Shared Utilities Guide

## Overview

skimatik generates shared utility functions that eliminate code duplication across repositories while maintaining full type safety and performance. These utilities are available in every generated package and can be used in custom repository extensions.

## 🔧 Phase 1: Database Operation Utilities

### Generated Functions

Every generated package includes these database operation utilities:

#### `ExecuteQueryRow(ctx, db, operation, entity, query, args...)`
Executes single-row queries (CREATE, GET, UPDATE operations) with consistent error handling.

```go
// Generated usage in Create operation
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    query := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at`
    
    row := ExecuteQueryRow(ctx, r.db, "create", "Users", query, params.Name, params.Email)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    return &user, HandleQueryRowError("create", "Users", err)
}
```

#### `ExecuteQuery(ctx, db, operation, entity, query, args...)`
Executes multi-row queries (LIST operations) with automatic error handling.

```go
// Generated usage in List operation
func (r *UsersRepository) List(ctx context.Context) ([]Users, error) {
    query := `SELECT id, name, email, created_at FROM users ORDER BY created_at DESC`
    
    rows, err := ExecuteQuery(ctx, r.db, "list", "Users", query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []Users
    for rows.Next() {
        var user Users
        err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
        if err != nil {
            return nil, HandleDatabaseError("scan", "Users", err)
        }
        results = append(results, user)
    }
    
    return results, HandleRowsResult("Users", rows)
}
```

#### `ExecuteNonQuery(ctx, db, operation, entity, query, args...)`
Executes operations that don't return data (DELETE operations).

```go
// Generated usage in Delete operation
func (r *UsersRepository) Delete(ctx context.Context, id uuid.UUID) error {
    query := `DELETE FROM users WHERE id = $1`
    return ExecuteNonQuery(ctx, r.db, "delete", "Users", query, id)
}
```

### Using in Custom Repositories

These utilities are **public functions** available for cross-package use:

```go
// Your custom repository extending generated ones
type UserService struct {
    *repositories.UsersRepository  // Embed generated repository
    profileRepo *repositories.ProfilesRepository
}

// Custom business logic using shared utilities
func (s *UserService) CreateUserWithProfile(ctx context.Context, userData repositories.CreateUsersParams, profileData string) (*repositories.Users, error) {
    // Use shared database utilities in custom operations
    query := `
        WITH new_user AS (
            INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at
        ), new_profile AS (
            INSERT INTO user_profiles (user_id, bio) 
            SELECT id, $3 FROM new_user
            RETURNING user_id
        )
        SELECT id, name, email, created_at FROM new_user
    `
    
    row := repositories.ExecuteQueryRow(ctx, s.db, "create_user_with_profile", "Users", query, 
        userData.Name, userData.Email, profileData)
    
    var user repositories.Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    return &user, repositories.HandleQueryRowError("create_user_with_profile", "Users", err)
}
```

## 🔄 Phase 2: Retry Operation Utilities

### Generated Functions

#### `RetryOperation[T](ctx, config, operation, fn)`
Generic retry function for single-result operations.

```go
// Generated usage in WithRetry methods
func (r *UsersRepository) CreateWithRetry(ctx context.Context, params CreateUsersParams) (*Users, error) {
    return RetryOperation(ctx, DefaultRetryConfig, "create", func(ctx context.Context) (*Users, error) {
        return r.Create(ctx, params)
    })
}
```

#### `RetryOperationSlice[T](ctx, config, operation, fn)`
Generic retry function for slice-result operations.

```go
// Generated usage for list operations
func (r *UsersRepository) ListWithRetry(ctx context.Context) ([]Users, error) {
    return RetryOperationSlice(ctx, DefaultRetryConfig, "list", func(ctx context.Context) ([]Users, error) {
        return r.List(ctx)
    })
}
```

#### `ShouldRetryError(err)`
Determines if an error is worth retrying (connection issues, deadlocks, etc.).

### Retry Configuration

```go
// Default configuration (generated in every package)
var DefaultRetryConfig = RetryConfig{
    MaxRetries: 3,
    BaseDelay:  100 * time.Millisecond,
}

// Custom configuration
customConfig := RetryConfig{
    MaxRetries: 5,
    BaseDelay:  200 * time.Millisecond,
}
```

### Using in Custom Repositories

```go
// Custom business logic with retry support
func (s *UserService) CreateUserWithProfileWithRetry(ctx context.Context, userData repositories.CreateUsersParams, profileData string) (*repositories.Users, error) {
    return repositories.RetryOperation(ctx, repositories.DefaultRetryConfig, "create_user_with_profile", func(ctx context.Context) (*repositories.Users, error) {
        return s.CreateUserWithProfile(ctx, userData, profileData)
    })
}

// Custom retry configuration for sensitive operations
func (s *UserService) CreateUserWithCustomRetry(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error) {
    customConfig := repositories.RetryConfig{
        MaxRetries: 5,
        BaseDelay:  500 * time.Millisecond,
    }
    
    return repositories.RetryOperation(ctx, customConfig, "create_user_custom", func(ctx context.Context) (*repositories.Users, error) {
        return s.UsersRepository.Create(ctx, params)
    })
}
```

## 📋 Complete Example: Custom Repository with Shared Utilities

```go
package services

import (
    "context"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/nhalm/pgxkit"
    "your-project/repositories"
)

// Define your own interface based on business needs
type UserInterface interface {
    CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error)
    CreateUserWithProfile(ctx context.Context, userData repositories.CreateUsersParams, profileData string) (*repositories.Users, error)
    GetActiveUsers(ctx context.Context) ([]repositories.Users, error)
}

// Create repository that implements interface and embeds generated repository
type UserRepository struct {
    *repositories.UsersRepository  // All generated methods available
    profileRepo *repositories.ProfilesRepository
}

func NewUserRepository(db *pgxkit.DB) UserInterface {
    return &UserRepository{
        UsersRepository: repositories.NewUsersRepository(db),
        profileRepo:     repositories.NewProfilesRepository(db),
    }
}

// Implement interface using generated methods (no custom logic needed)
func (r *UserRepository) CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error) {
    return r.UsersRepository.Create(ctx, params)
}

// Custom business logic using shared utilities
func (s *UserService) CreateUserWithProfile(ctx context.Context, userData repositories.CreateUsersParams, profileData string) (*repositories.Users, error) {
    return repositories.RetryOperation(ctx, repositories.DefaultRetryConfig, "create_user_with_profile", func(ctx context.Context) (*repositories.Users, error) {
        // Use shared database utilities for complex operations
        query := `
            WITH new_user AS (
                INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at
            ), new_profile AS (
                INSERT INTO user_profiles (user_id, bio) 
                SELECT id, $3 FROM new_user
                RETURNING user_id
            )
            SELECT id, name, email, created_at FROM new_user
        `
        
        row := repositories.ExecuteQueryRow(ctx, s.db, "create_user_with_profile", "Users", query, 
            userData.Name, userData.Email, profileData)
        
        var user repositories.Users
        err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
        if err != nil {
            return nil, repositories.HandleQueryRowError("create_user_with_profile", "Users", err)
        }
        
        return &user, nil
    })
}

// Custom queries with shared error handling
func (r *UserRepository) GetActiveUsers(ctx context.Context) ([]repositories.Users, error) {
    return repositories.RetryOperationSlice(ctx, repositories.DefaultRetryConfig, "get_active_users", func(ctx context.Context) ([]repositories.Users, error) {
        query := `SELECT id, name, email, created_at FROM users WHERE is_active = true ORDER BY created_at DESC`
        
        rows, err := repositories.ExecuteQuery(ctx, r.db, "get_active_users", "Users", query)
        if err != nil {
            return nil, err
        }
        defer rows.Close()
        
        var results []repositories.Users
        for rows.Next() {
            var user repositories.Users
            err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
            if err != nil {
                return nil, repositories.HandleDatabaseError("scan", "Users", err)
            }
            results = append(results, user)
        }
        
        return results, repositories.HandleRowsResult("Users", rows)
    })
}

// Service layer uses the interface, fulfilled by the user's repository
type UserService struct {
    userRepo UserInterface  // Property of interface type
}

func NewUserService(userRepo UserInterface) *UserService {
    return &UserService{
        userRepo: userRepo,
    }
}

// Service methods delegate to repository through interface
func (s *UserService) RegisterUser(ctx context.Context, name, email string) (*repositories.Users, error) {
    params := repositories.CreateUsersParams{
        Name:  name,
        Email: email,
    }
    return s.userRepo.CreateUser(ctx, params)
}

func (s *UserService) GetUserDashboard(ctx context.Context) ([]repositories.Users, error) {
    // Business logic can use any interface methods
    return s.userRepo.GetActiveUsers(ctx)
}
```

### Usage in Application

```go
func main() {
    db, _ := pgxkit.New(ctx, "postgres://...")
    
    // Create repository that implements interface
    userRepo := NewUserRepository(db)
    
    // Service has property of interface type, fulfilled by repository
    userService := NewUserService(userRepo)
    
    // Use service for business operations
    user, err := userService.RegisterUser(ctx, "John", "john@example.com")
    dashboard, err := userService.GetUserDashboard(ctx)
}
```

## 🎯 Benefits Summary

### For Generated Code
- **90% reduction** in template duplication
- **Consistent error handling** across all repositories
- **Zero runtime overhead** - utilities generate concrete code
- **Full type safety** maintained

### For Implementers
- **Same patterns** for custom and generated code
- **Shared utilities** available across packages
- **Easy composition** with embedding pattern
- **Consistent retry logic** for all operations

### Architecture
- **No interfaces generated** - teams define their own based on domain needs
- **Clean embedding** - generated repositories work perfectly with composition
- **Cross-package utilities** - shared functions available everywhere
- **Zero reflection** - all benefits through compile-time generation

This approach eliminates duplication while maintaining the flexibility and performance that makes skimatik powerful for production use.

## Related Documentation

- **[Embedding Patterns](Embedding-Patterns)** - Repository composition and extension patterns
- **[Error Handling Guide](Error-Handling-Guide)** - Comprehensive error management strategies
- **[Examples & Tutorials](Examples-and-Tutorials)** - Hands-on examples with real applications 