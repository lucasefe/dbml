package introspect

import (
	"database/sql"
	"testing"
)

func TestMapPostgreSQLTypeToDBML(t *testing.T) {
	tests := []struct {
		name             string
		dataType         string
		udtName          string
		charMaxLength    sql.NullInt64
		numericPrecision sql.NullInt64
		numericScale     sql.NullInt64
		expected         string
	}{
		{"integer", "integer", "int4", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "int"},
		{"bigint", "bigint", "int8", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "bigint"},
		{"smallint", "smallint", "int2", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "smallint"},
		{"boolean", "boolean", "bool", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "boolean"},
		{"varchar with length", "character varying", "varchar", sql.NullInt64{Valid: true, Int64: 255}, sql.NullInt64{}, sql.NullInt64{}, "varchar(255)"},
		{"varchar without length", "character varying", "varchar", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "varchar"},
		{"char with length", "character", "char", sql.NullInt64{Valid: true, Int64: 10}, sql.NullInt64{}, sql.NullInt64{}, "char(10)"},
		{"text", "text", "text", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "text"},
		{"decimal with precision", "numeric", "numeric", sql.NullInt64{}, sql.NullInt64{Valid: true, Int64: 10}, sql.NullInt64{Valid: true, Int64: 2}, "decimal(10,2)"},
		{"decimal without precision", "numeric", "numeric", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "decimal"},
		{"real", "real", "float4", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "float"},
		{"double", "double precision", "float8", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "double"},
		{"timestamp", "timestamp without time zone", "timestamp", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "timestamp"},
		{"timestamptz", "timestamp with time zone", "timestamptz", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "timestamptz"},
		{"date", "date", "date", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "date"},
		{"time", "time without time zone", "time", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "time"},
		{"uuid", "uuid", "uuid", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "uuid"},
		{"json", "json", "json", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "json"},
		{"jsonb", "jsonb", "jsonb", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "jsonb"},
		{"bytea", "bytea", "bytea", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "binary"},
		{"user-defined", "user-defined", "custom_enum", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "text"},
		{"array type", "array", "_int4", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "text"},
		{"unknown type", "custom_type", "custom_type", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "custom_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapPostgreSQLTypeToDBML(tt.dataType, tt.udtName, tt.charMaxLength, tt.numericPrecision, tt.numericScale)
			if result != tt.expected {
				t.Errorf("MapPostgreSQLTypeToDBML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPostgreSQLTypeMapper(t *testing.T) {
	customMappings := map[string]string{
		"citext": "varchar",
		"ltree":  "text",
	}

	mapper := NewPostgreSQLTypeMapper(customMappings)

	t.Run("custom mapping by dataType", func(t *testing.T) {
		result := mapper.MapType("citext", "citext", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result != "varchar" {
			t.Errorf("Expected 'varchar' for citext, got '%s'", result)
		}
	})

	t.Run("custom mapping by udtName", func(t *testing.T) {
		result := mapper.MapType("user-defined", "ltree", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result != "text" {
			t.Errorf("Expected 'text' for ltree, got '%s'", result)
		}
	})

	t.Run("fallback to default mapping", func(t *testing.T) {
		result := mapper.MapType("integer", "int4", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result != "int" {
			t.Errorf("Expected 'int' for integer, got '%s'", result)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		result := mapper.MapType("CITEXT", "citext", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result != "varchar" {
			t.Errorf("Expected 'varchar' for CITEXT (case insensitive), got '%s'", result)
		}
	})
}

func TestPostgreSQLTypeMapperNilMappings(t *testing.T) {
	mapper := NewPostgreSQLTypeMapper(nil)

	result := mapper.MapType("integer", "int4", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
	if result != "int" {
		t.Errorf("Expected 'int' for integer with nil mappings, got '%s'", result)
	}
}

func TestNormalizeCustomType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected string
	}{
		{"array type with underscore", "_int4", "text"},
		{"regular custom type", "address", "text"},
		{"unknown custom type", "some_custom_type", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeCustomType(tt.typeName)
			if result != tt.expected {
				t.Errorf("NormalizeCustomType(%s) = %s, want %s", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestNormalizeTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected string
	}{
		{"known custom type address", "address", "text"},
		{"known custom type status", "status", "text"},
		{"unknown type", "random_type", "text"},
		{"case insensitive", "ADDRESS", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTypeName(tt.typeName)
			if result != tt.expected {
				t.Errorf("NormalizeTypeName(%s) = %s, want %s", tt.typeName, result, tt.expected)
			}
		})
	}
}
