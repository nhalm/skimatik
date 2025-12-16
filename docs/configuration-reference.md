# Configuration Reference

## Overview

skimatik is configured through a YAML file (default: `skimatik.yaml`). This guide documents all available configuration options based on the actual implementation.

## Quick Start

**Minimal working configuration:**

```yaml
database:
  dsn: "postgres://user:pass@localhost:5432/mydb"
  schema: "public"

output:
  directory: "./repositories"
  package: "repositories"

default_functions: "all"

tables:
  users:
  posts:
```

## Complete Configuration Structure

```yaml
# Database connection (required)
database:
  dsn: "postgres://user:pass@localhost:5432/dbname"
  schema: "public"

# Output settings (required)
output:
  directory: "./repositories"
  package: "repositories"

# Default functions for all tables (optional)
default_functions: "all"  # or ["create", "get", "update", "delete", "list", "paginate"]

# Tables to generate (optional if using queries)
tables:
  users:  # Generate all default_functions
  posts:
    functions: ["create", "get", "list"]  # Override for specific table
  comments:

# Query-based generation (optional)
queries:
  directory: "./database/queries"

# Custom type mappings (optional)
types:
  mappings:
    my_enum: "MyEnumType"

# Enable verbose logging (optional)
verbose: true
```

---

## Configuration Sections

### Database Configuration

**Required fields:**

#### `database.dsn`
- **Type**: String
- **Required**: Yes
- **Description**: PostgreSQL connection string

**Format:**
```
postgres://[user[:password]@][host][:port][/dbname][?options]
```

**Examples:**
```yaml
database:
  dsn: "postgres://postgres:password@localhost:5432/mydb"
  dsn: "postgres://user@localhost/mydb?sslmode=require"
  dsn: "postgres://user:pass@db.example.com:5432/production"
```

#### `database.schema`
- **Type**: String
- **Required**: No
- **Default**: `"public"`
- **Description**: PostgreSQL schema to introspect and use for nullability checks

**Examples:**
```yaml
database:
  schema: "public"
  schema: "app"
  schema: "accounting"
```

---

### Output Configuration

#### `output.directory`
- **Type**: String
- **Required**: No
- **Default**: `"./repositories"`
- **Description**: Directory where generated code will be written

**Examples:**
```yaml
output:
  directory: "./repositories"
  directory: "./internal/db/generated"
  directory: "./pkg/models"
```

#### `output.package`
- **Type**: String
- **Required**: No
- **Default**: `"repositories"`
- **Description**: Go package name for generated code

**Examples:**
```yaml
output:
  package: "repositories"
  package: "generated"
  package: "models"
```

---

### Default Functions

#### `default_functions`
- **Type**: String or Array
- **Required**: No
- **Default**: All functions (`["create", "get", "update", "delete", "list", "paginate"]`)
- **Description**: Functions to generate for all tables unless overridden

**String format** (shorthand):
```yaml
default_functions: "all"
```

Expands to: `["create", "get", "update", "delete", "list", "paginate"]`

**Array format** (explicit):
```yaml
default_functions:
  - create
  - get
  - list
  - paginate
```

**Available functions:**
- `create` - Insert new records
- `get` - Fetch by primary key
- `update` - Update existing records
- `delete` - Delete records
- `list` - Fetch all records
- `paginate` - Cursor-based pagination

---

### Tables Configuration

#### `tables`
- **Type**: Map of table names to table configuration
- **Required**: No (but required if not using `queries`)
- **Description**: Tables to generate repositories for

**Structure:**
```yaml
tables:
  table_name:  # Use default_functions
  other_table:
    functions: [...]  # Override default_functions
```

**Examples:**

Generate all default functions for users and posts:
```yaml
default_functions: "all"
tables:
  users:
  posts:
```

Per-table function overrides:
```yaml
default_functions: "all"
tables:
  users:  # Gets all functions
  posts:
    functions: ["create", "list"]  # Only create and list
  comments:
    functions: []  # No functions (struct only)
```

No default functions, specify per table:
```yaml
tables:
  users:
    functions: ["create", "get", "update", "delete"]
  posts:
    functions: ["create", "get", "list"]
```

---

### Queries Configuration

#### `queries.directory`
- **Type**: String
- **Required**: No
- **Description**: Directory containing `.sql` query files

Query files use SQL comments for annotations:
```sql
-- name: GetUserByEmail :one
SELECT id, name, email FROM users WHERE email = $1;

-- name: ListActiveUsers :many
SELECT id, name, email FROM users WHERE is_active = true;

-- name: CreateUser :one
INSERT INTO users (id, name, email) VALUES ($1, $2, $3) RETURNING *;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false WHERE id = $1;
```

**Annotation syntax:**
- `:one` - Returns single record (error if not found)
- `:many` - Returns array of records
- `:paginated` - Returns array with pagination support (generates two functions)
- `:exec` - Executes without returning data

**Pagination with :paginated:**

The `:paginated` query type extracts ORDER BY direction from your SQL and generates paginated query functions:

```sql
-- name: GetPublishedPosts :paginated
SELECT id, title, content, published_at
FROM posts
WHERE is_published = true
ORDER BY published_at DESC
```

**Generated Functions:**

This generates TWO functions:

```go
// 1. Regular function - returns all results using your ORDER BY
func (r *PostsQueries) GetPublishedPosts(ctx context.Context) ([]GetPublishedPostsResult, error)

// 2. Paginated function - uses ORDER BY direction from your SQL
func (r *PostsQueries) GetPublishedPostsPaginated(
    ctx context.Context,
    params PaginationParams,     // Only NextCursor supported (forward-only)
) (*PaginationResult[GetPublishedPostsResult], error)
```

**How It Works:**
- Sort direction is extracted from ORDER BY at code generation time
- DESC ordering uses `<` comparison for cursor pagination
- ASC ordering uses `>` comparison for cursor pagination
- No runtime `orderBy` parameter - ordering is fixed by your SQL

**Requirements:**
- ORDER BY column must be in the SELECT clause
- Only simple column references supported (no expressions)
- Forward-only pagination (NextCursor only)

**Usage Example:**
```go
// First page
result, err := queries.GetPublishedPostsPaginated(
    ctx,
    repositories.PaginationParams{
        Limit: 20,
    },
)

// Next page (forward-only)
if result.HasMore {
    nextResult, err := queries.GetPublishedPostsPaginated(
        ctx,
        repositories.PaginationParams{
            Limit:      20,
            NextCursor: result.NextCursor,
        },
    )
}
```

**Validation Errors:**
- Generation-time: ORDER BY column not in SELECT list
- Generation-time: ORDER BY uses expression instead of column reference

**Example configuration:**
```yaml
queries:
  directory: "./database/queries"
```

**File structure:**
```
database/
  queries/
    users.sql
    posts.sql
    comments.sql
```

---

### Types Configuration

#### `types.mappings`
- **Type**: Map of PostgreSQL types to Go types
- **Required**: No
- **Description**: Custom type mappings for PostgreSQL types

**Example:**
```yaml
types:
  mappings:
    user_status: "UserStatus"
    payment_state: "PaymentState"
```

With these mappings:
- `user_status` column → `UserStatus` Go type
- `payment_state` column → `PaymentState` Go type

You must provide the Go type definitions in your code.

---

### Verbose

#### `verbose`
- **Type**: Boolean
- **Required**: No
- **Default**: `false`
- **Description**: Enable verbose logging output

**Example:**
```yaml
verbose: true
```

Output includes:
- Database connection details
- Tables discovered
- Queries parsed
- Files generated

---

## CLI Flags

```bash
skimatik [options]
```

**Available flags:**

- `--config` - Path to configuration file (default: `skimatik.yaml`)
- `--verbose` - Enable verbose logging (overrides config file)
- `--version` - Show version information
- `--help` - Show help message

**Examples:**
```bash
skimatik                                    # Use skimatik.yaml
skimatik --config custom.yaml               # Use custom config
skimatik --config skimatik.yaml --verbose   # Enable verbose output
skimatik --version                          # Show version
```

---

## Complete Examples

### Example 1: Simple Blog

```yaml
database:
  dsn: "postgres://postgres:password@localhost:5432/blog"
  schema: "public"

output:
  directory: "./repository/generated"
  package: "generated"

default_functions: "all"

tables:
  users:
  posts:
  comments:

verbose: true
```

### Example 2: Custom Functions Per Table

```yaml
database:
  dsn: "postgres://app:secret@localhost:5432/production"
  schema: "public"

output:
  directory: "./internal/db/repositories"
  package: "repositories"

tables:
  users:
    functions: ["create", "get", "update", "list", "paginate"]
  posts:
    functions: ["create", "get", "delete", "list"]
  audit_logs:
    functions: ["create", "list"]
  sessions:
    functions: ["create", "get", "delete"]
```

### Example 3: Query-Based Generation

```yaml
database:
  dsn: "postgres://readonly:pass@replica:5432/analytics"
  schema: "public"

output:
  directory: "./queries/generated"
  package: "queries"

queries:
  directory: "./sql/analytics"

verbose: true
```

### Example 4: Mixed Tables and Queries

```yaml
database:
  dsn: "postgres://app:pass@localhost:5432/myapp"
  schema: "public"

output:
  directory: "./db/generated"
  package: "db"

default_functions: "all"

tables:
  users:
  posts:

queries:
  directory: "./db/queries"

verbose: true
```

### Example 5: Custom Type Mappings

```yaml
database:
  dsn: "postgres://user:pass@localhost:5432/mydb"
  schema: "public"

output:
  directory: "./models"
  package: "models"

default_functions: "all"

tables:
  payments:
  invoices:

types:
  mappings:
    payment_status: "PaymentStatus"
    invoice_state: "InvoiceState"
```

---

## Validation

skimatik validates configuration on startup. Common errors:

### Configuration Validation

**Missing required fields:**
```
Error: database connection string (DSN) is required
```

**No generation targets:**
```
Error: must enable either table generation (--tables) or query generation (--queries)
```

**Invalid queries directory:**
```
Error: queries directory does not exist: ./database/queries
```

**Output directory creation failed:**
```
Error: failed to create output directory: permission denied
```

**default_functions validation:**
```
Error: invalid string value for default_functions: "some_value" (only 'all' is supported)
Error: default_functions array must contain only strings
Error: default_functions must be a string ('all') or array of strings
```

### Table Validation

**Primary key errors:**
```
Error: table has no primary key
Error: composite primary keys are not supported
Error: primary key column id not found
Error: primary key column id must be UUID type, got integer. skimatik requires UUID v7 primary keys for consistent time-ordered pagination. Please migrate your table to use UUID primary keys
```

### Query Validation

**Basic query validation:**
```
Error: query name cannot be empty
Error: query SQL cannot be empty
Error: query type cannot be empty
Error: query name 'Get-Users' is not a valid Go identifier
Error: query type :one requires SELECT statement or CTE
Error: query type :exec cannot use SELECT statement or CTE
```

**Parameter annotation validation:**
```
Error: duplicate parameter annotation for $1
Error: parameter annotations must be sequential starting at $1, missing $2
Error: invalid Go type "BadType" for parameter $1
Error: parameter count mismatch: query expects 2 parameters, found 3
```

**Result annotation validation:**
```
Error: duplicate result annotation for column "email"
Error: invalid Go type "BadType" for result column "status"
Error: query has result annotations but no columns were detected
Error: result annotation for column 'missing_column' not found in query results
```

**:paginated validation:**
```
Error: ORDER BY column "missing_col" not found in SELECT list for query GetPosts
```

---

## Related Documentation

- [Quick Start Guide](quick-start) - Get started with skimatik
- [Type Mapping](type-mapping) - PostgreSQL to Go type conversions
- [Database Migrations](database-migrations) - Managing schema changes
