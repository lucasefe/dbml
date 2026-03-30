package dbml

import (
	"database/sql"

	"github.com/lucasefe/dbml/introspect"
	"github.com/lucasefe/dbml/schema"
)

// Type aliases — the canonical definitions live in the schema and introspect subpackages.
type Schema = schema.Schema
type Table = schema.Table
type Column = schema.Column
type Index = schema.Index
type Reference = schema.Reference
type Enum = schema.Enum
type TypeMapper = introspect.TypeMapper
type PostgreSQLTypeMapper = introspect.PostgreSQLTypeMapper

var DefaultTypeMappings = introspect.DefaultTypeMappings

func NewPostgreSQLTypeMapper(customMappings map[string]string) *PostgreSQLTypeMapper {
	return introspect.NewPostgreSQLTypeMapper(customMappings)
}

func MapPostgreSQLTypeToDBML(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	return introspect.MapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
}

func IntrospectDatabase(db *sql.DB, schemaNames []string) (*Schema, error) {
	return introspect.Database(db, introspect.WithSchemas(schemaNames...))
}

func IntrospectAllSchemas(db *sql.DB) (*Schema, error) {
	return introspect.Database(db, introspect.WithAllSchemas())
}

func IntrospectDatabaseWithMapper(db *sql.DB, schemaNames []string, mapper TypeMapper) (*Schema, error) {
	opts := []introspect.Option{introspect.WithSchemas(schemaNames...)}
	if mapper != nil {
		opts = append(opts, introspect.WithTypeMapper(mapper))
	}
	return introspect.Database(db, opts...)
}

func IntrospectAllSchemasWithMapper(db *sql.DB, mapper TypeMapper) (*Schema, error) {
	opts := []introspect.Option{introspect.WithAllSchemas()}
	if mapper != nil {
		opts = append(opts, introspect.WithTypeMapper(mapper))
	}
	return introspect.Database(db, opts...)
}

func NormalizeCustomType(typeName string) string {
	return introspect.NormalizeCustomType(typeName)
}

func NormalizeTypeName(typeName string) string {
	return introspect.NormalizeTypeName(typeName)
}
