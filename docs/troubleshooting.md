# Troubleshooting Guide

Common issues and solutions when working with skimatik.

## Configuration Errors

### "database.dsn is required"

**Cause**: Missing or empty DSN in configuration or environment variables.

**Fix**: Add database connection string to your configuration file or set environment variable:

```yaml
# skimatik.yaml
database:
  dsn: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
```

Or set environment variable:
```bash
export DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"
```

### "YAML parse error" or "unknown field"

**Cause**: Invalid YAML syntax or incorrect field names that don't match the configuration schema.

**Fix**:
1. Verify YAML syntax (indentation, colons, quotes)
2. Check field names match the configuration reference
3. Common mistakes:
   - `database_url` ❌ (should be `database.dsn`)
   - `output_dir` ❌ (should be `output.directory`)
   - `pkg` ❌ (should be `output.package`)

See [Configuration Reference](configuration-reference) for correct field names.

## Schema Errors

### "Composite primary keys are not supported"

**Cause**: Table has a multi-column primary key.

**Fix**: Skimatik requires single-column primary keys. Composite keys are not supported. Restructure your table to use a single primary key column:

```sql
CREATE TABLE your_table (
    id BIGSERIAL PRIMARY KEY,
    -- other columns that were part of composite key become regular columns or unique constraints
);
```

### "Table 'X' not found in schema 'public'"

**Cause**: Table doesn't exist in the specified schema, or wrong schema configured.

**Fix**:
1. Verify table exists: `\dt` in psql
2. Check schema name in configuration
3. If using non-public schema:

```yaml
database:
  dsn: "postgres://..."
  schema: "your_schema_name"  # Default is "public"
```

### "Column type 'X' not supported"

**Cause**: PostgreSQL column type is not yet mapped to Go type.

**Fix**: Use custom type mapping in configuration:

```yaml
custom_types:
  your_custom_type: "YourGoType"
```

See [Type Mapping Reference](type-mapping) for supported types.

## Generation Errors

### "Failed to analyze query"

**Cause**: SQL syntax error in query file, or query references non-existent tables/columns.

**Fix**:
1. Validate SQL syntax by running query directly in psql
2. Check table/column names are correct
3. Verify query annotations are properly formatted:

```sql
-- name: GetUserByEmail :one
SELECT id, name, email FROM users WHERE email = $1;
```

### "No tables specified for generation"

**Cause**: Configuration doesn't specify which tables or queries to generate.

**Fix**: Add tables or queries section to configuration:

```yaml
# Generate from tables
tables:
  users:
  posts:

# Or generate from query files
queries:
  directory: "./database/queries"
```

### "Generated code has syntax errors"

**Cause**:
- Bug in code generation template
- Invalid characters in table/column names
- Type mapping issue

**Fix**:
1. Check for special characters in database names
2. Run `go fmt` on generated files
3. Check generator logs with `verbose: true`
4. Report issue with schema details

## Compilation Errors

### "undefined: uuid"

**Cause**: Missing import for uuid package.

**Fix**: Ensure generated files include:

```go
import "github.com/google/uuid"
```

Install the package if needed:
```bash
go get github.com/google/uuid
```

### "cannot use string as *string"

**Cause**: Type mismatch with nullable fields. Nullable database columns map to pointer types.

**Fix**: Use pointers for nullable values:

```go
// For nullable columns
var email *string
if userEmail != "" {
    email = &userEmail
}

params := CreateUsersParams{
    Name:  "John",
    Email: email,  // *string for nullable column
}
```

### "undefined: pgxkit"

**Cause**: Missing pgxkit dependency.

**Fix**: Install required dependency:

```bash
go get github.com/nhalm/pgxkit
```

## Runtime Errors

### "connection refused" or "could not connect to server"

**Cause**: PostgreSQL server not running or connection details incorrect.

**Fix**:
1. Verify PostgreSQL is running: `pg_isready`
2. Check connection string (host, port, credentials)
3. Verify network access and firewall rules
4. Test connection: `psql "postgres://..."`

### "pq: relation does not exist"

**Cause**: Generated code references table that doesn't exist (schema changed after generation).

**Fix**: Regenerate code after schema changes:

```bash
skimatik --config=skimatik.yaml
```

### "unique constraint violation"

**Cause**: Attempting to insert duplicate value for unique column.

**Fix**: Generated code automatically wraps these as `ErrAlreadyExists`:

```go
user, err := repo.Create(ctx, db, params)
if err != nil {
    if IsAlreadyExists(err) {
        // Handle duplicate
        return fmt.Errorf("user already exists")
    }
    return err
}
```

### "foreign key constraint violation"

**Cause**: Attempting to reference non-existent related record.

**Fix**: Generated code wraps these as `ErrInvalidReference`. Use the helper function:

```go
post, err := repo.Create(ctx, db, params)
if err != nil {
    if IsInvalidReference(err) {
        // Handle invalid foreign key
        return fmt.Errorf("invalid author_id")
    }
    return err
}
```

Access detailed error information using `DatabaseError`:

```go
var dbErr *generated.DatabaseError
if errors.As(err, &dbErr) && dbErr.Type == generated.ErrInvalidReference {
    // dbErr.Detail contains the foreign key constraint details
    log.Printf("Foreign key violation: %s", dbErr.Detail)
}
```

## Performance Issues

### Slow query generation

**Cause**: Large number of tables or complex queries.

**Fix**:
1. Generate only needed tables:
```yaml
tables:
  users:
  posts:
  # Don't specify tables you don't need
```

2. Use selective query generation
3. Exclude test/temporary tables

### Large generated files

**Cause**: Tables with many columns generate verbose code.

**Fix**:
1. This is expected and not a problem for compilation
2. Consider splitting large tables if appropriate
3. Use query-based generation for specific operations instead of full table CRUD

## Getting Help

If you encounter issues not covered here:

1. **Check verbose output**: Add `verbose: true` to configuration
2. **Review logs**: Look for specific error messages and context
3. **Verify setup**: Ensure PostgreSQL version compatibility (10+)
4. **Check examples**: Review [tutorial-blog-app](tutorial-blog-app) for working setup
5. **Report issues**: [GitHub Issues](https://github.com/nhalm/skimatik/issues) with:
   - Configuration file
   - Database schema (sanitized)
   - Error messages
   - Generated code snippet (if applicable)

## Related Documentation

- **[Configuration Reference](configuration-reference)** - Complete configuration options
- **[Quick Start Guide](quick-start)** - Installation and setup
- **[Type Mapping](type-mapping)** - PostgreSQL to Go type mappings
- **[Error Handling Guide](error-handling)** - Working with generated errors
