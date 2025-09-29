// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"strings"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/formatter"
	"github.com/kyle-williams-1/bsonic/language"
	"github.com/kyle-williams-1/bsonic/registry"
	"go.mongodb.org/mongo-driver/bson"

	// Import packages to trigger their init functions
	_ "github.com/kyle-williams-1/bsonic/formatter/mongo"
	_ "github.com/kyle-williams-1/bsonic/language/lucene"
)

// Parser represents a query parser for the selected language and MongoDB formatter.
type Parser struct {
	// Config holds the language and formatter configuration
	Config *config.Config
	// Language parser instance
	languageParser language.Parser
	// Formatter instance (generic)
	formatter formatter.Formatter[bson.M]
}

// NewParser creates a parser based on the language type using the registry.
func NewParser(langType config.LanguageType) (language.Parser, error) {
	return registry.DefaultRegistry.Languages.GetLanguage(langType)
}

// NewFormatter creates a formatter based on the formatter type using the registry.
func NewFormatter(formatterType config.FormatterType) (formatter.Formatter[bson.M], error) {
	return registry.DefaultRegistry.Formatters.GetFormatter(formatterType)
}

// NewMongoFormatter creates a MongoDB BSON formatter with proper typing.
func NewMongoFormatter() formatter.Formatter[bson.M] {
	formatter, _ := NewFormatter(config.FormatterMongo)
	return formatter
}

// New creates a new parser instance with default configuration.
func New() *Parser {
	cfg := config.Default()
	languageParser, _ := NewParser(cfg.Language)
	formatter, _ := NewFormatter(cfg.Formatter)

	return &Parser{
		Config:         cfg,
		languageParser: languageParser,
		formatter:      formatter,
	}
}

// NewWithConfig creates a new parser with custom configuration.
func NewWithConfig(cfg *config.Config) (*Parser, error) {
	// Validate the configuration using the registry
	if err := registry.DefaultRegistry.ValidateConfig(cfg); err != nil {
		return nil, err
	}

	languageParser, err := NewParser(cfg.Language)
	if err != nil {
		return nil, err
	}

	formatter, err := NewFormatter(cfg.Formatter)
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
