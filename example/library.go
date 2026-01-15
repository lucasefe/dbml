//go:build ignore

// This file demonstrates various ways to use the dbml package as a library.
// Run with: go run using-library.go <connection_string>
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/lucasefe/dbml"
	"github.com/lucasefe/dbml/generator"
	"github.com/lucasefe/dbml/introspect"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run using-library.go <connection_string>")
		fmt.Println("Example: go run using-library.go 'postgres://user:password@localhost/dbname?sslmode=disable'")
		os.Exit(1)
	}

	connStr := os.Args[1]

	fmt.Println("=== Example 1: Basic Usage ===")
	basicUsage(connStr)

	fmt.Println("\n=== Example 2: Custom Type Mapping ===")
	customTypeMapping(connStr)

	fmt.Println("\n=== Example 3: Working with Schema Directly ===")
	workingWithSchema(connStr)

	fmt.Println("\n=== Example 4: Using Subpackages with Functional Options ===")
	usingSubpackages(connStr)

	fmt.Println("\n=== Example 5: Bytes Output ===")
	bytesOutput(connStr)
}

// basicUsage shows the simplest way to generate DBML
func basicUsage(connStr string) {
	dbmlContent, err := dbml.GenerateFromConnectionString(connStr, nil)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	printPreview("Basic output", dbmlContent)
}

// customTypeMapping demonstrates how to customize PostgreSQL to DBML type mappings
func customTypeMapping(connStr string) {
	// Method 1: Simple map-based overrides
	config := &dbml.Config{
		Schemas: []string{"public"},
		TypeMappings: map[string]string{
			"citext": "varchar",  // Map citext extension type to varchar
			"ltree":  "text",     // Map ltree extension type to text
			"hstore": "json",     // Map hstore to json
		},
	}

	dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	printPreview("With custom type mappings", dbmlContent)

	// Method 2: Using TypeMapper interface (for more control)
	mapper := dbml.NewPostgreSQLTypeMapper(map[string]string{
		"citext": "varchar",
		"ltree":  "text",
	})

	config2 := &dbml.Config{
		Schemas:    []string{"public"},
		TypeMapper: mapper,
	}

	dbmlContent2, err := dbml.GenerateFromConnectionString(connStr, config2)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	printPreview("With TypeMapper interface", dbmlContent2)
}

// workingWithSchema demonstrates introspecting and manipulating the schema directly
func workingWithSchema(connStr string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening connection: %v", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("Error pinging database: %v", err)
		return
	}

	// Step 1: Introspect the database
	schema, err := dbml.IntrospectDatabase(db, []string{"public"})
	if err != nil {
		log.Printf("Error introspecting database: %v", err)
		return
	}

	fmt.Printf("Found %d tables\n", len(schema.Tables))

	// Step 2: Filter out unwanted tables
	schema = dbml.FilterTables(schema, []string{
		"migrations",
		"schema_migrations",
		"ar_internal_metadata",
	})

	fmt.Printf("After filtering: %d tables\n", len(schema.Tables))

	// Step 3: Inspect the schema programmatically
	for _, table := range schema.Tables {
		fmt.Printf("  - %s.%s (%d columns)\n", table.Schema, table.Name, len(table.Columns))
	}

	// Step 4: Generate DBML
	dbmlOutput := dbml.GenerateDBML(schema)
	printPreview("Generated from filtered schema", dbmlOutput)
}

// usingSubpackages demonstrates the new subpackage APIs with functional options
func usingSubpackages(connStr string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening connection: %v", err)
		return
	}
	defer db.Close()

	// Use the introspect package with functional options
	schema, err := introspect.Database(db,
		introspect.WithSchemas("public"),
		introspect.WithExcludeTables("migrations", "schema_migrations"),
		introspect.WithTypeMappings(map[string]string{
			"citext": "varchar",
		}),
	)
	if err != nil {
		log.Printf("Error introspecting: %v", err)
		return
	}

	fmt.Printf("Introspected %d tables using functional options\n", len(schema.Tables))

	// Use the generator package (returns []byte)
	output, err := generator.Generate(schema)
	if err != nil {
		log.Printf("Error generating: %v", err)
		return
	}

	printPreview("Generated using subpackages", string(output))

	// Alternative: Use FromConnectionString for convenience
	schema2, err := introspect.FromConnectionString(connStr,
		introspect.WithAllSchemas(),
		introspect.WithExcludeTables("migrations"),
	)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("FromConnectionString found %d tables\n", len(schema2.Tables))
}

// bytesOutput demonstrates functions that return []byte for better performance
func bytesOutput(connStr string) {
	config := &dbml.Config{
		Schemas: []string{"public"},
	}

	// Get output as bytes (more efficient for file writing)
	dbmlBytes, err := dbml.GenerateFromConnectionStringBytes(connStr, config)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Generated %d bytes of DBML\n", len(dbmlBytes))

	// Write directly to file
	outputFile := "example-output.dbml"
	if err := os.WriteFile(outputFile, dbmlBytes, 0644); err != nil {
		log.Printf("Error writing file: %v", err)
		return
	}

	fmt.Printf("Written to %s\n", outputFile)

	// Clean up
	os.Remove(outputFile)
}

// printPreview prints a preview of the generated DBML
func printPreview(title, content string) {
	fmt.Printf("\n%s:\n", title)
	fmt.Println("---")
	if len(content) > 300 {
		fmt.Printf("%s...\n", content[:300])
	} else {
		fmt.Print(content)
	}
	fmt.Println("---")
}
