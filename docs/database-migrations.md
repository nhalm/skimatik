# Database Migrations

Skimatik follows a database-first approach where code is generated from your PostgreSQL schema. This guide shows how to use database migrations with skimatik using [golang-migrate](https://github.com/golang-migrate/migrate), maintaining a clean workflow from schema changes to regenerated code.

## Overview

The recommended workflow combines database migrations with skimatik's code generation:

1. **Create migration** - Define schema changes in SQL migration files
2. **Apply migration** - Update the database schema
3. **Regenerate code** - Run skimatik to update Go code
4. **Update application** - Modify your application code as needed

This approach ensures your database schema and Go code stay perfectly synchronized.

## Setting Up Migrations

### 1. Install golang-migrate

Choose your installation method:

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/migrate

# Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 2. Create Migrations Directory

```bash
mkdir -p database/migrations
```

### 3. Add Migration Commands to Makefile

```makefile
DATABASE_URL = postgres://postgres:password@localhost:5432/myapp?sslmode=disable

migrate-up: ## Apply all pending migrations
	migrate -database "$(DATABASE_URL)" -path database/migrations up

migrate-down: ## Rollback one migration
	migrate -database "$(DATABASE_URL)" -path database/migrations down 1

migrate-status: ## Show migration status
	migrate -database "$(DATABASE_URL)" -path database/migrations version

migrate-create: ## Create a new migration
	@if [ -z "$(NAME)" ]; then \
		echo "Please provide a migration name: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir database/migrations -seq $(NAME)
```

## Migration Workflow

### Creating Your First Migration

1. **Create the initial schema migration:**

```bash
make migrate-create NAME=initial_schema
```

This creates two files:
- `database/migrations/000001_initial_schema.up.sql`
- `database/migrations/000001_initial_schema.down.sql`

2. **Write your schema with skimatik requirements:**

```sql
-- 000001_initial_schema.up.sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table with UUID v7 primary key
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create posts table
CREATE TABLE posts (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    author_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add indexes
CREATE INDEX idx_posts_author_id ON posts(author_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
```

3. **Write the rollback migration:**

```sql
-- 000001_initial_schema.down.sql
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";
```

4. **Apply the migration and regenerate code:**

```bash
make migrate-up          # Apply migration
make generate            # Regenerate skimatik code
```

## UUID v7 Primary Keys

Skimatik requires UUID primary keys for all tables. When creating tables in migrations:

### PostgreSQL with uuid-ossp extension:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    -- other columns...
);
```

### PostgreSQL 17+ with built-in UUIDv7:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- other columns...
);
```

### Important Notes:
- All tables must have a column named `id` of type `UUID`
- The UUID must be the primary key
- Always provide a DEFAULT value for auto-generation
- Composite primary keys are not supported

## Common Migration Patterns

### Adding a New Table

```bash
make migrate-create NAME=add_categories
```

```sql
-- up migration
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL UNIQUE,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add junction table for many-to-many
CREATE TABLE post_categories (
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, category_id)
);

-- down migration
DROP TABLE IF EXISTS post_categories;
DROP TABLE IF EXISTS categories;
```

### Adding Columns

```bash
make migrate-create NAME=add_post_status
```

```sql
-- up migration
ALTER TABLE posts 
ADD COLUMN status TEXT NOT NULL DEFAULT 'draft' 
    CHECK (status IN ('draft', 'published', 'archived'));

ALTER TABLE posts 
ADD COLUMN published_at TIMESTAMPTZ;

-- Add index for common queries
CREATE INDEX idx_posts_status ON posts(status) WHERE status = 'published';

-- down migration
DROP INDEX IF EXISTS idx_posts_status;
ALTER TABLE posts DROP COLUMN IF EXISTS published_at;
ALTER TABLE posts DROP COLUMN IF EXISTS status;
```

### Adding Indexes

```bash
make migrate-create NAME=add_performance_indexes
```

```sql
-- up migration
-- Composite index for common queries
CREATE INDEX idx_posts_status_published_at 
    ON posts(status, published_at DESC) 
    WHERE status = 'published';

-- Full-text search
CREATE INDEX idx_posts_search 
    ON posts USING gin(to_tsvector('english', title || ' ' || content));

-- down migration
DROP INDEX IF EXISTS idx_posts_search;
DROP INDEX IF EXISTS idx_posts_status_published_at;
```

### Modifying Columns Safely

```bash
make migrate-create NAME=expand_user_bio
```

```sql
-- up migration
-- Add new column first
ALTER TABLE users ADD COLUMN bio_new TEXT;

-- Copy data with transformation if needed
UPDATE users SET bio_new = bio;

-- Drop old and rename new
ALTER TABLE users DROP COLUMN bio;
ALTER TABLE users RENAME COLUMN bio_new TO bio;

-- down migration
ALTER TABLE users ADD COLUMN bio_old VARCHAR(500);
UPDATE users SET bio_old = LEFT(bio, 500);
ALTER TABLE users DROP COLUMN bio;
ALTER TABLE users RENAME COLUMN bio_old TO bio;
```

## Development Workflow

### 1. Standard Development Cycle

```bash
# 1. Create migration
make migrate-create NAME=add_user_roles

# 2. Edit migration files
vim database/migrations/*_add_user_roles.*.sql

# 3. Apply migration
make migrate-up

# 4. Regenerate skimatik code
make generate

# 5. Update application code to use new schema
vim service/user_service.go
```

### 2. Iterating on Schema Design

During development, you might need to refine your schema:

```bash
# Check current version
make migrate-status

# Rollback if needed
make migrate-down

# Edit migration files
vim database/migrations/*.sql

# Reapply
make migrate-up

# Regenerate
make generate
```

### 3. Handling Migration Errors

If a migration fails:

1. Check the error message for constraint violations or syntax errors
2. Fix the migration file
3. If partially applied, you may need to manually clean up
4. Rerun the migration

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Database Migrations
on:
  push:
    branches: [main]

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install golang-migrate
        run: |
          curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
          sudo mv migrate /usr/local/bin/
      
      - name: Run migrations
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
        run: |
          migrate -database "$DATABASE_URL" -path database/migrations up
      
      - name: Generate code
        run: make generate
      
      - name: Run tests
        run: make test
```

### Docker Compose Integration

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: password
      POSTGRES_DB: myapp
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  migrate:
    image: migrate/migrate
    depends_on:
      - postgres
    volumes:
      - ./database/migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://postgres:password@postgres:5432/myapp?sslmode=disable",
      "up"
    ]

volumes:
  postgres_data:
```

## Testing with Migrations

### Test Database Setup

```bash
# Create test database
createdb myapp_test

# Run migrations on test database
DATABASE_URL="postgres://postgres:password@localhost:5432/myapp_test?sslmode=disable" make migrate-up

# Run tests
go test ./...
```

### Integration Test Helper

```go
func setupTestDB(t *testing.T) *pgxkit.DB {
    // Connect to test database
    db := pgxkit.NewDB()
    err := db.Connect(context.Background(), testDatabaseURL)
    require.NoError(t, err)
    
    // Clean up after test
    t.Cleanup(func() {
        db.Shutdown(context.Background())
    })
    
    return db
}
```

## Best Practices

### 1. Migration Naming
- Use descriptive names: `add_user_roles`, not `update_users`
- Include table names when relevant
- Keep names concise but clear

### 2. Migration Content
- Keep migrations focused on one logical change
- Always write both up and down migrations
- Test rollbacks during development
- Include necessary indexes with new columns

### 3. Schema Evolution
- Add columns with defaults or as nullable first
- Populate data in a separate migration if needed
- Then add constraints in a final migration
- This prevents locking issues on large tables

### 4. Data Migrations
- Keep DDL (schema) and DML (data) migrations separate
- For data migrations, use transactions when possible
- Consider performance impact on large tables
- Test with production-like data volumes

### 5. Version Control
- Commit migration files immediately after creation
- Never modify committed migrations
- If a migration has issues, create a new one to fix it
- Review migrations carefully before merging

## Troubleshooting

### Common Issues

**Migration lock stuck:**
```bash
# Force unlock (use carefully)
migrate -database "$DATABASE_URL" -path database/migrations force VERSION
```

**Dirty database state:**
```bash
# Check current version
migrate -database "$DATABASE_URL" -path database/migrations version

# Force to specific version
migrate -database "$DATABASE_URL" -path database/migrations force 3
```

**Schema out of sync:**
```bash
# Regenerate from current schema
make generate

# Or rollback and reapply
make migrate-down
make migrate-up
make generate
```

### Migration Table

golang-migrate creates a `schema_migrations` table to track applied migrations. This table should be excluded from skimatik generation:

```yaml
# skimatik.yaml
exclude:
  - "schema_migrations"
```

## Example: Complete Feature Addition

Here's a complete example adding a comments feature:

```bash
# 1. Create the migration
make migrate-create NAME=add_comments

# 2. Write the migration
cat > database/migrations/*_add_comments.up.sql << 'EOF'
-- Create comments table
CREATE TABLE comments (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL CHECK (length(content) >= 1),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add indexes for common queries
CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_comments_author_id ON comments(author_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);

-- Add comment count to posts for performance
ALTER TABLE posts ADD COLUMN comment_count INTEGER NOT NULL DEFAULT 0;

-- Create trigger to maintain comment count
CREATE OR REPLACE FUNCTION update_post_comment_count() 
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE posts SET comment_count = comment_count + 1 
        WHERE id = NEW.post_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE posts SET comment_count = comment_count - 1 
        WHERE id = OLD.post_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER maintain_comment_count
AFTER INSERT OR DELETE ON comments
FOR EACH ROW EXECUTE FUNCTION update_post_comment_count();
EOF

# 3. Write the down migration
cat > database/migrations/*_add_comments.down.sql << 'EOF'
DROP TRIGGER IF EXISTS maintain_comment_count ON comments;
DROP FUNCTION IF EXISTS update_post_comment_count();
ALTER TABLE posts DROP COLUMN IF EXISTS comment_count;
DROP TABLE IF EXISTS comments;
EOF

# 4. Apply and regenerate
make migrate-up
make generate

# 5. Now you have generated code for comments!
```

## Next Steps

- Review the [Quick Start](quick-start.md) guide for initial setup
- See [Configuration Reference](configuration-reference.md) for skimatik options
- Check the [example application](../example-app) for a complete implementation