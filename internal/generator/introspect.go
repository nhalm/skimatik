package generator

import (
	"context"
	"fmt"

	"github.com/nhalm/pgxkit/v2"
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

// getTables retrieves all tables in the schema with their columns and metadata
func (i *Introspector) getTables(ctx context.Context) ([]table, error) {
	tableNames, err := i.getTableNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %w", err)
	}

	if len(tableNames) == 0 {
		return []table{}, nil
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

	foreignKeysMap, err := i.getAllTableForeignKeys(ctx, tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}

	tables := make([]table, 0, len(tableNames))
	for _, tableName := range tableNames {
		table := table{
			Name:        tableName,
			Schema:      i.schema,
			Columns:     columnsMap[tableName],
			PrimaryKey:  primaryKeysMap[tableName],
			Indexes:     indexesMap[tableName],
			ForeignKeys: foreignKeysMap[tableName],
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// getTablesByName retrieves the named tables from the configured schema,
// bypassing any include/exclude filters. The returned map is keyed by table
// name; names that do not exist in the schema are silently omitted rather
// than surfaced as zero-value Table structs. An empty input slice
// short-circuits to an empty map without touching the database.
func (i *Introspector) getTablesByName(ctx context.Context, names []string) (map[string]table, error) {
	if len(names) == 0 {
		return map[string]table{}, nil
	}

	existing, err := i.filterExistingTableNames(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve table names: %w", err)
	}
	if len(existing) == 0 {
		return map[string]table{}, nil
	}

	columnsMap, err := i.getAllTableColumns(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	primaryKeysMap, err := i.getAllTablePrimaryKeys(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}

	indexesMap, err := i.getAllTableIndexes(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}

	foreignKeysMap, err := i.getAllTableForeignKeys(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}

	result := make(map[string]table, len(existing))
	for _, tableName := range existing {
		result[tableName] = table{
			Name:        tableName,
			Schema:      i.schema,
			Columns:     columnsMap[tableName],
			PrimaryKey:  primaryKeysMap[tableName],
			Indexes:     indexesMap[tableName],
			ForeignKeys: foreignKeysMap[tableName],
		}
	}
	return result, nil
}

// filterExistingTableNames returns the subset of `names` that exist as base
// tables in the configured schema.
func (i *Introspector) filterExistingTableNames(ctx context.Context, names []string) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		  AND table_name = ANY($2)
		ORDER BY table_name
	`

	rows, err := i.db.Query(ctx, query, i.schema, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var found []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		found = append(found, name)
	}
	return found, rows.Err()
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
func (i *Introspector) getAllTableColumns(ctx context.Context, tableNames []string) (map[string][]column, error) {
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

	columnsMap := make(map[string][]column)
	for rows.Next() {
		var tableName string
		var col column
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

// getAllTableIndexes retrieves all indexes for all tables in a single query.
// Column ordering is taken from pg_index.indkey so callers can reliably check
// the leading-column identity of each index. Expression columns surface as
// the empty string; INCLUDE-clause covering columns are excluded via
// pos < indnkeyatts; concurrently-built or otherwise unusable indexes are
// filtered via indisvalid AND indisready.
func (i *Introspector) getAllTableIndexes(ctx context.Context, tableNames []string) (map[string][]index, error) {
	query := `
		WITH idx AS (
			SELECT
				c.relname           AS table_name,
				ic.relname          AS index_name,
				x.indisunique       AS is_unique,
				x.indkey            AS indkey,
				x.indnkeyatts       AS indnkeyatts,
				x.indrelid          AS indrelid,
				generate_subscripts(x.indkey, 1) AS pos
			FROM pg_index x
			JOIN pg_class c       ON c.oid = x.indrelid
			JOIN pg_class ic      ON ic.oid = x.indexrelid
			JOIN pg_namespace n   ON n.oid = c.relnamespace
			WHERE n.nspname = $1
			  AND c.relname = ANY($2)
			  AND NOT x.indisprimary
			  AND x.indisvalid
			  AND x.indisready
		)
		SELECT
			idx.table_name,
			idx.index_name,
			idx.is_unique,
			idx.pos,
			COALESCE(a.attname, '') AS column_name
		FROM idx
		LEFT JOIN pg_attribute a
			ON a.attrelid = idx.indrelid
			AND a.attnum  = idx.indkey[idx.pos]
		WHERE idx.pos < idx.indnkeyatts
		ORDER BY idx.table_name, idx.index_name, idx.pos
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type key struct{ table, index string }
	type entry struct {
		isUnique bool
		columns  []string
	}
	order := make(map[string][]string) // table -> ordered list of index names
	seen := make(map[key]*entry)

	for rows.Next() {
		var tableName, indexName, columnName string
		var isUnique bool
		var pos int

		if err := rows.Scan(&tableName, &indexName, &isUnique, &pos, &columnName); err != nil {
			return nil, err
		}

		k := key{tableName, indexName}
		e, ok := seen[k]
		if !ok {
			e = &entry{isUnique: isUnique}
			seen[k] = e
			order[tableName] = append(order[tableName], indexName)
		}
		e.columns = append(e.columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	indexesMap := make(map[string][]index, len(order))
	for tableName, names := range order {
		for _, name := range names {
			e := seen[key{tableName, name}]
			indexesMap[tableName] = append(indexesMap[tableName], index{
				Name:     name,
				Columns:  e.columns,
				IsUnique: e.isUnique,
			})
		}
	}

	return indexesMap, nil
}

// getAllTableForeignKeys retrieves foreign-key constraints for all tables in
// the configured schema. Returns one row per FK column; composite FKs surface
// as multiple rows sharing a ConstraintName. Only same-schema references are
// returned.
func (i *Introspector) getAllTableForeignKeys(ctx context.Context, tableNames []string) (map[string][]foreignKey, error) {
	query := `
		SELECT
			tc.table_name        AS child_table,
			tc.constraint_name   AS constraint_name,
			kcu.column_name      AS child_column,
			kcu.ordinal_position AS ordinal_position,
			ccu.table_name       AS referenced_table,
			ccu.column_name      AS referenced_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_schema = kcu.constraint_schema
			AND tc.constraint_name  = kcu.constraint_name
			AND tc.table_schema     = kcu.table_schema
			AND tc.table_name       = kcu.table_name
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_schema = rc.constraint_schema
			AND tc.constraint_name  = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu
			ON rc.unique_constraint_schema = ccu.constraint_schema
			AND rc.unique_constraint_name  = ccu.constraint_name
			AND kcu.position_in_unique_constraint = (
				SELECT kcu2.ordinal_position
				FROM information_schema.key_column_usage kcu2
				WHERE kcu2.constraint_schema = ccu.constraint_schema
				  AND kcu2.constraint_name   = ccu.constraint_name
				  AND kcu2.column_name       = ccu.column_name
			)
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema    = $1
		  AND tc.table_name      = ANY($2)
		ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position
	`

	rows, err := i.db.Query(ctx, query, i.schema, tableNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string][]foreignKey)
	for rows.Next() {
		var (
			childTable       string
			constraintName   string
			childColumn      string
			ordinalPosition  int
			referencedTable  string
			referencedColumn string
		)

		if err := rows.Scan(
			&childTable,
			&constraintName,
			&childColumn,
			&ordinalPosition,
			&referencedTable,
			&referencedColumn,
		); err != nil {
			return nil, err
		}

		fkMap[childTable] = append(fkMap[childTable], foreignKey{
			ConstraintName:   constraintName,
			ColumnName:       childColumn,
			ReferencedTable:  referencedTable,
			ReferencedColumn: referencedColumn,
		})
	}

	return fkMap, rows.Err()
}
