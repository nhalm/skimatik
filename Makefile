# Makefile for skimatik project

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOLANGCI_LINT_VERSION=v2.11.4
BLUEPRINT_VET_VERSION=v0.2.0
CUSTOM_GCL=./bin/custom-gcl

BINARY_NAME=skimatik
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/skimatik
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_COMPOSE=docker-compose -f build/docker-compose.yml

TEST_DB_URL=postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test?sslmode=disable

.PHONY: default
default: help

.PHONY: build
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p bin
	$(GOBUILD) -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "✅ Binary built: $(BINARY_PATH)"

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GOMOD) tidy
	$(GOTEST) -v -timeout 30s -short ./...
	@echo "✅ Unit tests completed"

.PHONY: test-integration
test-integration:
	@echo "Starting database..."
	@$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for database..."
	@bash -c 'for i in {1..30}; do if pg_isready -h localhost -p 5432 -U skimatik -d skimatik_test >/dev/null 2>&1; then break; fi; sleep 1; done'
	@echo "Running integration tests..."
	$(GOMOD) tidy
	TEST_DATABASE_URL=$(TEST_DB_URL) $(GOTEST) -v -timeout 30s ./...
	@echo "✅ Integration tests completed"

.PHONY: test-example-app
test-example-app: build $(CUSTOM_GCL)
	@echo "Running example-app tests..."
	@cd example-app && $(MAKE) test
	@echo "Linting generated example-app code..."
	@cd example-app && ../$(CUSTOM_GCL) run ./...
	@echo "✅ Example app tests completed"

.PHONY: test-all
test-all: test-unit test-integration test-example-app
	@echo "✅ All tests completed"

.PHONY: lint
lint: $(CUSTOM_GCL) install-blueprint-sql-check
	@echo "Running linter..."
	go fmt ./...
	@$(CUSTOM_GCL) run ./...
	@if [ -d example-app/internal/repository/generated ] && [ -n "$$(ls -A example-app/internal/repository/generated 2>/dev/null)" ]; then \
		echo "Linting example-app..."; \
		cd example-app && ../$(CUSTOM_GCL) run ./...; \
	else \
		echo "Skipping example-app lint: generated code not present — run 'make test-example-app' to validate"; \
	fi
	@echo "Running blueprint-sql-check (example-app queries)..."
	@blueprint-sql-check example-app/database/queries
	@echo "✅ Linting completed"

# Build the custom golangci-lint binary that bundles blueprint-vet's analyzers
# (configured by .custom-gcl.yml). The binary lands at ./bin/custom-gcl and
# replaces both `golangci-lint run` and the standalone `blueprint-vet` invocation.
$(CUSTOM_GCL): .custom-gcl.yml install-golangci-lint
	@echo "Building custom-gcl from .custom-gcl.yml..."
	@golangci-lint custom

.PHONY: install-golangci-lint
install-golangci-lint:
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: install-blueprint-sql-check
install-blueprint-sql-check:
	@command -v blueprint-sql-check >/dev/null 2>&1 || go install github.com/nhalm/blueprint-vet/cmd/blueprint-sql-check@$(BLUEPRINT_VET_VERSION)

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf test_output/
	@rm -rf test-output/
	@$(DOCKER_COMPOSE) down -v --remove-orphans >/dev/null 2>&1 || true
	@cd example-app && $(MAKE) clean 2>/dev/null || true
	@echo "✅ Cleanup completed"

.PHONY: help
help:
	@echo ""
	@echo "skimatik - Database-first code generator for PostgreSQL"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build            Build the skimatik binary"
	@echo "  test-unit        Run unit tests (no database)"
	@echo "  test-integration Run integration tests (auto-starts database)"
	@echo "  test-example-app Run example-app tests (auto-setup)"
	@echo "  test-all         Run all tests"
	@echo "  lint             Run linter and formatter"
	@echo "  clean            Remove build artifacts and stop services"
	@echo "  help             Show this help"
	@echo ""
