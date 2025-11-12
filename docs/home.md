# skimatik Documentation

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

4. **Production patterns included**: Cursor-based pagination using UUID v7 time-ordering, retry logic for transient failures, comprehensive error handling. These aren't afterthoughts—they're built into every generated repository.

5. **Designed for extension**: Embed generated repositories into your own types. Add business logic, implement domain interfaces, compose repositories. The generated CRUD and utilities are your foundation, not your limitation.

**Compared to ORMs**: ORMs hide the database, generate inefficient queries, and make debugging painful. They discourage developers from learning how databases work, leading to applications that need more resources and scale poorly.

**Compared to sqlc**: sqlc gives you full SQL control and type safety through static SQL parsing, which is excellent. skimatik takes it further by using PostgreSQL's own query analyzer and information schema—we generate code based on what the database actually understands about your schema. You also get automatic CRUD repositories with built-in pagination, plus shared utilities for retry logic and error handling. With sqlc, you write every query manually, including basic operations.

**When to use skimatik**: You believe your data model is your domain model. You want to leverage PostgreSQL's capabilities fully—complex queries, CTEs, proper indexing. You value compile-time safety and zero reflection. You want your database schema to be self-documenting and accessible to tools.

**When to choose something else**: You're prototyping and don't care about performance. You prefer defining your domain in application code. You need to support multiple databases (though in practice, nobody switches databases).

## ✨ Features

- **Database-First**: Works with existing PostgreSQL databases, no schema migrations required
- **Type-Safe**: Generates fully typed Go code using pgx with comprehensive PostgreSQL type support
- **Bidirectional Pagination**: Table-based repositories include efficient cursor-based pagination using UUID v7 with forward/backward navigation and runtime sort selection
- **Zero Dependencies**: Generated code only requires pgx - no external pagination or ORM dependencies
- **Shared Utilities**: Eliminates code duplication with reusable database operation and retry patterns
- **Repository Embedding**: Generated repositories designed for clean composition and extension
- **Table-Based Generation**: Complete CRUD repositories with automatic bidirectional pagination for all database tables
- **Query-Based Generation**: Custom functions from SQL files with sqlc-style annotations
- **Compile-Time Safe Pagination**: `cursor_columns` annotation generates paginated queries with allowlist-validated sort columns
- **UUID v7 Optimized**: Time-ordered pagination with consistent performance for table-based operations
- **Production Ready**: Clean, formatted code following Go best practices

## 📚 Documentation Navigation

### Getting Started
- **[Quick Start Guide](quick-start)** - Installation and basic usage
- **[Examples & Tutorials](examples)** - Hands-on learning with real applications

### Developer Guides
- **[Shared Utilities Guide](shared-utilities)** - Database operations, retry logic, and error handling utilities
- **[Embedding Patterns](embedding-patterns)** - Repository composition and extension patterns
- **[Error Handling Guide](error-handling)** - Comprehensive error management strategies

### Reference Documentation
- **[Configuration Reference](configuration-reference)** - Complete configuration options
- **[Type Mapping](type-mapping)** - PostgreSQL to Go type mappings
- **[Database Migrations](database-migrations)** - Schema management with golang-migrate

## 🚀 Quick Start

### Installation

```bash
go install github.com/nhalm/skimatik/cmd/skimatik@latest
```

### Basic Usage

1. **Generate table-based repositories:**
```bash
skimatik --config="skimatik.yaml"
```

2. **Generated code example:**
```go
// users_generated.go
import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/nhalm/pgxkit"
)

type Users struct {
    Id        uuid.UUID  `json:"id" db:"id"`
    Name      string     `json:"name" db:"name"`
    Email     *string    `json:"email" db:"email"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

func (u Users) GetID() uuid.UUID { return u.Id }

type UsersRepository struct {
    db             *pgxkit.DB
    generateIdFunc func() uuid.UUID
}

func NewUsersRepository(db *pgxkit.DB, idGen func() uuid.UUID) *UsersRepository {
    if idGen == nil {
        idGen = UUIDv7
    }
    return &UsersRepository{
        db:             db,
        generateIdFunc: idGen,
    }
}

func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
    // Generated CRUD operations with shared utilities
}
```

## 🎯 Key Benefits

### For Developers
- **90% Less Duplication**: Shared utilities eliminate repetitive code
- **Type Safety**: Full compile-time checking maintained
- **IDE Support**: Perfect autocomplete, refactoring, and debugging
- **Zero Runtime Overhead**: All utilities generate concrete code

### For Architecture
- **Clean Embedding**: Generated repositories work perfectly with composition
- **Interface Freedom**: Teams define interfaces based on domain needs
- **Easy Testing**: Mock interfaces, not repositories
- **Maintainable**: Regeneration doesn't affect custom code

### For Production
- **Error Resilience**: Built-in retry logic for transient failures
- **Observability**: Comprehensive logging and error context
- **Performance**: No reflection, direct database operations
- **Reliability**: Battle-tested error handling patterns

## 💡 Philosophy

skimatik follows a **database-first, composition-friendly** approach:

1. **Your Database Schema is the Source of Truth** - We introspect existing PostgreSQL databases
2. **Teams Define Interfaces** - You create interfaces that match your domain needs
3. **We Generate Implementations** - Complete repositories with all CRUD operations
4. **You Embed and Extend** - Use composition to add business logic
5. **Shared Utilities Eliminate Duplication** - Common patterns centralized across all code

This approach ensures that generated code integrates seamlessly into your architecture while providing maximum flexibility for domain-specific requirements.

## 🏗️ Recommended Application Structure

skimatik works best with a clean multi-layer architecture:

```
your-project/
├── api/                    # HTTP handlers, routes, middleware
│   ├── handlers/
│   └── middleware/
├── service/                # Business logic and workflows
│   ├── user_service.go
│   └── order_service.go
├── repository/             # Generated data access layer
│   └── generated/          # skimatik generated code
│       ├── users_queries.go
│       ├── orders_queries.go
│       └── pagination.go
├── database/               # Database schema and queries
│   ├── schema.sql
│   └── queries/            # SQL files with annotations
│       ├── users.sql
│       └── orders.sql
└── main.go                 # Dependency injection & wiring
```

### Layer Responsibilities
- **`api/`** - HTTP concerns, request/response handling
- **`service/`** - Business rules, workflows, orchestration
- **`repository/generated/`** - Type-safe data access (skimatik generates)
- **`database/queries/`** - SQL files with annotations (you write)

## 📖 Complete Example Application

Want to see skimatik in action? The **[Complete Blog Application Example](examples#-complete-blog-application-example)** demonstrates a full-stack Go application with HTTP API, service layer, and database persistence using generated repositories.

**Features demonstrated:**
- 🔗 **Complete HTTP API** with REST endpoints
- 🏗️ **Repository embedding** patterns with custom business logic  
- 📊 **Real database schema** with foreign key relationships
- ⚡ **Generated + custom queries** from both tables and SQL files
- 🧪 **Integration testing** that validates the complete pipeline

```bash
# Try it yourself
git clone https://github.com/nhalm/skimatik.git
cd skimatik/example-app
make setup && make generate && make run
# Application starts at http://localhost:8080
```

---

**Next Steps**: Start with the [Quick Start Guide](quick-start) or dive into the [Complete Blog Application Example](examples#-complete-blog-application-example) to see the full architecture in action. 