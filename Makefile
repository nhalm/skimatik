# Makefile for skimatik project

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint
BLUEPRINT_VET_VERSION=v0.1.0

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
test-example-app: build install-blueprint-vet
	@echo "Running example-app tests..."
	@cd example-app && $(MAKE) test
	@echo "Running blueprint-vet on generated example-app code..."
	@cd example-app && blueprint-vet -nofmtprint=false ./...
	@echo "✅ Example app tests completed"

.PHONY: test-all
test-all: test-unit test-integration test-example-app
	@echo "✅ All tests completed"

.PHONY: lint
lint: install-blueprint-vet
	@echo "Running linter..."
	go fmt ./...
	$(GOLINT) run ./...
	@echo "Running blueprint-vet (skimatik)..."
	@blueprint-vet -nofmtprint=false ./...
	@if [ -d example-app/internal/repository/generated ] && [ -n "$$(ls -A example-app/internal/repository/generated 2>/dev/null)" ]; then \
		echo "Running blueprint-vet (example-app)..."; \
		cd example-app && blueprint-vet -nofmtprint=false ./...; \
	else \
		echo "Skipping blueprint-vet (example-app): generated code not present — run 'make test-example-app' to validate"; \
	fi
	@echo "Running blueprint-sql-check (example-app queries)..."
	@blueprint-sql-check example-app/database/queries
	@echo "✅ Linting completed"

# Install blueprint-vet binaries if not already on PATH (or out of date).
# Pinned to BLUEPRINT_VET_VERSION; nofmtprint is disabled because skimatik is a
# code generator — fmt.Sprintf/Fprintf are how the generator builds source code,
# not runtime logging. See docs/blueprint-vet.md.
.PHONY: install-blueprint-vet
install-blueprint-vet:
	@command -v blueprint-vet >/dev/null 2>&1 || go install github.com/nhalm/blueprint-vet/cmd/blueprint-vet@$(BLUEPRINT_VET_VERSION)
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
