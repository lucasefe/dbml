# PostgreSQL to DBML Generator

A Go package that generates DBML (Database Markup Language) files from PostgreSQL database connections.

## Features

- Extracts database schema from PostgreSQL
- Generates clean DBML syntax
- Supports tables, columns, primary keys, foreign keys, and indexes
- Configurable schema filtering and table exclusion
- PostgreSQL data type mapping to DBML types

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
go build -o dbml ./cmd/dbml
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

### Go Package Usage

#### Basic Usage

```go
package main

import (
    "database/sql"
    "fmt"
    "log"

    "github.com/lucasefe/dbml"
    _ "github.com/lib/pq"
)

func main() {
    // Using connection string
    connStr := "postgres://user:password@localhost/dbname?sslmode=disable"
    
    dbmlContent, err := dbml.GenerateFromConnectionString(connStr, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(dbmlContent)
}
```

### With Configuration

```go
config := &dbml.Config{
    Schemas:       []string{"public", "auth"},
    ExcludeTables: []string{"migrations", "schema_migrations"},
}

dbmlContent, err := dbml.GenerateFromConnectionString(connStr, config)
if err != nil {
    log.Fatal(err)
}

fmt.Println(dbmlContent)
```

### Direct to File

```go
err := dbml.WriteToFileFromConnectionString(connStr, "database.dbml", config)
if err != nil {
    log.Fatal(err)
}
```

### Using Existing Database Connection

```go
db, err := sql.Open("postgres", connStr)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

dbmlContent, err := dbml.GenerateFromConnection(db, config)
if err != nil {
    log.Fatal(err)
}

fmt.Println(dbmlContent)
```

## Example

Run the example:

```bash
cd example
go run main.go "postgres://user:password@localhost/dbname?sslmode=disable" output.dbml
```

## API Reference

### Types

```go
type Config struct {
    Schemas           []string // Specific schemas to include (empty means all non-system schemas)
    ExcludeTables     []string // Tables to exclude from generation
    IncludeAllSchemas bool     // If true, includes all non-system schemas
}
```

### Functions

- `GenerateFromConnection(db *sql.DB, config *Config) (string, error)`
- `GenerateFromConnectionString(connStr string, config *Config) (string, error)`
- `WriteToFile(db *sql.DB, filename string, config *Config) error`
- `WriteToFileFromConnectionString(connStr, filename string, config *Config) error`

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

## Sample Output

```dbml
Table users {
  id int [pk, not null, increment]
  email varchar(255) [not null]
  name varchar(100)
  created_at timestamp [not null, default: `now()`]
  is_active boolean [not null, default: `true`]

  unique email
}

Table posts {
  id int [pk, not null, increment]
  user_id int [not null]
  title varchar(200) [not null]
  content text
  created_at timestamp [not null, default: `now()`]

  index user_id
}

Ref: posts.user_id > users.id [delete: cascade]
```

## Requirements

- Go 1.21 or higher
- PostgreSQL database
- github.com/lib/pq driver

## License

MIT