# Error Handling Guide

## Overview

skimatik generates comprehensive error handling patterns that provide structured, actionable error information for application developers. This guide covers the complete error handling strategy from generated repositories to custom business logic.

## 🎯 Error Handling Philosophy

- **Structured Errors**: Clear error types with specific context
- **Actionable Information**: Errors include enough detail for debugging and user feedback
- **Consistent Patterns**: Same error handling across generated and custom code
- **Production Ready**: Appropriate error levels and logging integration

## 🚨 Generated Error Types

### Core Error Types

#### `NotFoundError`
Used when a requested resource doesn't exist. The error handling automatically converts `pgx.ErrNoRows` to `ErrNotFound`.

```go
import (
    "context"
    "fmt"

    "github.com/google/uuid"
)

// Generated usage in Get operations
func (r *UsersRepository) Get(ctx context.Context, id uuid.UUID) (*Users, error) {
    query := `SELECT id, name, email, created_at FROM users WHERE id = $1`

    row := ExecuteQueryRow(ctx, r.db, "get", "Users", query, id)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    if err := HandleQueryRowError("get", "Users", err); err != nil {
        return nil, err
    }
    return &user, nil
}

// Usage in application code
user, err := userRepo.Get(ctx, userID)
if err != nil {
    if IsNotFound(err) {
        return nil, fmt.Errorf("user not found")
    }
    return nil, fmt.Errorf("database error: %w", err)
}
```

#### `AlreadyExistsError`
Used when attempting to create a resource that already exists. The error handling automatically detects PostgreSQL unique constraint violations (error code 23505) and converts them to `ErrAlreadyExists`.

```go
// Generated usage in Create operations
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    query := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at`

    row := ExecuteQueryRow(ctx, r.db, "create", "Users", query, params.Name, params.Email)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    if err := HandleQueryRowError("create", "Users", err); err != nil {
        return nil, err
    }
    return &user, nil
}

// Usage in application code
user, err := userRepo.Create(ctx, params)
if err != nil {
    if IsAlreadyExists(err) {
        return nil, fmt.Errorf("user with email %s already exists", params.Email)
    }
    return nil, fmt.Errorf("failed to create user: %w", err)
}
```

#### `InvalidReferenceError`
Used when a foreign key constraint is violated. The error handling automatically detects PostgreSQL foreign key violations (error code 23503) and converts them to `ErrInvalidReference`.

```go
// Generated usage when referencing non-existent foreign key
func (r *PostsRepository) Create(ctx context.Context, params CreatePostsParams) (*Posts, error) {
    query := `INSERT INTO posts (title, user_id) VALUES ($1, $2) RETURNING id, title, user_id, created_at`

    row := ExecuteQueryRow(ctx, r.db, "create", "Posts", query, params.Title, params.UserID)
    var post Posts
    err := row.Scan(&post.Id, &post.Title, &post.UserID, &post.CreatedAt)
    if err := HandleQueryRowError("create", "Posts", err); err != nil {
        return nil, err
    }
    return &post, nil
}

// Usage in application code
post, err := postRepo.Create(ctx, params)
if err != nil {
    if IsInvalidReference(err) {
        return nil, fmt.Errorf("user %s does not exist", params.UserID)
    }
    return nil, fmt.Errorf("failed to create post: %w", err)
}
```

#### `ValidationError`
Used when database constraints are violated. The error handling automatically detects PostgreSQL check constraint violations (error code 23514) and NOT NULL violations (error code 23502) and converts them to `ErrValidationFailed` or `ErrRequiredField`.

```go
// Database-level validation (PostgreSQL CHECK constraints, NOT NULL)
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    query := `INSERT INTO users (name, email, age) VALUES ($1, $2, $3) RETURNING id, name, email, age, created_at`

    row := ExecuteQueryRow(ctx, r.db, "create", "Users", query, params.Name, params.Email, params.Age)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.Age, &user.CreatedAt)
    // HandleQueryRowError automatically detects constraint violations
    if err := HandleQueryRowError("create", "Users", err); err != nil {
        return nil, err
    }
    return &user, nil
}

// Usage in application code
user, err := userRepo.Create(ctx, params)
if err != nil {
    if IsValidationError(err) {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    return nil, fmt.Errorf("failed to create user: %w", err)
}

// Application-level validation (custom business logic)
func (s *UserService) CreateUser(ctx context.Context, params CreateUsersParams) (*Users, error) {
    // Custom validation before database operation
    if params.Email == "" {
        return nil, fmt.Errorf("email is required")
    }
    if !isValidEmail(params.Email) {
        return nil, fmt.Errorf("invalid email format: %s", params.Email)
    }

    return s.usersRepo.Create(ctx, params)
}
```

#### `DatabaseError`
Used for general database operation failures. All database errors are automatically wrapped in a `DatabaseError` struct that provides structured error information including operation, entity, and error type.

```go
// Generated usage for connection and query errors
func (r *UsersRepository) List(ctx context.Context) ([]Users, error) {
    query := `SELECT id, name, email, created_at FROM users ORDER BY created_at DESC`

    rows, err := ExecuteQuery(ctx, r.db, "list", "Users", query)
    if err != nil {
        // ExecuteQuery uses HandleDatabaseError for connection/query issues
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

## 🔍 Error Detection Functions

### Generated Helper Functions

Every generated package includes these error detection utilities:

```go
// Check specific error types
func IsNotFound(err error) bool
func IsAlreadyExists(err error) bool
func IsInvalidReference(err error) bool
func IsValidationError(err error) bool
func IsConnectionError(err error) bool
func IsTimeout(err error) bool

// Usage example
if err != nil {
    switch {
    case IsNotFound(err):
        // Handle resource not found
        return nil, fmt.Errorf("resource not found")
    case IsAlreadyExists(err):
        // Handle duplicate resource
        return nil, fmt.Errorf("resource already exists")
    case IsInvalidReference(err):
        // Handle foreign key violation
        return nil, fmt.Errorf("referenced resource does not exist")
    case IsValidationError(err):
        // Handle validation failure
        var dbErr *DatabaseError
        if errors.As(err, &dbErr) {
            return nil, fmt.Errorf("validation error: %s", dbErr.Detail)
        }
        return nil, fmt.Errorf("validation error: %w", err)
    case IsTimeout(err):
        // Handle timeout
        return nil, fmt.Errorf("operation timed out")
    case IsConnectionError(err):
        // Handle connection issues
        return nil, fmt.Errorf("database connection failed")
    default:
        // Handle other database errors
        return nil, fmt.Errorf("database error: %w", err)
    }
}
```

## 🛠️ Using Errors in Custom Code

### Pattern 1: Basic Error Handling

```go
import (
    "context"
    "fmt"

    "github.com/google/uuid"
)

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
    user, err := s.userRepo.Get(ctx, id)
    if err != nil {
        if IsNotFound(err) {
            return nil, fmt.Errorf("user %s not found", id)
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}
```

### Pattern 2: Error Context Enhancement

```go
func (s *UserService) CreateUserWithProfile(ctx context.Context, userData CreateUsersParams, profileData string) (*User, error) {
    return RetryOperation(ctx, DefaultRetryConfig, "create_user_with_profile", func(ctx context.Context) (*User, error) {
        user, err := s.userRepo.Create(ctx, userData)
        if err != nil {
            if IsAlreadyExists(err) {
                return nil, fmt.Errorf("user with email %s already exists", userData.Email)
            }
            if IsValidationError(err) {
                return nil, fmt.Errorf("user data validation failed: %w", err)
            }
            return nil, fmt.Errorf("failed to create user: %w", err)
        }
        
        _, err = s.profileRepo.Create(ctx, CreateProfileParams{
            UserID: user.Id,
            Bio:    profileData,
        })
        if err != nil {
            // Enhance error with business context
            if IsValidationError(err) {
                return nil, fmt.Errorf("profile validation failed for user %s: %w", user.Id, err)
            }
            return nil, fmt.Errorf("failed to create profile for user %s: %w", user.Id, err)
        }
        
        return user, nil
    })
}
```

### Pattern 3: Error Mapping for APIs

```go
func mapDatabaseErrorToHTTPStatus(err error) (int, string) {
    switch {
    case IsNotFound(err):
        return 404, "Resource not found"
    case IsAlreadyExists(err):
        return 409, "Resource already exists"
    case IsInvalidReference(err):
        return 422, "Referenced resource does not exist"
    case IsValidationError(err):
        return 400, "Invalid input data"
    case IsTimeout(err):
        return 408, "Request timeout"
    case IsConnectionError(err):
        return 503, "Service temporarily unavailable"
    default:
        return 500, "Internal server error"
    }
}

// HTTP handler example
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var params CreateUsersParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }
    
    user, err := h.userService.CreateUser(r.Context(), params)
    if err != nil {
        statusCode, message := mapDatabaseErrorToHTTPStatus(err)
        
        // Log full error for debugging
        log.Printf("CreateUser error: %v", err)
        
        // Return user-friendly message
        http.Error(w, message, statusCode)
        return
    }
    
    json.NewEncoder(w).Encode(user)
}
```

## 📊 Error Logging and Monitoring

### Structured Logging

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
)

func (s *UserService) logError(operation string, err error, context map[string]interface{}) {
    logData := map[string]interface{}{
        "operation": operation,
        "error":     err.Error(),
        "timestamp": time.Now(),
    }

    // Add context
    for k, v := range context {
        logData[k] = v
    }

    // Add error type information
    switch {
    case IsNotFound(err):
        logData["error_type"] = "not_found"
        logData["severity"] = "info"  // Expected condition
    case IsAlreadyExists(err):
        logData["error_type"] = "already_exists"
        logData["severity"] = "warning"
    case IsInvalidReference(err):
        logData["error_type"] = "invalid_reference"
        logData["severity"] = "warning"
    case IsValidationError(err):
        logData["error_type"] = "validation"
        logData["severity"] = "warning"
        if validationErr, ok := err.(*ValidationError); ok {
            logData["field"] = validationErr.Field
            logData["details"] = validationErr.Details
        }
    case IsTimeout(err):
        logData["error_type"] = "timeout"
        logData["severity"] = "error"
    case IsConnectionError(err):
        logData["error_type"] = "connection"
        logData["severity"] = "critical"
    default:
        logData["error_type"] = "database"
        logData["severity"] = "error"
    }

    // Use structured logger (e.g., logrus, zap)
    logger.WithFields(logData).Log(logData["severity"], "Database operation failed")
}

// Usage in service methods
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
    user, err := s.userRepo.Get(ctx, id)
    if err != nil {
        s.logError("get_user", err, map[string]interface{}{
            "user_id": id,
            "method":  "GetUser",
        })

        if IsNotFound(err) {
            return nil, fmt.Errorf("user not found")
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}
```

### Metrics and Monitoring

```go
type ErrorMetrics struct {
    notFoundCount    prometheus.Counter
    validationCount  prometheus.Counter
    timeoutCount     prometheus.Counter
    connectionCount  prometheus.Counter
    databaseCount    prometheus.Counter
}

func (m *ErrorMetrics) RecordError(operation string, err error) {
    labels := prometheus.Labels{"operation": operation}
    
    switch {
    case IsNotFound(err):
        m.notFoundCount.With(labels).Inc()
    case IsValidationError(err):
        m.validationCount.With(labels).Inc()
    case IsTimeout(err):
        m.timeoutCount.With(labels).Inc()
    case IsConnectionError(err):
        m.connectionCount.With(labels).Inc()
    default:
        m.databaseCount.With(labels).Inc()
    }
}
```

## ⚡ Error Handling with Retry Logic

### Smart Retry Based on Error Type

```go
func (r *UsersRepository) CreateWithRetry(ctx context.Context, params CreateUsersParams) (*Users, error) {
    return RetryOperation(ctx, DefaultRetryConfig, "create", func(ctx context.Context) (*Users, error) {
        user, err := r.Create(ctx, params)
        if err != nil {
            // Don't retry validation or already exists errors
            if IsValidationError(err) || IsAlreadyExists(err) {
                return nil, err  // No retry for these
            }
            // Retry for connection, timeout, and other database errors
            return nil, err
        }
        return user, nil
    })
}
```

### Custom Retry Configuration

```go
// Custom retry for critical operations
func (s *UserService) CreateCriticalUser(ctx context.Context, params CreateUsersParams) (*User, error) {
    criticalRetryConfig := RetryConfig{
        MaxRetries: 5,
        BaseDelay:  500 * time.Millisecond,
    }
    
    return RetryOperation(ctx, criticalRetryConfig, "create_critical_user", func(ctx context.Context) (*User, error) {
        user, err := s.userRepo.Create(ctx, params)
        if err != nil {
            // Log retry attempts
            s.logError("create_critical_user_attempt", err, map[string]interface{}{
                "user_email": params.Email,
                "attempt":    "retry",
            })
        }
        return user, err
    })
}
```

## 🎯 Best Practices

### 1. Error Context Enhancement
Always add business context to database errors:

```go
// Good
return nil, fmt.Errorf("failed to create user profile for user %s: %w", userID, err)

// Bad
return nil, err
```

### 2. Appropriate Error Types
Use the right error detection:

```go
// Good - specific error handling
if IsNotFound(err) {
    return nil, fmt.Errorf("user not found")
}

// Bad - generic error handling
if err != nil {
    return nil, err
}
```

### 3. Error Logging Levels
Use appropriate logging levels:

```go
// Info level for expected conditions
if IsNotFound(err) {
    log.Info("User not found", "user_id", id)
}

// Error level for unexpected failures
if IsConnectionError(err) {
    log.Error("Database connection failed", "error", err)
}
```

### 4. Don't Retry Non-Retriable Errors
```go
// Good - selective retry
if IsValidationError(err) || IsAlreadyExists(err) {
    return nil, err  // Don't retry
}

// Bad - retry everything
return RetryOperation(ctx, config, "operation", func(ctx context.Context) (*T, error) {
    return someOperation(ctx)
})
```

## 🔗 Integration with External Systems

### Error Translation for APIs

```go
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

func translateDatabaseError(err error) APIError {
    switch {
    case IsNotFound(err):
        return APIError{
            Code:    "RESOURCE_NOT_FOUND",
            Message: "The requested resource was not found",
        }
    case IsAlreadyExists(err):
        if existsErr, ok := err.(*AlreadyExistsError); ok {
            return APIError{
                Code:    "RESOURCE_ALREADY_EXISTS",
                Message: "Resource already exists",
                Details: map[string]interface{}{
                    "field": existsErr.Field,
                    "value": existsErr.Value,
                },
            }
        }
    case IsValidationError(err):
        if validationErr, ok := err.(*ValidationError); ok {
            return APIError{
                Code:    "VALIDATION_ERROR",
                Message: validationErr.Message,
                Details: map[string]interface{}{
                    "field":   validationErr.Field,
                    "details": validationErr.Details,
                },
            }
        }
    }
    
    return APIError{
        Code:    "INTERNAL_ERROR",
        Message: "An internal error occurred",
    }
}
```

This comprehensive error handling approach ensures that your applications can gracefully handle database errors while providing meaningful feedback to users and developers.

## Related Documentation

- **[Shared Utilities Guide](shared-utilities)** - Database operations, retry logic, and error handling utilities
- **[Embedding Patterns](embedding-patterns)** - Repository composition and extension patterns
- **[Examples & Tutorials](examples)** - Hands-on examples with real applications 