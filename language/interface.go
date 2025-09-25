// Package language provides interfaces for query language parsers.
package language

// AST represents a parsed query abstract syntax tree.
type AST interface{}

// Parser represents a query language parser.
type Parser interface {
	Parse(query string) (AST, error)
}
