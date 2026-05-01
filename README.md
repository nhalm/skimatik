# skimatik

[![Go Version](https://img.shields.io/github/go-mod/go-version/nhalm/skimatik)](https://golang.org/doc/devel/release.html)
[![CI Status](https://github.com/nhalm/skimatik/actions/workflows/ci.yml/badge.svg)](https://github.com/nhalm/skimatik/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nhalm/skimatik)](https://goreportcard.com/report/github.com/nhalm/skimatik)
[![Release](https://img.shields.io/github/v/release/nhalm/skimatik)](https://github.com/nhalm/skimatik/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A database-first code generator for PostgreSQL that creates type-safe Go repositories with built-in bidirectional cursor-based pagination. Generate clean, efficient CRUD operations and custom query functions directly from your database schema.

## Why skimatik?

**Data is your domain**. Not business logic, not application code—your database schema is the true representation of your domain model. When you start with an ORM, you're defining your domain in application code, hiding critical information from the database where it belongs. This creates problems:

- **Domain knowledge lives in the wrong place**: ORMs encourage logic in the application layer that should be constraints, indexes, and views in the database
- **Tools can't help you**: Gen AI, data analysis tools, and database introspection can't understand your domain when it's scattered across application code
- **Inefficient patterns emerge**: ORMs make it easy to run 100 queries when a single CTE could do the job, wasting CPU and memory assembling data that PostgreSQL could deliver complete
- **Database illiteracy spreads**: When ORM-generated queries appear in logs as indecipherable messes, developers avoid learning how databases actually work—missing the chance to build faster, more scalable applications

**skimatik takes a different approach**: Your PostgreSQL schema is the source of truth.

**How it works**:

1. **Database-aware generation**: Uses PostgreSQL's own query analyzer and planner to generate code. If the database says a column is NOT NULL, you get native Go types (`string`, `uuid.UUID`). Nullable? You get pointers (`*string`, `*uuid.UUID`). No wrapper types to unwrap.

2. **Leverage PostgreSQL's power**: Want to join 5 tables with a CTE? Write the SQL. Get a type-safe function generated from it. The database does what it's incredible at—assembling complex data. Your application gets clean result types with zero reflection.

3. **Self-documenting data layer**: Your database schema plus generated repositories create a clear contract. Other developers, AI tools, and data consumers can understand your domain by looking at the database—not spelunking through application code.

4. **Production patterns included**: Bidirectional cursor-based pagination using UUID v7 time-ordering with runtime sort selection, retry logic for transient failures, comprehensive error handling. These aren't afterthoughts—they're built into every generated repository.

5. **Designed for extension**: Embed generated repositories into your own types. Add business logic, implement domain interfaces, compose repositories. The generated CRUD and utilities are your foundation, not your limitation.

**Compared to ORMs**: ORMs hide the database, generate inefficient queries, and make debugging painful. They discourage developers from learning how databases work, leading to applications that need more resources and scale poorly.

**Compared to sqlc**: sqlc gives you full SQL control and type safety through static SQL parsing, which is excellent. skimatik takes it further by using PostgreSQL's own query analyzer and information schema—we generate code based on what the database actually understands about your schema. You also get automatic CRUD repositories with built-in pagination, plus shared utilities for retry logic and error handling. With sqlc, you write every query manually, including basic operations.

**When to use skimatik**: You believe your data model is your domain model. You want to leverage PostgreSQL's capabilities fully—complex queries, CTEs, proper indexing. You value compile-time safety and zero reflection. You want your database schema to be self-documenting and accessible to tools.

**When to choose something else**: You're prototyping and don't care about performance. You prefer defining your domain in application code. You need to support multiple databases (though in practice, nobody switches databases).

## Features

- **Database-First**: Works with existing PostgreSQL databases, no schema migrations required
- **Type-Safe**: Generates fully typed Go code using pgx with comprehensive PostgreSQL type support
- **Bidirectional Pagination**: Every list operation includes efficient cursor-based pagination using UUID v7 with forward/backward navigation and runtime sort selection
- **Zero Dependencies**: Generated code only requires pgx - no external pagination or ORM dependencies
- **Shared Utilities**: Eliminates code duplication with reusable database operation and retry patterns
- **Repository Embedding**: Generated repositories designed for clean composition and extension
- **Strict-Linter Clean**: Generated code passes the lint set strict downstream consumers commonly run — `errcheck`, `errorlint`, `govet`, `ineffassign`, `rowserrcheck`, `sqlclosecheck`, `staticcheck`, `unused`, plus the [blueprint-vet](https://github.com/nhalm/blueprint-vet) conformance rules. Verified end-to-end on every CI run by regenerating + linting the [example-app](./example-app/).

## Quick Start

```bash
# Install skimatik
go install github.com/nhalm/skimatik/cmd/skimatik@latest

# Create a configuration file
cat > skimatik.yaml << EOF
database:
  dsn: "postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable"
  schema: "public"

output:
  directory: "./repository/generated"
  package: "repository"

default_functions: "all"

tables:
  users:
  posts:
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
type UsersRepository struct {
    db             *pgxkit.DB
    generateIdFunc func() uuid.UUID
}

// Constructor - pass nil for idGen to use default UUID v7 generator
userRepo := generated.NewUsersRepository(db, nil)

// Type-safe user retrieval
user, err := userRepo.Get(ctx, userID)

// Built-in cursor-based pagination
result, err := userRepo.ListPaginated(ctx, generated.PaginationParams{
    Limit:      20,
    NextCursor: cursor,
    OrderBy:    "created_at",
})

// Custom queries from SQL files (generated as separate structs)
userQueries := generated.NewUsersQueries(db)
activeUsers, err := userQueries.GetActiveUsers(ctx, 50)
userByEmail, err := userQueries.GetUserByEmail(ctx, "user@example.com")
searchResults, err := userQueries.SearchUsers(ctx, "john", 20)

// Optional parameters with nullable types
allUsers, err := userQueries.ListUsersWithOptionalFilters(ctx, 100, nil, nil)
activeOnly := true
filtered, err := userQueries.ListUsersWithOptionalFilters(ctx, 100, &activeOnly, nil)
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
- Tables must have single-column primary keys (composite keys not supported)

## Installation

```bash
go install github.com/nhalm/skimatik/cmd/skimatik@latest
```

For more installation options and detailed setup instructions, see the [Quick Start Guide](https://github.com/nhalm/skimatik/wiki/quick-start).

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](https://github.com/nhalm/skimatik/wiki/contributing) for details.

After cloning, bootstrap dev tools and activate the Git hooks:

```bash
make setup
```

This installs lefthook and wires up `pre-commit` (`make lint` + `make test-unit` in parallel), `commit-msg` (Conventional Commits format on the subject line), and `pre-push` (`make test-integration`) per `lefthook.yml`.

## License

skimatik is licensed under the [MIT License](LICENSE).

## Support

- **[Documentation Wiki](https://github.com/nhalm/skimatik/wiki)** - Comprehensive guides and references
- **[GitHub Issues](https://github.com/nhalm/skimatik/issues)** - Bug reports and feature requests
- **[Discussions](https://github.com/nhalm/skimatik/discussions)** - Community help and questions

---

Built with ❤️ for the Go community. Making PostgreSQL development delightful, one repository at a time.