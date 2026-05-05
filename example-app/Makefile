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
	@echo "Testing application..."
	@go build -o /tmp/example-app-test ./cmd/blog
	@if command -v curl >/dev/null 2>&1; then \
		set -e; \
		lsof -ti:8080 | xargs kill -9 2>/dev/null || true; \
		sleep 1; \
		DATABASE_URL="$(DATABASE_URL)" /tmp/example-app-test & echo $$! > /tmp/example-app.pid; \
		sleep 2; \
		trap 'kill $$(cat /tmp/example-app.pid) 2>/dev/null || true; rm -f /tmp/example-app.pid /tmp/example-app-test' EXIT; \
		echo "Testing endpoints..."; \
		curl -f http://localhost:8080/api/health; \
		curl -f http://localhost:8080/api/users; \
		curl -f http://localhost:8080/api/posts; \
		curl -f http://localhost:8080/api/posts/with-stats; \
		curl -f http://localhost:8080/api/posts/featured; \
		curl -f "http://localhost:8080/api/users/search?q=test"; \
		echo ""; \
		echo "Audit demo: POST + PATCH on /api/users (audited table)..."; \
		EMAIL="audit-demo-$$$$@example.com"; \
		CREATED_JSON=$$(curl -sf -X POST http://localhost:8080/api/users \
			-H 'Content-Type: application/json' \
			-d "{\"name\":\"Audit Demo Original\",\"email\":\"$$EMAIL\"}"); \
		echo "Created: $$CREATED_JSON"; \
		USER_ID=$$(echo "$$CREATED_JSON" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p'); \
		if [ -z "$$USER_ID" ]; then echo "could not parse created user id"; exit 1; fi; \
		echo "Asserting one open audit row after Create..."; \
		AUDIT_AFTER_CREATE=$$(curl -sf "http://localhost:8080/api/users/$$USER_ID/audit"); \
		echo "Audit after create: $$AUDIT_AFTER_CREATE"; \
		echo "$$AUDIT_AFTER_CREATE" | grep -q '"count":1' || { echo "expected 1 audit row after create"; exit 1; }; \
		curl -sf -X PATCH "http://localhost:8080/api/users/$$USER_ID/name" \
			-H 'Content-Type: application/json' \
			-d '{"name":"Audit Demo Renamed"}'; \
		echo "Asserting two audit rows (one closed, one open) after Update..."; \
		AUDIT_AFTER_UPDATE=$$(curl -sf "http://localhost:8080/api/users/$$USER_ID/audit"); \
		echo "Audit after update: $$AUDIT_AFTER_UPDATE"; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q '"count":2' || { echo "expected 2 audit rows after update"; exit 1; }; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q 'Audit Demo Original' || { echo "expected closed audit row to retain original name"; exit 1; }; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q 'Audit Demo Renamed' || { echo "expected open audit row to carry new name"; exit 1; }; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q '"valid_to":"' || { echo "expected one closed audit row with non-null valid_to"; exit 1; }; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q '"valid_to":null' || { echo "expected one open audit row with null valid_to"; exit 1; }; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q '"version":1' || { echo "expected closed audit row with version=1"; exit 1; }; \
		echo "$$AUDIT_AFTER_UPDATE" | grep -q '"version":2' || { echo "expected open audit row with version=2"; exit 1; }; \
		FINAL_USER=$$(curl -sf "http://localhost:8080/api/users/$$USER_ID"); \
		echo "Final user: $$FINAL_USER"; \
		echo "$$FINAL_USER" | grep -q '"name":"Audit Demo Renamed"' || { echo "expected GET user to show updated name"; exit 1; }; \
	fi
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
