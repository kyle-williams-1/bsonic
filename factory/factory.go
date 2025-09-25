// Package factory provides factory functions for creating parsers and formatters.
package factory

import (
	"fmt"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/formatter"
	bsonformatter "github.com/kyle-williams-1/bsonic/formatter/bson"
	"github.com/kyle-williams-1/bsonic/language"
	"github.com/kyle-williams-1/bsonic/language/lucene"
	"go.mongodb.org/mongo-driver/bson"
)

// CreateParser creates a parser based on the language type.
func CreateParser(langType config.LanguageType) (language.Parser, error) {
	switch langType {
	case config.LanguageLucene:
		return lucene.New(), nil
	default:
		return nil, fmt.Errorf("unsupported language type: %s", langType)
	}
}

// CreateTextSearchParser creates a text search parser based on the language type.
func CreateTextSearchParser(langType config.LanguageType) (language.TextSearchParser, error) {
	switch langType {
	case config.LanguageLucene:
		return lucene.New(), nil
	default:
		return nil, fmt.Errorf("unsupported language type: %s", langType)
	}
}

// CreateFormatter creates a formatter based on the formatter type.
func CreateFormatter(formatterType config.FormatterType) (formatter.Formatter[bson.M], error) {
	switch formatterType {
	case config.FormatterBSON:
		return bsonformatter.New(), nil
	default:
		return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
	}
}

// CreateBSONFormatter creates a BSON formatter with proper typing.
func CreateBSONFormatter() formatter.Formatter[bson.M] {
	return bsonformatter.New()
}
