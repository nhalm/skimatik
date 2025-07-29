# Example Blog Application Makefile
# Demonstrates full skimatik workflow: database setup → code generation → running app

# Configuration
DB_NAME = blog
DB_USER = postgres
DB_PASSWORD = password
DB_HOST = localhost
DB_PORT = 5432
DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Docker container name
CONTAINER_NAME = blog-db

# Skimatik configuration
QUERIES_DIR = database/queries
OUTPUT_DIR = repository/generated
PACKAGE_NAME = generated

.PHONY: help setup db-start db-stop db-create db-migrate db-reset generate clean build run test dev logs stop-all

# Default target
help: ## Show this help message
	@echo "Example Blog Application - skimatik demonstration"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Quick start: make dev"

# Complete development setup
setup: db-start db-create db-migrate generate ## Complete setup: database + schema + code generation
	@echo "✅ Setup complete! Ready to run 'make run'"

dev: setup run ## Full development workflow: setup + run application

# Database targets
db-start: ## Start PostgreSQL database in Docker
	@echo "🐘 Starting PostgreSQL database..."
	@if [ $$(docker ps -q -f name=$(CONTAINER_NAME)) ]; then \
		echo "Database container already running"; \
	else \
		docker run --name $(CONTAINER_NAME) \
			-e POSTGRES_DB=$(DB_NAME) \
			-e POSTGRES_USER=$(DB_USER) \
			-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
			-p $(DB_PORT):5432 \
			-d postgres:15-alpine; \
		echo "Waiting for database to be ready..."; \
		sleep 5; \
		while ! docker exec $(CONTAINER_NAME) pg_isready -U $(DB_USER) -d $(DB_NAME) > /dev/null 2>&1; do \
			echo "Waiting for database..."; \
			sleep 2; \
		done; \
		echo "✅ Database is ready"; \
	fi

db-stop: ## Stop and remove database container
	@echo "🛑 Stopping database..."
	@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true
	@echo "✅ Database stopped"

db-create: ## Create database (if not exists)
	@echo "🗄️  Ensuring database exists..."
	@docker exec -i $(CONTAINER_NAME) psql -U $(DB_USER) -tc "SELECT 1 FROM pg_database WHERE datname = '$(DB_NAME)'" | grep -q 1 || \
		docker exec -i $(CONTAINER_NAME) psql -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME);"
	@echo "✅ Database ready"

db-migrate: ## Apply database schema and sample data
	@echo "📋 Applying database schema..."
	@docker exec -i $(CONTAINER_NAME) psql -U $(DB_USER) -d $(DB_NAME) < database/schema.sql
	@echo "✅ Schema applied with sample data"

db-reset: db-stop db-start db-create db-migrate ## Reset database: stop, start, recreate schema
	@echo "♻️  Database reset complete"

# Code generation targets
generate: ## Generate Go code from SQL queries using skimatik
	@echo "⚡ Generating code with skimatik..."
	@mkdir -p $(OUTPUT_DIR)
	@if ! command -v skimatik >/dev/null 2>&1; then \
		echo "❌ skimatik not found. Please install it first:"; \
		echo "   go install github.com/nhalm/skimatik/cmd/skimatic@latest"; \
		echo "   (or build locally: make -C .. build)"; \
		exit 1; \
	fi
	@skimatik \
		--database-url="$(DATABASE_URL)" \
		--queries-dir="$(QUERIES_DIR)" \
		--output-dir="$(OUTPUT_DIR)" \
		--package-name="$(PACKAGE_NAME)" \
		--verbose
	@echo "✅ Code generation complete"

# Application targets
build: generate ## Build the application binary
	@echo "🔨 Building application..."
	@go mod tidy
	@go build -o bin/blog-api .
	@echo "✅ Build complete: bin/blog-api"

run: ## Run the application (requires generate)
	@echo "🚀 Starting blog API server..."
	@if [ ! -d "$(OUTPUT_DIR)" ]; then \
		echo "❌ Generated code not found. Run 'make generate' first"; \
		exit 1; \
	fi
	@export DATABASE_URL="$(DATABASE_URL)" && go run .

test: ## Run tests
	@echo "🧪 Running tests..."
	@go mod tidy
	@go test -v ./...

# Development targets
logs: ## Show database logs
	@docker logs -f $(CONTAINER_NAME)

clean: ## Clean generated code and build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@rm -rf bin/
	@echo "✅ Clean complete"

stop-all: db-stop ## Stop all services
	@echo "🛑 All services stopped"

# Utility targets
db-shell: ## Connect to database shell
	@docker exec -it $(CONTAINER_NAME) psql -U $(DB_USER) -d $(DB_NAME)

db-status: ## Check database status
	@echo "📊 Database status:"
	@if [ $$(docker ps -q -f name=$(CONTAINER_NAME)) ]; then \
		echo "✅ Container running"; \
		docker exec $(CONTAINER_NAME) pg_isready -U $(DB_USER) -d $(DB_NAME) && echo "✅ Database ready" || echo "❌ Database not ready"; \
	else \
		echo "❌ Container not running"; \
	fi

show-config: ## Show current configuration
	@echo "📋 Configuration:"
	@echo "  Database URL: $(DATABASE_URL)"
	@echo "  Queries Dir:  $(QUERIES_DIR)"
	@echo "  Output Dir:   $(OUTPUT_DIR)"
	@echo "  Package:      $(PACKAGE_NAME)"

# API testing targets (requires curl)
test-api: ## Test API endpoints (requires running server)
	@echo "🧪 Testing API endpoints..."
	@echo "Health check:"
	@curl -s http://localhost:8080/api/health | jq '.' || echo "❌ Health check failed"
	@echo "\nActive users:"
	@curl -s http://localhost:8080/api/users | jq '.users[] | {id, name, email}' || echo "❌ Users endpoint failed"
	@echo "\nPublished posts:"
	@curl -s http://localhost:8080/api/posts | jq '.posts[] | {id, title, author_id}' || echo "❌ Posts endpoint failed"

# Documentation
demo: ## Show complete workflow demonstration
	@echo "🎬 skimatik Example Application Demo"
	@echo "=================================="
	@echo ""
	@echo "This Makefile demonstrates the complete skimatik workflow:"
	@echo ""
	@echo "1. 📋 Database Schema (database/schema.sql)"
	@echo "   - PostgreSQL with UUID v7 primary keys"
	@echo "   - Realistic blog schema: users, posts, comments, tags"
	@echo ""
	@echo "2. 📝 SQL Queries (database/queries/*.sql)"
	@echo "   - Custom SQL with sqlc-style annotations"
	@echo "   - Complex joins, aggregations, business logic"
	@echo ""
	@echo "3. ⚡ Code Generation (skimatik)"
	@echo "   - Generates type-safe Go functions from SQL"
	@echo "   - Creates repository/generated/*.go"
	@echo ""
	@echo "4. 🏗️ Application Layers"
	@echo "   - Service layer: business logic"
	@echo "   - API layer: HTTP handlers"
	@echo "   - Clean architecture with dependency injection"
	@echo ""
	@echo "Quick commands:"
	@echo "  make dev     # Complete setup and run"
	@echo "  make setup   # Database + generation only"
	@echo "  make test-api # Test the running API"
	@echo "" 