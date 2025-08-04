package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucasefe/dbml"
)

const (
	defaultDatabaseURL = "DATABASE_URL"
	version           = "1.0.0"
)

type Config struct {
	DatabaseURL       string
	OutputFile        string
	Schemas           []string
	ExcludeTables     []string
	IncludeAllSchemas bool
	ShowVersion       bool
	ShowHelp          bool
}

func main() {
	config := parseFlags()

	if config.ShowVersion {
		fmt.Printf("dbml version %s\n", version)
		os.Exit(0)
	}

	if config.ShowHelp {
		printUsage()
		os.Exit(0)
	}

	// Get database URL from environment if not provided
	if config.DatabaseURL == "" {
		config.DatabaseURL = os.Getenv(defaultDatabaseURL)
	}

	if config.DatabaseURL == "" {
		fmt.Fprintf(os.Stderr, "Error: Database URL is required. Provide via --url flag or %s environment variable.\n", defaultDatabaseURL)
		printUsage()
		os.Exit(1)
	}

	// Generate DBML
	dbmlConfig := &dbml.Config{
		Schemas:           config.Schemas,
		ExcludeTables:     config.ExcludeTables,
		IncludeAllSchemas: config.IncludeAllSchemas,
	}

	dbmlContent, err := dbml.GenerateFromConnectionString(config.DatabaseURL, dbmlConfig)
	if err != nil {
		log.Fatalf("Failed to generate DBML: %v", err)
	}

	// Output to file or stdout
	if config.OutputFile != "" {
		err := os.WriteFile(config.OutputFile, []byte(dbmlContent), 0644)
		if err != nil {
			log.Fatalf("Failed to write to file %s: %v", config.OutputFile, err)
		}
		fmt.Fprintf(os.Stderr, "DBML written to %s (%d bytes)\n", config.OutputFile, len(dbmlContent))
	} else {
		fmt.Print(dbmlContent)
	}
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.DatabaseURL, "url", "", "PostgreSQL connection URL (can also use DATABASE_URL env var)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&config.OutputFile, "o", "", "Output file path (short form)")
	
	var schemasFlag string
	flag.StringVar(&schemasFlag, "schemas", "", "Comma-separated list of schemas to include (default: public)")
	flag.StringVar(&schemasFlag, "s", "", "Comma-separated list of schemas to include (short form)")
	
	var excludeTablesFlag string
	flag.StringVar(&excludeTablesFlag, "exclude-tables", "", "Comma-separated list of tables to exclude")
	flag.StringVar(&excludeTablesFlag, "x", "", "Comma-separated list of tables to exclude (short form)")
	
	flag.BoolVar(&config.IncludeAllSchemas, "all-schemas", false, "Include all non-system schemas")
	flag.BoolVar(&config.IncludeAllSchemas, "a", false, "Include all non-system schemas (short form)")
	
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")
	flag.BoolVar(&config.ShowVersion, "v", false, "Show version information (short form)")
	
	flag.BoolVar(&config.ShowHelp, "help", false, "Show help information")
	flag.BoolVar(&config.ShowHelp, "h", false, "Show help information (short form)")

	flag.Parse()

	// Parse schemas
	if schemasFlag != "" {
		config.Schemas = strings.Split(schemasFlag, ",")
		for i, schema := range config.Schemas {
			config.Schemas[i] = strings.TrimSpace(schema)
		}
	}

	// Parse exclude tables
	if excludeTablesFlag != "" {
		config.ExcludeTables = strings.Split(excludeTablesFlag, ",")
		for i, table := range config.ExcludeTables {
			config.ExcludeTables[i] = strings.TrimSpace(table)
		}
	}

	// Handle environment variables for other options
	if envSchemas := os.Getenv("DBML_SCHEMAS"); envSchemas != "" && schemasFlag == "" {
		config.Schemas = strings.Split(envSchemas, ",")
		for i, schema := range config.Schemas {
			config.Schemas[i] = strings.TrimSpace(schema)
		}
	}

	if envExclude := os.Getenv("DBML_EXCLUDE_TABLES"); envExclude != "" && excludeTablesFlag == "" {
		config.ExcludeTables = strings.Split(envExclude, ",")
		for i, table := range config.ExcludeTables {
			config.ExcludeTables[i] = strings.TrimSpace(table)
		}
	}

	if os.Getenv("DBML_ALL_SCHEMAS") == "true" && !config.IncludeAllSchemas {
		config.IncludeAllSchemas = true
	}

	return config
}

func printUsage() {
	fmt.Printf(`dbml - Generate DBML from PostgreSQL databases

USAGE:
    dbml [OPTIONS]

OPTIONS:
    -url, --url <URL>              PostgreSQL connection URL
    -o, --output <FILE>            Output file (default: stdout)
    -s, --schemas <SCHEMAS>        Comma-separated schemas to include (default: public)
    -x, --exclude-tables <TABLES>  Comma-separated tables to exclude
    -a, --all-schemas              Include all non-system schemas
    -v, --version                  Show version
    -h, --help                     Show help

ENVIRONMENT VARIABLES:
    DATABASE_URL                   PostgreSQL connection URL
    DBML_SCHEMAS                   Comma-separated schemas to include
    DBML_EXCLUDE_TABLES           Comma-separated tables to exclude
    DBML_ALL_SCHEMAS              Set to 'true' to include all schemas

EXAMPLES:
    # Generate DBML for public schema to stdout
    dbml --url "postgres://user:pass@localhost/db"

    # Generate DBML for all schemas to file
    dbml --url "postgres://user:pass@localhost/db" --all-schemas --output schema.dbml

    # Use environment variable for database URL
    export DATABASE_URL="postgres://user:pass@localhost/db"
    dbml --schemas "public,auth" --exclude-tables "migrations,_temp"

    # Generate DBML to stdout (useful for piping)
    dbml | head -20

CONNECTION STRING FORMAT:
    postgres://[user[:password]@][host][:port][/dbname][?param1=value1&...]

    Examples:
    - postgres://localhost/mydb
    - postgres://user:secret@localhost:5432/mydb?sslmode=disable
    - postgres://user@localhost/mydb?sslmode=require

`)
}