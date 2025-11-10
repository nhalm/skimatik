package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/nhalm/skimatik/internal/generator"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func getVersion() string {
	if version != "dev" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" && info.Main.Version != "" {
		return info.Main.Version
	}

	return "dev"
}

func getFullVersion() string {
	v := getVersion()
	if commit != "none" {
		commitHash := commit
		if len(commit) > 7 {
			commitHash = commit[:7]
		}
		v = fmt.Sprintf("%s (commit: %s)", v, commitHash)
	}
	if date != "unknown" {
		v = fmt.Sprintf("%s built at %s", v, date)
	}
	return v
}

func main() {
	var (
		config      = flag.String("config", "skimatik.yaml", "Path to YAML configuration file")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging output")
		help        = flag.Bool("help", false, "Show detailed help and examples")
		showVersion = flag.Bool("version", false, "Show version information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `skimatik - Database-first code generator for PostgreSQL

USAGE:
    skimatik [options]

DESCRIPTION:
    Generate type-safe Go repositories with built-in pagination from PostgreSQL databases.
    Supports both table-based generation (CRUD operations) and query-based generation
    (custom SQL with sqlc-style annotations).

REQUIREMENTS:
    - PostgreSQL 12+ database
    - Tables must have UUID primary keys for pagination
    - Go 1.24+ for generated code

OPTIONS:
`)
		flag.PrintDefaults()

		fmt.Fprintf(os.Stderr, `
EXAMPLES:
    # Generate repositories using configuration file (recommended)
    skimatik

    # Generate with custom config file
    skimatik --config="./my-config.yaml"

    # Use configuration file with custom path
    skimatik --config="custom-config.yaml"

    # Verbose output for debugging
    skimatik --verbose

CONFIGURATION FILE:
    Create skimatik.yaml:
        database:
          dsn: "postgres://user:pass@localhost/mydb"
          schema: "public"
        output:
          directory: "./repositories"
          package: "repositories"
        # Generate all functions by default (recommended)
        default_functions: "all"
        tables:
          users:
          posts:
          comments:
            functions: ["create", "delete"]  # Override for specific tables
        verbose: true

    Alternative (verbose format still supported):
        tables:
          users:
            functions: ["create", "get", "update", "delete", "list", "paginate"]
          posts:
            functions: ["create", "get", "list", "paginate"]

GENERATED FILES:
    Table-based generation (*_generated.go):
    - Struct representing the table
    - Repository with CRUD operations
    - Built-in pagination support with cursor-based queries
    - Type-safe parameter structs

    Query-based generation (*_queries_generated.go):
    - Struct representing query results
    - Functions matching your SQL annotations (:one, :many, :exec)
    - Type-safe parameter structs
    - No automatic pagination (write your own SQL with LIMIT/OFFSET)

    Shared files:
    - pagination.go: Common pagination types and utilities
    - database_operations.go: Shared database operation utilities
    - retry_operations.go: Retry logic for transient failures
    - errors.go: Error handling utilities

PAGINATION:
    Table-based repositories include efficient cursor-based pagination:
    - ListPaginated(ctx, PaginationParams) (*PaginationResult[T], error)
    - Uses UUID v7 time-ordering for consistent results
    - O(log n) performance regardless of dataset size

    Query-based functions use your SQL as-is (add LIMIT/OFFSET as needed)

MORE INFO:
    Documentation: https://github.com/nhalm/skimatik
    Examples:      https://github.com/nhalm/skimatik/tree/main/examples
    Issues:        https://github.com/nhalm/skimatik/issues

`)
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("skimatik %s\n", getFullVersion())
		fmt.Println("Database-first code generator for PostgreSQL")
		fmt.Println("https://github.com/nhalm/skimatik")
		os.Exit(0)
	}

	cfg, err := generator.LoadConfig(*config)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	if *verbose {
		cfg.Verbose = true
	}

	gen := generator.New(cfg, getVersion())
	ctx := context.Background()

	if err := gen.Generate(ctx); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Printf("Successfully generated code in %s\n", cfg.OutputDir)
}
