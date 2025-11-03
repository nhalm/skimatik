# Skimatik: Intelligent Result Type Generation (No SQL Parser Required)

## Problem

Currently, skimatik generates query result structs with ALL `pgtype.*` types, regardless of whether columns are NOT NULL in the schema. This creates a mismatch with table structs and requires manual type conversion.

**Current behavior:**
```sql
SELECT id, status FROM payment_links;
```

Generates:
```go
type Result struct {
    Id     pgtype.UUID  // Should be uuid.UUID (NOT NULL)
    Status pgtype.Text  // Should be string (NOT NULL)
}
```

## Solution: Use PostgreSQL + pgx FieldDescriptions (No Parser!)

PostgreSQL and pgx together tell us everything we need - **no SQL parser required**.

### The Approach

When generating code for a query:

1. **Execute the query with LIMIT 0** (returns no rows, just column metadata)
2. **Get FieldDescriptions from pgx** - gives us table OID and column number for each result column
3. **Look up nullability in pg_catalog** - match OID + column number to schema
4. **Generate appropriate Go types:**
   - **Table columns with NOT NULL** → Native Go types (`uuid.UUID`, `string`, `int`, `time.Time`)
   - **Table columns that are nullable** → Go pointers (`*uuid.UUID`, `*string`, `*int`, `*time.Time`)
   - **Computed columns (COUNT aggregates)** → Native Go types (`int64`) - COUNT never returns NULL
   - **Computed columns (everything else)** → Go pointers (`*int64`, `*string`, etc) - assume nullable

### How It Works

```go
// 1. Execute query (no data returned)
rows, err := conn.Query(ctx, querySQL + " LIMIT 0")
fieldDescriptions := rows.FieldDescriptions()
rows.Close()

// 2. For each column in the result
for _, fd := range fieldDescriptions {
    // pgx gives us:
    // - fd.Name: column name in result
    // - fd.TableOID: which table (0 if computed/expression)
    // - fd.TableAttributeNumber: which column in table (0 if computed)
    // - fd.DataTypeOID: PostgreSQL type OID

    // Get PostgreSQL type name (uuid, int4, text, etc)
    pgType := lookupTypeName(fd.DataTypeOID)

    if fd.TableOID != 0 && fd.TableAttributeNumber != 0 {
        // This is a table column - look up nullability from schema
        var notNull bool
        conn.QueryRow(ctx, `
            SELECT a.attnotnull
            FROM pg_attribute a
            WHERE a.attrelid = $1 AND a.attnum = $2
        `, fd.TableOID, fd.TableAttributeNumber).Scan(&notNull)

        if notNull {
            // NOT NULL column → native Go type
            goType = nativeGoType(pgType)  // uuid.UUID, string, int, time.Time
        } else {
            // Nullable column → pointer
            goType = pointerType(pgType)   // *uuid.UUID, *string, *int, *time.Time
        }
    } else {
        // Computed/expression column
        if isCountAggregate(querySQL, fd.Name) {
            // COUNT(*) or COUNT(col) → always NOT NULL
            goType = nativeGoType(pgType)  // int64 (COUNT returns bigint)
        } else {
            // All other computed columns → nullable pointer
            // (SUM, AVG, MAX, MIN, expressions, etc can return NULL)
            goType = pointerType(pgType)   // *int64, *string, *time.Time, etc
        }
    }
}

// Helper: Detect COUNT aggregates in SELECT clause
func isCountAggregate(querySQL, columnName string) bool {
    // Simple regex to detect COUNT in SELECT for this column
    selectClause := extractSelectClause(querySQL)
    pattern := fmt.Sprintf(`(?i)COUNT\s*\([^)]*\)\s*(?:AS\s+)?%s\b`,
                          regexp.QuoteMeta(columnName))
    matched, _ := regexp.MatchString(pattern, selectClause)
    return matched
}
```

### Type Mapping

```
PostgreSQL Type → Go (NOT NULL)     | Go (Nullable)
----------------|---------------------|----------------------
uuid            | uuid.UUID           | *uuid.UUID
varchar/text    | string              | *string
int2/int4/int8  | int                 | *int
timestamptz     | time.Time           | *time.Time
boolean         | bool                | *bool
jsonb           | json.RawMessage     | *json.RawMessage
```

**Important:** All PostgreSQL integer types (`int2`, `int4`, `int8`) map to Go's `int`, not sized integers. This follows the project guideline to always use `int` instead of `int32` or `int64`.

**Note:** Generated query execution code handles conversion from `pgtype.*` to pointers internally. The struct uses native Go types and pointers, but scanning still uses `pgtype.*` for nullable columns.

### Why Pointers Instead of pgtype in Structs?

Users interact with idiomatic Go:
```go
result, _ := queries.GetStats(ctx)

if result.Total != nil {
    fmt.Printf("Total: %d\n", *result.Total)  // Simple pointer check
}
```

Instead of:
```go
if result.Total.Valid {
    fmt.Printf("Total: %d\n", result.Total.Int64)  // pgtype API
}
```

Generated code handles the conversion:
```go
var totalPg pgtype.Int8  // Scan into pgtype for nullable column
err := row.Scan(&result.Id, &totalPg)

if totalPg.Valid {
    val := int(totalPg.Int64)  // Convert to int
    result.Total = &val
}
```

## Examples

### Example 1: Simple Query

**Query:**
```sql
-- name: GetPaymentLink :one
SELECT id, tenant_id, status, description, created_at
FROM payment_links
WHERE id = $1;
```

**FieldDescriptions tell us:**
```
Column 0: id
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 1
  → Lookup: payment_links.id, NOT NULL = true
  → Generate: uuid.UUID

Column 1: tenant_id
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 2
  → Lookup: payment_links.tenant_id, NOT NULL = true
  → Generate: uuid.UUID

Column 2: status
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 4
  → Lookup: payment_links.status, NOT NULL = true
  → Generate: string

Column 3: description
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 3
  → Lookup: payment_links.description, NOT NULL = false
  → Generate: pgtype.Text

Column 4: created_at
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 9
  → Lookup: payment_links.created_at, NOT NULL = true
  → Generate: time.Time
```

**Generated:**
```go
type GetPaymentLinkResult struct {
    Id          uuid.UUID   // NOT NULL → native type
    TenantId    uuid.UUID   // NOT NULL → native type
    Status      string      // NOT NULL → native type
    Description *string     // Nullable → pointer
    CreatedAt   time.Time   // NOT NULL → native type
}
```

### Example 2: JOIN Query

**Query:**
```sql
-- name: GetPaymentWithLink :one
SELECT
    pl.id as link_id,
    pl.status as link_status,
    p.amount,
    p.created_at as payment_created
FROM payment_links pl
LEFT JOIN payments p ON p.payment_link_id = pl.id
WHERE pl.id = $1;
```

**FieldDescriptions tell us:**
```
Column 0: link_id
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 1
  → payment_links.id, NOT NULL = true → uuid.UUID

Column 1: link_status
  TableOID: 16865 (payment_links)
  TableAttributeNumber: 4
  → payment_links.status, NOT NULL = true → string

Column 2: amount
  TableOID: 16902 (payments)
  TableAttributeNumber: 5
  → payments.amount, NOT NULL = true → int

Column 3: payment_created
  TableOID: 16902 (payments)
  TableAttributeNumber: 9
  → payments.created_at, NOT NULL = true → time.Time
```

**Generated:**
```go
type GetPaymentWithLinkResult struct {
    LinkId          uuid.UUID  // payment_links.id NOT NULL → native type
    LinkStatus      string     // payment_links.status NOT NULL → native type
    Amount          int        // payments.amount NOT NULL → native type
    PaymentCreated  time.Time  // payments.created_at NOT NULL → native type
}
```

### Example 3: Query with Computed Columns

**Query:**
```sql
-- name: GetLinkStats :one
SELECT
    pl.id,
    pl.status,
    COUNT(*) as payment_count,
    SUM(p.amount) as total_amount,
    COALESCE(MAX(p.created_at), pl.created_at) as latest_activity
FROM payment_links pl
LEFT JOIN payments p ON p.payment_link_id = pl.id
WHERE pl.id = $1
GROUP BY pl.id, pl.status, pl.created_at;
```

**FieldDescriptions tell us:**
```
Column 0: id
  TableOID: 16865, TableAttributeNumber: 1
  → payment_links.id, NOT NULL = true → uuid.UUID

Column 1: status
  TableOID: 16865, TableAttributeNumber: 4
  → payment_links.status, NOT NULL = true → string

Column 2: payment_count
  TableOID: 0, TableAttributeNumber: 0
  → COMPUTED (COUNT) → pgtype.Int8

Column 3: total_amount
  TableOID: 0, TableAttributeNumber: 0
  → COMPUTED (SUM) → pgtype.Int8

Column 4: latest_activity
  TableOID: 0, TableAttributeNumber: 0
  → COMPUTED (COALESCE) → pgtype.Timestamptz
```

**Generated:**
```go
type GetLinkStatsResult struct {
    Id             uuid.UUID   // Table column, NOT NULL → native type
    Status         string      // Table column, NOT NULL → native type
    PaymentCount   int         // COUNT aggregate → native type (never NULL)
    TotalAmount    *int        // SUM aggregate → pointer (can be NULL)
    LatestActivity *time.Time  // COALESCE computed → pointer (can be NULL)
}
```

**Note:** `PaymentCount` uses `int` (not `*int`) because `COUNT(*)` is guaranteed to return a value (0 if no rows). PostgreSQL returns `bigint` for COUNT, but we map it to Go's `int`.

### Example 4: CTE (Common Table Expression)

**Query:**
```sql
-- name: GetTenantSummary :one
WITH link_counts AS (
    SELECT
        tenant_id,
        COUNT(*) as total_links
    FROM payment_links
    GROUP BY tenant_id
)
SELECT
    lc.tenant_id,
    lc.total_links,
    pl.id as sample_link_id,
    pl.status as sample_status
FROM link_counts lc
LEFT JOIN payment_links pl ON pl.tenant_id = lc.tenant_id
WHERE lc.tenant_id = $1
LIMIT 1;
```

**FieldDescriptions tell us:**
```
Column 0: tenant_id
  TableOID: 16865, TableAttributeNumber: 2
  → payment_links.tenant_id, NOT NULL = true → uuid.UUID

Column 1: total_links
  TableOID: 0, TableAttributeNumber: 0
  → COMPUTED (from CTE) → pgtype.Int8

Column 2: sample_link_id
  TableOID: 16865, TableAttributeNumber: 1
  → payment_links.id, NOT NULL = true → uuid.UUID

Column 3: sample_status
  TableOID: 16865, TableAttributeNumber: 4
  → payment_links.status, NOT NULL = true → string
```

**Generated:**
```go
type GetTenantSummaryResult struct {
    TenantId      uuid.UUID   // Table column, NOT NULL → native type
    TotalLinks    int         // COUNT in CTE → native type (never NULL)
    SampleLinkId  uuid.UUID   // Table column, NOT NULL → native type
    SampleStatus  string      // Table column, NOT NULL → native type
}
```

**Note:** Even though `TotalLinks` comes from a CTE, skimatik detects it's a `COUNT(*)` and generates `int` instead of `*int`.

## Implementation in Skimatik

### Step 1: Execute Query During Code Generation

```go
func generateQueryCode(db *pgx.Conn, query Query) error {
    // Execute query with LIMIT 0 to get metadata only
    queryWithLimit := query.SQL + " LIMIT 0"
    rows, err := db.Query(context.Background(), queryWithLimit)
    if err != nil {
        return fmt.Errorf("failed to analyze query: %w", err)
    }

    // Get field descriptions
    fieldDescriptions := rows.FieldDescriptions()
    rows.Close()

    // Analyze each column...
    for _, fd := range fieldDescriptions {
        col := analyzeColumn(db, fd)
        // ...
    }
}
```

### Step 2: Analyze Each Column

```go
type ColumnInfo struct {
    Name       string
    GoType     string
    IsNullable bool
    SourceInfo string // For debugging: "payment_links.id" or "COMPUTED"
}

func analyzeColumn(db *pgx.Conn, fd pgconn.FieldDescription) ColumnInfo {
    // Get PostgreSQL type name
    var pgTypeName string
    db.QueryRow(context.Background(),
        "SELECT typname FROM pg_type WHERE oid = $1",
        fd.DataTypeOID).Scan(&pgTypeName)

    if fd.TableOID != 0 && fd.TableAttributeNumber != 0 {
        // Table column - get full info
        var tableName, columnName string
        var notNull bool

        db.QueryRow(context.Background(), `
            SELECT c.relname, a.attname, a.attnotnull
            FROM pg_class c
            JOIN pg_attribute a ON a.attrelid = c.oid
            WHERE c.oid = $1 AND a.attnum = $2
        `, fd.TableOID, fd.TableAttributeNumber).Scan(&tableName, &columnName, &notNull)

        return ColumnInfo{
            Name:       fd.Name,
            GoType:     chooseGoType(pgTypeName, notNull),
            IsNullable: !notNull,
            SourceInfo: fmt.Sprintf("%s.%s", tableName, columnName),
        }
    } else {
        // Computed column
        return ColumnInfo{
            Name:       fd.Name,
            GoType:     pgTypeToGoType(pgTypeName, true), // Always nullable for computed
            IsNullable: true,
            SourceInfo: "COMPUTED",
        }
    }
}

func chooseGoType(pgType string, notNull bool) string {
    if notNull {
        return nativeGoType(pgType)  // uuid.UUID, string, int, time.Time
    }
    return pgtypeType(pgType)  // pgtype.UUID, pgtype.Text, etc
}
```

### Step 3: Generate Struct

```go
func generateResultStruct(w io.Writer, queryName string, columns []ColumnInfo) {
    fmt.Fprintf(w, "type %sResult struct {\n", queryName)

    for _, col := range columns {
        fieldName := toCamelCase(col.Name)
        fmt.Fprintf(w, "\t%s %s `json:\"%s\" db:\"%s\"`",
            fieldName, col.GoType, col.Name, col.Name)

        // Optional: add comment showing source
        fmt.Fprintf(w, " // %s\n", col.SourceInfo)
    }

    fmt.Fprintf(w, "}\n")
}
```

## Benefits

1. **No SQL parser needed** - PostgreSQL + pgx do all the work
2. **Handles all query types** - Simple, JOINs, CTEs, subqueries, window functions
3. **Correct types automatically** - Native Go types for NOT NULL, pgtype for nullable/computed
4. **Zero manual annotations** - Works out of the box for all queries
5. **Type safety** - Compiler catches mismatches between query and usage

## Edge Cases

### Ambiguous Columns (Same Name from Different Tables)

PostgreSQL handles this - FieldDescriptions give us the correct source table:

```sql
SELECT pl.status, p.status FROM payment_links pl JOIN payments p ON ...
```

Result columns get distinct OIDs, skimatik looks up the correct table for each.

### Columns from Subqueries/CTEs

If the column comes from a subquery/CTE, `TableOID = 0` (computed), so we use pgtype for safety.

### Functions that Preserve NOT NULL

Functions like `COALESCE(col, 'default')` always return non-null, but pgx reports `TableOID = 0`, so we conservatively use pgtype. This is safe but not optimal.

**Future enhancement**: Detect specific functions and infer nullability.

## Migration Path

### Phase 1: Implement for New Queries

Generate smart types for new queries, keep existing generated code unchanged.

### Phase 2: Opt-in Regeneration

Add flag `--smart-types` to regenerate with new logic.

### Phase 3: Default Behavior

Make smart types the default, allow opt-out with `--conservative-types`.

## PostgreSQL Guarantees for COUNT

Based on PostgreSQL documentation, `COUNT` has special nullability behavior:

> "Except for count, these functions return a null value when no rows are selected. In particular, sum of no rows returns null, not zero as one might expect."

This means:
- **`COUNT(*)`** → Always returns `0` or greater, never NULL
- **`COUNT(column)`** → Always returns `0` or greater, never NULL
- **`SUM(column)`** → Returns NULL on empty set
- **`AVG(column)`** → Returns NULL on empty set
- **`MAX(column)`** → Returns NULL on empty set
- **`MIN(column)`** → Returns NULL on empty set

Skimatik uses simple pattern matching to detect COUNT aggregates in the SELECT clause and generates non-nullable `int` for those columns.

## Summary

By leveraging PostgreSQL's query analysis and pgx's FieldDescriptions, skimatik can generate optimal Go types for query results without building a SQL parser:

1. **Table columns** → Look up NOT NULL from schema → use native types or pointers
2. **COUNT aggregates** → Detect with pattern matching → use native `int` (never NULL)
3. **Other computed columns** → Use nullable pointers (safe default)

This makes generated code idiomatic Go while maintaining type safety.
