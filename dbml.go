package dbml

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type Config struct {
	Schema       string
	ExcludeTables []string
}

func GenerateFromConnection(db *sql.DB, config *Config) (string, error) {
	if config == nil {
		config = &Config{Schema: "public"}
	}
	if config.Schema == "" {
		config.Schema = "public"
	}

	schema, err := IntrospectDatabase(db, config.Schema)
	if err != nil {
		return "", fmt.Errorf("failed to introspect database: %w", err)
	}

	if len(config.ExcludeTables) > 0 {
		schema = filterTables(schema, config.ExcludeTables)
	}

	return GenerateDBML(schema), nil
}

func GenerateFromConnectionString(connStr string, config *Config) (string, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return "", fmt.Errorf("failed to ping database: %w", err)
	}

	return GenerateFromConnection(db, config)
}

func WriteToFile(db *sql.DB, filename string, config *Config) error {
	dbmlContent, err := GenerateFromConnection(db, config)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(dbmlContent), 0644)
}

func WriteToFileFromConnectionString(connStr, filename string, config *Config) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return WriteToFile(db, filename, config)
}

func filterTables(schema *Schema, excludeTables []string) *Schema {
	excludeMap := make(map[string]bool)
	for _, table := range excludeTables {
		excludeMap[table] = true
	}

	filteredTables := make([]Table, 0)
	for _, table := range schema.Tables {
		if !excludeMap[table.Name] {
			filteredTables = append(filteredTables, table)
		}
	}

	return &Schema{Tables: filteredTables}
}