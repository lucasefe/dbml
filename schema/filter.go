package schema

// FilterTables removes tables from the schema that match the exclude list.
// It returns a new Schema with the filtered tables; the original is not modified.
func FilterTables(s *Schema, excludeTables []string) *Schema {
	excludeMap := make(map[string]bool)
	for _, table := range excludeTables {
		excludeMap[table] = true
	}

	filteredTables := make([]Table, 0)
	for _, table := range s.Tables {
		if !excludeMap[table.Name] {
			filteredTables = append(filteredTables, table)
		}
	}

	return &Schema{Tables: filteredTables}
}
