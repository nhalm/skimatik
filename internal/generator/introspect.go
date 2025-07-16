package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/nhalm/pgxkit"
)

// Introspector handles database schema introspection
type Introspector struct {
	db     *pgxkit.DB
	schema string
}

// NewIntrospector creates a new introspector instance
func NewIntrospector(db *pgxkit.DB, schema string) *Introspector {
	return &Introspector{
		db:     db,
		schema: schema,
	}
}

// GetTables retrieves all tables in the schema with their columns and metadata
func (i *Introspector) GetTables(ctx context.Context) ([]Table, error) {
	// First, get all tables in the schema
	tableNames, err := i.getTableNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %w", err)
	}

	var tables []Table
	for _, tableName := range tableNames {
		table, err := i.getTableDetails(ctx, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get details for table %s: %w", tableName, err)
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// getTableNames retrieves all table names in the schema
func (i *Introspector) getTableNames(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := i.db.Query(ctx, query, i.schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, tableName)
	}

	return tableNames, rows.Err()
}

// getTableDetails retrieves detailed information about a specific table
func (i *Introspector) getTableDetails(ctx context.Context, tableName string) (Table, error) {
	table := Table{
		Name:   tableName,
		Schema: i.schema,
	}

	// Get columns
	columns, err := i.getTableColumns(ctx, tableName)
	if err != nil {
		return table, fmt.Errorf("failed to get columns: %w", err)
	}
	table.Columns = columns

	// Get primary key
	primaryKey, err := i.getTablePrimaryKey(ctx, tableName)
	if err != nil {
		return table, fmt.Errorf("failed to get primary key: %w", err)
	}
	table.PrimaryKey = primaryKey

	// Get indexes
	indexes, err := i.getTableIndexes(ctx, tableName)
	if err != nil {
		return table, fmt.Errorf("failed to get indexes: %w", err)
	}
	table.Indexes = indexes

	return table, nil
}

// getTableColumns retrieves all columns for a table
func (i *Introspector) getTableColumns(ctx context.Context, tableName string) ([]Column, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			CASE 
				WHEN data_type = 'ARRAY' THEN true 
				ELSE false 
			END as is_array,
			CASE 
				WHEN data_type = 'ARRAY' THEN 
					REPLACE(REPLACE(udt_name, '_', ''), 'varchar', 'text')
				ELSE 
					CASE 
						WHEN data_type = 'character varying' THEN 'varchar'
						WHEN data_type = 'timestamp without time zone' THEN 'timestamp'
						WHEN data_type = 'timestamp with time zone' THEN 'timestamptz'
						ELSE data_type
					END
			END as normalized_type
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var isNullable string
		var defaultValue *string
		var maxLength *int

		err := rows.Scan(
			&col.Name,
			&col.Type,
			&isNullable,
			&defaultValue,
			&maxLength,
			&col.IsArray,
			&col.Type, // This overwrites the original data_type with normalized_type
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if defaultValue != nil {
			col.DefaultValue = *defaultValue
		}
		if maxLength != nil {
			col.MaxLength = *maxLength
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// getTablePrimaryKey retrieves the primary key columns for a table
func (i *Introspector) getTablePrimaryKey(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = $1 
		  AND tc.table_name = $2
		  AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY kcu.ordinal_position
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKey []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKey = append(primaryKey, columnName)
	}

	return primaryKey, rows.Err()
}

// getTableIndexes retrieves all indexes for a table
func (i *Introspector) getTableIndexes(ctx context.Context, tableName string) ([]Index, error) {
	query := `
		SELECT 
			i.indexname,
			i.indexdef,
			CASE WHEN i.indexdef LIKE '%UNIQUE%' THEN true ELSE false END as is_unique
		FROM pg_indexes i
		WHERE i.schemaname = $1 AND i.tablename = $2
		  AND i.indexname NOT LIKE '%_pkey'  -- Exclude primary key indexes
		ORDER BY i.indexname
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var indexName, indexDef string
		var isUnique bool

		if err := rows.Scan(&indexName, &indexDef, &isUnique); err != nil {
			return nil, err
		}

		// Parse column names from index definition
		columns := i.parseIndexColumns(indexDef)

		index := Index{
			Name:     indexName,
			Columns:  columns,
			IsUnique: isUnique,
		}
		indexes = append(indexes, index)
	}

	return indexes, rows.Err()
}

// parseIndexColumns extracts column names from an index definition
func (i *Introspector) parseIndexColumns(indexDef string) []string {
	// This is a simplified parser for index definitions
	// Example: "CREATE INDEX idx_name ON table_name USING btree (column1, column2)"

	// Find the part between parentheses
	start := strings.Index(indexDef, "(")
	end := strings.LastIndex(indexDef, ")")

	if start == -1 || end == -1 || start >= end {
		return []string{}
	}

	columnsPart := indexDef[start+1 : end]

	// Split by comma and clean up
	var columns []string
	for _, col := range strings.Split(columnsPart, ",") {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}

		// Handle quoted column names
		if strings.HasPrefix(col, "\"") && strings.Contains(col, "\"") {
			// Find the closing quote
			endQuote := strings.Index(col[1:], "\"")
			if endQuote != -1 {
				col = col[:endQuote+2] // Include both quotes
			}
		} else {
			// For unquoted columns, remove any function calls or expressions
			if spaceIndex := strings.Index(col, " "); spaceIndex != -1 {
				col = col[:spaceIndex]
			}
		}

		if col != "" {
			columns = append(columns, col)
		}
	}

	return columns
}
