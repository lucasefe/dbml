package dbml

import (
	"fmt"
	"strings"
)

func GenerateDBML(schema *Schema) string {
	var builder strings.Builder

	for _, table := range schema.Tables {
		generateTable(&builder, table)
		builder.WriteString("\n")
	}

	for _, table := range schema.Tables {
		generateReferences(&builder, table)
	}

	return builder.String()
}

func generateTable(builder *strings.Builder, table Table) {
	tableName := table.Name
	if table.Schema != "" && table.Schema != "public" {
		tableName = fmt.Sprintf("%s.%s", table.Schema, table.Name)
	}
	builder.WriteString(fmt.Sprintf("Table %s {\n", tableName))

	for _, column := range table.Columns {
		generateColumn(builder, column)
	}

	if len(table.Indexes) > 0 {
		builder.WriteString("\n")
		generateIndexes(builder, table.Indexes)
	}

	builder.WriteString("}\n")
}

func generateColumn(builder *strings.Builder, column Column) {
	builder.WriteString(fmt.Sprintf("  %s %s", column.Name, column.Type))

	var attributes []string

	if column.IsPrimaryKey {
		attributes = append(attributes, "pk")
	}

	if !column.Nullable && !column.IsPrimaryKey {
		attributes = append(attributes, "not null")
	}

	if column.DefaultValue != nil {
		defaultVal := *column.DefaultValue
		if strings.HasPrefix(defaultVal, "nextval(") {
			attributes = append(attributes, "increment")
		} else {
			attributes = append(attributes, fmt.Sprintf("default: `%s`", defaultVal))
		}
	}

	if len(attributes) > 0 {
		builder.WriteString(fmt.Sprintf(" [%s]", strings.Join(attributes, ", ")))
	}

	builder.WriteString("\n")
}

func generateIndexes(builder *strings.Builder, indexes []Index) {
	builder.WriteString("  indexes {\n")
	for _, index := range indexes {
		if index.Unique {
			if len(index.Columns) == 1 {
				builder.WriteString(fmt.Sprintf("    (%s) [unique]\n", index.Columns[0]))
			} else {
				builder.WriteString(fmt.Sprintf("    (%s) [unique]\n", strings.Join(index.Columns, ", ")))
			}
		} else {
			if len(index.Columns) == 1 {
				builder.WriteString(fmt.Sprintf("    %s\n", index.Columns[0]))
			} else {
				builder.WriteString(fmt.Sprintf("    (%s)\n", strings.Join(index.Columns, ", ")))
			}
		}
	}
	builder.WriteString("  }\n")
}

func generateReferences(builder *strings.Builder, table Table) {
	for _, ref := range table.References {
		generateReference(builder, ref)
	}
}

func generateReference(builder *strings.Builder, ref Reference) {
	fromTable := getQualifiedTableName(ref.FromTable, ref.FromSchema)
	toTable := getQualifiedTableName(ref.ToTable, ref.ToSchema)
	
	fromRef := fromTable
	if len(ref.FromColumns) == 1 {
		fromRef = fmt.Sprintf("%s.%s", fromTable, ref.FromColumns[0])
	} else {
		fromRef = fmt.Sprintf("%s.(%s)", fromTable, strings.Join(ref.FromColumns, ", "))
	}

	toRef := toTable
	if len(ref.ToColumns) == 1 {
		toRef = fmt.Sprintf("%s.%s", toTable, ref.ToColumns[0])
	} else {
		toRef = fmt.Sprintf("%s.(%s)", toTable, strings.Join(ref.ToColumns, ", "))
	}

	builder.WriteString(fmt.Sprintf("Ref: %s > %s", fromRef, toRef))

	var refAttributes []string
	if ref.OnDelete != "NO ACTION" && ref.OnDelete != "" {
		refAttributes = append(refAttributes, fmt.Sprintf("delete: %s", strings.ToLower(ref.OnDelete)))
	}
	if ref.OnUpdate != "NO ACTION" && ref.OnUpdate != "" {
		refAttributes = append(refAttributes, fmt.Sprintf("update: %s", strings.ToLower(ref.OnUpdate)))
	}

	if len(refAttributes) > 0 {
		builder.WriteString(fmt.Sprintf(" [%s]", strings.Join(refAttributes, ", ")))
	}

	builder.WriteString("\n")
}

func getQualifiedTableName(tableName, schemaName string) string {
	if schemaName != "" && schemaName != "public" {
		return fmt.Sprintf("%s.%s", schemaName, tableName)
	}
	return tableName
}