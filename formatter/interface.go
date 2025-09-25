// Package formatter provides interfaces for query result formatters.
package formatter

import "go.mongodb.org/mongo-driver/bson"

// Formatter represents a query result formatter for a specific output type.
type Formatter[T any] interface {
	Format(ast interface{}) (T, error)
}

// Type aliases for formatter types
type BSONFormatter = Formatter[bson.M]
