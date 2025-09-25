// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"errors"
	"strings"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/factory"
	"github.com/kyle-williams-1/bsonic/formatter"
	"github.com/kyle-williams-1/bsonic/language"
	"go.mongodb.org/mongo-driver/bson"
)

// SearchMode represents the type of search to perform
type SearchMode int

const (
	// SearchModeDisabled disables text search functionality (default behavior)
	SearchModeDisabled SearchMode = iota
	// SearchModeText performs MongoDB text search using $text operator
	SearchModeText
)

// Parser represents a query parser for MongoDB BSON filters.
type Parser struct {
	// SearchMode determines the type of search to perform
	SearchMode SearchMode
	// Config holds the language and formatter configuration
	Config *config.Config
	// Language parser instance
	languageParser language.Parser
	// Formatter instance (generic)
	formatter formatter.Formatter[bson.M]
}

// New creates a new BSON parser instance with default configuration.
func New() *Parser {
	cfg := config.Default()
	languageParser, _ := factory.CreateParser(cfg.Language)
	formatter, _ := factory.CreateFormatter(cfg.Formatter)

	return &Parser{
		SearchMode:     SearchModeDisabled,
		Config:         cfg,
		languageParser: languageParser,
		formatter:      formatter,
	}
}

// NewWithTextSearch creates a new BSON parser instance with text search enabled.
func NewWithTextSearch() *Parser {
	cfg := config.Default()
	languageParser, _ := factory.CreateParser(cfg.Language)
	formatter, _ := factory.CreateFormatter(cfg.Formatter)

	return &Parser{
		SearchMode:     SearchModeText,
		Config:         cfg,
		languageParser: languageParser,
		formatter:      formatter,
	}
}

// NewWithConfig creates a new parser with custom configuration.
func NewWithConfig(cfg *config.Config) (*Parser, error) {
	languageParser, err := factory.CreateParser(cfg.Language)
	if err != nil {
		return nil, err
	}

	formatter, err := factory.CreateFormatter(cfg.Formatter)
	if err != nil {
		return nil, err
	}

	return &Parser{
		SearchMode:     SearchModeDisabled,
		Config:         cfg,
		languageParser: languageParser,
		formatter:      formatter,
	}, nil
}

// SetSearchMode sets the search mode for the parser.
func (p *Parser) SetSearchMode(mode SearchMode) {
	p.SearchMode = mode
}

// Parse converts a query string into a BSON document.
// This is the recommended way to parse queries for most use cases.
func Parse(query string) (bson.M, error) {
	parser := New()
	return parser.Parse(query)
}

// Parse converts a query string into a BSON document.
func (p *Parser) Parse(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// If text search is enabled, handle all query types appropriately
	if p.SearchMode == SearchModeText {
		// Check if the language parser supports text search
		if textSearchParser, ok := p.languageParser.(language.TextSearchParser); ok {
			// Check if this is a mixed query (field searches + text search) first
			if textSearchParser.IsMixedQuery(query) {
				fieldAST, textTerms, err := textSearchParser.ParseMixedQuery(query)
				if err != nil {
					return nil, err
				}

				var conditions []bson.M

				if fieldAST != nil {
					fieldBSON, err := p.formatter.Format(fieldAST)
					if err != nil {
						return nil, err
					}
					conditions = append(conditions, fieldBSON)
				}

				if textTerms != "" {
					conditions = append(conditions, bson.M{"$text": bson.M{"$search": textTerms}})
				}

				if len(conditions) == 0 {
					return bson.M{}, nil
				} else if len(conditions) == 1 {
					return conditions[0], nil
				}
				return bson.M{"$and": conditions}, nil
			}

			// Check if this should be a text search query (no field:value pairs)
			if p.shouldUseTextSearch(query) {
				return p.parseTextSearch(query)
			}

			// If we get here, it's a pure field search with text search enabled
			// Parse it as a regular field query
			return p.parseFieldQuery(query)
		} else {
			// Fallback for parsers that don't support text search
			// Check if this should be a text search query (no field:value pairs)
			if p.shouldUseTextSearch(query) {
				return p.parseTextSearch(query)
			}

			// Parse as regular field query
			return p.parseFieldQuery(query)
		}
	}

	// Text search is disabled, parse as regular field query
	return p.parseFieldQuery(query)
}

// shouldUseTextSearch determines if a query should use text search instead of field searches.
func (p *Parser) shouldUseTextSearch(query string) bool {
	if p.SearchMode != SearchModeText {
		return false
	}

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
	parts := strings.Fields(trimmed)
	for _, part := range parts {
		if part == "AND" || part == "OR" || part == "NOT" {
			return false
		}
	}

	// If we get here, it's a simple text search query
	return true
}

// parseTextSearch parses a text search query and returns a BSON document with $text operator.
func (p *Parser) parseTextSearch(query string) (bson.M, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return bson.M{}, nil
	}

	// Only SearchModeText is supported for text search
	if p.SearchMode != SearchModeText {
		return nil, errors.New("text search requires SearchModeText")
	}

	return bson.M{"$text": bson.M{"$search": trimmed}}, nil
}

// parseFieldQuery parses a field-only query (without text search terms).
func (p *Parser) parseFieldQuery(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// Check if the language parser supports text search and validation
	if textSearchParser, ok := p.languageParser.(language.TextSearchParser); ok {
		if p.SearchMode != SearchModeText {
			if err := textSearchParser.ValidateFieldQuery(query); err != nil {
				return nil, err
			}
		}

		// Use the language parser's ParseFieldQuery method
		result, err := textSearchParser.ParseFieldQuery(query)
		if err != nil {
			return nil, err
		}

		// If it's already a BSON.M, return it directly
		if bsonResult, ok := result.(bson.M); ok {
			return bsonResult, nil
		}

		// Otherwise, format the AST
		return p.formatter.Format(result)
	}

	// Fallback to the original behavior for parsers that don't support text search
	ast, err := p.languageParser.Parse(query)
	if err != nil {
		return nil, err
	}

	return p.formatter.Format(ast)
}
