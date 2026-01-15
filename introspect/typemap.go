package introspect

import (
	"database/sql"
	"fmt"
	"strings"
)

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
//	mapper := introspect.NewPostgreSQLTypeMapper(map[string]string{
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
	return MapPostgreSQLTypeToDBML(dataType, udtName, charMaxLength, numericPrecision, numericScale)
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
func MapPostgreSQLTypeToDBML(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
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
		return NormalizeCustomType(udtName)
	case "array":
		return NormalizeCustomType(udtName)
	default:
		return dataType
	}
}

// NormalizeCustomType converts PostgreSQL custom types (including array types)
// to DBML-compatible type names.
func NormalizeCustomType(typeName string) string {
	if strings.HasPrefix(typeName, "_") {
		baseType := strings.TrimPrefix(typeName, "_")
		return NormalizeTypeName(baseType)
	}
	return NormalizeTypeName(typeName)
}

// NormalizeTypeName converts a type name to a valid DBML identifier.
// Unknown types default to "text" for DBML compatibility.
func NormalizeTypeName(typeName string) string {
	switch strings.ToLower(typeName) {
	case "address", "contact_method", "offering", "provider", "carrier", "direction", "status", "business_type", "industry", "cta":
		return "text"
	default:
		return "text"
	}
}
