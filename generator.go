package dbml

import (
	"github.com/lucasefe/dbml/generator"
	"github.com/lucasefe/dbml/schema"
)

// GenerateDBML converts a Schema into DBML-formatted text.
func GenerateDBML(s *Schema) string {
	result, _ := generator.GenerateString(s)
	return result
}

// GenerateDBMLBytes converts a Schema into DBML-formatted bytes.
func GenerateDBMLBytes(s *Schema) []byte {
	result, _ := generator.Generate(s)
	return result
}

func GetQualifiedTableName(tableName, schemaName string) string {
	return generator.GetQualifiedTableName(tableName, schemaName)
}

func FilterTables(s *Schema, excludeTables []string) *Schema {
	return schema.FilterTables(s, excludeTables)
}
