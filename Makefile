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
TEST_DB_URL=postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test?sslmode=disable
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
	@bash -c 'for i in {1..30}; do if pg_isready -h localhost -p 5432 -U skimatik -d skimatik_test >/dev/null 2>&1; then break; fi; sleep 1; done'
	@echo "Running test data migrations..."
	@./test/run_migrations.sh
	@echo "✅ Development environment ready!"
	@echo "Database URL: $(TEST_DB_URL)"

# Example app integration test (validates end-to-end code generation and functionality)
# Tests: code generation, compilation, database connectivity, API endpoints, and repository functionality
# Note: Requires database to be set up separately (use with existing CI database or run 'make setup' first locally)
.PHONY: example-app-test
example-app-test: build
	@echo "🧪 Running example-app comprehensive integration test..."
	@echo "   This tests the complete skimatik workflow:"
	@echo "   • Code generation from schema and queries"
	@echo "   • Repository and service integration"
	@echo "   • Real database operations"
	@echo "   • API endpoint functionality"
	@cd example-app && $(MAKE) generate && $(MAKE) test
	@echo "✅ Example app integration test completed successfully"
	@echo "   All generated repositories and services are working correctly!"

# Clean example app
.PHONY: example-app-clean
example-app-clean:
	@echo "🧹 Cleaning example app..."
	@cd example-app && $(MAKE) clean

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
	@echo "🔧 skimatik - Database-first code generator for PostgreSQL"
	@echo ""
	@echo "📋 USAGE:"
	@echo "  make <target>    Run a specific target"
	@echo "  make             Show this help message"
	@echo ""
	@echo "🚀 ESSENTIAL TARGETS:"
	@echo "  build              Build the skimatik binary"
	@echo "  test               Run unit tests only (no database required)"
	@echo "  integration-test   Run integration tests (auto-starts database)"
	@echo "  example-app-test   Comprehensive end-to-end test with real application (validates full workflow)"
	@echo "  test-all           Run all tests (unit + integration)"
	@echo "  lint               Run linter and code formatter"
	@echo "  dev-setup          Setup development environment with database"
	@echo "  clean              Remove build artifacts and stop services"
	@echo ""
	@echo "💡 QUICK START:"
	@echo "  make build       # Build the tool"
	@echo "  make test        # Run unit tests (no database needed)"
	@echo "  make integration-test  # Run integration tests (auto-starts database)"
	@echo ""
	@echo "📚 MORE INFO:"
	@echo "  ./bin/skimatik --help    # CLI tool usage and options"
	@echo "  https://github.com/nhalm/skimatik    # Documentation"
	@echo "" 