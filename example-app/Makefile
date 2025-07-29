# Example Blog Application - Simple Makefile

# Configuration
DATABASE_URL = postgres://postgres:password@localhost:5432/blog?sslmode=disable

.PHONY: help setup generate run clean

help: ## Show available commands
	@echo "Example Blog Application"
	@echo ""
	@echo "Commands:"
	@echo "  make setup     - Start database and apply schema"
	@echo "  make generate  - Generate Go code with skimatik"
	@echo "  make run       - Run the application"
	@echo "  make clean     - Clean generated code"
	@echo ""
	@echo "Quick start: make setup && make generate && make run"

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
	@skimatik
	@echo "✅ Code generated"

run: ## Run the application
	@echo "🚀 Starting server..."
	@export DATABASE_URL="$(DATABASE_URL)" && go run .

clean: ## Clean up
	@echo "🧹 Cleaning..."
	@rm -rf repository/generated/
	@docker stop blog-db 2>/dev/null || true
	@docker rm blog-db 2>/dev/null || true
	@echo "✅ Clean" 