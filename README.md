# skimatik

[![Go Version](https://img.shields.io/github/go-mod/go-version/nhalm/skimatik)](https://golang.org/doc/devel/release.html)
[![CI Status](https://github.com/nhalm/skimatik/actions/workflows/ci.yml/badge.svg)](https://github.com/nhalm/skimatik/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nhalm/skimatik)](https://goreportcard.com/report/github.com/nhalm/skimatik)
[![Release](https://img.shields.io/github/v/release/nhalm/skimatik)](https://github.com/nhalm/skimatik/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A database-first code generator for PostgreSQL that creates type-safe Go repositories with built-in cursor-based pagination. Generate clean, efficient CRUD operations and custom query functions directly from your database schema.

## Features

- **Database-First**: Works with existing PostgreSQL databases, no schema migrations required
- **Type-Safe**: Generates fully typed Go code using pgx with comprehensive PostgreSQL type support
- **Built-in Pagination**: Every list operation includes efficient cursor-based pagination using UUID v7
- **Zero Dependencies**: Generated code only requires pgx - no external pagination or ORM dependencies
- **Shared Utilities**: Eliminates code duplication with reusable database operation and retry patterns
- **Repository Embedding**: Generated repositories designed for clean composition and extension
- **Production Ready**: Clean, formatted code following Go best practices

## Quick Start

```bash
# Install skimatik
go install github.com/nhalm/skimatik/cmd/skimatik@latest

# Create a configuration file
cat > skimatik.yaml << EOF
database:
  host: localhost
  port: 5432
  name: mydb
  user: postgres
  password: postgres

output:
  package: repository
  dir: ./repository/generated

tables:
  - users
  - posts
EOF

# Generate repositories
skimatik

# Use the generated code
```

## Documentation

For comprehensive documentation, examples, and guides, visit the **[skimatik Wiki](https://github.com/nhalm/skimatik/wiki)**.

### Key Documentation

- **[Quick Start Guide](https://github.com/nhalm/skimatik/wiki/quick-start)** - Installation and basic usage
- **[Examples & Tutorials](https://github.com/nhalm/skimatik/wiki/examples)** - Real-world usage examples
- **[Configuration Reference](https://github.com/nhalm/skimatik/wiki/configuration-reference)** - Complete configuration options
- **[Database Migrations](https://github.com/nhalm/skimatik/wiki/database-migrations)** - Schema management with golang-migrate

## Example

Here's a simple example of what skimatik generates:

```go
// Generated repository with full CRUD operations
type UserRepository struct {
    db         pgxkit.DBConn
    operations *DatabaseOperations
}

// Type-safe user retrieval
user, err := userRepo.GetByID(ctx, userID)

// Built-in cursor-based pagination
page, err := userRepo.ListPaginated(ctx, ListUsersParams{
    Limit:  20,
    Cursor: cursor,
})

// Custom queries from SQL files with semantic parameter names
activeUsers, err := userRepo.GetActiveUsers(ctx, 50) // limit parameter
userByEmail, err := userRepo.GetUserByEmail(ctx, "user@example.com") // email parameter
searchResults, err := userRepo.SearchUsers(ctx, "john", 20) // searchTerm, limit parameters

// Optional parameters with nullable types
allUsers, err := userRepo.ListUsersWithFilters(ctx, 100, nil, nil) // all users
activeOnly := true
activeUsers, err := userRepo.ListUsersWithFilters(ctx, 100, &activeOnly, nil) // filtered by status
```

### Nullable Parameters for Optional Filters

For queries with optional filter parameters, use `-- param:` annotations:

```sql
-- name: ListUsersWithFilters :many
-- param: $1 limit int
-- param: $2 is_active *bool
-- param: $3 name_filter *string
SELECT id, name, email, is_active, created_at
FROM users
WHERE ($2::boolean IS NULL OR is_active = $2)
  AND ($3::text IS NULL OR name ILIKE $3)
ORDER BY created_at DESC
LIMIT $1;
```

This generates:

```go
func (q *Queries) ListUsersWithFilters(
    ctx context.Context,
    limit int,
    isActive *bool,
    nameFilter *string,
) ([]ListUsersWithFiltersResult, error)
```

**Rules:**
- If ANY parameter has a `-- param:` annotation, ALL parameters must be annotated
- Parameters must be sequential: `$1, $2, $3, ...`
- Use pointer types (`*string`, `*int`, etc.) for optional parameters
- Your SQL must use `IS NULL OR` pattern for optional parameters

## Requirements

- Go 1.24+
- PostgreSQL (any version supported by pgx)
- Tables must have UUID v7 primary keys for pagination support

## Installation

```bash
go install github.com/nhalm/skimatik/cmd/skimatik@latest
```

For more installation options and detailed setup instructions, see the [Quick Start Guide](https://github.com/nhalm/skimatik/wiki/quick-start).

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](https://github.com/nhalm/skimatik/wiki/contributing) for details.

## License

skimatik is licensed under the [MIT License](LICENSE).

## Support

- **[Documentation Wiki](https://github.com/nhalm/skimatik/wiki)** - Comprehensive guides and references
- **[GitHub Issues](https://github.com/nhalm/skimatik/issues)** - Bug reports and feature requests
- **[Discussions](https://github.com/nhalm/skimatik/discussions)** - Community help and questions

---

Built with ❤️ for the Go community. Making PostgreSQL development delightful, one repository at a time.