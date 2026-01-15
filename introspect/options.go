package introspect

// Option configures introspection behavior.
type Option func(*options)

type options struct {
	schemas           []string
	excludeTables     []string
	includeAllSchemas bool
	typeMapper        TypeMapper
}

func defaultOptions() *options {
	return &options{
		schemas: []string{"public"},
	}
}

// WithSchemas specifies which database schemas to introspect.
// If not specified, defaults to ["public"].
func WithSchemas(schemas ...string) Option {
	return func(o *options) {
		o.schemas = schemas
	}
}

// WithExcludeTables specifies tables to exclude from introspection.
func WithExcludeTables(tables ...string) Option {
	return func(o *options) {
		o.excludeTables = tables
	}
}

// WithAllSchemas includes all non-system schemas in the introspection.
// This overrides WithSchemas.
func WithAllSchemas() Option {
	return func(o *options) {
		o.includeAllSchemas = true
	}
}

// WithTypeMapper sets a custom type mapper for converting database types to DBML types.
// If not specified, uses the default PostgreSQL type mapper.
func WithTypeMapper(mapper TypeMapper) Option {
	return func(o *options) {
		o.typeMapper = mapper
	}
}

// WithTypeMappings provides custom type mappings as a simple map.
// This is a convenience alternative to WithTypeMapper for simple use cases.
// Keys are PostgreSQL type names (case-insensitive), values are DBML types.
func WithTypeMappings(mappings map[string]string) Option {
	return func(o *options) {
		o.typeMapper = NewPostgreSQLTypeMapper(mappings)
	}
}
