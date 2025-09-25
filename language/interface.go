// Package language provides interfaces for query language parsers.
package language

// AST represents a parsed query abstract syntax tree.
type AST interface{}

// Parser represents a query language parser.
type Parser interface {
	Parse(query string) (AST, error)
}

// TextSearchParser represents a parser that supports text search functionality.
type TextSearchParser interface {
	Parser
	// IsMixedQuery determines if a query contains both field searches and text search terms.
	IsMixedQuery(query string) bool
	// ValidateFieldQuery validates that a field query doesn't contain standalone text terms when text search is disabled.
	ValidateFieldQuery(query string) error
	// ParseMixedQuery parses a mixed query containing both field searches and text search.
	// Returns the AST for the field parts and text search terms separately.
	ParseMixedQuery(query string) (fieldAST interface{}, textTerms string, err error)
	// ParseFieldQuery parses a field-only query (without text search terms).
	ParseFieldQuery(query string) (interface{}, error)
}
