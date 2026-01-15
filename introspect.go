package dbml

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Schema represents a database schema containing multiple tables.
// It is the top-level container returned by introspection functions.
type Schema struct {
	// Tables contains all tables found in the introspected schema(s).
	Tables []Table
}

// Table represents a database table with its columns, primary keys,
// indexes, and foreign key references.
type Table struct {
	// Name is the table name without schema qualification.
	Name string
	// Schema is the database schema containing this table (e.g., "public").
	Schema string
	// Columns contains all columns in the table, ordered by ordinal position.
	Columns []Column
	// PrimaryKeys lists column names that form the primary key.
	PrimaryKeys []string
	// Indexes contains non-primary-key indexes on the table.
	Indexes []Index
	// References contains foreign key relationships from this table to other tables.
	References []Reference
}

// Column represents a database column within a table.
type Column struct {
	// Name is the column name.
	Name string
	// Type is the DBML-compatible type (e.g., "int", "varchar(255)", "timestamp").
	Type string
	// Nullable indicates whether the column allows NULL values.
	Nullable bool
	// DefaultValue is the column's default value expression, or nil if none.
	DefaultValue *string
	// IsPrimaryKey indicates whether this column is part of the primary key.
	IsPrimaryKey bool
}

// Index represents a database index on one or more columns.
type Index struct {
	// Name is the index name.
	Name string
	// Columns lists the column names included in the index.
	Columns []string
	// Unique indicates whether this is a unique index.
	Unique bool
}

// Reference represents a foreign key relationship between tables.
type Reference struct {
	// FromTable is the table containing the foreign key.
	FromTable string
	// FromSchema is the schema of the table containing the foreign key.
	FromSchema string
	// FromColumns lists the column names in the foreign key.
	FromColumns []string
	// ToTable is the referenced table.
	ToTable string
	// ToSchema is the schema of the referenced table.
	ToSchema string
	// ToColumns lists the referenced column names.
	ToColumns []string
	// OnDelete is the referential action on delete (e.g., "CASCADE", "SET NULL").
	OnDelete string
	// OnUpdate is the referential action on update.
	OnUpdate string
}

// TypeMapper defines the interface for converting database types to DBML types.
// Implement this interface to customize type mapping behavior.
type TypeMapper interface {
	// MapType converts a database column type to a DBML type string.
	// dataType is the base data type (e.g., "integer", "varchar")
	// udtName is the user-defined type name for custom types
	// charMaxLength, numericPrecision, numericScale provide type modifiers
	MapType(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string
}

// PostgreSQLTypeMapper provides PostgreSQL to DBML type conversion.
// It supports custom type overrides via the CustomMappings field.
type PostgreSQLTypeMapper struct {
	// CustomMappings allows overriding default type mappings.
	// Keys are PostgreSQL type names (case-insensitive), values are DBML types.
	CustomMappings map[string]string
}

// NewPostgreSQLTypeMapper creates a new TypeMapper with optional custom mappings.
// If customMappings is nil, only default mappings are used.
//
// Example:
//
//	mapper := dbml.NewPostgreSQLTypeMapper(map[string]string{
//	    "citext": "varchar",
//	    "ltree":  "text",
//	})
func NewPostgreSQLTypeMapper(customMappings map[string]string) *PostgreSQLTypeMapper {
	return &PostgreSQLTypeMapper{CustomMappings: customMappings}
}

// MapType implements TypeMapper for PostgreSQL databases.
// It checks CustomMappings first, then falls back to default mappings.
func (m *PostgreSQLTypeMapper) MapType(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	// Check custom mappings first (case-insensitive)
	if m.CustomMappings != nil {
		if mapped, ok := m.CustomMappings[strings.ToLower(dataType)]; ok {
			return mapped
		}
		// Also check the UDT name for custom types
		if mapped, ok := m.CustomMappings[strings.ToLower(udtName)]; ok {
			return mapped
		}
	}
	// Fall back to default implementation
	return mapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
}

// IntrospectDatabase queries a PostgreSQL database and returns its schema structure.
// It extracts tables, columns, primary keys, indexes, and foreign key references
// from the specified schema names. If schemaNames is empty, it defaults to ["public"].
func IntrospectDatabase(db *sql.DB, schemaNames []string) (*Schema, error) {
	return IntrospectDatabaseWithMapper(db, schemaNames, nil)
}

// IntrospectDatabaseWithMapper queries a PostgreSQL database and returns its schema structure
// using a custom type mapper. If mapper is nil, the default PostgreSQL type mappings are used.
func IntrospectDatabaseWithMapper(db *sql.DB, schemaNames []string, mapper TypeMapper) (*Schema, error) {
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
			columns, err := getColumnsWithMapper(db, schemaName, table.Name, mapper)
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

// IntrospectAllSchemas queries a PostgreSQL database and returns schemas for all
// non-system schemas. It excludes information_schema, pg_catalog, pg_toast, and
// other PostgreSQL internal schemas.
func IntrospectAllSchemas(db *sql.DB) (*Schema, error) {
	return IntrospectAllSchemasWithMapper(db, nil)
}

// IntrospectAllSchemasWithMapper queries a PostgreSQL database and returns schemas
// for all non-system schemas using a custom type mapper.
// If mapper is nil, the default PostgreSQL type mappings are used.
func IntrospectAllSchemasWithMapper(db *sql.DB, mapper TypeMapper) (*Schema, error) {
	schemas, err := getAllSchemas(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	return IntrospectDatabaseWithMapper(db, schemas, mapper)
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
	return getColumnsWithMapper(db, schemaName, tableName, nil)
}

func getColumnsWithMapper(db *sql.DB, schemaName, tableName string, mapper TypeMapper) ([]Column, error) {
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

		// Use custom mapper if provided, otherwise use default
		if mapper != nil {
			col.Type = mapper.MapType(dataType, udtName, charMaxLength, numericPrecision, numericScale)
		} else {
			col.Type = mapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
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

	// Convert map back to slice and sort by key for deterministic output
	var keys []string
	for key := range referenceMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	var references []Reference
	for _, key := range keys {
		references = append(references, referenceMap[key])
	}

	return references, rows.Err()
}

// DefaultTypeMappings contains the standard PostgreSQL to DBML type mappings.
// This can be used as a reference when creating custom type mappers.
var DefaultTypeMappings = map[string]string{
	"integer":                     "int",
	"int4":                        "int",
	"bigint":                      "bigint",
	"int8":                        "bigint",
	"smallint":                    "smallint",
	"int2":                        "smallint",
	"boolean":                     "boolean",
	"bool":                        "boolean",
	"text":                        "text",
	"character varying":           "varchar",
	"varchar":                     "varchar",
	"character":                   "char",
	"char":                        "char",
	"numeric":                     "decimal",
	"decimal":                     "decimal",
	"real":                        "float",
	"float4":                      "float",
	"double precision":            "double",
	"float8":                      "double",
	"timestamp without time zone": "timestamp",
	"timestamp":                   "timestamp",
	"timestamp with time zone":    "timestamptz",
	"timestamptz":                 "timestamptz",
	"date":                        "date",
	"time without time zone":      "time",
	"time":                        "time",
	"time with time zone":         "timetz",
	"timetz":                      "timetz",
	"uuid":                        "uuid",
	"json":                        "json",
	"jsonb":                       "jsonb",
	"bytea":                       "binary",
}

// MapPostgreSQLTypeToDBML converts a PostgreSQL data type to its DBML equivalent.
// It handles varchar lengths, numeric precision/scale, and custom types.
// For types with length/precision modifiers (varchar, char, decimal), include
// the appropriate NullInt64 values to get properly formatted output.
func MapPostgreSQLTypeToDBML(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	return mapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
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
		return NormalizeCustomType(udtName)
	case "array":
		// For arrays, use the base type with array notation
		return NormalizeCustomType(udtName)
	default:
		return dataType
	}
}

// NormalizeCustomType converts PostgreSQL custom types (including array types)
// to DBML-compatible type names. Array types in PostgreSQL start with underscore
// (e.g., "_int4" for integer[]).
func NormalizeCustomType(typeName string) string {
	// Remove common PostgreSQL array suffixes and make type names DBML-compatible
	if strings.HasPrefix(typeName, "_") {
		// Array types in PostgreSQL start with underscore
		baseType := strings.TrimPrefix(typeName, "_")
		return NormalizeTypeName(baseType)
	}

	return NormalizeTypeName(typeName)
}

// normalizeCustomType is an alias for backward compatibility.
func normalizeCustomType(typeName string) string {
	return NormalizeCustomType(typeName)
}

// NormalizeTypeName converts a type name to a valid DBML identifier.
// Unknown types default to "text" for DBML compatibility.
func NormalizeTypeName(typeName string) string {
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

// normalizeTypeName is an alias for backward compatibility.
func normalizeTypeName(typeName string) string {
	return NormalizeTypeName(typeName)
}