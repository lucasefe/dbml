package schema

import "testing"

func TestFilterTables(t *testing.T) {
	s := &Schema{
		Tables: []Table{
			{Name: "users", Schema: "public"},
			{Name: "posts", Schema: "public"},
			{Name: "migrations", Schema: "public"},
			{Name: "schema_versions", Schema: "public"},
		},
	}

	excludeTables := []string{"migrations", "schema_versions"}
	filtered := FilterTables(s, excludeTables)

	if len(filtered.Tables) != 2 {
		t.Errorf("Expected 2 tables after filtering, got %d", len(filtered.Tables))
	}

	expectedTables := map[string]bool{"users": true, "posts": true}
	for _, table := range filtered.Tables {
		if !expectedTables[table.Name] {
			t.Errorf("Unexpected table in filtered result: %s", table.Name)
		}
	}
}

func TestFilterTablesEmpty(t *testing.T) {
	s := &Schema{
		Tables: []Table{
			{Name: "users", Schema: "public"},
		},
	}

	// Filter with empty exclude list
	filtered := FilterTables(s, []string{})

	if len(filtered.Tables) != 1 {
		t.Errorf("Expected 1 table when exclude list is empty, got %d", len(filtered.Tables))
	}
}

func TestFilterTablesOriginalUnmodified(t *testing.T) {
	s := &Schema{
		Tables: []Table{
			{Name: "users", Schema: "public"},
			{Name: "migrations", Schema: "public"},
		},
	}

	FilterTables(s, []string{"migrations"})

	// Original should still have 2 tables
	if len(s.Tables) != 2 {
		t.Errorf("Original schema was modified, expected 2 tables, got %d", len(s.Tables))
	}
}
