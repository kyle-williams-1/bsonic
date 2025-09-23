// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"fmt"
	"strings"

	"github.com/grindlemire/go-lucene"
	"go.mongodb.org/mongo-driver/bson"
)

// SearchMode represents the type of search to perform.
// This determines how queries without field:value pairs are handled.
type SearchMode int

const (
	// SearchModeDisabled disables text search functionality (default behavior).
	// Queries without field:value pairs will return an error.
	SearchModeDisabled SearchMode = iota

	// SearchModeText performs MongoDB text search using $text operator.
	// Queries without field:value pairs will be treated as text search queries.
	SearchModeText
)

// Parser represents a Lucene-style query parser for MongoDB BSON filters.
// It uses the go-lucene library for parsing and converts the AST to BSON.
type Parser struct {
	// SearchMode determines the type of search to perform
	SearchMode SearchMode
	// driver handles the actual query parsing using go-lucene and BSON conversion
	driver *BSONDriver
	// preprocessor handles query preprocessing to fix parsing issues
	preprocessor *QueryPreprocessor
}

// New creates a new BSON parser instance with text search disabled.
// This is the recommended constructor for most use cases.
func New() *Parser {
	return &Parser{
		SearchMode:   SearchModeDisabled,
		driver:       NewBSONDriver(SearchModeDisabled),
		preprocessor: NewQueryPreprocessor(),
	}
}

// NewWithTextSearch creates a new BSON parser instance with text search enabled.
// Use this when you need to support text search queries without field:value pairs.
func NewWithTextSearch() *Parser {
	return &Parser{
		SearchMode:   SearchModeText,
		driver:       NewBSONDriver(SearchModeText),
		preprocessor: NewQueryPreprocessor(),
	}
}

// SetSearchMode sets the search mode for the parser.
// This can be used to change the search mode at runtime.
func (p *Parser) SetSearchMode(mode SearchMode) {
	p.SearchMode = mode
	p.driver.SetSearchMode(mode)
}

// Parse converts a Lucene-style query string into a BSON document.
// This is the recommended way to parse queries for most use cases.
// It creates a new parser instance internally.
func Parse(query string) (bson.M, error) {
	parser := New()
	return parser.Parse(query)
}

// Parse converts a Lucene-style query string into a BSON document.
// The parsing process uses the go-lucene library for parsing and converts the AST to BSON.
// Returns an error if the query is invalid or if text search is disabled but a text search query is provided.
func (p *Parser) Parse(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// Preprocess the query to fix common parsing issues
	processedQuery := p.preprocessor.PreprocessQuery(query)

	// Check if this is a text search query (no field:value pairs and text search mode enabled)
	if p.SearchMode == SearchModeText && p.isTextSearchQuery(processedQuery) {
		return bson.M{"$text": bson.M{"$search": strings.TrimSpace(processedQuery)}}, nil
	}

	// Check if this is a text search query but text search mode is disabled
	if p.SearchMode == SearchModeDisabled && p.isTextSearchQuery(processedQuery) {
		return nil, fmt.Errorf("text search queries are not supported when text search mode is disabled")
	}

	// Parse the query using go-lucene
	expr, err := lucene.Parse(processedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Convert the AST to BSON using the custom driver
	return p.driver.RenderExpression(expr)
}

// isTextSearchQuery determines if a query should be treated as text search
func (p *Parser) isTextSearchQuery(query string) bool {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return false
	}

	// Check if query contains any field:value pairs
	if strings.Contains(trimmed, ":") {
		return false
	}

	// Check if query contains logical operators without field:value pairs
	// This would be a mixed query that needs special handling
	if p.hasOperators(trimmed) {
		return false
	}

	// If we get here, it's a simple text search query
	return true
}

// hasOperators checks if a query contains logical operators
func (p *Parser) hasOperators(query string) bool {
	upperQuery := strings.ToUpper(query)
	operators := []string{" AND ", " OR ", " NOT ", "(", ")"}

	for _, op := range operators {
		if strings.Contains(upperQuery, op) {
			return true
		}
	}

	return false
}
