# PostgreSQL to DBML Generator

A Go package that generates DBML (Database Markup Language) files from PostgreSQL database connections.

## Features

- Extracts database schema from PostgreSQL
- Generates clean DBML syntax
- Supports tables, columns, primary keys, foreign keys, and indexes
- Configurable schema filtering and table exclusion
- PostgreSQL data type mapping to DBML types

## Installation

```bash
go get github.com/lucasefe/dbml
```

## Usage

### Basic Usage

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
    Schema:        "public",
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
    Schema        string   // Database schema name (default: "public")
    ExcludeTables []string // Tables to exclude from generation
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