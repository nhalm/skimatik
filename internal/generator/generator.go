package generator

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nhalm/pgxkit"
)

// dummyQuerier is a placeholder for pgxkit's generic requirement
type dummyQuerier struct {
	pool *pgxpool.Pool
}

// newDummyQuerier creates a dummy querier for pgxkit
func newDummyQuerier(pool *pgxpool.Pool) *dummyQuerier {
	return &dummyQuerier{pool: pool}
}

// WithTx implements the pgxkit.Querier interface
func (d *dummyQuerier) WithTx(tx pgx.Tx) pgxkit.Querier {
	return &dummyQuerier{pool: d.pool} // Return self since we don't use transactions in generation
}

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

	// Connect to database using pgxkit
	dsn := pgxkit.GetDSN()
	if g.config.DSN != "" {
		dsn = g.config.DSN
	}

	// Use pgxkit for connection management
	conn, err := pgxkit.NewConnection(ctx, dsn, newDummyQuerier)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	// Get the underlying pgxpool.Pool
	g.db = conn.GetDB()

	// Test connection
	if err := g.db.Ping(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Create introspector
	g.introspect = NewIntrospector(g.db, g.config.Schema)

	// Create code generator
	g.codegen = NewCodeGenerator(g.config)

	// Generate code
	if err := g.generateCode(ctx); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	return nil
}

// generateCode performs the actual code generation
func (g *Generator) generateCode(ctx context.Context) error {
	// Handle table generation
	if g.config.Tables {
		// Get tables from database
		tables, err := g.introspect.GetTables(ctx)
		if err != nil {
			return fmt.Errorf("failed to introspect tables: %w", err)
		}

		log.Printf("Found %d tables to generate", len(tables))

		// Generate code for each table
		for _, table := range tables {
			log.Printf("Generating code for table: %s", table.Name)

			// Validate that the table has a valid UUID primary key
			pkCol := table.GetPrimaryKeyColumn()
			if pkCol == nil {
				log.Printf("Skipping table %s: no single primary key column found", table.Name)
				continue
			}

			// Use the type mapper to validate the primary key is a UUID
			typeMapper := NewTypeMapper(nil)
			if err := typeMapper.ValidateUUIDPrimaryKey(pkCol); err != nil {
				log.Printf("Skipping table %s: %v", table.Name, err)
				continue
			}

			if err := g.codegen.GenerateTableRepository(table); err != nil {
				return fmt.Errorf("failed to generate code for table %s: %w", table.Name, err)
			}
		}

		// Generate shared files
		if err := g.codegen.GenerateSharedPaginationTypes(); err != nil {
			return fmt.Errorf("failed to generate shared files: %w", err)
		}
	}

	// Handle query generation
	if g.config.QueriesDir != "" {
		log.Printf("Generating code for queries in directory: %s", g.config.QueriesDir)

		// Parse queries from SQL files
		parser := NewQueryParser(g.config.QueriesDir)
		queries, err := parser.ParseQueries()
		if err != nil {
			return fmt.Errorf("failed to parse queries: %w", err)
		}

		log.Printf("Found %d queries to generate", len(queries))

		// Analyze queries to extract parameters and columns
		analyzer := NewQueryAnalyzer(g.db)
		for i := range queries {
			if err := analyzer.AnalyzeQuery(ctx, &queries[i]); err != nil {
				return fmt.Errorf("failed to analyze query %s: %w", queries[i].Name, err)
			}
		}

		// Generate code for queries
		if err := g.codegen.GenerateQueries(queries); err != nil {
			return fmt.Errorf("failed to generate query code: %w", err)
		}
	}

	return nil
}
