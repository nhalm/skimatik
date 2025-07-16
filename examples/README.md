# dbutil-gen Example

Complete REST API demonstrating generated repositories with CRUD operations and pagination.

## Quick Start

```bash
# From project root
make test-setup      # Start database
make test-generate   # Generate repositories

# Run example
cd examples
go run main.go

# Test endpoints
curl http://localhost:8080/health
curl http://localhost:8080/users
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com"}'
```

## Features Demonstrated

- CRUD operations with generated repositories
- Cursor-based pagination
- Error handling and HTTP responses
- Foreign key relationships (users ↔ posts)
- Integration with Chi router and standard HTTP libraries

The example shows how to integrate dbutil-gen generated code into a real application. 