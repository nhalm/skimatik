package generator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nhalm/pgxkit/v2"
)

// Generator handles the code generation process
type Generator struct {
	config     *Config
	db         *pgxkit.DB
	introspect *Introspector
	codegen    *CodeGenerator
	version    string
}

// New creates a new generator instance
func New(config *Config, version string) *Generator {
	return &Generator{
		config:  config,
		version: version,
	}
}

// Generate runs the complete generation process and returns the list of
// file paths written. The caller is responsible for any user-facing logging.
func (g *Generator) Generate(ctx context.Context) ([]string, error) {
	// Validate configuration
	if err := g.config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Connect to database
	if err := g.connect(ctx); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	defer g.shutdownDB()

	// Initialize components
	g.introspect = NewIntrospector(g.db, g.config.Schema)
	g.codegen = NewCodeGenerator(g.config, g.version)

	if g.config.Verbose {
		slog.Info("Connected to database", "schema", g.config.Schema)
	}

	// Generate table-based repositories
	if g.config.Tables {
		if err := g.generateTablesStage(ctx); err != nil {
			return nil, err
		}
	}

	// Generate query-based code
	if g.config.QueriesDir != "" {
		if err := g.generateQueriesStage(ctx); err != nil {
			return nil, err
		}
	}

	if g.config.Verbose {
		slog.Info("Successfully generated code", "output_dir", g.config.OutputDir)
	}

	return g.codegen.GeneratedFiles(), nil
}

// shutdownDB shuts down the database connection with a bounded timeout.
// Errors are logged at warn level rather than returned because shutdown
// happens in a deferred call.
func (g *Generator) shutdownDB() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := g.db.Shutdown(shutdownCtx); err != nil {
		slog.Warn("database shutdown encountered error", "error", err)
	}
}

// generateTablesStage emits the shared support files plus per-table
// repository code. It is the table-mode entry point used by Generate.
func (g *Generator) generateTablesStage(ctx context.Context) error {
	if err := g.generateSharedFiles(true); err != nil {
		return err
	}
	if err := g.generateTables(ctx); err != nil {
		return fmt.Errorf("table generation failed: %w", err)
	}
	return nil
}

// generateQueriesStage emits any shared support files needed by query-only
// generation, then generates per-query code. Shared files are only written
// when table generation didn't already write them.
func (g *Generator) generateQueriesStage(ctx context.Context) error {
	// Ensure shared files exist for queries even if tables are disabled.
	// When Tables is true, generateTablesStage already emitted these.
	if !g.config.Tables {
		if err := g.generateSharedFiles(false); err != nil {
			return err
		}
	}
	if err := g.generateQueries(ctx); err != nil {
		return fmt.Errorf("query generation failed: %w", err)
	}
	return nil
}

// generateSharedFiles emits the shared pagination/error/database-operations
// files. When includeIDGenerators is true, it also emits the shared ID
// generator helpers (only required when generating table repositories).
func (g *Generator) generateSharedFiles(includeIDGenerators bool) error {
	if err := g.generateSharedPaginationTypes(); err != nil {
		return fmt.Errorf("shared pagination types generation failed: %w", err)
	}
	if err := g.generateSharedErrors(); err != nil {
		return fmt.Errorf("shared error handling generation failed: %w", err)
	}
	if err := g.generateSharedDatabaseOperations(); err != nil {
		return fmt.Errorf("shared database operations generation failed: %w", err)
	}
	if includeIDGenerators {
		if err := g.generateSharedIDGenerators(); err != nil {
			return fmt.Errorf("shared ID generators generation failed: %w", err)
		}
	}
	return nil
}

// connect establishes a connection to the PostgreSQL database
func (g *Generator) connect(ctx context.Context) error {
	// Use pgxkit for connection management
	db := pgxkit.NewDB()
	err := db.Connect(ctx, g.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	g.db = db
	return nil
}

// generateTables generates repositories for database tables
func (g *Generator) generateTables(ctx context.Context) error {
	if g.config.Verbose {
		slog.Info("Starting table introspection")
	}

	// Get all tables in the schema
	tables, err := g.introspect.GetTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect tables: %w", err)
	}

	if g.config.Verbose {
		slog.Info("Found tables in schema", "count", len(tables), "schema", g.config.Schema)
	}

	// Filter tables based on include patterns
	var filteredTables []Table
	for _, table := range tables {
		if g.config.ShouldIncludeTable(table.Name) {
			filteredTables = append(filteredTables, table)
		}
	}

	if g.config.Verbose {
		slog.Info("Generating code for tables after filtering", "count", len(filteredTables))
	}

	// Generate code for each table
	for _, table := range filteredTables {
		if g.config.Verbose {
			slog.Info("Generating repository for table", "table", table.Name)
		}

		// Validate table has single-column primary key
		if err := g.validateTablePrimaryKey(table); err != nil {
			return fmt.Errorf("table %s validation failed: %w", table.Name, err)
		}

		// Generate repository code
		if err := g.codegen.GenerateTableRepository(table); err != nil {
			return fmt.Errorf("failed to generate repository for table %s: %w", table.Name, err)
		}
	}

	return nil
}

// generateSharedPaginationTypes generates the shared pagination types file
func (g *Generator) generateSharedPaginationTypes() error {
	return g.codegen.GenerateSharedPaginationTypes()
}

// generateSharedErrors generates the shared error handling utilities file
func (g *Generator) generateSharedErrors() error {
	return g.codegen.GenerateSharedErrors()
}

// generateSharedDatabaseOperations generates the shared database operation utilities file
func (g *Generator) generateSharedDatabaseOperations() error {
	return g.codegen.GenerateSharedDatabaseOperations()
}

// generateSharedIDGenerators generates the shared ID generator utilities file
func (g *Generator) generateSharedIDGenerators() error {
	return g.codegen.GenerateSharedIDGenerators()
}

// generateQueries generates code from SQL query files
func (g *Generator) generateQueries(ctx context.Context) error {
	if g.config.Verbose {
		slog.Info("Starting query generation from directory", "dir", g.config.QueriesDir)
	}

	// Parse SQL files
	parser := NewQueryParser(g.config.QueriesDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		return fmt.Errorf("failed to parse queries: %w", err)
	}

	if g.config.Verbose {
		slog.Info("Found queries to generate", "count", len(queries))
	}

	// Analyze queries against database
	analyzer := NewQueryAnalyzer(g.db, g.config.Schema)
	for i := range queries {
		if g.config.Verbose {
			slog.Info("Analyzing query", "name", queries[i].Name)
		}

		if err := analyzer.AnalyzeQuery(ctx, &queries[i]); err != nil {
			return fmt.Errorf("failed to analyze query %s: %w", queries[i].Name, err)
		}
	}

	// Generate code for queries
	if err := g.codegen.GenerateQueries(queries); err != nil {
		return fmt.Errorf("failed to generate query code: %w", err)
	}

	return nil
}

// validateTablePrimaryKey ensures the table has a single-column primary key
func (g *Generator) validateTablePrimaryKey(table Table) error {
	if len(table.PrimaryKey) == 0 {
		return fmt.Errorf("table has no primary key")
	}

	if len(table.PrimaryKey) > 1 {
		return fmt.Errorf("composite primary keys are not supported")
	}

	pkColumn := table.PrimaryKey[0]
	column := table.GetColumn(pkColumn)
	if column == nil {
		return fmt.Errorf("primary key column %s not found", pkColumn)
	}

	return nil
}
