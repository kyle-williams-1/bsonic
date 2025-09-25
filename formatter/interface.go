// Package formatter provides interfaces for query result formatters.
package formatter

import "go.mongodb.org/mongo-driver/bson"

// Formatter represents a query result formatter for a specific output type.
type Formatter[T any] interface {
	Format(ast interface{}) (T, error)
}

// TextSearchFormatter represents a formatter that can handle text search operations.
type TextSearchFormatter[T any] interface {
	Formatter[T]
	// FormatTextSearch formats text search terms into the output format.
	FormatTextSearch(textTerms string) (T, error)
	// FormatMixedQuery formats a mixed query with both field and text search components.
	FormatMixedQuery(fieldResult T, textTerms string) (T, error)
}

// Type aliases for formatter types
type BSONFormatter = Formatter[bson.M]
type BSONTextSearchFormatter = TextSearchFormatter[bson.M]
