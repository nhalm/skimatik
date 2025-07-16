package generator

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Generator handles the code generation process
type Generator struct {
	config     *Config
	db         *pgxpool.Pool
	introspect *Introspector
	codegen    *CodeGenerator
}

// New creates a new generator instance
func New(config *Config) *Generator {
	return &Generator{
		config: config,
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
	defer g.db.Close()

	// Initialize components
	g.introspect = NewIntrospector(g.db, g.config.Schema)
	g.codegen = NewCodeGenerator(g.config)

	if g.config.Verbose {
		log.Printf("Connected to database, schema: %s", g.config.Schema)
	}

	// Generate table-based repositories
	if g.config.Tables {
		// Generate shared pagination types file first
		if err := g.generateSharedPaginationTypes(); err != nil {
			return fmt.Errorf("shared pagination types generation failed: %w", err)
		}

		if err := g.generateTables(ctx); err != nil {
			return fmt.Errorf("table generation failed: %w", err)
		}
	}

	// Generate query-based code
	if g.config.QueriesDir != "" {
		if err := g.generateQueries(ctx); err != nil {
			return fmt.Errorf("query generation failed: %w", err)
		}
	}

	return nil
}

// connect establishes a connection to the PostgreSQL database
func (g *Generator) connect(ctx context.Context) error {
	config, err := pgxpool.ParseConfig(g.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Configure connection pool for introspection
	config.MaxConns = 5
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	g.db = pool
	return nil
}

// generateTables generates repositories for database tables
func (g *Generator) generateTables(ctx context.Context) error {
	if g.config.Verbose {
		log.Println("Starting table introspection...")
	}

	// Get all tables in the schema
	tables, err := g.introspect.GetTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect tables: %w", err)
	}

	if g.config.Verbose {
		log.Printf("Found %d tables in schema '%s'", len(tables), g.config.Schema)
	}

	// Filter tables based on include patterns
	var filteredTables []Table
	for _, table := range tables {
		if g.config.ShouldIncludeTable(table.Name) {
			filteredTables = append(filteredTables, table)
		}
	}

	if g.config.Verbose {
		log.Printf("Generating code for %d tables after filtering", len(filteredTables))
	}

	// Generate code for each table
	for _, table := range filteredTables {
		if g.config.Verbose {
			log.Printf("Generating repository for table: %s", table.Name)
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

// generateQueries generates code from SQL query files
func (g *Generator) generateQueries(ctx context.Context) error {
	if g.config.Verbose {
		log.Printf("Starting query generation from directory: %s", g.config.QueriesDir)
	}

	// Parse SQL files
	parser := NewQueryParser(g.config.QueriesDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		return fmt.Errorf("failed to parse queries: %w", err)
	}

	if g.config.Verbose {
		log.Printf("Found %d queries to generate", len(queries))
	}

	// Analyze queries against database
	analyzer := NewQueryAnalyzer(g.db)
	for i := range queries {
		if g.config.Verbose {
			log.Printf("Analyzing query: %s", queries[i].Name)
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
			"dbutil-gen requires UUID v7 primary keys for consistent time-ordered pagination. "+
			"Please migrate your table to use UUID primary keys", pkColumn, column.Type)
	}

	return nil
}
