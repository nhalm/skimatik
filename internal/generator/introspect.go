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
	tableNames, err := i.getTableNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %w", err)
	}

	if len(tableNames) == 0 {
		return []Table{}, nil
	}

	columnsMap, err := i.getAllTableColumns(ctx, tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	primaryKeysMap, err := i.getAllTablePrimaryKeys(ctx, tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}

	indexesMap, err := i.getAllTableIndexes(ctx, tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}

	tables := make([]Table, 0, len(tableNames))
	for _, tableName := range tableNames {
		table := Table{
			Name:       tableName,
			Schema:     i.schema,
			Columns:    columnsMap[tableName],
			PrimaryKey: primaryKeysMap[tableName],
			Indexes:    indexesMap[tableName],
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

// getAllTableColumns retrieves all columns for all tables in a single query
func (i *Introspector) getAllTableColumns(ctx context.Context, tableNames []string) (map[string][]Column, error) {
	query := `
		SELECT
			table_name,
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			ordinal_position,
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
		WHERE table_schema = $1 AND table_name = ANY($2)
		ORDER BY table_name, ordinal_position
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnsMap := make(map[string][]Column)
	for rows.Next() {
		var tableName string
		var col Column
		var isNullable string
		var defaultValue *string
		var maxLength *int
		var ordinalPosition int

		err := rows.Scan(
			&tableName,
			&col.Name,
			&col.Type,
			&isNullable,
			&defaultValue,
			&maxLength,
			&ordinalPosition,
			&col.IsArray,
			&col.Type,
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

		columnsMap[tableName] = append(columnsMap[tableName], col)
	}

	return columnsMap, rows.Err()
}

// getAllTablePrimaryKeys retrieves primary key columns for all tables in a single query
func (i *Introspector) getAllTablePrimaryKeys(ctx context.Context, tableNames []string) (map[string][]string, error) {
	query := `
		SELECT
			tc.table_name,
			kcu.column_name,
			kcu.ordinal_position
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		WHERE tc.table_schema = $1
		  AND tc.table_name = ANY($2)
		  AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY tc.table_name, kcu.ordinal_position
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	primaryKeysMap := make(map[string][]string)
	for rows.Next() {
		var tableName string
		var columnName string
		var ordinalPosition int

		if err := rows.Scan(&tableName, &columnName, &ordinalPosition); err != nil {
			return nil, err
		}

		primaryKeysMap[tableName] = append(primaryKeysMap[tableName], columnName)
	}

	return primaryKeysMap, rows.Err()
}

// getAllTableIndexes retrieves all indexes for all tables in a single query
func (i *Introspector) getAllTableIndexes(ctx context.Context, tableNames []string) (map[string][]Index, error) {
	query := `
		SELECT
			i.tablename,
			i.indexname,
			i.indexdef,
			CASE WHEN i.indexdef LIKE '%UNIQUE%' THEN true ELSE false END as is_unique
		FROM pg_indexes i
		WHERE i.schemaname = $1
		  AND i.tablename = ANY($2)
		  AND i.indexname NOT LIKE '%_pkey'
		ORDER BY i.tablename, i.indexname
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexesMap := make(map[string][]Index)
	for rows.Next() {
		var tableName string
		var indexName, indexDef string
		var isUnique bool

		if err := rows.Scan(&tableName, &indexName, &indexDef, &isUnique); err != nil {
			return nil, err
		}

		columns := i.parseIndexColumns(indexDef)

		index := Index{
			Name:     indexName,
			Columns:  columns,
			IsUnique: isUnique,
		}
		indexesMap[tableName] = append(indexesMap[tableName], index)
	}

	return indexesMap, rows.Err()
}

// parseIndexColumns extracts column names from an index definition
func (i *Introspector) parseIndexColumns(indexDef string) []string {
	start := strings.Index(indexDef, "(")
	end := strings.LastIndex(indexDef, ")")

	if start == -1 || end == -1 || start >= end {
		return []string{}
	}

	columnsPart := indexDef[start+1 : end]

	var columns []string
	for _, col := range strings.Split(columnsPart, ",") {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}

		if strings.HasPrefix(col, "\"") && strings.Contains(col, "\"") {
			endQuote := strings.Index(col[1:], "\"")
			if endQuote != -1 {
				col = col[:endQuote+2]
			}
		} else {
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
