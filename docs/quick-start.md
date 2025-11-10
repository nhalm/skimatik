# Quick Start Guide

Get up and running with skimatik in just a few minutes. This guide will walk you through installation, configuration, and generating your first repositories.

## 🚀 See It In Action

Want to see a complete working example first? Check out the **[Complete Blog Application Example](examples#-complete-blog-application-example)** which demonstrates skimatik with a full HTTP API, database schema, and generated repositories.

```bash
# Quick demo with the example app
git clone https://github.com/nhalm/skimatik.git
cd skimatik/example-app
make setup && make generate && make run
```

## Prerequisites

- **Go 1.24+** - Required for installation and generated code
- **PostgreSQL** - Any version supported by pgx
- **Database with UUID v7 primary keys** - Required for pagination support

## Installation

### Option 1: Install Latest Release (Recommended)

```bash
go install github.com/nhalm/skimatik/cmd/skimatik@latest
```

### Option 2: Install Specific Version

```bash
go install github.com/nhalm/skimatik/cmd/skimatik@v1.0.0
```

### Option 3: Build from Source

```bash
git clone https://github.com/nhalm/skimatik.git
cd skimatik
make build
./bin/skimatik --version
```

## Setting Up Database Migrations

Skimatik works with your existing database schema. We recommend using [golang-migrate](https://github.com/golang-migrate/migrate) to manage schema changes:

### Install golang-migrate

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/migrate
```

### Create Your First Migration

```bash
# Create migrations directory
mkdir -p database/migrations

# Create initial schema migration
migrate create -ext sql -dir database/migrations -seq initial_schema
```

### Write Your Schema

Edit `database/migrations/000001_initial_schema.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Apply Migrations

```bash
migrate -database "postgres://username:password@localhost:5432/database" \
        -path database/migrations up
```

See the complete [Database Migrations](database-migrations.md) guide for detailed workflows and best practices.

## Quick Setup

### 1. Create Configuration File

Create `skimatik.yaml` in your project root:

```yaml
database:
  dsn: "postgres://username:password@localhost:5432/database"
  schema: "public"

output:
  directory: "./repositories"
  package: "repositories"

default_functions: "all"  # Generate all CRUD operations

tables:
  users:
  posts:
  comments:

verbose: true
```

### 2. Generate Repositories

```bash
skimatik --config=skimatik.yaml
```

### 3. View Generated Code

```bash
ls repositories/
# Output:
# users_generated.go
# posts_generated.go  
# comments_generated.go
# pagination.go
# errors.go
# database_operations.go
# retry_operations.go
```

## Basic Usage Examples

### 1. Simple CRUD Operations

```go
package main

import (
    "context"
    "log"
    
    "github.com/nhalm/pgxkit"
    "your-project/repositories"
)

func main() {
    // Connect to database
    db := pgxkit.NewDB()
    err := db.Connect(context.Background(), "postgres://...")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Shutdown(context.Background())
    
    // Create repository
    userRepo := repositories.NewUsersRepository(db)
    
    // Create user
    user, err := userRepo.Create(ctx, repositories.CreateUsersParams{
        Name:  "John Doe",
        Email: "john@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Get user by ID
    fetchedUser, err := userRepo.Get(ctx, user.Id)
    if err != nil {
        log.Fatal(err)
    }
    
    // List all users
    users, err := userRepo.List(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Update user
    updatedUser, err := userRepo.Update(ctx, user.Id, repositories.UpdateUsersParams{
        Name: "Jane Doe",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Delete user
    err = userRepo.Delete(ctx, user.Id)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Pagination

```go
// List first page (10 items)
result, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
    Limit: 10,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Users: %+v\n", result.Items)
fmt.Printf("Has more: %v\n", result.HasMore)

// Get next page using cursor
if result.HasMore {
    nextResult, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
        Limit:  10,
        Cursor: result.NextCursor,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 3. Error Handling

```go
import "your-project/repositories"

user, err := userRepo.Get(ctx, userID)
if err != nil {
    if repositories.IsNotFound(err) {
        // Handle not found
        return nil, fmt.Errorf("user not found")
    }
    
    if repositories.IsAlreadyExists(err) {
        // Handle duplicate
        return nil, fmt.Errorf("user already exists")
    }
    
    // Handle other database errors
    return nil, fmt.Errorf("database error: %w", err)
}
```

### 4. Repository Embedding

```go
// Define your domain interface
type UserManager interface {
    CreateUser(ctx context.Context, params repositories.CreateUsersParams) (*repositories.Users, error)
    GetActiveUsers(ctx context.Context) ([]repositories.Users, error)
}

// Implement by embedding generated repository
type UserService struct {
    *repositories.UsersRepository // Embed for CRUD operations
    db *pgxkit.DB                 // Explicit db field for custom operations
}

func NewUserService(db *pgxkit.DB) UserManager {
    return &UserService{
        UsersRepository: repositories.NewUsersRepository(db),
        db:             db,
    }
}

// Add custom business logic using shared utilities
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
    
    return users, repositories.HandleRowsResult("Users", rows)
}
```

## Configuration Options

For complete configuration reference, see [Configuration Reference](configuration-reference.md).

### Database Configuration

```yaml
database:
  dsn: "postgres://user:pass@localhost:5432/db"  # Required
  schema: "public"                               # Default: "public"
```

### Output Configuration

```yaml
output:
  directory: "./repositories"    # Default: "./repositories"
  package: "repositories"        # Default: "repositories"
```

### Default Functions

```yaml
default_functions: "all"  # Or specify: ["create", "get", "update", "delete", "list", "paginate"]
```

### Tables

```yaml
tables:
  users:   # Use default_functions
  posts:
    functions: ["create", "get", "list"]  # Override per table
  comments:
```

## Environment Variables

skimatik supports standard PostgreSQL environment variables:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"

# Or use individual components
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_USER="myuser"
export POSTGRES_PASSWORD="mypass"
export POSTGRES_DB="mydb"
export POSTGRES_SSLMODE="disable"
```

## CLI Options

```bash
# Basic usage
skimatik

# Use custom config file
skimatik --config=custom.yaml

# Enable verbose logging
skimatik --verbose

# Show version
skimatik --version

# Show help
skimatik --help
```

## Troubleshooting

### Common Issues

**1. "Table does not have UUID primary key"**
```
Error: Table 'users' does not have a UUID primary key. Only UUID v7 primary keys are supported for pagination.
```
Solution: Ensure your tables use UUID v7 primary keys for pagination support.

**2. "Cannot connect to database"**
```
Error: failed to connect to database: connection refused
```
Solution: Check your database connection string and ensure PostgreSQL is running.

**3. "Permission denied"**
```
Error: permission denied for schema public
```
Solution: Ensure your database user has read permissions on the schema.

## Next Steps

- **[Examples](examples.md)** - Hands-on examples with real applications
- **[Shared Utilities Guide](shared-utilities.md)** - Learn about built-in utilities for common patterns
- **[Embedding Patterns](embedding-patterns.md)** - Advanced repository composition patterns
- **[Configuration Reference](configuration-reference.md)** - Complete configuration documentation

## Best Practices

1. **Use UUID v7 primary keys** for optimal pagination performance
2. **Define domain interfaces** instead of using repositories directly
3. **Embed repositories** in services for business logic
4. **Use shared utilities** to maintain consistency across custom code
5. **Regenerate safely** - custom code won't be overwritten 