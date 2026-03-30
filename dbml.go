package dbml

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Config specifies options for DBML generation.
type Config struct {
	// Schemas lists specific database schemas to include.
	// If empty and IncludeAllSchemas is false, defaults to ["public"].
	Schemas []string
	// ExcludeTables lists table names to exclude from generation.
	ExcludeTables []string
	// IncludeAllSchemas, when true, includes all non-system schemas.
	// This overrides the Schemas field.
	IncludeAllSchemas bool
	// TypeMapper allows customizing how database types are mapped to DBML types.
	// If nil, uses the default PostgreSQL type mapper.
	TypeMapper TypeMapper
	// TypeMappings provides a simple way to override specific type mappings
	// without implementing TypeMapper. These are applied before default mappings.
	// This is a convenience alternative to TypeMapper for simple use cases.
	// If TypeMapper is also set, TypeMapper takes precedence.
	TypeMappings map[string]string
}

// GenerateFromConnection generates DBML from an existing database connection.
// If config is nil, defaults to introspecting only the "public" schema.
// Returns the generated DBML as a string.
func GenerateFromConnection(db *sql.DB, config *Config) (string, error) {
	if config == nil {
		config = &Config{Schemas: []string{"public"}}
	}

	// Determine which type mapper to use
	var mapper TypeMapper
	if config.TypeMapper != nil {
		mapper = config.TypeMapper
	} else if len(config.TypeMappings) > 0 {
		mapper = NewPostgreSQLTypeMapper(config.TypeMappings)
	}

	var schema *Schema
	var err error

	if config.IncludeAllSchemas {
		schema, err = IntrospectAllSchemasWithMapper(db, mapper)
	} else if len(config.Schemas) == 0 {
		schema, err = IntrospectDatabaseWithMapper(db, []string{"public"}, mapper)
	} else {
		schema, err = IntrospectDatabaseWithMapper(db, config.Schemas, mapper)
	}

	if err != nil {
		return "", fmt.Errorf("failed to introspect database: %w", err)
	}

	if len(config.ExcludeTables) > 0 {
		schema = FilterTables(schema, config.ExcludeTables)
	}

	return GenerateDBML(schema), nil
}

// GenerateFromConnectionString generates DBML from a PostgreSQL connection string.
// It opens a connection, introspects the database, and returns the generated DBML.
// If config is nil, defaults to introspecting only the "public" schema.
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

// WriteToFile generates DBML from an existing database connection and writes it to a file.
func WriteToFile(db *sql.DB, filename string, config *Config) error {
	dbmlContent, err := GenerateFromConnection(db, config)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(dbmlContent), 0644)
}

// WriteToFileFromConnectionString generates DBML from a PostgreSQL connection string
// and writes it to a file.
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

// GenerateFromConnectionBytes generates DBML from an existing database connection as bytes.
func GenerateFromConnectionBytes(db *sql.DB, config *Config) ([]byte, error) {
	result, err := GenerateFromConnection(db, config)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

// GenerateFromConnectionStringBytes generates DBML from a PostgreSQL connection string as bytes.
func GenerateFromConnectionStringBytes(connStr string, config *Config) ([]byte, error) {
	result, err := GenerateFromConnectionString(connStr, config)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}
