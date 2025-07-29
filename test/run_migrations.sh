#!/bin/bash

# Migration runner for skimatik test environment
# This script runs all migration files in the test/migrations directory

set -e

# Configuration
DB_URL="${DATABASE_URL:-postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test?sslmode=disable}"
MIGRATIONS_DIR="$(dirname "$0")/migrations"

echo "Running migrations..."
echo "Database URL: $DB_URL"
echo "Migrations directory: $MIGRATIONS_DIR"

# Check if migrations directory exists
if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "Migrations directory not found: $MIGRATIONS_DIR"
    exit 1
fi

# Run each migration file in order
for migration_file in "$MIGRATIONS_DIR"/*.sql; do
    if [ -f "$migration_file" ]; then
        echo "Applying migration: $(basename "$migration_file")"
        psql "$DB_URL" -f "$migration_file"
        if [ $? -eq 0 ]; then
            echo "✓ Migration applied successfully: $(basename "$migration_file")"
        else
            echo "✗ Migration failed: $(basename "$migration_file")"
            exit 1
        fi
    fi
done

echo "All migrations completed successfully!" 