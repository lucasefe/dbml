// Package introspect provides database introspection capabilities for PostgreSQL.
// It extracts schema information including tables, columns, primary keys,
// foreign keys, and indexes.
//
// Basic usage:
//
//	schema, err := introspect.Database(db,
//	    introspect.WithSchemas("public", "auth"),
//	    introspect.WithExcludeTables("migrations"),
//	)
//
// With custom type mapping:
//
//	mapper := introspect.NewPostgreSQLTypeMapper(map[string]string{
//	    "citext": "varchar",
//	})
//	schema, err := introspect.Database(db, introspect.WithTypeMapper(mapper))
package introspect

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/lucasefe/dbml/schema"

	_ "github.com/lib/pq"
)

// Database introspects a PostgreSQL database and returns its schema.
// Use options to customize which schemas and tables to include.
func Database(db *sql.DB, opts ...Option) (*schema.Schema, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	var schemaNames []string
	if o.includeAllSchemas {
		schemas, err := getAllSchemas(db)
		if err != nil {
			return nil, fmt.Errorf("failed to get schemas: %w", err)
		}
		schemaNames = schemas
	} else {
		schemaNames = o.schemas
	}

	result, err := introspectSchemas(db, schemaNames, o.typeMapper)
	if err != nil {
		return nil, err
	}

	if len(o.excludeTables) > 0 {
		result = schema.FilterTables(result, o.excludeTables)
	}

	return result, nil
}

// FromConnectionString connects to a PostgreSQL database and introspects it.
// This is a convenience function that handles connection management.
func FromConnectionString(connStr string, opts ...Option) (*schema.Schema, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return Database(db, opts...)
}

func introspectSchemas(db *sql.DB, schemaNames []string, mapper TypeMapper) (*schema.Schema, error) {
	if len(schemaNames) == 0 {
		schemaNames = []string{"public"}
	}

	result := &schema.Schema{}

	for _, schemaName := range schemaNames {
		tables, err := getTables(db, schemaName)
		if err != nil {
			return nil, fmt.Errorf("failed to get tables for schema %s: %w", schemaName, err)
		}

		for _, table := range tables {
			columns, err := getColumns(db, schemaName, table.Name, mapper)
			if err != nil {
				return nil, fmt.Errorf("failed to get columns for table %s.%s: %w", schemaName, table.Name, err)
			}
			table.Columns = columns

			primaryKeys, err := getPrimaryKeys(db, schemaName, table.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get primary keys for table %s.%s: %w", schemaName, table.Name, err)
			}
			table.PrimaryKeys = primaryKeys

			for i := range table.Columns {
				for _, pk := range primaryKeys {
					if table.Columns[i].Name == pk {
						table.Columns[i].IsPrimaryKey = true
						break
					}
				}
			}

			indexes, err := getIndexes(db, schemaName, table.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get indexes for table %s.%s: %w", schemaName, table.Name, err)
			}
			table.Indexes = indexes

			references, err := getForeignKeys(db, schemaName, table.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get foreign keys for table %s.%s: %w", schemaName, table.Name, err)
			}
			table.References = references

			result.Tables = append(result.Tables, table)
		}
	}

	return result, nil
}

func getAllSchemas(db *sql.DB) ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast', 'pg_temp_1', 'pg_toast_temp_1')
		ORDER BY schema_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		schemas = append(schemas, schemaName)
	}

	return schemas, rows.Err()
}

func getTables(db *sql.DB, schemaName string) ([]schema.Table, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, schema.Table{
			Name:   tableName,
			Schema: schemaName,
		})
	}

	return tables, rows.Err()
}

func getColumns(db *sql.DB, schemaName, tableName string, mapper TypeMapper) ([]schema.Column, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			c.is_nullable,
			c.column_default,
			COALESCE(c.udt_name, c.data_type) as udt_name
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var col schema.Column
		var dataType string
		var charMaxLength, numericPrecision, numericScale sql.NullInt64
		var isNullable string
		var columnDefault sql.NullString
		var udtName string

		err := rows.Scan(
			&col.Name,
			&dataType,
			&charMaxLength,
			&numericPrecision,
			&numericScale,
			&isNullable,
			&columnDefault,
			&udtName,
		)
		if err != nil {
			return nil, err
		}

		if mapper != nil {
			col.Type = mapper.MapType(dataType, udtName, charMaxLength, numericPrecision, numericScale)
		} else {
			col.Type = MapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
		}
		col.Nullable = isNullable == "YES"
		if columnDefault.Valid {
			col.DefaultValue = &columnDefault.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func getPrimaryKeys(db *sql.DB, schemaName, tableName string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND kcu.table_schema = $1
			AND kcu.table_name = $2
		ORDER BY kcu.ordinal_position
	`

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	return primaryKeys, rows.Err()
}

func getIndexes(db *sql.DB, schemaName, tableName string) ([]schema.Index, error) {
	query := `
		SELECT
			i.indexname,
			array_agg(a.attname ORDER BY array_position(idx.indkey::int[], a.attnum)) as columns,
			i.indexdef LIKE '%UNIQUE%' as is_unique
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.tablename
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_class ic ON ic.relname = i.indexname
		JOIN pg_index idx ON idx.indexrelid = ic.oid AND idx.indrelid = c.oid
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(idx.indkey)
		WHERE n.nspname = $1 AND i.tablename = $2
			AND NOT idx.indisprimary
		GROUP BY i.indexname, i.indexdef
		ORDER BY i.indexname
	`

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []schema.Index
	for rows.Next() {
		var index schema.Index
		var columnsArray string
		var isUnique bool

		err := rows.Scan(&index.Name, &columnsArray, &isUnique)
		if err != nil {
			return nil, err
		}

		columnsArray = strings.Trim(columnsArray, "{}")
		index.Columns = strings.Split(columnsArray, ",")
		index.Unique = isUnique

		indexes = append(indexes, index)
	}

	return indexes, rows.Err()
}

func getForeignKeys(db *sql.DB, schemaName, tableName string) ([]schema.Reference, error) {
	query := `
		SELECT DISTINCT
			kcu1.column_name,
			kcu2.table_schema AS foreign_table_schema,
			kcu2.table_name AS foreign_table_name,
			kcu2.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule,
			kcu1.ordinal_position
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu1
			ON kcu1.constraint_name = rc.constraint_name
			AND kcu1.table_schema = rc.constraint_schema
		JOIN information_schema.key_column_usage kcu2
			ON kcu2.constraint_name = rc.unique_constraint_name
			AND kcu2.table_schema = rc.unique_constraint_schema
			AND kcu2.ordinal_position = kcu1.ordinal_position
		WHERE kcu1.table_schema = $1 AND kcu1.table_name = $2
		ORDER BY kcu1.ordinal_position
	`

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	referenceMap := make(map[string]schema.Reference)
	for rows.Next() {
		var ref schema.Reference
		var fromColumn, toColumn string
		var ordinalPosition int

		err := rows.Scan(
			&fromColumn,
			&ref.ToSchema,
			&ref.ToTable,
			&toColumn,
			&ref.OnDelete,
			&ref.OnUpdate,
			&ordinalPosition,
		)
		if err != nil {
			return nil, err
		}

		ref.FromTable = tableName
		ref.FromSchema = schemaName
		ref.FromColumns = []string{fromColumn}
		ref.ToColumns = []string{toColumn}

		key := fmt.Sprintf("%s.%s.%s->%s.%s.%s",
			schemaName, tableName, fromColumn,
			ref.ToSchema, ref.ToTable, toColumn)

		if existing, exists := referenceMap[key]; exists {
			if ref.OnDelete != "NO ACTION" && ref.OnDelete != "" && existing.OnDelete == "NO ACTION" {
				existing.OnDelete = ref.OnDelete
			}
			if ref.OnUpdate != "NO ACTION" && ref.OnUpdate != "" && existing.OnUpdate == "NO ACTION" {
				existing.OnUpdate = ref.OnUpdate
			}
			referenceMap[key] = existing
		} else {
			referenceMap[key] = ref
		}
	}

	var keys []string
	for key := range referenceMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var references []schema.Reference
	for _, key := range keys {
		references = append(references, referenceMap[key])
	}

	return references, rows.Err()
}
