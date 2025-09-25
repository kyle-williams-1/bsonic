// Package lucene provides Lucene-style syntax parsing functionality.
package lucene

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/kyle-williams-1/bsonic/language"
	"go.mongodb.org/mongo-driver/bson"
)

// Participle Grammar structures for Lucene-style queries

// ParticipleQuery is the root of the Participle AST
type ParticipleQuery struct {
	Expression *ParticipleExpression `@@`
}

// ParticipleExpression handles OR operations (lowest precedence)
type ParticipleExpression struct {
	Or []*ParticipleAndExpression `@@ ( "OR" @@ )*`
}

// ParticipleAndExpression handles AND operations (higher precedence than OR)
type ParticipleAndExpression struct {
	And []*ParticipleNotExpression `@@ ( "AND" @@ )*`
}

// ParticipleNotExpression handles NOT operations (highest precedence)
type ParticipleNotExpression struct {
	Not  *ParticipleNotExpression `"NOT" @@`
	Term *ParticipleTerm          `| @@`
}

// ParticipleTerm represents individual query terms
type ParticipleTerm struct {
	FieldValue *ParticipleFieldValue `@@`
	Group      *ParticipleGroup      `| @@`
	TextSearch *string               `| @TextTerm`
}

// ParticipleFieldValue represents field:value pairs
type ParticipleFieldValue struct {
	Field string           `@TextTerm ":"`
	Value *ParticipleValue `@@`
}

// ParticipleValue represents a value that can be a text term or quoted string
type ParticipleValue struct {
	TextTerms    []string `@TextTerm+`
	String       *string  `| @String`
	SingleString *string  `| @SingleString`
	Bracketed    *string  `| @Bracketed`
	DateTime     *string  `| @DateTime`
	TimeString   *string  `| @TimeString`
}

// ParticipleGroup represents parenthesized expressions
type ParticipleGroup struct {
	Expression *ParticipleExpression `"(" @@ ")"`
}

// Lexer definition for Lucene-style queries
var luceneLexer = lexer.MustSimple([]lexer.SimpleRule{
	// Whitespace
	{Name: "Whitespace", Pattern: `\s+`},
	// Logical operators
	{Name: "AND", Pattern: `AND`},
	{Name: "OR", Pattern: `OR`},
	{Name: "NOT", Pattern: `NOT`},
	// Parentheses
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	// Quoted strings - must come before TextTerm
	{Name: "String", Pattern: `"([^"\\]|\\.)*"`},
	// Single quoted strings - must come before TextTerm
	{Name: "SingleString", Pattern: `'([^'\\]|\\.)*'`},
	// Date ranges and other bracketed expressions
	{Name: "Bracketed", Pattern: `\[[^\]]+\]`},
	// Datetime strings with colons (ISO format, etc.)
	{Name: "DateTime", Pattern: `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`},
	// Time strings with colons
	{Name: "TimeString", Pattern: `\d{2}:\d{2}:\d{2}(\.\d+)?`},
	// Colon separator - must come after datetime patterns
	{Name: "Colon", Pattern: `:`},
	// Text terms (can be field names or values) - pattern includes wildcards
	{Name: "TextTerm", Pattern: `[^:\s\[\]()]+`},
})

// Parser instance using Participle
var participleParser = participle.MustBuild[ParticipleQuery](
	participle.Lexer(luceneLexer),
	participle.Unquote("String", "SingleString"),
	participle.UseLookahead(2),
	participle.Elide("Whitespace"),
)

// Parser represents a Lucene-style query parser.
type Parser struct{}

// New creates a new Lucene parser instance.
func New() *Parser {
	return &Parser{}
}

// Parse parses a Lucene-style query string into an AST.
func (p *Parser) Parse(query string) (language.AST, error) {
	return participleParser.ParseString("", query)
}

// IsMixedQuery determines if a query contains both field searches and text search terms.
func (p *Parser) IsMixedQuery(query string) bool {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return false
	}

	// Check if query contains field:value pairs
	hasFieldPairs := strings.Contains(trimmed, ":")
	if !hasFieldPairs {
		return false
	}

	// Simple check: if we have colons (field:value pairs) and the query is not just field:value pairs,
	// then it's likely a mixed query
	parts := strings.Fields(trimmed)
	hasTextTerms := false

	for _, part := range parts {
		if !strings.Contains(part, ":") && part != "AND" && part != "OR" && part != "NOT" && part != "(" && part != ")" {
			hasTextTerms = true
			break
		}
	}

	return hasFieldPairs && hasTextTerms
}

// ParseMixedQuery parses a mixed query containing both field searches and text search.
func (p *Parser) ParseMixedQuery(query string) (interface{}, string, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, "", nil
	}

	parts := strings.Fields(trimmed)
	var fieldParts []string
	var textParts []string

	for _, part := range parts {
		if strings.Contains(part, ":") || part == "AND" || part == "OR" || part == "NOT" || part == "(" || part == ")" {
			fieldParts = append(fieldParts, part)
		} else {
			textParts = append(textParts, part)
		}
	}

	var fieldAST interface{}
	var textTerms string

	if len(fieldParts) > 0 {
		fieldQuery := strings.Join(fieldParts, " ")
		ast, err := p.ParseFieldQuery(fieldQuery)
		if err != nil {
			return nil, "", err
		}
		fieldAST = ast
	}

	if len(textParts) > 0 {
		textTerms = strings.Join(textParts, " ")
	}

	return fieldAST, textTerms, nil
}

// ValidateFieldQuery validates that a field query doesn't contain standalone text terms when text search is disabled.
func (p *Parser) ValidateFieldQuery(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	if !strings.Contains(trimmed, ":") {
		words := strings.Fields(trimmed)
		for _, word := range words {
			if word != "AND" && word != "OR" && word != "NOT" && word != "(" && word != ")" {
				return fmt.Errorf("text search term '%s' found but text search is disabled", word)
			}
		}
	}

	return nil
}

// ParseFieldQuery parses a field-only query (without text search terms).
// This method returns the AST, which will be formatted by the main parser.
func (p *Parser) ParseFieldQuery(query string) (interface{}, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	ast, err := p.Parse(query)
	if err != nil {
		return nil, err
	}

	// Return the AST for the main parser to format
	return ast, nil
}
