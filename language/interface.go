// Package language provides interfaces for query language parsers.
package language

// Parser represents a query language parser.
type Parser interface {
	Parse(query string) (interface{}, error)
}
