// Package dbml provides tools for generating DBML (Database Markup Language)
// files from PostgreSQL database schemas.
//
// The package supports introspecting PostgreSQL databases to extract schema
// information including tables, columns, primary keys, foreign keys, and indexes,
// then generating DBML-formatted output.
//
// # Basic Usage
//
// Generate DBML from a connection string:
//
//	import "github.com/lucasefe/dbml"
//
//	dbmlContent, err := dbml.GenerateFromConnectionString(connStr, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Print(dbmlContent)
//
// # Configuration
//
// Use Config to customize which schemas and tables to include:
//
//	config := &dbml.Config{
//	    Schemas:       []string{"public", "auth"},
//	    ExcludeTables: []string{"migrations", "schema_versions"},
//	}
//	dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)
//
// # Working with Schema Directly
//
// For more control, you can introspect the database and generate DBML separately:
//
//	db, err := sql.Open("postgres", connStr)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Introspect the database
//	schema, err := dbml.IntrospectDatabase(db, []string{"public"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Optionally filter tables
//	schema = dbml.FilterTables(schema, []string{"temp_table"})
//
//	// Generate DBML
//	dbmlOutput := dbml.GenerateDBML(schema)
//
// # Subpackages
//
// For advanced use cases, consider using the subpackages directly:
//
//   - github.com/lucasefe/dbml/schema - Data structures for representing database schemas
//   - github.com/lucasefe/dbml/introspect - Database introspection with functional options
//   - github.com/lucasefe/dbml/generator - DBML generation with []byte output
//
// # Custom Type Mapping
//
// You can customize how PostgreSQL types are mapped to DBML types:
//
//	mapper := dbml.NewPostgreSQLTypeMapper(map[string]string{
//	    "citext": "varchar",
//	    "ltree":  "text",
//	})
//	config := &dbml.Config{
//	    TypeMapper: mapper,
//	}
//	dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)
package dbml
