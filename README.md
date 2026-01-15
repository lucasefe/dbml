# PostgreSQL to DBML Generator

A Go library and CLI tool that generates DBML (Database Markup Language) files from PostgreSQL database connections.

## Features

- Extracts database schema from PostgreSQL
- Generates clean DBML syntax
- Supports tables, columns, primary keys, foreign keys, and indexes
- Configurable schema filtering and table exclusion
- PostgreSQL data type mapping to DBML types
- Custom type mapping support
- Library-first design with subpackages for advanced use cases

## Installation

### Go Package
```bash
go get github.com/lucasefe/dbml
```

### CLI Binary
```bash
# Install globally
go install github.com/lucasefe/dbml/cmd/dbml@latest

# Or build locally
git clone https://github.com/lucasefe/dbml.git
cd dbml
make build
```

## Usage

### CLI Usage

```bash
# Generate DBML for public schema to stdout
dbml --url "postgres://user:pass@localhost/db"

# Generate DBML for all schemas to file
dbml --url "postgres://user:pass@localhost/db" --all-schemas --output schema.dbml

# Use environment variable for database URL
export DATABASE_URL="postgres://user:pass@localhost/db"
dbml --schemas "public,auth" --exclude-tables "migrations,_temp"

# Generate DBML to stdout (useful for piping)
dbml | head -20
```

#### CLI Options
- `--url, -u`: PostgreSQL connection URL
- `--output, -o`: Output file path (default: stdout)
- `--schemas, -s`: Comma-separated schemas to include (default: public)
- `--exclude-tables, -x`: Comma-separated tables to exclude
- `--all-schemas, -a`: Include all non-system schemas
- `--version, -v`: Show version
- `--help, -h`: Show help

#### Environment Variables
- `DATABASE_URL`: PostgreSQL connection URL
- `DBML_SCHEMAS`: Comma-separated schemas to include
- `DBML_EXCLUDE_TABLES`: Comma-separated tables to exclude
- `DBML_ALL_SCHEMAS`: Set to 'true' to include all schemas

### Go Library Usage

#### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/lucasefe/dbml"
)

func main() {
    connStr := "postgres://user:password@localhost/dbname?sslmode=disable"

    dbmlContent, err := dbml.GenerateFromConnectionString(connStr, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(dbmlContent)
}
```

#### With Configuration

```go
config := &dbml.Config{
    Schemas:       []string{"public", "auth"},
    ExcludeTables: []string{"migrations", "schema_migrations"},
}

dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)
if err != nil {
    log.Fatal(err)
}
```

#### Custom Type Mapping

Override default PostgreSQL to DBML type mappings:

```go
// Simple approach: use TypeMappings map
config := &dbml.Config{
    TypeMappings: map[string]string{
        "citext": "varchar",
        "ltree":  "text",
    },
}

dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)

// Advanced approach: implement TypeMapper interface
mapper := dbml.NewPostgreSQLTypeMapper(map[string]string{
    "citext": "varchar",
    "ltree":  "text",
})

config := &dbml.Config{
    TypeMapper: mapper,
}
```

#### Return Bytes Instead of String

For better performance when writing to files:

```go
// Returns []byte
dbmlBytes, err := dbml.GenerateFromConnectionStringBytes(connStr, config)
if err != nil {
    log.Fatal(err)
}

// Write directly
os.WriteFile("schema.dbml", dbmlBytes, 0644)
```

#### Working with Schema Directly

For more control, introspect and generate separately:

```go
import (
    "database/sql"
    "github.com/lucasefe/dbml"
    _ "github.com/lib/pq"
)

db, err := sql.Open("postgres", connStr)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Introspect the database
schema, err := dbml.IntrospectDatabase(db, []string{"public", "auth"})
if err != nil {
    log.Fatal(err)
}

// Filter tables programmatically
schema = dbml.FilterTables(schema, []string{"temp_table", "migrations"})

// Generate DBML
dbmlOutput := dbml.GenerateDBML(schema)
fmt.Println(dbmlOutput)
```

### Using Subpackages (Advanced)

For advanced use cases, use the subpackages directly with functional options:

```go
import (
    "database/sql"
    "os"

    "github.com/lucasefe/dbml/introspect"
    "github.com/lucasefe/dbml/generator"
    _ "github.com/lib/pq"
)

func main() {
    db, _ := sql.Open("postgres", connStr)
    defer db.Close()

    // Introspect with functional options
    schema, err := introspect.Database(db,
        introspect.WithSchemas("public", "auth"),
        introspect.WithExcludeTables("migrations", "schema_versions"),
        introspect.WithTypeMappings(map[string]string{
            "citext": "varchar",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Generate DBML (returns []byte)
    output, err := generator.Generate(schema)
    if err != nil {
        log.Fatal(err)
    }

    os.Stdout.Write(output)
}
```

Or use the connection string convenience function:

```go
schema, err := introspect.FromConnectionString(connStr,
    introspect.WithAllSchemas(),
    introspect.WithExcludeTables("migrations"),
)
```

## API Reference

### Root Package Types

```go
// Config specifies options for DBML generation.
type Config struct {
    Schemas           []string          // Schemas to include (default: ["public"])
    ExcludeTables     []string          // Tables to exclude
    IncludeAllSchemas bool              // Include all non-system schemas
    TypeMapper        TypeMapper        // Custom type mapper (optional)
    TypeMappings      map[string]string // Simple type overrides (optional)
}

// TypeMapper interface for custom type mapping
type TypeMapper interface {
    MapType(dataType, udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string
}

// Schema represents a database schema
type Schema struct {
    Tables []Table
}

type Table struct {
    Name        string
    Schema      string
    Columns     []Column
    PrimaryKeys []string
    Indexes     []Index
    References  []Reference
}
```

### Root Package Functions

```go
// Generation functions (return string)
func GenerateFromConnection(db *sql.DB, config *Config) (string, error)
func GenerateFromConnectionString(connStr string, config *Config) (string, error)
func GenerateDBML(schema *Schema) string

// Generation functions (return []byte)
func GenerateFromConnectionBytes(db *sql.DB, config *Config) ([]byte, error)
func GenerateFromConnectionStringBytes(connStr string, config *Config) ([]byte, error)
func GenerateDBMLBytes(schema *Schema) []byte

// File writing
func WriteToFile(db *sql.DB, filename string, config *Config) error
func WriteToFileFromConnectionString(connStr, filename string, config *Config) error

// Introspection
func IntrospectDatabase(db *sql.DB, schemaNames []string) (*Schema, error)
func IntrospectDatabaseWithMapper(db *sql.DB, schemaNames []string, mapper TypeMapper) (*Schema, error)
func IntrospectAllSchemas(db *sql.DB) (*Schema, error)

// Utilities
func FilterTables(schema *Schema, excludeTables []string) *Schema
func NewPostgreSQLTypeMapper(customMappings map[string]string) *PostgreSQLTypeMapper
func MapPostgreSQLTypeToDBML(dataType, udtName string, ...) string
func GetQualifiedTableName(tableName, schemaName string) string
```

### Subpackages

#### `github.com/lucasefe/dbml/schema`

Data structures for representing database schemas:
- `Schema`, `Table`, `Column`, `Index`, `Reference` types
- `FilterTables(s *Schema, excludeTables []string) *Schema`

#### `github.com/lucasefe/dbml/introspect`

Database introspection with functional options:
- `Database(db *sql.DB, opts ...Option) (*schema.Schema, error)`
- `FromConnectionString(connStr string, opts ...Option) (*schema.Schema, error)`

Options:
- `WithSchemas(schemas ...string)` - Specify schemas to introspect
- `WithExcludeTables(tables ...string)` - Exclude specific tables
- `WithAllSchemas()` - Include all non-system schemas
- `WithTypeMapper(mapper TypeMapper)` - Custom type mapper
- `WithTypeMappings(mappings map[string]string)` - Simple type overrides

#### `github.com/lucasefe/dbml/generator`

DBML generation:
- `Generate(s *schema.Schema) ([]byte, error)` - Returns bytes
- `GenerateString(s *schema.Schema) (string, error)` - Returns string
- `GetQualifiedTableName(tableName, schemaName string) string`

## PostgreSQL Data Type Mapping

| PostgreSQL Type | DBML Type |
|----------------|-----------|
| integer, int4 | int |
| bigint, int8 | bigint |
| smallint, int2 | smallint |
| boolean, bool | boolean |
| varchar | varchar |
| char | char |
| text | text |
| numeric, decimal | decimal |
| real, float4 | float |
| double precision, float8 | double |
| timestamp | timestamp |
| timestamptz | timestamptz |
| date | date |
| time | time |
| timetz | timetz |
| uuid | uuid |
| json | json |
| jsonb | jsonb |
| bytea | binary |

Custom types and arrays are normalized to `text` by default. Use `TypeMappings` or `TypeMapper` to customize.

## Sample Output

```dbml
Table users {
  id int [pk, increment]
  email varchar(255) [not null]
  name varchar(100)
  created_at timestamp [not null, default: `now()`]
  is_active boolean [not null, default: `true`]

  indexes {
    (email) [unique]
  }
}

Table posts {
  id int [pk, increment]
  user_id int [not null]
  title varchar(200) [not null]
  content text
  created_at timestamp [not null, default: `now()`]

  indexes {
    user_id
  }
}

Ref: posts.user_id > users.id [delete: cascade]
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Build for all platforms
make build-all
```

## Requirements

- Go 1.21 or higher
- PostgreSQL database
- github.com/lib/pq driver

## License

MIT
