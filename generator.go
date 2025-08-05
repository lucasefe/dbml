package dbml

import (
	"fmt"
	"sort"
	"strings"
)

func GenerateDBML(schema *Schema) string {
	var builder strings.Builder

	// Sort tables by schema.name for consistent output
	sortedTables := make([]Table, len(schema.Tables))
	copy(sortedTables, schema.Tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		if sortedTables[i].Schema != sortedTables[j].Schema {
			return sortedTables[i].Schema < sortedTables[j].Schema
		}
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		generateTable(&builder, table)
		builder.WriteString("\n")
	}

	// Collect and sort all references
	var allReferences []Reference
	for _, table := range sortedTables {
		for _, ref := range table.References {
			allReferences = append(allReferences, ref)
		}
	}

	// Sort references for consistent output
	sort.Slice(allReferences, func(i, j int) bool {
		refI := allReferences[i]
		refJ := allReferences[j] 
		
		// Sort by from table schema.name first
		fromTableI := getQualifiedTableName(refI.FromTable, refI.FromSchema)
		fromTableJ := getQualifiedTableName(refJ.FromTable, refJ.FromSchema)
		if fromTableI != fromTableJ {
			return fromTableI < fromTableJ
		}
		
		// Then by from column
		if len(refI.FromColumns) > 0 && len(refJ.FromColumns) > 0 {
			if refI.FromColumns[0] != refJ.FromColumns[0] {
				return refI.FromColumns[0] < refJ.FromColumns[0]
			}
		}
		
		// Then by to table schema.name
		toTableI := getQualifiedTableName(refI.ToTable, refI.ToSchema)
		toTableJ := getQualifiedTableName(refJ.ToTable, refJ.ToSchema)
		if toTableI != toTableJ {
			return toTableI < toTableJ
		}
		
		// Finally by to column
		if len(refI.ToColumns) > 0 && len(refJ.ToColumns) > 0 {
			return refI.ToColumns[0] < refJ.ToColumns[0]
		}
		
		return false
	})

	// Generate sorted references
	for _, ref := range allReferences {
		generateReference(&builder, ref)
	}

	return builder.String()
}

func generateTable(builder *strings.Builder, table Table) {
	tableName := table.Name
	if table.Schema != "" && table.Schema != "public" {
		tableName = fmt.Sprintf("%s.%s", table.Schema, table.Name)
	}
	builder.WriteString(fmt.Sprintf("Table %s {\n", tableName))

	// Sort columns by name for consistent output
	sortedColumns := make([]Column, len(table.Columns))
	copy(sortedColumns, table.Columns)
	sort.Slice(sortedColumns, func(i, j int) bool {
		return sortedColumns[i].Name < sortedColumns[j].Name
	})

	for _, column := range sortedColumns {
		generateColumn(builder, column)
	}

	if len(table.Indexes) > 0 {
		builder.WriteString("\n")
		// Sort indexes by name for consistent output
		sortedIndexes := make([]Index, len(table.Indexes))
		copy(sortedIndexes, table.Indexes)
		sort.Slice(sortedIndexes, func(i, j int) bool {
			return sortedIndexes[i].Name < sortedIndexes[j].Name
		})
		generateIndexes(builder, sortedIndexes)
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