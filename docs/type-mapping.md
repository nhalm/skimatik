# Type Mapping Reference

Skimatik uses intelligent, idiomatic Go type mappings that prioritize ergonomics and type safety.

## Design Philosophy

- **All integers map to `int`** - Whether your column is `SMALLINT`, `INTEGER`, or `BIGINT`, it maps to Go's native `int`
- **NOT NULL columns use native Go types** - Direct types like `int`, `string`, `uuid.UUID`, `time.Time`
- **Nullable columns use pointers** - NULL support via `*int`, `*string`, `*uuid.UUID`, `*time.Time`
- **No dependencies on pgtype** - Pure Go types only

## Type Mapping Table

| PostgreSQL Type | NOT NULL Go Type | NULLABLE Go Type |
|----------------|------------------|------------------|
| `SMALLINT`, `INT2` | `int` | `*int` |
| `INTEGER`, `INT`, `INT4` | `int` | `*int` |
| `BIGINT`, `INT8` | `int` | `*int` |
| `TEXT`, `VARCHAR` | `string` | `*string` |
| `BOOLEAN`, `BOOL` | `bool` | `*bool` |
| `UUID` | `uuid.UUID` | `*uuid.UUID` |
| `TIMESTAMP`, `TIMESTAMPTZ` | `time.Time` | `*time.Time` |
| `DATE` | `time.Time` | `*time.Time` |
| `JSON`, `JSONB` | `json.RawMessage` | `*json.RawMessage` |
| `BYTEA` | `[]byte` | `*[]byte` |
| `REAL`, `FLOAT4` | `float32` | `*float32` |
| `DOUBLE PRECISION`, `FLOAT8` | `float64` | `*float64` |
| `NUMERIC`, `DECIMAL` | `float64` | `*float64` |

## Array Types

Array columns map to Go slices:

| PostgreSQL Type | NOT NULL Go Type | NULLABLE Go Type |
|----------------|------------------|------------------|
| `INTEGER[]` | `[]int` | `[]*int` |
| `TEXT[]` | `[]string` | `[]*string` |
| `UUID[]` | `[]uuid.UUID` | `[]*uuid.UUID` |

## Example: Table Structure

Given this PostgreSQL table:

```sql
CREATE TABLE users (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT,  -- nullable
    age        INTEGER NOT NULL,
    score      BIGINT,  -- nullable
    created_at TIMESTAMPTZ NOT NULL
);
```

Skimatik generates:

```go
type Users struct {
    Id        uuid.UUID   `json:"id" db:"id"`
    Name      string      `json:"name" db:"name"`
    Email     *string     `json:"email" db:"email"`        // pointer for nullable
    Age       int         `json:"age" db:"age"`            // int for INTEGER
    Score     *int        `json:"score" db:"score"`        // pointer for nullable BIGINT
    CreatedAt time.Time   `json:"created_at" db:"created_at"`
}
```

## Example: Query Results

Custom queries use the same intelligent type mappings:

```sql
-- queries/users.sql
-- name: GetUserStats :many
SELECT
    COUNT(*) as user_count,
    AVG(age) as avg_age,
    MAX(score) as max_score
FROM users;
```

Generates:

```go
type GetUserStatsResult struct {
    UserCount int       `json:"user_count" db:"user_count"`  // COUNT never NULL
    AvgAge    *float64  `json:"avg_age" db:"avg_age"`        // AVG can be NULL
    MaxScore  *int      `json:"max_score" db:"max_score"`    // MAX can be NULL
}
```

## Result Type Annotations

You can override auto-detected types using `-- result:` annotations:

```sql
-- name: GetUserCount :one
-- result: total int
SELECT COUNT(*) as total FROM users;
```

This forces `total` to be `int` instead of the auto-detected type.

## Why `int` Instead of `int32`/`int64`?

Skimatik uses Go's native `int` type for all PostgreSQL integers because:

1. **More idiomatic** - Go developers expect `int` for counts, IDs, quantities
2. **Ergonomic** - No casting needed when using values (`for i := 0; i < count; i++`)
3. **Sufficient range** - `int` is 64-bit on all modern platforms
4. **Consistent** - Same type regardless of column's exact PostgreSQL integer type

If you need precise integer sizes, use custom type mappings in your config.

## Custom Type Mappings

Override default mappings in `skimatik.yaml`:

```yaml
type_mappings:
  my_enum_type: "MyEnumType"
  custom_type: "domain.CustomType"
```

## Required Imports

Skimatik automatically generates the necessary imports:

```go
import (
    "encoding/json"           // for json.RawMessage
    "time"                    // for time.Time
    "github.com/google/uuid"  // for uuid.UUID
)
```

No pgtype dependencies needed!

## Working with Nullable Fields

Nullable fields use pointers, so check for `nil`:

```go
user, err := repo.GetByID(ctx, id)
if err != nil {
    return err
}

// Check nullable email
if user.Email != nil {
    sendEmail(*user.Email)
}

// Set nullable field
newEmail := "user@example.com"
user.Email = &newEmail
```

## See Also

- [Configuration Reference](configuration-reference) - Custom type mappings
- [Examples](examples) - Real usage examples
- [Quick Start](quick-start) - Getting started guide
