# Makefile for skimatik project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Project parameters
BINARY_NAME=skimatik
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/skimatic
DOCKER_COMPOSE=docker-compose -f build/docker-compose.yml

# Test parameters
TEST_DB_URL=postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test?sslmode=disable
TEST_TIMEOUT=30s

# Default target - show help
.PHONY: default
default: help

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "✅ Binary built: $(BINARY_PATH)"

# Run unit tests only (no database required)
.PHONY: test
test:
	@echo "Running unit tests..."
	$(GOMOD) tidy
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -short ./...
	@echo "✅ Unit tests completed"

# Run integration tests (requires database)
.PHONY: integration-test
integration-test: dev-setup
	@echo "Running integration tests..."
	$(GOMOD) tidy
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./...
	@echo "✅ Integration tests completed"

# Run all tests (unit + integration)
.PHONY: test-all
test-all:
	@echo "Running all tests..."
	$(GOMOD) tidy
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./...
	@echo "✅ All tests completed"

# Generate test output for development/testing
.PHONY: test-generate
test-generate: build dev-setup
	@echo "Generating test output..."
	@mkdir -p test-output
	@echo "database:" > test-output/config.yaml
	@echo "  dsn: \"$(TEST_DB_URL)\"" >> test-output/config.yaml
	@echo "  schema: \"public\"" >> test-output/config.yaml
	@echo "output:" >> test-output/config.yaml
	@echo "  directory: \"./test-output\"" >> test-output/config.yaml
	@echo "  package: \"testgen\"" >> test-output/config.yaml
	@echo "default_functions: \"all\"" >> test-output/config.yaml
	@echo "tables:" >> test-output/config.yaml
	@echo "  users:" >> test-output/config.yaml
	@echo "  posts:" >> test-output/config.yaml
	@echo "  comments:" >> test-output/config.yaml
	@echo "verbose: true" >> test-output/config.yaml
	$(BINARY_PATH) --config=test-output/config.yaml
	@echo "✅ Test generation completed"
	@echo "📁 Generated files in: ./test-output/"
	@echo "🔍 Key files to check:"
	@echo "   - test-output/database_operations.go  # New shared utilities"
	@echo "   - test-output/errors.go               # Shared error handling"
	@echo "   - test-output/pagination.go           # Shared pagination types"
	@echo "   - test-output/users_generated.go      # Example repository"

# Run linter and formatter
.PHONY: lint
lint:
	@echo "Running linter and formatter..."
	go fmt ./...
	$(GOLINT) run ./...
	@echo "✅ Linting completed"

# Setup development environment
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@echo "Starting PostgreSQL database..."
	$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for database to be ready..."
	@bash -c 'for i in {1..30}; do if pg_isready -h localhost -p 5432 -U dbutil -d dbutil_test >/dev/null 2>&1; then break; fi; sleep 1; done'
	@echo "Running test data migrations..."
	@./test/run_migrations.sh
	@echo "✅ Development environment ready!"
	@echo "Database URL: $(TEST_DB_URL)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf test_output/
	@rm -rf test-output/
	@$(DOCKER_COMPOSE) down -v --remove-orphans >/dev/null 2>&1 || true
	@echo "✅ Cleanup completed"

# Show help
.PHONY: help
help:
	@echo ""
	@echo "🔧 dbutil-gen - Database-first code generator for PostgreSQL"
	@echo ""
	@echo "📋 USAGE:"
	@echo "  make <target>    Run a specific target"
	@echo "  make             Show this help message"
	@echo ""
	@echo "🚀 ESSENTIAL TARGETS:"
	@echo "  build            Build the dbutil-gen binary"
	@echo "  test             Run unit tests only (no database required)"
	@echo "  integration-test Run integration tests (auto-starts database)"
	@echo "  test-all         Run all tests (unit + integration)"
	@echo "  test-generate    Generate test output for development/testing"
	@echo "  lint             Run linter and code formatter"
	@echo "  dev-setup        Setup development environment with database"
	@echo "  clean            Remove build artifacts and stop services"
	@echo ""
	@echo "💡 QUICK START:"
	@echo "  make build       # Build the tool"
	@echo "  make test        # Run unit tests (no database needed)"
	@echo "  make integration-test  # Run integration tests (auto-starts database)"
	@echo ""
	@echo "📚 MORE INFO:"
	@echo "  ./bin/dbutil-gen --help    # CLI tool usage and options"
	@echo "  https://github.com/nhalm/dbutil    # Documentation"
	@echo "" 