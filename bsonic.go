// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"strings"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/factory"
	"github.com/kyle-williams-1/bsonic/formatter"
	"github.com/kyle-williams-1/bsonic/language"
	"go.mongodb.org/mongo-driver/bson"
)

// Parser represents a query parser for MongoDB BSON filters.
type Parser struct {
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
		Config:         cfg,
		languageParser: languageParser,
		formatter:      formatter,
	}, nil
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

	// Parse the query and let the formatter handle it
	ast, err := p.languageParser.Parse(query)
	if err != nil {
		return nil, err
	}

	return p.formatter.Format(ast)
}
