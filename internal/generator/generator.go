package generator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nhalm/pgxkit"
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

// Generate runs the complete generation process
func (g *Generator) Generate(ctx context.Context) error {
	// Validate configuration
	if err := g.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Connect to database
	if err := g.connect(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := g.db.Shutdown(shutdownCtx); err != nil {
			slog.Warn("database shutdown encountered error", "error", err)
		}
	}()

	// Initialize components
	g.introspect = NewIntrospector(g.db, g.config.Schema)
	g.codegen = NewCodeGenerator(g.config, g.version)

	if g.config.Verbose {
		slog.Info("Connected to database", "schema", g.config.Schema)
	}

	// Generate table-based repositories
	if g.config.Tables {
		// Generate shared files first
		if err := g.generateSharedPaginationTypes(); err != nil {
			return fmt.Errorf("shared pagination types generation failed: %w", err)
		}

		if err := g.generateSharedErrors(); err != nil {
			return fmt.Errorf("shared error handling generation failed: %w", err)
		}

		if err := g.generateSharedDatabaseOperations(); err != nil {
			return fmt.Errorf("shared database operations generation failed: %w", err)
		}

		if err := g.generateSharedRetryOperations(); err != nil {
			return fmt.Errorf("shared retry operations generation failed: %w", err)
		}

		if err := g.generateSharedIDGenerators(); err != nil {
			return fmt.Errorf("shared ID generators generation failed: %w", err)
		}

		if err := g.generateTables(ctx); err != nil {
			return fmt.Errorf("table generation failed: %w", err)
		}
	}

	// Generate query-based code
	if g.config.QueriesDir != "" {
		// Ensure shared database operations exist for queries even if tables are disabled
		if !g.config.Tables {
			if err := g.generateSharedErrors(); err != nil {
				return fmt.Errorf("shared error handling generation failed: %w", err)
			}

			if err := g.generateSharedDatabaseOperations(); err != nil {
				return fmt.Errorf("shared database operations generation failed: %w", err)
			}
		}

		if err := g.generateQueries(ctx); err != nil {
			return fmt.Errorf("query generation failed: %w", err)
		}
	}

	if g.config.Verbose {
		slog.Info("Successfully generated code", "output_dir", g.config.OutputDir)
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

		// Validate table has UUID primary key
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

// generateSharedRetryOperations generates the shared retry operation utilities file
func (g *Generator) generateSharedRetryOperations() error {
	return g.codegen.GenerateSharedRetryOperations()
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

// validateTablePrimaryKey ensures the table has a UUID primary key
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

	if !column.IsUUID() {
		return fmt.Errorf("primary key column %s must be UUID type, got %s. "+
			"skimatik requires UUID v7 primary keys for consistent time-ordered pagination. "+
			"Please migrate your table to use UUID primary keys", pkColumn, column.Type)
	}

	return nil
}
