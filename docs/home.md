# skimatik Documentation

[![Go Version](https://img.shields.io/github/go-mod/go-version/nhalm/skimatik)](https://golang.org/doc/devel/release.html)
[![CI Status](https://github.com/nhalm/skimatik/actions/workflows/ci.yml/badge.svg)](https://github.com/nhalm/skimatik/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nhalm/skimatik)](https://goreportcard.com/report/github.com/nhalm/skimatik)
[![Release](https://img.shields.io/github/v/release/nhalm/skimatik)](https://github.com/nhalm/skimatik/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A database-first code generator for PostgreSQL that creates type-safe Go repositories with built-in cursor-based pagination. Generate clean, efficient CRUD operations and custom query functions directly from your database schema.

## ✨ Features

- **Database-First**: Works with existing PostgreSQL databases, no schema migrations required
- **Type-Safe**: Generates fully typed Go code using pgx with comprehensive PostgreSQL type support
- **Built-in Pagination**: Every list operation includes efficient cursor-based pagination using UUID v7
- **Zero Dependencies**: Generated code only requires pgx - no external pagination or ORM dependencies
- **Shared Utilities**: Eliminates code duplication with reusable database operation and retry patterns
- **Repository Embedding**: Generated repositories designed for clean composition and extension
- **Table-Based Generation**: Complete CRUD repositories for all database tables
- **Query-Based Generation**: Custom functions from SQL files with sqlc-style annotations
- **UUID v7 Optimized**: Time-ordered pagination with consistent performance
- **Production Ready**: Clean, formatted code following Go best practices

## 📚 Documentation Navigation

### Getting Started
- **[Quick Start Guide](Quick-Start-Guide)** - Installation and basic usage
- **[Examples & Tutorials](Examples-and-Tutorials)** - Hands-on learning with real applications

### Developer Guides
- **[Shared Utilities Guide](Shared-Utilities-Guide)** - Database operations, retry logic, and error handling utilities
- **[Embedding Patterns](Embedding-Patterns)** - Repository composition and extension patterns
- **[Error Handling Guide](Error-Handling-Guide)** - Comprehensive error management strategies

### Reference Documentation
- **[Configuration Reference](Configuration-Reference)** - Complete configuration options
- **[Type Mapping Reference](Type-Mapping-Reference)** - PostgreSQL to Go type mappings
- **[CLI Reference](CLI-Reference)** - Command-line interface documentation

## 🚀 Quick Start

### Installation

```bash
go install github.com/nhalm/skimatik/cmd/skimatic@latest
```

### Basic Usage

1. **Generate table-based repositories:**
```bash
skimatik --config="skimatik.yaml"
```

2. **Generated code example:**
```go
// users_generated.go
type Users struct {
    Id        uuid.UUID          `json:"id" db:"id"`
    Name      string             `json:"name" db:"name"`
    Email     string             `json:"email" db:"email"`
    CreatedAt pgtype.Timestamptz `json:"created_at" db:"created_at"`
}

func (u Users) GetID() uuid.UUID { return u.Id }

type UsersRepository struct {
    db *pgxkit.DB
}

func NewUsersRepository(db *pgxkit.DB) *UsersRepository {
    return &UsersRepository{db: db}
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