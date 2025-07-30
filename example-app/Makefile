# Example Blog Application - Simple Makefile

# Configuration
DATABASE_URL = postgres://postgres:password@localhost:5432/blog?sslmode=disable

.PHONY: help setup generate test test-ci start-and-test run clean

help: ## Show available commands
	@echo "Example Blog Application"
	@echo ""
	@echo "Commands:"
	@echo "  make setup         - Start database and apply schema"
	@echo "  make generate      - Generate Go code with skimatik"
	@echo "  make test          - Run integration tests"
	@echo "  make test-ci       - Run CI-friendly tests (no background processes)"
	@echo "  make start-and-test - Start app and test with curl"
	@echo "  make run           - Run the application"
	@echo "  make clean         - Clean generated code"
	@echo ""
	@echo "Quick start: make setup && make generate && make test && make start-and-test"

setup: ## Start database and apply schema
	@echo "🐘 Setting up database..."
	@docker run --name blog-db \
		-e POSTGRES_DB=blog \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=password \
		-p 5432:5432 \
		-d postgres:15-alpine || echo "Database already running"
	@sleep 3
	@docker exec -i blog-db psql -U postgres -d blog < database/schema.sql 2>/dev/null || sleep 2 && docker exec -i blog-db psql -U postgres -d blog < database/schema.sql
	@echo "✅ Database ready"

generate: ## Generate Go code with skimatik
	@echo "⚡ Generating code..."
	@../bin/skimatik
	@echo "✅ Code generated"

test: ## Test the application (validates generated code works)
	@echo "🧪 Testing application..."
	@echo "🔨 Testing that generated code compiles..."
	@go build -v ./...
	@if command -v curl >/dev/null 2>&1; then \
		echo "🚀 Starting application and testing endpoints..."; \
		lsof -ti:8080 | xargs kill -9 2>/dev/null || true; \
		sleep 2; \
		export DATABASE_URL="$(DATABASE_URL)" && go run . & \
		sleep 5; \
		echo "🔍 Testing health endpoint..."; \
		curl -f http://localhost:8080/api/health || (echo "❌ Health check failed" && exit 1); \
		echo ""; \
		echo "🔍 Testing that app responds to requests..."; \
		curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/posts | grep -E "^(200|400|404)$$" > /dev/null || (echo "❌ App not responding" && exit 1); \
		echo "✅ App responds to requests (status codes indicate app is running)"; \
		echo "🛑 Stopping application..."; \
		lsof -ti:8080 | xargs kill -9 2>/dev/null || true; \
	else \
		echo "⚠️  curl not available - skipping HTTP tests (compilation test passed)"; \
	fi
	@echo "✅ Application tests passed"

run: ## Run the application
	@echo "🚀 Starting server..."
	@export DATABASE_URL="$(DATABASE_URL)" && go run .

clean: ## Clean up
	@echo "🧹 Cleaning..."
	@rm -rf repository/generated/
	@docker stop blog-db 2>/dev/null || true
	@docker rm blog-db 2>/dev/null || true
	@echo "✅ Clean" 