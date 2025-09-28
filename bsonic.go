// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"fmt"
	"strings"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/formatter"
	mongoformatter "github.com/kyle-williams-1/bsonic/formatter/mongo"
	"github.com/kyle-williams-1/bsonic/language"
	"github.com/kyle-williams-1/bsonic/language/lucene"
	"go.mongodb.org/mongo-driver/bson"
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

// NewParser creates a parser based on the language type.
func NewParser(langType config.LanguageType) (language.Parser, error) {
	switch langType {
	case config.LanguageLucene:
		return lucene.New(), nil
	default:
		return nil, fmt.Errorf("unsupported language type: %s", langType)
	}
}

// NewFormatter creates a formatter based on the formatter type.
func NewFormatter(formatterType config.FormatterType) (formatter.Formatter[bson.M], error) {
	switch formatterType {
	case config.FormatterMongo:
		return mongoformatter.New(), nil
	default:
		return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
	}
}

// NewMongoFormatter creates a MongoDB BSON formatter with proper typing.
func NewMongoFormatter() formatter.Formatter[bson.M] {
	return mongoformatter.New()
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
