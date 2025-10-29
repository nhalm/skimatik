# Skimatik: Nullable Parameter Support

## Problem Statement

Currently, skimatik generates query methods with non-nullable parameters only. This makes it difficult to write flexible List/filter queries where parameters are optional.

**Example Problem:**
```sql
-- name: ListPaymentLinks :many
SELECT * FROM payment_links
WHERE tenant_id = $1
  AND ($2::text IS NULL OR status = $2)
  AND ($3::timestamptz IS NULL OR created_at >= $3)
```

Currently generates (broken):
```go
func ListPaymentLinks(ctx context.Context, tenantId uuid.UUID, status string, createdAt time.Time)
```

Should generate:
```go
func ListPaymentLinks(ctx context.Context, tenantId uuid.UUID, status *string, createdAt *time.Time)
```

## Proposed Solution

Add optional parameter type annotations in SQL query comments using this syntax:

```
-- param: $N parameter_name go_type
```

Where:
- `$N` is the positional parameter number (required, explicit)
- `parameter_name` is a descriptive name (required, for documentation/readability)
- `go_type` is the Go type to generate (required, supports pointer types)

## Syntax Examples

### Basic Usage
```sql
-- name: ListPaymentLinks :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 status *string
-- param: $3 created_after *time.Time
-- param: $4 limit int
SELECT * FROM payment_links
WHERE tenant_id = $1
  AND ($2::text IS NULL OR status = $2)
  AND ($3::timestamptz IS NULL OR created_at >= $3)
LIMIT $4;
```

Generated:
```go
func (q *PaymentLinksQueries) ListPaymentLinks(
    ctx context.Context,
    tenantId uuid.UUID,
    status *string,
    createdAfter *time.Time,
    limit int,
) ([]ListPaymentLinksResult, error)
```

### Mixed Nullable/Non-Nullable
```sql
-- name: GetPaymentsByStatus :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 status string
-- param: $3 min_amount *int
SELECT * FROM payments
WHERE tenant_id = $1
  AND status = $2
  AND ($3::integer IS NULL OR amount >= $3);
```

Generated:
```go
func (q *PaymentsQueries) GetPaymentsByStatus(
    ctx context.Context,
    tenantId uuid.UUID,
    status string,
    minAmount *int,
) ([]GetPaymentsByStatusResult, error)
```

## Supported Go Types

The annotation supports any valid Go type:

**Scalar types:**
- `string`, `*string`
- `int`, `*int`
- `bool`, `*bool`
- `float64`, `*float64`

**Time types:**
- `time.Time`, `*time.Time`

**UUID types:**
- `uuid.UUID`, `*uuid.UUID`

**JSON types:**
- `json.RawMessage`, `*json.RawMessage`

**Custom types:**
- Any valid Go type identifier

## Implementation Requirements

### 1. Parsing Logic

When parsing query files (`.sql`), skimatik should:

1. Scan for `-- param:` comment lines before the query SQL
2. Parse each line into: `($N, parameter_name, go_type)`
3. Store parameter metadata keyed by position (`$N`)
4. Validate:
   - Parameter positions are sequential starting at `$1`
   - No duplicate positions
   - All parameters in SQL have annotations if any are annotated
   - Go type is valid (basic syntax check)

Example parse result:
```go
type ParameterAnnotation struct {
    Position int        // 1, 2, 3...
    Name     string     // "tenant_id", "status"
    GoType   string     // "uuid.UUID", "*string"
}
```

### 2. Code Generation Logic

When generating query methods:

**Without annotations (current behavior):**
- Infer types from PostgreSQL column types
- Generate non-nullable parameters

**With annotations:**
- Use annotated Go type directly
- Generate parameters in order: `$1`, `$2`, `$3`, etc.
- Use `parameter_name` in camelCase for Go parameter names
- Preserve pointer types (`*string` generates `*string`, not `string`)

**Parameter name conversion:**
- `tenant_id` → `tenantId`
- `created_after` → `createdAfter`
- `min_amount` → `minAmount`

### 3. Type Handling

For pointer types, the generated code must pass values correctly to pgx:

```go
func (q *Queries) ListPaymentLinks(
    ctx context.Context,
    tenantId uuid.UUID,
    status *string,
    createdAfter *time.Time,
    limit int,
) ([]ListPaymentLinksResult, error) {
    query := `SELECT ... WHERE tenant_id = $1 AND ($2::text IS NULL OR status = $2) ...`

    // pgx handles nil pointers correctly - they become SQL NULL
    rows, err := ExecuteQuery(ctx, q.db, "ListPaymentLinks", "ListPaymentLinksResult",
        query, tenantId, status, createdAfter, limit)
    // ... rest of implementation
}
```

**Key insight:** pgx's `Exec`/`Query` methods already handle `nil` pointer arguments correctly - they become SQL `NULL`. No special conversion needed.

### 4. Validation Rules

**At parse time:**
- If ANY parameter has an annotation, ALL parameters must be annotated
- Parameter positions must be sequential: `$1, $2, $3, ...` (no gaps)
- No duplicate parameter positions
- Go type syntax must be valid (basic check: valid identifier, optional `*` prefix)

**Example error messages:**
```
Error parsing query "ListPaymentLinks":
  - Parameter $2 is annotated but $1 is not. If using param annotations, all parameters must be annotated.

Error parsing query "ListPaymentLinks":
  - Duplicate parameter annotation for $2

Error parsing query "ListPaymentLinks":
  - Invalid Go type "*invalid type" for parameter $3
```

## SQL Pattern Requirements

For nullable parameters, the SQL MUST use the `IS NULL OR` pattern:

```sql
-- Correct
WHERE ($2::text IS NULL OR status = $2)

-- Also correct
WHERE $2::text IS NULL OR status = $2

-- Incorrect (will fail at runtime if status is nil)
WHERE status = $2
```

**Note:** Skimatik does NOT validate this - it's the developer's responsibility to write correct SQL. The annotation only affects Go type generation.

## Backward Compatibility

**Queries without annotations continue to work exactly as before:**
```sql
-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1 AND tenant_id = $2;
```

Still generates:
```go
func GetPaymentByID(ctx context.Context, id uuid.UUID, tenantId uuid.UUID) (*Payment, error)
```

**Annotations are opt-in** - only needed for queries with optional parameters.

## Examples

### Example 1: Simple List with Optional Status Filter
```sql
-- name: ListProducts :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 active *bool
-- param: $3 limit int
SELECT id, name, description, active, created_at
FROM products
WHERE tenant_id = $1
  AND deleted_at IS NULL
  AND ($2::boolean IS NULL OR active = $2)
ORDER BY created_at DESC
LIMIT $3;
```

Usage:
```go
// List all products
allProducts, err := queries.ListProducts(ctx, tenantID, nil, 25)

// List only active products
activeOnly := true
activeProducts, err := queries.ListProducts(ctx, tenantID, &activeOnly, 25)
```

### Example 2: Date Range Filtering
```sql
-- name: ListPaymentsByDateRange :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 start_date *time.Time
-- param: $3 end_date *time.Time
-- param: $4 limit int
SELECT id, amount, status, created_at
FROM payments
WHERE tenant_id = $1
  AND deleted_at IS NULL
  AND ($2::timestamptz IS NULL OR created_at >= $2)
  AND ($3::timestamptz IS NULL OR created_at <= $3)
ORDER BY created_at DESC
LIMIT $4;
```

Usage:
```go
// All payments
all, err := queries.ListPaymentsByDateRange(ctx, tenantID, nil, nil, 100)

// Payments after date
startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
after, err := queries.ListPaymentsByDateRange(ctx, tenantID, &startDate, nil, 100)

// Payments in range
endDate := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
inRange, err := queries.ListPaymentsByDateRange(ctx, tenantID, &startDate, &endDate, 100)
```

### Example 3: Complex Multi-Filter Query
```sql
-- name: SearchPaymentLinks :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 status *string
-- param: $3 min_amount *int
-- param: $4 max_amount *int
-- param: $5 created_after *time.Time
-- param: $6 created_before *time.Time
-- param: $7 starting_after *uuid.UUID
-- param: $8 limit int
SELECT id, tenant_id, status, created_at, updated_at
FROM payment_links
WHERE tenant_id = $1
  AND deleted_at IS NULL
  AND ($2::text IS NULL OR status = $2)
  AND ($3::integer IS NULL OR current_uses >= $3)
  AND ($4::integer IS NULL OR max_uses <= $4)
  AND ($5::timestamptz IS NULL OR created_at >= $5)
  AND ($6::timestamptz IS NULL OR created_at <= $6)
  AND ($7::uuid IS NULL OR id > $7)
ORDER BY id ASC
LIMIT $8;
```

## Implementation Checklist

- [ ] Add parameter annotation parsing to SQL parser
- [ ] Store parameter metadata with query definition
- [ ] Validate parameter annotations (sequential, no duplicates, all-or-nothing)
- [ ] Update code generation to use annotated types
- [ ] Update parameter naming (snake_case → camelCase)
- [ ] Add error messages for invalid annotations
- [ ] Test with nullable pointer types (`*string`, `*int`, `*time.Time`, `*uuid.UUID`)
- [ ] Test with non-nullable types (backward compatibility)
- [ ] Test with mixed nullable/non-nullable parameters
- [ ] Test queries without annotations (backward compatibility)
- [ ] Update documentation with examples
- [ ] Add integration tests for nullable parameter queries

## Testing Strategy

### Unit Tests
- Parse valid parameter annotations
- Parse invalid annotations (duplicate positions, gaps, invalid types)
- Enforce all-or-nothing rule (if one param annotated, all must be)
- Generate correct Go method signatures

### Integration Tests
Create test queries in `testdata/`:

```sql
-- testdata/nullable_params.sql

-- name: TestNullableString :many
-- param: $1 id uuid.UUID
-- param: $2 name *string
SELECT * FROM users WHERE tenant_id = $1 AND ($2::text IS NULL OR name = $2);

-- name: TestNullableTime :many
-- param: $1 id uuid.UUID
-- param: $2 created_after *time.Time
SELECT * FROM events WHERE user_id = $1 AND ($2::timestamptz IS NULL OR created_at >= $2);

-- name: TestMixedNullability :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 status string
-- param: $3 min_amount *int
SELECT * FROM payments
WHERE tenant_id = $1
  AND status = $2
  AND ($3::integer IS NULL OR amount >= $3);
```

Run skimatik generation and verify:
1. Generated signatures match expected types
2. Nil pointer parameters work at runtime (become SQL NULL)
3. Non-nil pointer parameters work correctly

## Documentation Updates

Add to skimatik README:

### Optional Parameters

For queries with optional filter parameters, use `-- param:` annotations:

```sql
-- name: ListUsers :many
-- param: $1 tenant_id uuid.UUID
-- param: $2 status *string
-- param: $3 limit int
SELECT * FROM users
WHERE tenant_id = $1
  AND ($2::text IS NULL OR status = $2)
LIMIT $3;
```

This generates:
```go
func (q *Queries) ListUsers(ctx context.Context, tenantId uuid.UUID, status *string, limit int)
```

Usage:
```go
// All users
allUsers, _ := queries.ListUsers(ctx, tenantID, nil, 100)

// Filter by status
activeStatus := "active"
activeUsers, _ := queries.ListUsers(ctx, tenantID, &activeStatus, 100)
```

**Rules:**
- If ANY parameter has a `-- param:` annotation, ALL parameters must be annotated
- Parameters must be sequential: `$1, $2, $3, ...`
- Use pointer types (`*string`, `*int`, etc.) for optional parameters
- Your SQL must use `IS NULL OR` pattern for optional parameters

## Future Enhancements (Optional)

These are NOT required for initial implementation:

1. **Auto-inference from `IS NULL` pattern:**
   - Detect `($N IS NULL OR ...)` pattern
   - Automatically make `$N` nullable without annotation
   - Falls back to annotations if ambiguous

2. **Named parameter syntax:**
   - Support `:param_name` syntax in addition to `$N`
   - Generate params struct instead of positional args

3. **Validation of SQL pattern:**
   - Warn if parameter is nullable but SQL doesn't use `IS NULL`
   - Warn if parameter is non-nullable but SQL uses `IS NULL`

## Summary

This feature adds minimal, explicit syntax for nullable parameters:
- **Syntax:** `-- param: $N name go_type`
- **Opt-in:** Only needed for queries with optional parameters
- **Type-safe:** Uses standard Go pointer types (`*string`, `*time.Time`)
- **Simple:** pgx already handles nil pointers as SQL NULL
- **Backward compatible:** Existing queries without annotations continue to work

This solves the common use case of flexible List/filter queries without requiring query builders or complex ORM patterns.
