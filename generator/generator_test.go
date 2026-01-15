package generator

import (
	"strings"
	"testing"

	"github.com/lucasefe/dbml/schema"
)

func TestGenerate(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:   "users",
				Schema: "public",
				Columns: []schema.Column{
					{Name: "id", Type: "int", Nullable: false, IsPrimaryKey: true},
					{Name: "email", Type: "varchar(255)", Nullable: false},
					{Name: "name", Type: "varchar(100)", Nullable: true},
				},
				PrimaryKeys: []string{"id"},
				Indexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
				},
			},
		},
	}

	result, err := Generate(s)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	dbml := string(result)

	expectedContains := []string{
		"Table users {",
		"id int [pk]",
		"email varchar(255) [not null]",
		"name varchar(100)",
		"indexes {",
		"(email) [unique]",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(dbml, expected) {
			t.Errorf("Generated DBML does not contain expected string: %s", expected)
		}
	}
}

func TestGenerateWithReferences(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:        "users",
				Schema:      "public",
				Columns:     []schema.Column{{Name: "id", Type: "int", IsPrimaryKey: true}},
				PrimaryKeys: []string{"id"},
			},
			{
				Name:   "posts",
				Schema: "public",
				Columns: []schema.Column{
					{Name: "id", Type: "int", IsPrimaryKey: true},
					{Name: "user_id", Type: "int", Nullable: false},
				},
				PrimaryKeys: []string{"id"},
				References: []schema.Reference{
					{
						FromTable:   "posts",
						FromSchema:  "public",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToSchema:    "public",
						ToColumns:   []string{"id"},
						OnDelete:    "CASCADE",
						OnUpdate:    "NO ACTION",
					},
				},
			},
		},
	}

	result, err := Generate(s)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	dbml := string(result)

	if !strings.Contains(dbml, "Ref: posts.user_id > users.id [delete: cascade]") {
		t.Errorf("Generated DBML missing expected reference: %s", dbml)
	}
}

func TestGenerateString(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:        "users",
				Schema:      "public",
				Columns:     []schema.Column{{Name: "id", Type: "int", IsPrimaryKey: true}},
				PrimaryKeys: []string{"id"},
			},
		},
	}

	result, err := GenerateString(s)
	if err != nil {
		t.Fatalf("GenerateString returned error: %v", err)
	}

	if !strings.Contains(result, "Table users {") {
		t.Errorf("GenerateString output missing expected content")
	}
}

func TestGetQualifiedTableName(t *testing.T) {
	tests := []struct {
		tableName  string
		schemaName string
		expected   string
	}{
		{"users", "public", "users"},
		{"users", "", "users"},
		{"users", "auth", "auth.users"},
		{"accounts", "billing", "billing.accounts"},
	}

	for _, tt := range tests {
		result := GetQualifiedTableName(tt.tableName, tt.schemaName)
		if result != tt.expected {
			t.Errorf("GetQualifiedTableName(%s, %s) = %s, want %s",
				tt.tableName, tt.schemaName, result, tt.expected)
		}
	}
}

func TestGenerateWithNonPublicSchema(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:        "users",
				Schema:      "auth",
				Columns:     []schema.Column{{Name: "id", Type: "int", IsPrimaryKey: true}},
				PrimaryKeys: []string{"id"},
			},
		},
	}

	result, err := Generate(s)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	dbml := string(result)

	if !strings.Contains(dbml, "Table auth.users {") {
		t.Errorf("Generated DBML should include schema prefix for non-public schema: %s", dbml)
	}
}

func TestGenerateWithDefaultValue(t *testing.T) {
	defaultVal := "now()"
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:   "users",
				Schema: "public",
				Columns: []schema.Column{
					{Name: "id", Type: "int", IsPrimaryKey: true},
					{Name: "created_at", Type: "timestamp", Nullable: false, DefaultValue: &defaultVal},
				},
				PrimaryKeys: []string{"id"},
			},
		},
	}

	result, err := Generate(s)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	dbml := string(result)

	if !strings.Contains(dbml, "default: `now()`") {
		t.Errorf("Generated DBML missing default value: %s", dbml)
	}
}

func TestGenerateWithAutoIncrement(t *testing.T) {
	defaultVal := "nextval('users_id_seq')"
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:   "users",
				Schema: "public",
				Columns: []schema.Column{
					{Name: "id", Type: "int", IsPrimaryKey: true, DefaultValue: &defaultVal},
				},
				PrimaryKeys: []string{"id"},
			},
		},
	}

	result, err := Generate(s)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	dbml := string(result)

	if !strings.Contains(dbml, "increment") {
		t.Errorf("Generated DBML missing increment attribute: %s", dbml)
	}
}
