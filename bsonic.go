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

				// Use the formatter's mixed query handling
				var fieldBSON bson.M
				if fieldAST != nil {
					fieldBSON, err = p.formatter.Format(fieldAST)
					if err != nil {
						return nil, err
					}
				}
				// Use the formatter's FormatMixedQuery method
				return p.formatter.(formatter.TextSearchFormatter[bson.M]).FormatMixedQuery(fieldBSON, textTerms)
			}

			// Check if this should be a text search query (no field:value pairs)
			if textSearchParser.ShouldUseTextSearch(query) {
				textTerms, err := textSearchParser.ParseTextSearch(query)
				if err != nil {
					return nil, err
				}

				// Use the formatter's text search handling
				return p.formatter.(formatter.TextSearchFormatter[bson.M]).FormatTextSearch(textTerms)
			}

			// If we get here, it's a pure field search with text search enabled
			// Parse it as a regular field query
			return p.parseFieldQuery(query)
		} else {
			// Fallback for parsers that don't support text search
			// This shouldn't happen in practice, but handle gracefully
			return p.parseFieldQuery(query)
		}
	}

	// Text search is disabled, parse as regular field query
	return p.parseFieldQuery(query)
}

// parseFieldQuery parses a field-only query (without text search terms).
func (p *Parser) parseFieldQuery(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// Check if the language parser supports text search and validation
	if textSearchParser, ok := p.languageParser.(language.TextSearchParser); ok {
		// Validate that this is not a mixed query or text-only query when text search is disabled
		if p.SearchMode != SearchModeText {
			if textSearchParser.IsMixedQuery(query) {
				return nil, errors.New("mixed query (field:value pairs + text terms) requires text search to be enabled")
			}
			if err := textSearchParser.ValidateFieldQuery(query); err != nil {
				return nil, err
			}
		}
	}

	// Parse the query and let the formatter handle it
	ast, err := p.languageParser.Parse(query)
	if err != nil {
		return nil, err
	}

	return p.formatter.Format(ast)
}
