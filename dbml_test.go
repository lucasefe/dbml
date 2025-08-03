package dbml

import (
	"database/sql"
	"testing"
)

func TestMapPostgreSQLTypeToDBML(t *testing.T) {
	tests := []struct {
		name             string
		dataType         string
		charMaxLength    sql.NullInt64
		numericPrecision sql.NullInt64
		numericScale     sql.NullInt64
		expected         string
	}{
		{"integer", "integer", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "int"},
		{"bigint", "bigint", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "bigint"},
		{"boolean", "boolean", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "boolean"},
		{"varchar with length", "character varying", sql.NullInt64{Valid: true, Int64: 255}, sql.NullInt64{}, sql.NullInt64{}, "varchar(255)"},
		{"varchar without length", "character varying", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "varchar"},
		{"text", "text", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "text"},
		{"decimal with precision", "numeric", sql.NullInt64{}, sql.NullInt64{Valid: true, Int64: 10}, sql.NullInt64{Valid: true, Int64: 2}, "decimal(10,2)"},
		{"timestamp", "timestamp without time zone", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "timestamp"},
		{"uuid", "uuid", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "uuid"},
		{"unknown type", "custom_type", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, "custom_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapPostgreSQLTypeToDBML(tt.dataType, tt.charMaxLength, tt.numericPrecision, tt.numericScale)
			if result != tt.expected {
				t.Errorf("mapPostgreSQLTypeToDBML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateDBML(t *testing.T) {
	schema := &Schema{
		Tables: []Table{
			{
				Name:   "users",
				Schema: "public",
				Columns: []Column{
					{Name: "id", Type: "int", Nullable: false, IsPrimaryKey: true},
					{Name: "email", Type: "varchar(255)", Nullable: false},
					{Name: "name", Type: "varchar(100)", Nullable: true},
				},
				PrimaryKeys: []string{"id"},
				Indexes: []Index{
					{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
				},
				References: []Reference{},
			},
			{
				Name:   "posts",
				Schema: "public",
				Columns: []Column{
					{Name: "id", Type: "int", Nullable: false, IsPrimaryKey: true},
					{Name: "user_id", Type: "int", Nullable: false},
					{Name: "title", Type: "varchar(200)", Nullable: false},
				},
				PrimaryKeys: []string{"id"},
				Indexes:     []Index{},
				References: []Reference{
					{
						FromTable:   "posts",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToColumns:   []string{"id"},
						OnDelete:    "CASCADE",
						OnUpdate:    "NO ACTION",
					},
				},
			},
		},
	}

	dbml := GenerateDBML(schema)

	expectedContains := []string{
		"Table users {",
		"id int [pk, not null]",
		"email varchar(255) [not null]",
		"name varchar(100)",
		"unique email",
		"Table posts {",
		"user_id int [not null]",
		"Ref: posts.user_id > users.id [delete: cascade]",
	}

	for _, expected := range expectedContains {
		if !containsString(dbml, expected) {
			t.Errorf("Generated DBML does not contain expected string: %s", expected)
		}
	}
}

func TestFilterTables(t *testing.T) {
	schema := &Schema{
		Tables: []Table{
			{Name: "users", Schema: "public"},
			{Name: "posts", Schema: "public"},
			{Name: "migrations", Schema: "public"},
			{Name: "schema_migrations", Schema: "public"},
		},
	}

	excludeTables := []string{"migrations", "schema_migrations"}
	filtered := filterTables(schema, excludeTables)

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

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}