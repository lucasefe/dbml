package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/lucasefe/dbml"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <connection_string> [output_file]")
		fmt.Println("Example: go run main.go 'postgres://user:password@localhost/dbname?sslmode=disable' output.dbml")
		os.Exit(1)
	}

	connStr := os.Args[1]
	outputFile := "output.dbml"
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	fmt.Printf("Connecting to database...\n")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Printf("Introspecting database schema...\n")

	config := &dbml.Config{
		Schema: "public",
	}

	dbmlContent, err := dbml.GenerateFromConnection(db, config)
	if err != nil {
		log.Fatalf("Failed to generate DBML: %v", err)
	}

	fmt.Printf("Writing DBML to file: %s\n", outputFile)

	if err := os.WriteFile(outputFile, []byte(dbmlContent), 0644); err != nil {
		log.Fatalf("Failed to write DBML file: %v", err)
	}

	fmt.Printf("Successfully generated DBML file: %s\n", outputFile)
	fmt.Printf("Generated %d bytes of DBML content\n", len(dbmlContent))

	fmt.Println("\nDBML content preview:")
	fmt.Println("---------------------")
	if len(dbmlContent) > 500 {
		fmt.Printf("%s...\n", dbmlContent[:500])
	} else {
		fmt.Println(dbmlContent)
	}
}

func exampleUsageWithConfig() {
	connStr := "postgres://user:password@localhost/dbname?sslmode=disable"

	config := &dbml.Config{
		Schema:        "public",
		ExcludeTables: []string{"migrations", "schema_migrations"},
	}

	dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)
	if err != nil {
		log.Fatalf("Failed to generate DBML: %v", err)
	}

	fmt.Println(dbmlContent)
}

func exampleDirectToFile() {
	connStr := "postgres://user:password@localhost/dbname?sslmode=disable"
	filename := "database_schema.dbml"

	config := &dbml.Config{
		Schema: "public",
	}

	err := dbml.WriteToFileFromConnectionString(connStr, filename, config)
	if err != nil {
		log.Fatalf("Failed to write DBML file: %v", err)
	}

	fmt.Printf("DBML written to %s\n", filename)
}