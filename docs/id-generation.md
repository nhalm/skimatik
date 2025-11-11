# ID Generation Guide

## Overview

Skimatik generates repository code that performs ID generation on the application side before inserting records into the database. This approach provides better control, testability, and flexibility compared to database-generated IDs.

## Why Application-Side ID Generation?

### Benefits

**1. UUID v7 by Default**
- Time-ordered IDs provide better database performance
- Natural chronological sorting
- Improved B-tree index performance compared to random UUIDs
- Compatible with all PostgreSQL versions

**2. Testability**
- Use deterministic ID generators in tests
- Predictable IDs simplify test assertions
- Easy to mock and control ID generation

**3. Flexibility**
- Choose different ID strategies per repository
- Support for custom ID formats (ULID, KSUID, Snowflake, etc.)
- No dependency on database extensions

**4. Cross-Database Compatibility**
- No need for `uuid-ossp` extension
- Works consistently across PostgreSQL versions
- Easier to migrate between databases

## Default Behavior

### Using the Default UUID v7 Generator

Generated repositories automatically use a UUID v7 generator by default:

```go
import (
    "github.com/nhalm/pgxkit"
    "your-project/repositories"
)

func main() {
    db := pgxkit.NewDB()
    defer db.Close()

    // UUID v7 generator is automatically set
    userRepo := repositories.NewUsersRepository(db)

    // IDs are automatically generated using UUID v7
    user, err := userRepo.Create(ctx, repositories.CreateUsersParams{
        Name:  "Jane Doe",
        Email: "jane@example.com",
    })
    // user.Id contains a time-ordered UUID v7
}
```

### How UUID v7 Works

UUID v7 embeds a timestamp in the first 48 bits, providing natural time-ordering:

```
xxxxxxxx-xxxx-7xxx-xxxx-xxxxxxxxxxxx
└─timestamp─┘ ver └───random────────┘
```

This means records inserted later have lexicographically larger IDs, which benefits:
- Database index performance
- Natural sorting in queries
- Debugging and troubleshooting

## Custom ID Generators

### UUID v4 (Random)

If you need random UUIDs instead of time-ordered, override the `GenerateIdFunc` property:

```go
import "github.com/google/uuid"

userRepo := repositories.NewUsersRepository(db)

// Override with UUID v4 generator
userRepo.GenerateIdFunc = func() uuid.UUID {
    return uuid.New() // UUID v4
}
```

### Deterministic IDs for Testing

Create predictable IDs for unit tests by setting the `GenerateIdFunc` property:

```go
func TestUserCreation(t *testing.T) {
    testID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

    userRepo := repositories.NewUsersRepository(db)

    // Override with deterministic generator
    userRepo.GenerateIdFunc = func() uuid.UUID { return testID }

    user, err := userRepo.Create(ctx, repositories.CreateUsersParams{
        Name:  "Test User",
        Email: "test@example.com",
    })

    require.NoError(t, err)
    assert.Equal(t, testID, user.Id)
}
```

### Sequential Test IDs

Generate sequential IDs for tests with multiple records:

```go
func TestMultipleUsers(t *testing.T) {
    counter := 0

    userRepo := repositories.NewUsersRepository(db)

    // Override with sequential generator
    userRepo.GenerateIdFunc = func() uuid.UUID {
        counter++
        return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", counter))
    }

    user1, _ := userRepo.Create(ctx, CreateUsersParams{Name: "User 1", Email: "user1@example.com"})
    user2, _ := userRepo.Create(ctx, CreateUsersParams{Name: "User 2", Email: "user2@example.com"})

    // user1.Id = 00000000-0000-0000-0000-000000000001
    // user2.Id = 00000000-0000-0000-0000-000000000002
}
```

## Alternative ID Formats

While skimatik generates code for UUID primary keys, you can adapt the pattern for other ID formats.

### ULID (Universally Unique Lexicographically Sortable Identifier)

ULIDs are similar to UUID v7 but with better readability:

```go
import "github.com/oklog/ulid/v2"

userRepo := repositories.NewUsersRepository(db)

// Override with ULID generator
userRepo.GenerateIdFunc = func() uuid.UUID {
    entropy := ulid.DefaultEntropy()
    id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
    return uuid.UUID(id)
}
```

### KSUID (K-Sortable Unique Identifier)

KSUIDs are time-ordered and URL-safe:

```go
import "github.com/segmentio/ksuid"

userRepo := repositories.NewUsersRepository(db)

// Override with KSUID generator
userRepo.GenerateIdFunc = func() uuid.UUID {
    id := ksuid.New()
    var uuidBytes [16]byte
    copy(uuidBytes[:], id.Bytes())
    return uuid.UUID(uuidBytes)
}
```

### Snowflake IDs (Twitter's Distributed ID)

For systems requiring int64 IDs, convert to UUID:

```go
import "github.com/bwmarrin/snowflake"

// Initialize snowflake node (typically in main)
node, _ := snowflake.NewNode(1)

userRepo := repositories.NewUsersRepository(db)

// Override with Snowflake generator
userRepo.GenerateIdFunc = func() uuid.UUID {
    id := node.Generate().Int64()
    var uuidBytes [16]byte
    binary.BigEndian.PutUint64(uuidBytes[8:], uint64(id))
    return uuid.UUID(uuidBytes)
}
```

## Migration from Database-Generated IDs

If you're migrating from database-generated IDs to application-generated IDs:

### Step 1: Update Database Schema

Remove DEFAULT clauses from your migrations:

```sql
-- Before
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL
);

-- After
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL
);
```

### Step 2: Update Repository Construction

Update all repository instantiation to use the new signature:

```go
// Before (if you had custom code)
userRepo := &UserRepository{db: db}

// After
userRepo := repositories.NewUsersRepository(db)
```

### Step 3: Regenerate Code

```bash
skimatik --config=skimatik.yaml
```

### Step 4: Deploy Changes

The generated repositories now handle ID generation automatically. No changes to your business logic are required.

## Best Practices

### 1. Use Default Generator in Production

Unless you have specific requirements, use the default UUID v7 generator:

```go
userRepo := repositories.NewUsersRepository(db)
```

### 2. Use Deterministic Generators for Tests

Tests become more predictable and easier to debug:

```go
func setupTestRepo(t *testing.T) *repositories.UsersRepository {
    db := setupTestDB(t)
    testID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

    repo := repositories.NewUsersRepository(db)
    repo.GenerateIdFunc = func() uuid.UUID { return testID }

    return repo
}
```

### 3. Consistent ID Strategy Per Service

Within a service, use the same ID generation strategy:

```go
type Service struct {
    userRepo  *repositories.UsersRepository
    orderRepo *repositories.OrdersRepository
}

func NewService(db *pgxkit.DB) *Service {
    // Both repos use default UUID v7
    return &Service{
        userRepo:  repositories.NewUsersRepository(db),
        orderRepo: repositories.NewOrdersRepository(db),
    }
}
```

### 4. Document Custom ID Strategies

If using non-default generators, document why:

```go
// Using UUID v4 instead of v7 to prevent ID prediction in public APIs
userRepo := repositories.NewUsersRepository(db)
userRepo.GenerateIdFunc = func() uuid.UUID {
    return uuid.New() // UUID v4
}
```

## Repository Structure

All generated repositories follow this pattern:

```go
type <Table>Repository struct {
    db             *pgxkit.DB
    GenerateIdFunc func() uuid.UUID // Public property for ID generation
}

func New<Table>Repository(db *pgxkit.DB) *<Table>Repository {
    return &<Table>Repository{
        db:             db,
        GenerateIdFunc: DefaultUUIDv7Generator, // UUID v7 by default
    }
}
```

Where:
- `db`: Database connection (required in constructor)
- `GenerateIdFunc`: Public property that can be overridden after construction
- Default: UUID v7 generator is automatically set

## Testing Patterns

### Unit Tests with Mock IDs

```go
func TestUserService_CreateUser(t *testing.T) {
    expectedID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

    db := setupTestDB(t)
    userRepo := repositories.NewUsersRepository(db)
    userRepo.GenerateIdFunc = func() uuid.UUID { return expectedID }

    userService := NewUserService(userRepo)

    user, err := userService.Register(ctx, "Test User", "test@example.com")

    require.NoError(t, err)
    assert.Equal(t, expectedID, user.Id)
}
```

### Integration Tests with Default Generator

```go
func TestIntegration_UserFlow(t *testing.T) {
    db := setupIntegrationDB(t)

    // Use default UUID v7 generator
    userRepo := repositories.NewUsersRepository(db)

    user, err := userRepo.Create(ctx, CreateUsersParams{
        Name:  "Integration Test",
        Email: "integration@example.com",
    })

    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, user.Id)
}
```

## Troubleshooting

### IDs Not Being Generated

**Problem**: `nil` UUID in created records

**Solution**: The default UUID v7 generator is automatically set. If you're seeing nil UUIDs, verify the repository was constructed properly:

```go
// Correct - default generator is set automatically
userRepo := repositories.NewUsersRepository(db)

// Custom generator override
userRepo.GenerateIdFunc = func() uuid.UUID {
    return uuid.New()
}
```

### ID Collision in Tests

**Problem**: Multiple tests use the same deterministic ID

**Solution**: Use unique IDs per test or sequential generator:

```go
func TestA(t *testing.T) {
    id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
    userRepo := repositories.NewUsersRepository(db)
    userRepo.GenerateIdFunc = func() uuid.UUID { return id }
}

func TestB(t *testing.T) {
    id := uuid.MustParse("00000000-0000-0000-0000-000000000002")
    userRepo := repositories.NewUsersRepository(db)
    userRepo.GenerateIdFunc = func() uuid.UUID { return id }
}
```

### Performance Concerns

**Question**: Is application-side ID generation slower?

**Answer**: No. UUID v7 generation is extremely fast (microseconds), and moving it to the application eliminates a database round-trip for ID generation.

## Related Documentation

- **[Quick Start](quick-start)** - Basic usage with default ID generation
- **[Database Migrations](database-migrations)** - Schema setup without DEFAULT clauses
- **[Embedding Patterns](embedding-patterns)** - Custom repository patterns with ID generators
- **[Examples](examples)** - Real-world usage examples
