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
DOCKER_COMPOSE=docker compose -f build/docker-compose.yml

TEST_DB_URL=postgres://skimatik:skimatik_test_password@localhost:15432/skimatik_test?sslmode=disable

.PHONY: default
default: help

# Bootstrap dev tools and activate Git hooks. Mirrors go-blueprint's
# setup / install-tools split, but folds `lefthook install` into setup
# because skimatik's setup has no other manual follow-up steps — there's
# no codegen step, no .env to fill in, so a one-shot bootstrap is
# friendlier than splitting it across two commands. golangci-lint and
# blueprint-sql-check stay out of install-tools because `make lint`
# bootstraps them on demand via the .custom-gcl.yml file dep.
.PHONY: setup
setup: install-tools
	@lefthook install
	@echo "✅ Setup complete"

.PHONY: install-tools
install-tools:
	@go install github.com/evilmartians/lefthook@latest
	@echo "✅ Dev tools installed"

.PHONY: build
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p bin
	$(GOBUILD) -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "✅ Binary built: $(BINARY_PATH)"

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	@$(GOMOD) tidy
	$(GOTEST) -v -race -timeout 30s -short ./...
	@echo "✅ Unit tests completed"

.PHONY: test-integration
test-integration:
	@echo "Starting database..."
	@$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for database..."
	@bash -c 'for i in {1..30}; do if pg_isready -h localhost -p 15432 -U skimatik -d skimatik_test >/dev/null 2>&1; then break; fi; sleep 1; done'
	@echo "Running integration tests..."
	@$(GOMOD) tidy
	TEST_DATABASE_URL=$(TEST_DB_URL) $(GOTEST) -v -race -parallel=1 -timeout 60s ./...
	@echo "✅ Integration tests completed"

.PHONY: test-example-app
test-example-app: build $(CUSTOM_GCL)
	@echo "Running example-app tests..."
	@cd example-app && $(MAKE) test
	@echo "Linting generated example-app code..."
	@cd example-app && ../$(CUSTOM_GCL) run ./...
	@echo "✅ Example app tests completed"

# `make lint` is the one-stop shop: format, lint (Go + blueprint-vet rules
# via the custom-gcl plugin), and SQL-annotation checks. It auto-installs
# any tools it needs and rebuilds ./bin/custom-gcl only when .custom-gcl.yml
# changes. Anything that wants to run "all the checks" — pre-commit hooks,
# CI, IDE-on-save — should just call this.
.PHONY: lint
lint: $(CUSTOM_GCL)
	@echo "Running linter..."
	@go fmt ./...
	@$(CUSTOM_GCL) run ./...
	@if [ -d example-app/internal/repository/generated ] && [ -n "$$(ls -A example-app/internal/repository/generated 2>/dev/null)" ]; then \
		echo "Linting example-app..."; \
		cd example-app && ../$(CUSTOM_GCL) run ./...; \
	else \
		echo "Skipping example-app lint: generated code not present — run 'make test-example-app' to validate"; \
	fi
	@command -v blueprint-sql-check >/dev/null 2>&1 || go install github.com/nhalm/blueprint-vet/cmd/blueprint-sql-check@$(BLUEPRINT_VET_VERSION)
	@blueprint-sql-check example-app/internal/repository/queries
	@echo "✅ Linting completed"

# Build the custom golangci-lint binary that bundles blueprint-vet's analyzers,
# configured by .custom-gcl.yml. Rebuilt only when .custom-gcl.yml changes.
$(CUSTOM_GCL): .custom-gcl.yml
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "Building custom-gcl from .custom-gcl.yml..."
	@golangci-lint custom

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
	@echo "First time:"
	@echo "  setup              Install dev tools and activate Git hooks (lefthook + lefthook install)"
	@echo "  install-tools      Install dev tools only (no hook activation)"
	@echo ""
	@echo "Daily workflow:"
	@echo "  build              Compile the skimatik binary"
	@echo "  test-unit          Fast tests (no database)"
	@echo "  lint               Format + lint everything (Go + blueprint-vet + SQL)"
	@echo ""
	@echo "Heavier targets:"
	@echo "  test-integration   Integration tests (auto-starts Postgres)"
	@echo "  test-example-app   Regenerate + test example-app end-to-end"
	@echo ""
	@echo "Other:"
	@echo "  clean              Remove build artifacts, stop services"
	@echo "  help               Show this help"
	@echo ""
