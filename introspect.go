package dbml

import (
	"database/sql"
	"fmt"
	"strings"
)

type Schema struct {
	Tables []Table
}

type Table struct {
	Name        string
	Schema      string
	Columns     []Column
	PrimaryKeys []string
	Indexes     []Index
	References  []Reference
}

type Column struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	IsPrimaryKey bool
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

type Reference struct {
	FromTable   string
	FromColumns []string
	ToTable     string
	ToColumns   []string
	OnDelete    string
	OnUpdate    string
}

func IntrospectDatabase(db *sql.DB, schemaName string) (*Schema, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	schema := &Schema{}

	tables, err := getTables(db, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	for _, table := range tables {
		columns, err := getColumns(db, schemaName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", table.Name, err)
		}
		table.Columns = columns

		primaryKeys, err := getPrimaryKeys(db, schemaName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get primary keys for table %s: %w", table.Name, err)
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
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", table.Name, err)
		}
		table.Indexes = indexes

		references, err := getForeignKeys(db, schemaName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get foreign keys for table %s: %w", table.Name, err)
		}
		table.References = references

		schema.Tables = append(schema.Tables, table)
	}

	return schema, nil
}

func getTables(db *sql.DB, schemaName string) ([]Table, error) {
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

	var tables []Table
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, Table{
			Name:   tableName,
			Schema: schemaName,
		})
	}

	return tables, rows.Err()
}

func getColumns(db *sql.DB, schemaName, tableName string) ([]Column, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			character_maximum_length,
			numeric_precision,
			numeric_scale,
			is_nullable,
			column_default
		FROM information_schema.columns 
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var dataType string
		var charMaxLength, numericPrecision, numericScale sql.NullInt64
		var isNullable string
		var columnDefault sql.NullString

		err := rows.Scan(
			&col.Name,
			&dataType,
			&charMaxLength,
			&numericPrecision,
			&numericScale,
			&isNullable,
			&columnDefault,
		)
		if err != nil {
			return nil, err
		}

		col.Type = mapPostgreSQLTypeToDBML(dataType, charMaxLength, numericPrecision, numericScale)
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

func getIndexes(db *sql.DB, schemaName, tableName string) ([]Index, error) {
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

	var indexes []Index
	for rows.Next() {
		var index Index
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

func getForeignKeys(db *sql.DB, schemaName, tableName string) ([]Reference, error) {
	query := `
		SELECT 
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
			ON kcu.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage ccu
			ON rc.unique_constraint_name = ccu.constraint_name
		WHERE kcu.table_schema = $1 AND kcu.table_name = $2
		ORDER BY kcu.ordinal_position
	`

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var references []Reference
	for rows.Next() {
		var ref Reference
		var fromColumn, toColumn string

		err := rows.Scan(
			&fromColumn,
			&ref.ToTable,
			&toColumn,
			&ref.OnDelete,
			&ref.OnUpdate,
		)
		if err != nil {
			return nil, err
		}

		ref.FromTable = tableName
		ref.FromColumns = []string{fromColumn}
		ref.ToColumns = []string{toColumn}

		references = append(references, ref)
	}

	return references, rows.Err()
}

func mapPostgreSQLTypeToDBML(dataType string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	switch strings.ToLower(dataType) {
	case "integer", "int4":
		return "int"
	case "bigint", "int8":
		return "bigint"
	case "smallint", "int2":
		return "smallint"
	case "boolean", "bool":
		return "boolean"
	case "character varying", "varchar":
		if charMaxLength.Valid {
			return fmt.Sprintf("varchar(%d)", charMaxLength.Int64)
		}
		return "varchar"
	case "character", "char":
		if charMaxLength.Valid {
			return fmt.Sprintf("char(%d)", charMaxLength.Int64)
		}
		return "char"
	case "text":
		return "text"
	case "numeric", "decimal":
		if numericPrecision.Valid && numericScale.Valid {
			return fmt.Sprintf("decimal(%d,%d)", numericPrecision.Int64, numericScale.Int64)
		}
		return "decimal"
	case "real", "float4":
		return "float"
	case "double precision", "float8":
		return "double"
	case "timestamp without time zone", "timestamp":
		return "timestamp"
	case "timestamp with time zone", "timestamptz":
		return "timestamptz"
	case "date":
		return "date"
	case "time without time zone", "time":
		return "time"
	case "time with time zone", "timetz":
		return "timetz"
	case "uuid":
		return "uuid"
	case "json":
		return "json"
	case "jsonb":
		return "jsonb"
	case "bytea":
		return "binary"
	default:
		return dataType
	}
}