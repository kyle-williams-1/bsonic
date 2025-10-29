// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"fmt"
	"strings"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/formatter"
	"github.com/kyle-williams-1/bsonic/formatter/mongo"
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
		return mongo.New(), nil
	default:
		return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
	}
}

// NewFormatterWithConfig creates a formatter based on the formatter type with config options.
func NewFormatterWithConfig(formatterType config.FormatterType, cfg *config.Config) (formatter.Formatter[bson.M], error) {
	switch formatterType {
	case config.FormatterMongo:
		return mongo.NewWithOptions(cfg.ReplaceIDWithMongoID, cfg.AutoConvertIDToObjectID), nil
	default:
		return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
	}
}

// NewMongoFormatter creates a MongoDB BSON formatter with proper typing.
func NewMongoFormatter() formatter.Formatter[bson.M] {
	return mongo.New()
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

	formatter, err := NewFormatterWithConfig(cfg.Formatter, cfg)
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

// ParseWithDefaults converts a query string into a BSON document using the provided default fields for unstructured queries.
// This function handles both structured queries (field:value pairs) and unstructured queries (free text).
// For unstructured queries, the free text is searched across all provided defaultFields using regex.
func ParseWithDefaults(defaultFields []string, query string) (bson.M, error) {
	if len(defaultFields) == 0 {
		return nil, fmt.Errorf("default fields cannot be empty")
	}

	// Create a parser with default fields configured
	cfg := config.Default().WithDefaultFields(defaultFields)
	parser, err := NewWithConfig(cfg)
	if err != nil {
		return nil, err
	}

	return parser.ParseWithDefaults(defaultFields, query)
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

	// Check if we have default fields configured
	if len(p.Config.DefaultFields) > 0 {
		// Use default fields for free text queries
		mongoFormatter, ok := p.formatter.(*mongo.MongoFormatter)
		if !ok {
			return nil, fmt.Errorf("formatter is not a MongoFormatter")
		}
		return mongoFormatter.FormatWithDefaults(ast, p.Config.DefaultFields)
	}

	// If no default fields are configured, return an error
	return nil, fmt.Errorf("no default fields are configured. Use ParseWithDefaults() or configure default fields in the parser config")
}

// ParseWithDefaults converts a query string into a BSON document using the provided default fields for unstructured queries.
// This method handles both structured queries (field:value pairs) and unstructured queries (free text).
// For unstructured queries, the free text is searched across all provided defaultFields using regex.
func (p *Parser) ParseWithDefaults(defaultFields []string, query string) (bson.M, error) {
	if len(defaultFields) == 0 {
		return nil, fmt.Errorf("default fields cannot be empty")
	}

	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// Parse the query and let the formatter handle it with default fields
	ast, err := p.languageParser.Parse(query)
	if err != nil {
		return nil, err
	}

	// Create a temporary formatter with the default fields
	mongoFormatter, ok := p.formatter.(*mongo.MongoFormatter)
	if !ok {
		return nil, fmt.Errorf("formatter is not a MongoFormatter")
	}

	// Always use default fields for ParseWithDefaults
	return mongoFormatter.FormatWithDefaults(ast, defaultFields)
}
