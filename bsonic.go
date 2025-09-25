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

	if p.SearchMode == SearchModeText {
		return p.parseWithTextSearch(query)
	}

	return p.parseFieldQuery(query)
}

// parseWithTextSearch handles parsing when text search is enabled
func (p *Parser) parseWithTextSearch(query string) (bson.M, error) {
	textSearchParser, ok := p.languageParser.(language.TextSearchParser)
	if !ok {
		// Fallback for parsers that don't support text search
		return p.parseFieldQuery(query)
	}

	if textSearchParser.IsMixedQuery(query) {
		return p.parseMixedQuery(query, textSearchParser)
	}

	if textSearchParser.ShouldUseTextSearch(query) {
		return p.parseTextOnlyQuery(query, textSearchParser)
	}

	// Pure field search with text search enabled
	return p.parseFieldQuery(query)
}

// parseMixedQuery handles mixed queries with both field and text search
func (p *Parser) parseMixedQuery(query string, textSearchParser language.TextSearchParser) (bson.M, error) {
	fieldAST, textTerms, err := textSearchParser.ParseMixedQuery(query)
	if err != nil {
		return nil, err
	}

	var fieldBSON bson.M
	if fieldAST != nil {
		fieldBSON, err = p.formatter.Format(fieldAST)
		if err != nil {
			return nil, err
		}
	}

	return p.formatter.(formatter.TextSearchFormatter[bson.M]).FormatMixedQuery(fieldBSON, textTerms)
}

// parseTextOnlyQuery handles text-only queries
func (p *Parser) parseTextOnlyQuery(query string, textSearchParser language.TextSearchParser) (bson.M, error) {
	textTerms, err := textSearchParser.ParseTextSearch(query)
	if err != nil {
		return nil, err
	}

	return p.formatter.(formatter.TextSearchFormatter[bson.M]).FormatTextSearch(textTerms)
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
