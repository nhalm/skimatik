# Example Blog Application

DATABASE_URL = postgres://postgres:password@localhost:15987/blog?sslmode=disable
MIGRATE_VERSION = v4.17.0

.PHONY: help setup generate test run clean install-migrate migrate-up migrate-down migrate-status migrate-create

help:
	@echo "Example Blog Application"
	@echo ""
	@echo "Commands:"
	@echo "  make setup         - Start database and run migrations"
	@echo "  make generate      - Generate Go code with skimatik"
	@echo "  make test          - Run tests (auto-setup)"
	@echo "  make run           - Run the application"
	@echo "  make clean         - Clean generated code and stop database"
	@echo ""
	@echo "Migration commands:"
	@echo "  make migrate-up     - Apply all pending migrations"
	@echo "  make migrate-down   - Rollback one migration"
	@echo "  make migrate-status - Show migration status"
	@echo "  make migrate-create NAME=<name> - Create a new migration"

install-migrate:
	@if ! command -v migrate >/dev/null 2>&1; then \
		echo "Installing golang-migrate..."; \
		if [ "$$(uname)" = "Darwin" ]; then \
			brew install golang-migrate; \
		elif [ "$$(uname)" = "Linux" ]; then \
			curl -L https://github.com/golang-migrate/migrate/releases/download/$(MIGRATE_VERSION)/migrate.linux-amd64.tar.gz | tar xvz && \
			sudo mv migrate /usr/local/bin/migrate; \
		else \
			echo "Please install golang-migrate manually"; \
			exit 1; \
		fi; \
	fi

setup: install-migrate
	@echo "Setting up database..."
	@docker run --name blog-db \
		-e POSTGRES_DB=blog \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=password \
		-p 15987:5432 \
		-d postgres:15-alpine 2>/dev/null || true
	@echo "Waiting for database..."
	@bash -c 'for i in {1..30}; do if pg_isready -h localhost -p 15987 -U postgres >/dev/null 2>&1; then break; fi; sleep 1; done'
	@echo "Running migrations..."
	@migrate -database "$(DATABASE_URL)" -path internal/database/migrations up 2>/dev/null || sleep 2 && migrate -database "$(DATABASE_URL)" -path internal/database/migrations up
	@echo "✅ Database ready"

generate:
	@echo "Generating code..."
	@../bin/skimatik
	@echo "✅ Code generated"

test: setup generate
	@DATABASE_URL="$(DATABASE_URL)" go test -tags=integration -count=1 -v ./...
	@echo "✅ Tests passed"

run:
	@export DATABASE_URL="$(DATABASE_URL)" && go run ./cmd/blog

migrate-up: install-migrate
	@migrate -database "$(DATABASE_URL)" -path internal/database/migrations up

migrate-down: install-migrate
	@migrate -database "$(DATABASE_URL)" -path internal/database/migrations down 1

migrate-status: install-migrate
	@migrate -database "$(DATABASE_URL)" -path internal/database/migrations version

migrate-create: install-migrate
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=<name>"; exit 1; fi
	@migrate create -ext sql -dir internal/database/migrations -seq $(NAME)

clean:
	@rm -rf internal/repository/generated/
	@docker stop blog-db 2>/dev/null || true
	@docker rm blog-db 2>/dev/null || true
	@echo "✅ Clean"
