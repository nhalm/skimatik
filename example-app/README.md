[![GitHub Workflow Status (branch)](https://img.shields.io/github/actions/workflow/status/golang-migrate/migrate/ci.yaml?branch=master)](https://github.com/golang-migrate/migrate/actions/workflows/ci.yaml?query=branch%3Amaster)
[![GoDoc](https://pkg.go.dev/badge/github.com/golang-migrate/migrate)](https://pkg.go.dev/github.com/golang-migrate/migrate/v4)
[![Coverage Status](https://img.shields.io/coveralls/github/golang-migrate/migrate/master.svg)](https://coveralls.io/github/golang-migrate/migrate?branch=master)
[![packagecloud.io](https://img.shields.io/badge/deb-packagecloud.io-844fec.svg)](https://packagecloud.io/golang-migrate/migrate?filter=debs)
[![Docker Pulls](https://img.shields.io/docker/pulls/migrate/migrate.svg)](https://hub.docker.com/r/migrate/migrate/)
![Supported Go Versions](https://img.shields.io/badge/Go-1.20%2C%201.21-lightgrey.svg)
[![GitHub Release](https://img.shields.io/github/release/golang-migrate/migrate.svg)](https://github.com/golang-migrate/migrate/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/golang-migrate/migrate/v4)](https://goreportcard.com/report/github.com/golang-migrate/migrate/v4)

# migrate

__Database migrations written in Go. Use as [CLI](#cli-usage) or import as [library](#use-in-your-go-project).__

* Migrate reads migrations from [sources](#migration-sources)
   and applies them in correct order to a [database](#databases).
* Drivers are "dumb", migrate glues everything together and makes sure the logic is bulletproof.
   (Keeps the drivers lightweight, too.)
* Database drivers don't assume things or try to correct user input. When in doubt, fail.

Forked from [mattes/migrate](https://github.com/mattes/migrate)

## Databases

Database drivers run migrations. [Add a new database?](database/driver.go)

* [PostgreSQL](database/postgres)
* [PGX v4](database/pgx)
* [PGX v5](database/pgx/v5)
* [Redshift](database/redshift)
* [Ql](database/ql)
* [Cassandra / ScyllaDB](database/cassandra)
* [SQLite](database/sqlite)
* [SQLite3](database/sqlite3) ([todo #165](https://github.com/mattes/migrate/issues/165))
* [SQLCipher](database/sqlcipher)
* [MySQL / MariaDB](database/mysql)
* [Neo4j](database/neo4j)
* [MongoDB](database/mongodb)
* [CrateDB](database/crate) ([todo #170](https://github.com/mattes/migrate/issues/170))
* [Shell](database/shell) ([todo #171](https://github.com/mattes/migrate/issues/171))
* [Google Cloud Spanner](database/spanner)
* [CockroachDB](database/cockroachdb)
* [YugabyteDB](database/yugabytedb)
* [ClickHouse](database/clickhouse)
* [Firebird](database/firebird)
* [MS SQL Server](database/sqlserver)
* [RQLite](database/rqlite)

### Database URLs

Database connection strings are specified via URLs. The URL format is driver dependent but generally has the form: `dbdriver://username:password@host:port/dbname?param1=true&param2=false`

Any [reserved URL characters](https://en.wikipedia.org/wiki/Percent-encoding#Percent-encoding_reserved_characters) need to be escaped. Note, the `%` character also [needs to be escaped](https://en.wikipedia.org/wiki/Percent-encoding#Percent-encoding_the_percent_character)

Explicitly, the following characters need to be escaped:
`!`, `#`, `$`, `%`, `&`, `'`, `(`, `)`, `*`, `+`, `,`, `/`, `:`, `;`, `=`, `?`, `@`, `[`, `]`

It's easiest to always run the URL parts of your DB connection URL (e.g. username, password, etc) through an URL encoder. See the example Python snippets below:

```bash
$ python3 -c 'import urllib.parse; print(urllib.parse.quote(input("String to encode: "), ""))'
String to encode: FAKEpassword!#$%&'()*+,/:;=?@[]
FAKEpassword%21%23%24%25%26%27%28%29%2A%2B%2C%2F%3A%3B%3D%3F%40%5B%5D
$ python2 -c 'import urllib; print urllib.quote(raw_input("String to encode: "), "")'
String to encode: FAKEpassword!#$%&'()*+,/:;=?@[]
FAKEpassword%21%23%24%25%26%27%28%29%2A%2B%2C%2F%3A%3B%3D%3F%40%5B%5D
$
```

## Migration Sources

Source drivers read migrations from local or remote sources. [Add a new source?](source/driver.go)

* [Filesystem](source/file) - read from filesystem
* [io/fs](source/iofs) - read from a Go [io/fs](https://pkg.go.dev/io/fs#FS)
* [Go-Bindata](source/go_bindata) - read from embedded binary data ([jteeuwen/go-bindata](https://github.com/jteeuwen/go-bindata))
* [pkger](source/pkger) - read from embedded binary data ([markbates/pkger](https://github.com/markbates/pkger))
* [GitHub](source/github) - read from remote GitHub repositories
* [GitHub Enterprise](source/github_ee) - read from remote GitHub Enterprise repositories
* [Bitbucket](source/bitbucket) - read from remote Bitbucket repositories
* [Gitlab](source/gitlab) - read from remote Gitlab repositories
* [AWS S3](source/aws_s3) - read from Amazon Web Services S3
* [Google Cloud Storage](source/google_cloud_storage) - read from Google Cloud Platform Storage

## CLI usage

* Simple wrapper around this library.
* Handles ctrl+c (SIGINT) gracefully.
* No config search paths, no config files, no magic ENV var injections.

__[CLI Documentation](cmd/migrate)__

### Basic usage

```bash
$ migrate -source file://path/to/migrations -database postgres://localhost:5432/database up 2
```

### Docker usage

```bash
$ docker run -v {{ migration dir }}:/migrations --network host migrate/migrate
    -path=/migrations/ -database postgres://localhost:5432/database up 2
```

## Use in your Go project

* API is stable and frozen for this release (v3 & v4).
* Uses [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies.
* To help prevent database corruptions, it supports graceful stops via `GracefulStop chan bool`.
* Bring your own logger.
* Uses `io.Reader` streams internally for low memory overhead.
* Thread-safe and no goroutine leaks.

__[Go Documentation](https://pkg.go.dev/github.com/golang-migrate/migrate/v4)__

```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/github"
)

func main() {
    m, err := migrate.New(
        "github://mattes:personal-access-token@mattes/migrate_test",
        "postgres://localhost:5432/database?sslmode=enable")
    m.Steps(2)
}
```

Want to use an existing database client?

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    db, err := sql.Open("postgres", "postgres://localhost:5432/database?sslmode=enable")
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    m, err := migrate.NewWithDatabaseInstance(
        "file:///migrations",
        "postgres", driver)
    m.Up() // or m.Step(2) if you want to explicitly set the number of migrations to run
}
```

## Getting started

Go to [getting started](GETTING_STARTED.md)

## Tutorials

* [CockroachDB](database/cockroachdb/TUTORIAL.md)
* [PostgreSQL](database/postgres/TUTORIAL.md)

(more tutorials to come)

## Migration files

Each migration has an up and down migration. [Why?](FAQ.md#why-two-separate-files-up-and-down-for-a-migration)

```bash
1481574547_create_users_table.up.sql
1481574547_create_users_table.down.sql
```

[Best practices: How to write migrations.](MIGRATIONS.md)

## Coming from another db migration tool?

Check out [migradaptor](https://github.com/musinit/migradaptor/).
*Note: migradaptor is not affliated or supported by this project*

## Versions

Version | Supported? | Import | Notes
--------|------------|--------|------
**master** | :white_check_mark: | `import "github.com/golang-migrate/migrate/v4"` | New features and bug fixes arrive here first |
**v4** | :white_check_mark: | `import "github.com/golang-migrate/migrate/v4"` | Used for stable releases |
**v3** | :x: | `import "github.com/golang-migrate/migrate"` (with package manager) or `import "gopkg.in/golang-migrate/migrate.v3"` (not recommended) | **DO NOT USE** - No longer supported |

## Development and Contributing

Yes, please! [`Makefile`](Makefile) is your friend,
read the [development guide](CONTRIBUTING.md).

Also have a look at the [FAQ](FAQ.md).

# Example Blog Application

This is a complete example application demonstrating skimatik's recommended multi-layer architecture. It implements a simple blog with users, posts, and comments.

## 🏗️ Architecture

This example demonstrates a clean multi-layer architecture with proper separation of concerns.

## 🚀 Quick Start

### Quick Start

**The example includes pre-generated code for demonstration purposes.** You can run the app immediately:

```bash
make setup     # Start database and run migrations
make run       # Start the API server
```

**To see the full workflow in action:**
```bash
make setup     # Start database and run migrations
make generate  # Regenerate Go code with skimatik  
make run       # Start the API server
```

**Available commands:**
```bash
make help      # Show all commands
make clean     # Clean up everything
```

Or test manually:
```bash
# Get active users
curl "http://localhost:8080/api/users"

# Get published posts
curl "http://localhost:8080/api/posts"

# Search users
curl "http://localhost:8080/api/users/search?q=alice"

# Get user statistics
curl "http://localhost:8080/api/users/{user-id}/stats"
```

## 📝 Database Migrations

This example uses [golang-migrate](https://github.com/golang-migrate/migrate) for database schema management. This provides version control, rollback support, and proper state tracking for your database schema.

### Migration Commands

```bash
# Apply all pending migrations
make migrate-up

# Rollback one migration
make migrate-down

# Check current migration version
make migrate-status

# Create a new migration
make migrate-create NAME=add_new_column
```

### Migration Structure

Migrations are stored in `database/migrations/`:
- `000001_initial_schema.up.sql` - Creates the base schema
- `000001_initial_schema.down.sql` - Rollback for base schema
- `000002_sample_data.up.sql` - Adds sample data
- `000003_add_view_count.up.sql` - Example of schema evolution

### Schema Evolution Example

When you need to modify the schema:

1. **Create a new migration**:
   ```bash
   make migrate-create NAME=add_user_avatar
   ```

2. **Edit the generated files**:
   ```sql
   -- 000004_add_user_avatar.up.sql
   ALTER TABLE users ADD COLUMN avatar_url TEXT;
   
   -- 000004_add_user_avatar.down.sql
   ALTER TABLE users DROP COLUMN avatar_url;
   ```

3. **Apply the migration**:
   ```bash
   make migrate-up
   ```

4. **Regenerate code**:
   ```bash
   make generate
   ```

## 📚 Layer Details

### Generated Layer (`repository/generated/`)
- **Query-based functions** - Generated from your SQL files with annotations
- **Generated by skimatik** - Don't modify these files manually  
- **Custom queries** - Exactly the SQL you wrote, with Go type safety

### Query Layer (`database/queries/`)
- **SQL files** - Your handwritten SQL with sqlc-style annotations
- **Type annotations** - `:one`, `:many`, `:exec`, `:paginated`
- **Custom logic** - Complex joins, aggregations, business queries

### Service Layer (`service/`)
- **Business logic** - Validation, workflows, orchestration
- **Cross-cutting concerns** - Transaction management, error handling
- **Domain operations** - Complex operations spanning multiple repositories

### API Layer (`api/`)
- **HTTP concerns** - Request/response handling, routing
- **Authentication** - JWT validation, user context
- **Serialization** - JSON marshaling, validation

## 🎯 Key Patterns Demonstrated

### 1. Interface-Driven Design
Services depend on repository interfaces, not implementations:
```go
type PostService interface {
    CreatePost(ctx context.Context, req CreatePostRequest) (*PostResponse, error)
    GetPost(ctx context.Context, id uuid.UUID) (*PostResponse, error)
}

type postService struct {
    postRepo repository.PostRepository  // Interface, not struct
    userRepo repository.UserRepository
}
```

### 2. Query-Based Generation
Write SQL, get type-safe Go functions:
```sql
-- name: GetPublishedPosts :many
SELECT p.id, p.title, u.name as author_name
FROM posts p JOIN users u ON p.author_id = u.id  
WHERE p.is_published = true
ORDER BY p.published_at DESC LIMIT $1;
```

skimatik generates:
```go
func (q *PostsQueries) GetPublishedPosts(ctx context.Context, limit int32) ([]GetPublishedPostsRow, error)
```

### 3. Domain Model Separation
Business models can differ from database models:
```go
// Database model (generated)
type Posts struct {
    ID       uuid.UUID
    Title    string
    Content  string
    AuthorID uuid.UUID
}

// Domain model (business layer)
type Post struct {
    ID          uuid.UUID
    Title       string
    Content     string
    AuthorName  string    // Enriched with author info
    CommentCount int      // Calculated field
}
```

### 4. Error Handling
Consistent error handling across all layers:
```go
func (s *postService) CreatePost(ctx context.Context, req CreatePostRequest) (*PostResponse, error) {
    if err := s.validatePostRequest(req); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    post, err := s.postRepo.CreatePost(ctx, req.Title, req.Content, req.AuthorID)
    if err != nil {
        return nil, fmt.Errorf("failed to create post: %w", err)
    }
    
    return s.toPostResponse(post), nil
}
```

## 🧪 Testing

The architecture makes testing easy:

### Unit Tests (Service Layer)
```go
func TestPostService_CreatePost(t *testing.T) {
    mockPostRepo := &mocks.PostRepository{}
    mockUserRepo := &mocks.UserRepository{}
    
    service := NewPostService(mockPostRepo, mockUserRepo)
    
    // Test business logic without database
}
```

### Integration Tests (Repository Layer)
```go
func TestPostRepository_Integration(t *testing.T) {
    testDB := setupTestDatabase(t)
    repo := NewPostRepository(testDB)
    
    // Test against real database
}
```

## 🔄 Regeneration Workflow

When your database schema changes:

1. **Create migration**: Use `make migrate-create NAME=your_change`
2. **Apply migration**: Run `make migrate-up` to update database
3. **Regenerate code**: Run `make generate` to update `repository/generated/`
4. **Update repositories**: Modify domain repositories if needed
5. **Update services**: Adjust business logic for schema changes
6. **Update API**: Modify handlers for new endpoints

Your custom code in `repository/`, `service/`, and `api/` remains intact!

**Note**: The generated files in `repository/generated/` are committed to this example for demonstration purposes. In your own projects, you may choose to add this directory to `.gitignore` and generate fresh on each developer's machine.

## 📖 Related Documentation

- **[Multi-Layer Architecture Guide](../docs/embedding-patterns.md)** - Detailed architecture patterns
- **[Quick Start Guide](../docs/quick-start.md)** - Basic skimatik usage
- **[Configuration Reference](../docs/configuration-reference.md)** - All configuration options
