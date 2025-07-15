# dbutil-gen

[![Go Version](https://img.shields.io/github/go-mod/go-version/nhalm/dbutil)](https://golang.org/doc/devel/release.html)
[![CI Status](https://github.com/nhalm/dbutil/actions/workflows/ci.yml/badge.svg)](https://github.com/nhalm/dbutil/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nhalm/dbutil)](https://goreportcard.com/report/github.com/nhalm/dbutil)
[![Release](https://img.shields.io/github/v/release/nhalm/dbutil)](https://github.com/nhalm/dbutil/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A database-first code generator for PostgreSQL that creates type-safe Go repositories with built-in cursor-based pagination. Generate clean, efficient CRUD operations and custom query functions directly from your database schema.

## ✨ Features

- **Database-First**: Works with existing PostgreSQL databases, no schema migrations required
- **Type-Safe**: Generates fully typed Go code using pgx with comprehensive PostgreSQL type support
- **Built-in Pagination**: Every list operation includes efficient cursor-based pagination using UUID v7
- **Zero Dependencies**: Generated code only requires pgx - no external pagination or ORM dependencies
- **Table-Based Generation**: Complete CRUD repositories for all database tables
- **Query-Based Generation**: Custom functions from SQL files with sqlc-style annotations
- **UUID v7 Optimized**: Time-ordered pagination with consistent performance
- **Production Ready**: Clean, formatted code following Go best practices

## 🚀 Quick Start

### Installation

```bash
go install github.com/nhalm/dbutil/cmd/dbutil-gen@latest
```

### Basic Usage

1. **Generate table-based repositories:**
```bash
dbutil-gen --dsn="postgres://user:pass@localhost/mydb" --tables --output="./repositories"
```

2. **Generated code example:**
```go
// users_generated.go
type Users struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (u Users) GetID() uuid.UUID { return u.ID }

// Complete CRUD operations
func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*Users, error)
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error)
func (r *UsersRepository) Update(ctx context.Context, id uuid.UUID, params UpdateUsersParams) (*Users, error)
func (r *UsersRepository) Delete(ctx context.Context, id uuid.UUID) error
func (r *UsersRepository) List(ctx context.Context) ([]Users, error)
func (r *UsersRepository) ListPaginated(ctx context.Context, params PaginationParams) (*PaginationResult[Users], error)
```

3. **Use in your application:**
```go
package main

import (
    "context"
    "log"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "your-project/repositories"
)

func main() {
    ctx := context.Background()
    
    // Connect to database
    conn, err := pgxpool.New(ctx, "postgres://user:pass@localhost/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Use generated repository
    userRepo := repositories.NewUsersRepository(conn)
    
    // Create a user
    user, err := userRepo.Create(ctx, repositories.CreateUsersParams{
        Name:  "John Doe",
        Email: "john@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // List users with pagination
    result, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
        Limit: 20,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Found %d users, has more: %v", len(result.Items), result.HasMore)
}
```

## 📋 Requirements

- **PostgreSQL 12+** (tested with PostgreSQL 15)
- **Go 1.21+** (requires generics support)
- **UUID v7 Primary Keys** (required for pagination)

### Database Requirements

All tables must have UUID primary keys for pagination to work. The tool will reject tables with non-UUID primary keys:

```sql
-- ✅ Supported
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL
);

-- ❌ Not supported (will be skipped with error message)
CREATE TABLE legacy_table (
    id SERIAL PRIMARY KEY,
    name TEXT
);
```

## 🛠️ Installation & Setup

### 1. Install the Tool

```bash
# Install latest version
go install github.com/nhalm/dbutil/cmd/dbutil-gen@latest

# Or download binary from releases
curl -L https://github.com/nhalm/dbutil/releases/latest/download/dbutil-gen-linux-amd64 -o dbutil-gen
chmod +x dbutil-gen
```

### 2. Docker Usage (Alternative)

You can also run dbutil-gen using Docker:

```bash
# Build the Docker image
docker build -t dbutil-gen .

# Run the tool in a container
docker run --rm -it --network host dbutil-gen --help

# Or use the Makefile
make docker-build
make docker-run
```

### 3. Prepare Your Database

Ensure your PostgreSQL database has UUID v7 primary keys:

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Example table with UUID primary key
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. Generate Code

```bash
# Basic table generation
dbutil-gen --dsn="postgres://user:pass@localhost/mydb" --tables

# With custom output directory and package
dbutil-gen \
  --dsn="postgres://user:pass@localhost/mydb" \
  --tables \
  --output="./internal/repositories" \
  --package="repositories"

# Exclude specific tables
dbutil-gen \
  --dsn="postgres://user:pass@localhost/mydb" \
  --tables \
  --exclude="migrations,schema_versions"
```

## 📖 Usage Guide

### Command Line Options

```bash
dbutil-gen [options]

Options:
  -dsn string
        PostgreSQL connection string (or use DATABASE_URL env var)
  -tables
        Generate table-based repositories
  -queries string
        Directory containing SQL query files for custom generation
  -output string
        Output directory for generated files (default "./repositories")
  -package string
        Package name for generated code (default "repositories")
  -schema string
        Database schema to introspect (default "public")
  -include string
        Comma-separated list of tables to include
  -exclude string
        Comma-separated list of tables to exclude
  -config string
        Path to configuration file
  -verbose
        Enable verbose logging
```

### Environment Variables

```bash
# Database connection
export DATABASE_URL="postgres://user:pass@localhost/mydb"

# Then run without --dsn
dbutil-gen --tables
```

### Configuration File

Create `dbutil-gen.yaml`:

```yaml
dsn: "postgres://user:pass@localhost/mydb"
output: "./internal/repositories"
package: "repositories"
schema: "public"
tables: true
exclude:
  - "migrations"
  - "schema_versions"
verbose: true
```

Use with:
```bash
dbutil-gen --config=dbutil-gen.yaml
```

### Table Filtering

```bash
# Include only specific tables
dbutil-gen --dsn="..." --tables --include="users,posts,comments"

# Exclude specific tables
dbutil-gen --dsn="..." --tables --exclude="migrations,temp_*"

# Include and exclude (include takes precedence)
dbutil-gen --dsn="..." --tables --include="users,posts" --exclude="posts"
# Result: only "users" table
```

## 🔄 Pagination

All generated repositories include efficient cursor-based pagination using UUID v7 time-ordering:

### Basic Pagination

```go
// Get first page
result, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
    Limit: 20, // Max 100, default 20
})

// Get next page
if result.HasMore {
    nextResult, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
        Cursor: result.NextCursor,
        Limit:  20,
    })
}
```

### Pagination Response

```go
type PaginationResult[T any] struct {
    Items      []T     `json:"items"`        // The actual data
    HasMore    bool    `json:"has_more"`     // True if more pages available
    NextCursor string  `json:"next_cursor"`  // Cursor for next page
    Total      *int    `json:"total"`        // Optional total count
}
```

### Integration with Web APIs

```go
func handleListUsers(w http.ResponseWriter, r *http.Request) {
    cursor := r.URL.Query().Get("cursor")
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 20
    }
    
    result, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
        Cursor: cursor,
        Limit:  limit,
    })
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    
    json.NewEncoder(w).Encode(result)
}
```

## 🗃️ Supported PostgreSQL Types

| PostgreSQL Type | Go Type | Notes |
|----------------|---------|--------|
| `uuid` | `uuid.UUID` | Required for primary keys |
| `text`, `varchar`, `char` | `string` | |
| `smallint`, `int2` | `int16` | |
| `integer`, `int`, `int4` | `int32` | |
| `bigint`, `int8` | `int64` | |
| `real`, `float4` | `float32` | |
| `double precision`, `float8` | `float64` | |
| `numeric`, `decimal` | `float64` | |
| `boolean`, `bool` | `bool` | |
| `date` | `time.Time` | |
| `time`, `timetz` | `time.Time` | |
| `timestamp`, `timestamptz` | `time.Time` | |
| `bytea` | `[]byte` | |
| `json`, `jsonb` | `json.RawMessage` | |
| `inet`, `cidr` | `string` | |
| `macaddr` | `string` | |
| Arrays | `[]T` | For any supported type T |

### Nullable Types

Nullable columns automatically use pgtype equivalents:

| PostgreSQL | Go Type (Nullable) |
|-----------|-------------------|
| `text` | `pgtype.Text` |
| `integer` | `pgtype.Int4` |
| `boolean` | `pgtype.Bool` |
| `timestamp` | `pgtype.Timestamptz` |
| etc. | `pgtype.*` |

## 🔧 Integration with go generate

Add to your Go files:

```go
//go:generate dbutil-gen --dsn="$DATABASE_URL" --tables --output="./repositories"
```

Then run:
```bash
go generate ./...
```

## 🏗️ Project Structure

Generated files follow a consistent pattern:

```
repositories/
├── pagination.go              # Shared pagination types (generated once)
├── users_generated.go         # Users table repository
├── posts_generated.go         # Posts table repository
├── comments_generated.go      # Comments table repository
└── ...
```

### Generated File Structure

Each `*_generated.go` file contains:

1. **Struct Definition**: Represents the database table
2. **GetID Method**: Required for pagination interface
3. **Repository Struct**: Holds database connection
4. **Constructor**: `NewXRepository(conn *pgxpool.Pool)`
5. **CRUD Operations**:
   - `GetByID(ctx, id) (*T, error)`
   - `Create(ctx, params) (*T, error)`
   - `Update(ctx, id, params) (*T, error)`
   - `Delete(ctx, id) error`
   - `List(ctx) ([]T, error)`
   - `ListPaginated(ctx, params) (*PaginationResult[T], error)`

## 🎯 Best Practices

### 1. Database Design

```sql
-- ✅ Good: UUID v7 primary keys
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ✅ Good: Proper foreign key relationships
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    title TEXT NOT NULL
);
```

### 2. Generated Code Usage

```go
// ✅ Good: Use dependency injection
type UserService struct {
    userRepo *repositories.UsersRepository
}

func NewUserService(conn *pgxpool.Pool) *UserService {
    return &UserService{
        userRepo: repositories.NewUsersRepository(conn),
    }
}

// ✅ Good: Handle pagination properly
func (s *UserService) ListUsers(cursor string, limit int) (*repositories.PaginationResult[repositories.Users], error) {
    return s.userRepo.ListPaginated(ctx, repositories.PaginationParams{
        Cursor: cursor,
        Limit:  limit,
    })
}
```

### 3. Error Handling

```go
// ✅ Good: Proper error handling
user, err := userRepo.GetByID(ctx, userID)
if err != nil {
    if err == pgx.ErrNoRows {
        return nil, ErrUserNotFound
    }
    return nil, fmt.Errorf("failed to get user: %w", err)
}
```

## 🚧 Migration Guide

### Migrating to UUID v7 Primary Keys

If you have existing tables with integer primary keys, here's how to migrate:

#### 1. For New Applications

Start with UUID v7 from the beginning:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- other columns...
);
```

#### 2. For Existing Applications

**Option A: Add UUID column alongside integer ID**

```sql
-- 1. Add UUID column
ALTER TABLE users ADD COLUMN uuid_id UUID DEFAULT gen_random_uuid();

-- 2. Make it NOT NULL
ALTER TABLE users ALTER COLUMN uuid_id SET NOT NULL;

-- 3. Add unique constraint
ALTER TABLE users ADD CONSTRAINT users_uuid_id_unique UNIQUE (uuid_id);

-- 4. Update foreign key tables to reference UUID
-- (This requires careful planning and migration)

-- 5. Eventually drop integer ID and rename
ALTER TABLE users DROP COLUMN id;
ALTER TABLE users RENAME COLUMN uuid_id TO id;
ALTER TABLE users ADD PRIMARY KEY (id);
```

**Option B: Create new tables with UUID and migrate data**

```sql
-- 1. Create new table with UUID
CREATE TABLE users_new (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    old_id INTEGER, -- temporary reference
    name TEXT NOT NULL,
    -- other columns...
);

-- 2. Migrate data
INSERT INTO users_new (old_id, name, ...)
SELECT id, name, ... FROM users;

-- 3. Update application to use new table
-- 4. Drop old table when ready
```

### Performance Considerations

- UUID v7 is time-ordered, providing good performance for pagination
- Consider partitioning for very large tables
- Index foreign key columns that reference UUIDs

## 🔍 Troubleshooting

### Common Issues

**1. "Table has non-UUID primary key"**
```
Error: primary key column id must be UUID type, got integer
```
Solution: Migrate to UUID primary keys (see Migration Guide above)

**2. "Connection refused"**
```
Error: failed to connect to database
```
Solution: Check your DSN and ensure PostgreSQL is running

**3. "Generated code doesn't compile"**
```
Error: missing go.sum entry for module
```
Solution: Run `go mod tidy` in your project directory

**4. "No tables found"**
```
Warning: No tables found in schema 'public'
```
Solution: Check schema name with `--schema` flag or verify database has tables

### Debug Mode

Use `--verbose` for detailed logging:

```bash
dbutil-gen --dsn="..." --tables --verbose
```

Output:
```
2025/01/15 10:30:00 Connected to database, schema: public
2025/01/15 10:30:00 Found 5 tables in schema 'public'
2025/01/15 10:30:00 Generating code for 3 tables after filtering
2025/01/15 10:30:00 Generating repository for table: users
Generated: repositories/users_generated.go
...
```

## 📊 Performance

- **Fast Generation**: 100 tables in ~15 seconds
- **Efficient Pagination**: O(log n) cursor-based queries with UUID v7 ordering
- **Zero Dependencies**: Generated code only requires `pgx`

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repository
git clone https://github.com/nhalm/dbutil.git
cd dbutil

# Install dependencies
go mod download

# Run tests
make test

# Start test database
make test-setup

# Run integration tests
make integration-test

# Test code generation
make test-generate
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver and toolkit
- [sqlc](https://github.com/kyleconroy/sqlc) - Inspiration for query-based generation
- [UUID v7](https://uuid7.com/) - Time-ordered UUID specification

---

**Made with ❤️ for the Go and PostgreSQL communities**