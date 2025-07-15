package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nhalm/skimatic/internal/generator"
)

func main() {
	var (
		config  = flag.String("config", "dbutil-gen.yaml", "Path to YAML configuration file")
		help    = flag.Bool("help", false, "Show detailed help and examples")
		version = flag.Bool("version", false, "Show version information")
	)

	// Custom usage function with better formatting
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `dbutil-gen - Database-first code generator for PostgreSQL

USAGE:
    dbutil-gen [options]

DESCRIPTION:
    Generate type-safe Go repositories with built-in pagination from PostgreSQL databases.
    Supports both table-based generation (CRUD operations) and query-based generation
    (custom SQL with sqlc-style annotations).

REQUIREMENTS:
    - PostgreSQL 12+ database
    - Tables must have UUID primary keys for pagination
    - Go 1.21+ for generated code

OPTIONS:
`)
		flag.PrintDefaults()

		fmt.Fprintf(os.Stderr, `
EXAMPLES:
    # Generate repositories using configuration file (recommended)
    dbutil-gen

    # Generate with custom config file
    dbutil-gen --config="./my-config.yaml"

    # Generate repositories for specific tables with CLI flags (basic usage)
    dbutil-gen --dsn="postgres://user:pass@localhost/mydb" --tables --include="users,posts,comments"

    # Use environment variable for connection (DATABASE_URL)
    export DATABASE_URL="postgres://user:pass@localhost/mydb"
    dbutil-gen --tables

    # Use POSTGRES_* environment variables for connection
    export POSTGRES_HOST="localhost"
    export POSTGRES_PORT="5432"
    export POSTGRES_USER="myuser"
    export POSTGRES_PASSWORD="mypass"
    export POSTGRES_DB="mydb"
    dbutil-gen --tables

    # Generate from SQL files with custom queries
    dbutil-gen --dsn="postgres://..." --queries="./sql" --output="./repositories"

    # Use configuration file
    dbutil-gen --config="dbutil-gen.yaml"

    # Verbose output for debugging
    dbutil-gen --dsn="postgres://..." --tables --verbose

ENVIRONMENT VARIABLES:
    DATABASE_URL       PostgreSQL connection string (alternative to --dsn)
    POSTGRES_HOST      Database host (default: localhost)
    POSTGRES_PORT      Database port (default: 5432)
    POSTGRES_USER      Database user (default: postgres)
    POSTGRES_PASSWORD  Database password (default: empty)
    POSTGRES_DB        Database name (default: postgres)
    POSTGRES_SSLMODE   SSL mode (default: disable)

CONFIGURATION FILE:
    Create dbutil-gen.yaml:
        database:
          dsn: "postgres://user:pass@localhost/mydb"
          schema: "public"
        output:
          directory: "./repositories"
          package: "repositories"
        tables:
          users:
            functions:
              - "create"
              - "get"
              - "update"
              - "delete"
              - "list"
          posts:
            functions:
              - "create"
              - "get"
              - "list"
          comments:
            functions:
              - "create"
              - "delete"
        verbose: true

GENERATED FILES:
    Each table generates a *_generated.go file with:
    - Struct representing the table
    - Repository with CRUD operations
    - Pagination support with cursor-based queries
    - Type-safe parameter structs

    Shared files:
    - pagination.go: Common pagination types and utilities

PAGINATION:
    All generated repositories include efficient cursor-based pagination:
    - ListPaginated(ctx, PaginationParams) (*PaginationResult[T], error)
    - Uses UUID v7 time-ordering for consistent results
    - O(log n) performance regardless of dataset size

MORE INFO:
    Documentation: https://github.com/nhalm/skimatic
    Examples:      https://github.com/nhalm/skimatic/tree/main/examples
    Issues:        https://github.com/nhalm/skimatic/issues

`)
	}

	flag.Parse()

	// Handle help and version flags
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Println("dbutil-gen version 2.0.0")
		fmt.Println("Database-first code generator for PostgreSQL")
		fmt.Println("https://github.com/nhalm/skimatic")
		os.Exit(0)
	}

	// Load configuration file
	cfg, err := generator.LoadConfig(*config)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	// Create and run generator
	gen := generator.New(cfg)
	ctx := context.Background()

	if err := gen.Generate(ctx); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Printf("Successfully generated code in %s\n", cfg.OutputDir)
}
