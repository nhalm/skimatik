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
Used when a requested resource doesn't exist.

```go
// Generated usage in Get operations
func (r *UsersRepository) Get(ctx context.Context, id uuid.UUID) (*Users, error) {
    query := `SELECT id, name, email, created_at FROM users WHERE id = $1`
    
    row := ExecuteQueryRow(ctx, r.db, "get", "Users", query, id)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, NewNotFoundError("Users", id.String())
        }
        return nil, HandleQueryRowError("get", "Users", err)
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
Used when attempting to create a resource that already exists.

```go
// Generated usage in Create operations
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    query := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at`
    
    row := ExecuteQueryRow(ctx, r.db, "create", "Users", query, params.Name, params.Email)
    var user Users
    err := row.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
    if err != nil {
        // PostgreSQL unique constraint violation
        if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
            return nil, NewAlreadyExistsError("Users", "email", params.Email)
        }
        return nil, HandleQueryRowError("create", "Users", err)
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

#### `ValidationError`
Used when input data fails validation rules.

```go
// Generated usage with parameter validation
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    // Validate required fields
    if params.Name == "" {
        return nil, NewValidationError("Users", "create", "name", "name cannot be empty", nil)
    }
    if params.Email == "" {
        return nil, NewValidationError("Users", "create", "email", "email cannot be empty", nil)
    }
    
    // Email format validation
    if !isValidEmail(params.Email) {
        return nil, NewValidationError("Users", "create", "email", "invalid email format", 
            map[string]interface{}{"provided": params.Email})
    }
    
    // Proceed with database operation...
}

// Usage in application code
user, err := userRepo.Create(ctx, params)
if err != nil {
    if IsValidation(err) {
        validationErr := err.(*ValidationError)
        return nil, fmt.Errorf("validation failed for %s: %s", validationErr.Field, validationErr.Message)
    }
    return nil, fmt.Errorf("failed to create user: %w", err)
}
```

#### `DatabaseError`
Used for general database operation failures.

```go
// Generated usage for connection and query errors
func (r *UsersRepository) List(ctx context.Context) ([]Users, error) {
    query := `SELECT id, name, email, created_at FROM users ORDER BY created_at DESC`
    
    rows, err := ExecuteQuery(ctx, r.db, "list", "Users", query)
    if err != nil {
        // ExecuteQuery returns DatabaseError for connection issues
        return nil, err  
    }
    defer rows.Close()
    
    var results []Users
    for rows.Next() {
        var user Users
        err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
        if err != nil {
            return nil, NewDatabaseError("Users", "scan", err)
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
func IsValidation(err error) bool
func IsDatabase(err error) bool
func IsTimeout(err error) bool
func IsConnection(err error) bool

// Usage example
if err != nil {
    switch {
    case IsNotFound(err):
        // Handle resource not found
        return nil, fmt.Errorf("resource not found")
    case IsAlreadyExists(err):
        // Handle duplicate resource
        return nil, fmt.Errorf("resource already exists")
    case IsValidation(err):
        // Handle validation failure
        validationErr := err.(*ValidationError)
        return nil, fmt.Errorf("validation error: %s", validationErr.Message)
    case IsTimeout(err):
        // Handle timeout
        return nil, fmt.Errorf("operation timed out")
    case IsConnection(err):
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
            if IsValidation(err) {
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
            if IsValidation(err) {
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
    case IsValidation(err):
        return 400, "Invalid input data"
    case IsTimeout(err):
        return 408, "Request timeout"
    case IsConnection(err):
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
    case IsValidation(err):
        logData["error_type"] = "validation"
        logData["severity"] = "warning"
        if validationErr, ok := err.(*ValidationError); ok {
            logData["field"] = validationErr.Field
            logData["details"] = validationErr.Details
        }
    case IsTimeout(err):
        logData["error_type"] = "timeout"
        logData["severity"] = "error"
    case IsConnection(err):
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
    case IsValidation(err):
        m.validationCount.With(labels).Inc()
    case IsTimeout(err):
        m.timeoutCount.With(labels).Inc()
    case IsConnection(err):
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
            if IsValidation(err) || IsAlreadyExists(err) {
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
if IsConnection(err) {
    log.Error("Database connection failed", "error", err)
}
```

### 4. Don't Retry Non-Retriable Errors
```go
// Good - selective retry
if IsValidation(err) || IsAlreadyExists(err) {
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
    case IsValidation(err):
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

- **[Shared Utilities Guide](Shared-Utilities-Guide)** - Database operations, retry logic, and error handling utilities
- **[Embedding Patterns](Embedding-Patterns)** - Repository composition and extension patterns
- **[Examples & Tutorials](Examples-and-Tutorials)** - Hands-on examples with real applications 