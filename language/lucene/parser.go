// Package lucene provides Lucene-style syntax parsing functionality.
package lucene

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/kyle-williams-1/bsonic/language"
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
	FreeText   *ParticipleFreeText   `| @@`
	Group      *ParticipleGroup      `| @@`
}

// ParticipleFieldValue represents field:value pairs
type ParticipleFieldValue struct {
	Field string           `@TextTerm ":"`
	Value *ParticipleValue `@@`
}

// ParticipleFreeText represents free text search queries (quoted strings without field names)
type ParticipleFreeText struct {
	Value *ParticipleQuotedValue `@@`
}

// ParticipleQuotedValue represents only quoted values for free text search
type ParticipleQuotedValue struct {
	String       *string `@String`
	SingleString *string `| @SingleString`
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
