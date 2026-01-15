// Package schema defines the data structures for representing database schemas.
// These types are used throughout the dbml package for introspection and generation.
package schema

// Schema represents a database schema containing multiple tables.
// It is the top-level container returned by introspection functions.
type Schema struct {
	// Tables contains all tables found in the introspected schema(s).
	Tables []Table
}

// Table represents a database table with its columns, primary keys,
// indexes, and foreign key references.
type Table struct {
	// Name is the table name without schema qualification.
	Name string
	// Schema is the database schema containing this table (e.g., "public").
	Schema string
	// Columns contains all columns in the table, ordered by ordinal position.
	Columns []Column
	// PrimaryKeys lists column names that form the primary key.
	PrimaryKeys []string
	// Indexes contains non-primary-key indexes on the table.
	Indexes []Index
	// References contains foreign key relationships from this table to other tables.
	References []Reference
}

// Column represents a database column within a table.
type Column struct {
	// Name is the column name.
	Name string
	// Type is the DBML-compatible type (e.g., "int", "varchar(255)", "timestamp").
	Type string
	// Nullable indicates whether the column allows NULL values.
	Nullable bool
	// DefaultValue is the column's default value expression, or nil if none.
	DefaultValue *string
	// IsPrimaryKey indicates whether this column is part of the primary key.
	IsPrimaryKey bool
}

// Index represents a database index on one or more columns.
type Index struct {
	// Name is the index name.
	Name string
	// Columns lists the column names included in the index.
	Columns []string
	// Unique indicates whether this is a unique index.
	Unique bool
}

// Reference represents a foreign key relationship between tables.
type Reference struct {
	// FromTable is the table containing the foreign key.
	FromTable string
	// FromSchema is the schema of the table containing the foreign key.
	FromSchema string
	// FromColumns lists the column names in the foreign key.
	FromColumns []string
	// ToTable is the referenced table.
	ToTable string
	// ToSchema is the schema of the referenced table.
	ToSchema string
	// ToColumns lists the referenced column names.
	ToColumns []string
	// OnDelete is the referential action on delete (e.g., "CASCADE", "SET NULL").
	OnDelete string
	// OnUpdate is the referential action on update.
	OnUpdate string
}
