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
	FromSchema  string
	FromColumns []string
	ToTable     string
	ToSchema    string
	ToColumns   []string
	OnDelete    string
	OnUpdate    string
}

func IntrospectDatabase(db *sql.DB, schemaNames []string) (*Schema, error) {
	if len(schemaNames) == 0 {
		schemaNames = []string{"public"}
	}

	schema := &Schema{}

	for _, schemaName := range schemaNames {
		tables, err := getTables(db, schemaName)
		if err != nil {
			return nil, fmt.Errorf("failed to get tables for schema %s: %w", schemaName, err)
		}

		for _, table := range tables {
			columns, err := getColumns(db, schemaName, table.Name)
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

			schema.Tables = append(schema.Tables, table)
		}
	}

	return schema, nil
}

func IntrospectAllSchemas(db *sql.DB) (*Schema, error) {
	schemas, err := getAllSchemas(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}
	
	return IntrospectDatabase(db, schemas)
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

	var columns []Column
	for rows.Next() {
		var col Column
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

		col.Type = mapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
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

	referenceMap := make(map[string]Reference)
	for rows.Next() {
		var ref Reference
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

		// Create a unique key for deduplication
		key := fmt.Sprintf("%s.%s.%s->%s.%s.%s", 
			schemaName, tableName, fromColumn,
			ref.ToSchema, ref.ToTable, toColumn)
		
		// Only keep the first occurrence (or merge if needed)
		if existing, exists := referenceMap[key]; exists {
			// If delete/update rules differ, prefer the more restrictive one
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

	// Convert map back to slice
	var references []Reference
	for _, ref := range referenceMap {
		references = append(references, ref)
	}

	return references, rows.Err()
}

func mapPostgreSQLTypeToDBML(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
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
	case "user-defined":
		// For USER-DEFINED types, use the actual type name but make it DBML-compatible
		return normalizeCustomType(udtName)
	case "array":
		// For arrays, use the base type with array notation
		return normalizeCustomType(udtName)
	default:
		return dataType
	}
}

func normalizeCustomType(typeName string) string {
	// Remove common PostgreSQL array suffixes and make type names DBML-compatible
	if strings.HasPrefix(typeName, "_") {
		// Array types in PostgreSQL start with underscore
		baseType := strings.TrimPrefix(typeName, "_")
		return normalizeTypeName(baseType)
	}
	
	return normalizeTypeName(typeName)
}

func normalizeTypeName(typeName string) string {
	// Convert custom types to valid DBML identifiers
	// For unknown types, default to 'text' to ensure DBML compatibility
	switch strings.ToLower(typeName) {
	case "address", "contact_method", "offering", "provider", "carrier", "direction", "status", "business_type", "industry", "cta":
		// Common custom types can be mapped to text for DBML compatibility
		return "text"
	default:
		// For truly unknown types, use text as fallback
		return "text"
	}
}