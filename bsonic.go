// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
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

// Parser represents a Lucene-style query parser for MongoDB BSON filters.
type Parser struct {
	// SearchMode determines the type of search to perform
	SearchMode SearchMode
}

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

// New creates a new BSON parser instance.
func New() *Parser {
	return &Parser{
		SearchMode: SearchModeDisabled,
	}
}

// NewWithTextSearch creates a new BSON parser instance with text search enabled.
func NewWithTextSearch() *Parser {
	return &Parser{
		SearchMode: SearchModeText,
	}
}

// SetSearchMode sets the search mode for the parser.
func (p *Parser) SetSearchMode(mode SearchMode) {
	p.SearchMode = mode
}

// Parse converts a Lucene-style query string into a BSON document.
// This is the recommended way to parse queries for most use cases.
func Parse(query string) (bson.M, error) {
	parser := &Parser{}
	return parser.Parse(query)
}

// Parse converts a Lucene-style query string into a BSON document.
func (p *Parser) Parse(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// If text search is enabled, handle all query types appropriately
	if p.SearchMode == SearchModeText {
		// Check if this is a mixed query (field searches + text search) first
		if p.isMixedQuery(query) {
			return p.parseMixedQuery(query)
		}

		// Check if this should be a text search query (no field:value pairs)
		if p.shouldUseTextSearch(query) {
			return p.parseTextSearch(query)
		}

		// If we get here, it's a pure field search with text search enabled
		// Parse it as a regular field query
		return p.parseFieldQuery(query)
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

// isMixedQuery determines if a query contains both field searches and text search terms.
func (p *Parser) isMixedQuery(query string) bool {
	if p.SearchMode != SearchModeText {
		return false
	}

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

// parseMixedQuery parses a mixed query containing both field searches and text search.
func (p *Parser) parseMixedQuery(query string) (bson.M, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return bson.M{}, nil
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

	var conditions []bson.M

	if len(fieldParts) > 0 {
		fieldQuery := strings.Join(fieldParts, " ")
		fieldBSON, err := p.parseFieldQuery(fieldQuery)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, fieldBSON)
	}

	if len(textParts) > 0 {
		textQuery := strings.Join(textParts, " ")
		conditions = append(conditions, bson.M{"$text": bson.M{"$search": textQuery}})
	}

	if len(conditions) == 0 {
		return bson.M{}, nil
	} else if len(conditions) == 1 {
		return conditions[0], nil
	}
	return bson.M{"$and": conditions}, nil
}

// parseFieldQuery parses a field-only query (without text search terms).
func (p *Parser) parseFieldQuery(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	if p.SearchMode != SearchModeText {
		if err := p.validateFieldQuery(query); err != nil {
			return nil, err
		}
	}

	ast, err := participleParser.ParseString("", query)
	if err != nil {
		return nil, err
	}

	return p.participleASTToBSON(ast), nil
}

// participleASTToBSON converts a Participle AST to MongoDB BSON format
func (p *Parser) participleASTToBSON(query *ParticipleQuery) bson.M {
	if query.Expression == nil {
		return bson.M{}
	}
	return p.participleExpressionToBSON(query.Expression)
}

// participleExpressionToBSON converts a ParticipleExpression to BSON
func (p *Parser) participleExpressionToBSON(expr *ParticipleExpression) bson.M {
	if len(expr.Or) == 0 {
		return bson.M{}
	}

	if len(expr.Or) == 1 {
		return p.participleAndExpressionToBSON(expr.Or[0])
	}

	var conditions []bson.M
	for _, andExpr := range expr.Or {
		conditions = append(conditions, p.participleAndExpressionToBSON(andExpr))
	}
	return bson.M{"$or": conditions}
}

// participleAndExpressionToBSON converts a ParticipleAndExpression to BSON
func (p *Parser) participleAndExpressionToBSON(andExpr *ParticipleAndExpression) bson.M {
	if len(andExpr.And) == 0 {
		return bson.M{}
	}

	if len(andExpr.And) == 1 {
		return p.participleNotExpressionToBSON(andExpr.And[0])
	}

	var conditions []bson.M
	directFields := bson.M{}
	hasComplexExpressions := false

	for _, notExpr := range andExpr.And {
		childBSON := p.participleNotExpressionToBSON(notExpr)

		if p.isSimpleFieldValue(childBSON) {
			// Check for field conflicts
			hasConflict := false
			for k := range childBSON {
				if _, exists := directFields[k]; exists {
					hasConflict = true
					break
				}
			}

			if !hasConflict && !hasComplexExpressions {
				// Merge simple field:value pairs directly only if no complex expressions
				for k, v := range childBSON {
					directFields[k] = v
				}
			} else {
				conditions = append(conditions, childBSON)
			}
		} else {
			hasComplexExpressions = true
			conditions = append(conditions, childBSON)
		}
	}

	if len(directFields) > 0 && len(conditions) > 0 {
		conditions = append(conditions, directFields)
		return bson.M{"$and": conditions}
	} else if len(conditions) > 0 {
		return bson.M{"$and": conditions}
	} else {
		return directFields
	}
}

// participleNotExpressionToBSON converts a ParticipleNotExpression to BSON
func (p *Parser) participleNotExpressionToBSON(notExpr *ParticipleNotExpression) bson.M {
	if notExpr.Not != nil {
		// Handle NOT operation
		childBSON := p.participleNotExpressionToBSON(notExpr.Not)
		return p.negateBSON(childBSON)
	}

	return p.participleTermToBSON(notExpr.Term)
}

// participleTermToBSON converts a ParticipleTerm to BSON
func (p *Parser) participleTermToBSON(term *ParticipleTerm) bson.M {
	if term.FieldValue != nil {
		return p.participleFieldValueToBSON(term.FieldValue)
	}

	if term.Group != nil {
		return p.participleExpressionToBSON(term.Group.Expression)
	}

	if term.TextSearch != nil {
		if p.SearchMode == SearchModeText {
			return bson.M{"$text": bson.M{"$search": *term.TextSearch}}
		}
		// If text search is disabled, treat as invalid
		return bson.M{}
	}

	return bson.M{}
}

// participleFieldValueToBSON converts a ParticipleFieldValue to BSON
func (p *Parser) participleFieldValueToBSON(fv *ParticipleFieldValue) bson.M {
	var valueStr string
	if len(fv.Value.TextTerms) > 0 {
		valueStr = strings.Join(fv.Value.TextTerms, " ")
	} else if fv.Value.String != nil {
		valueStr = *fv.Value.String
	} else if fv.Value.SingleString != nil {
		valueStr = *fv.Value.SingleString
	} else if fv.Value.Bracketed != nil {
		valueStr = *fv.Value.Bracketed
	} else if fv.Value.DateTime != nil {
		valueStr = *fv.Value.DateTime
	} else if fv.Value.TimeString != nil {
		valueStr = *fv.Value.TimeString
	}

	value, err := p.parseValue(valueStr)
	if err != nil {
		value = valueStr
	}
	return bson.M{fv.Field: value}
}

// negateBSON negates a BSON condition using De Morgan's law
func (p *Parser) negateBSON(condition bson.M) bson.M {
	if orClause, hasOr := condition["$or"]; hasOr {
		return bson.M{"$and": p.negateConditions(orClause.([]bson.M))}
	}

	if andClause, hasAnd := condition["$and"]; hasAnd {
		return bson.M{"$or": p.negateConditions(andClause.([]bson.M))}
	}

	result := bson.M{}
	for k, v := range condition {
		result[k] = bson.M{"$ne": v}
	}
	return result
}

// negateConditions negates a list of conditions by adding $ne operators
func (p *Parser) negateConditions(conditions []bson.M) []bson.M {
	var result []bson.M
	for _, condition := range conditions {
		negated := bson.M{}
		for k, v := range condition {
			negated[k] = bson.M{"$ne": v}
		}
		result = append(result, negated)
	}
	return result
}

// isSimpleFieldValue checks if a BSON condition is a simple field:value pair
func (p *Parser) isSimpleFieldValue(condition bson.M) bool {
	if len(condition) != 1 {
		return false
	}

	// Check if the condition itself has complex operators
	if _, hasOr := condition["$or"]; hasOr {
		return false
	}
	if _, hasAnd := condition["$and"]; hasAnd {
		return false
	}

	// Check if any field value contains complex operators
	for _, v := range condition {
		if vMap, ok := v.(bson.M); ok {
			for key := range vMap {
				if key == "$or" || key == "$and" {
					return false
				}
			}
		}
	}
	return true
}

// parseValue parses a value string, handling wildcards, dates, and special syntax
func (p *Parser) parseValue(valueStr string) (interface{}, error) {
	if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") && strings.Contains(strings.ToUpper(valueStr), " TO ") {
		return p.parseRange(valueStr)
	}

	if strings.HasPrefix(valueStr, ">=") || strings.HasPrefix(valueStr, "<=") || strings.HasPrefix(valueStr, ">") || strings.HasPrefix(valueStr, "<") {
		return p.parseComparison(valueStr)
	}

	if strings.Contains(valueStr, "*") {
		return p.parseWildcard(valueStr)
	}

	if date, err := p.parseDate(valueStr); err == nil {
		return date, nil
	}

	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil
	}

	if valueStr == "true" || valueStr == "false" {
		return valueStr == "true", nil
	}

	return valueStr, nil
}

// parseRange parses range queries like [start TO end] for both dates and numbers
func (p *Parser) parseRange(valueStr string) (interface{}, error) {
	rangeStr := strings.Trim(valueStr, "[]")
	parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
	if len(parts) != 2 {
		return nil, errors.New("invalid range format: expected [start TO end]")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	if p.isDateLike(startStr) || p.isDateLike(endStr) {
		return p.parseDateRange(startStr, endStr)
	}

	return p.parseNumberRange(startStr, endStr)
}

// parseComparison parses comparison operators like >value, <value, >=value, <=value
func (p *Parser) parseComparison(valueStr string) (interface{}, error) {
	var operator string
	var value string

	if strings.HasPrefix(valueStr, ">=") {
		operator = "$gte"
		value = valueStr[2:]
	} else if strings.HasPrefix(valueStr, "<=") {
		operator = "$lte"
		value = valueStr[2:]
	} else if strings.HasPrefix(valueStr, ">") {
		operator = "$gt"
		value = valueStr[1:]
	} else if strings.HasPrefix(valueStr, "<") {
		operator = "$lt"
		value = valueStr[1:]
	} else {
		return nil, errors.New("invalid comparison operator")
	}

	value = strings.TrimSpace(value)

	if p.isDateLike(value) {
		date, err := p.parseDate(value)
		if err != nil {
			return nil, err
		}
		return bson.M{operator: date}, nil
	}

	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %v", err)
	}
	return bson.M{operator: num}, nil
}

// isDateLike checks if a string looks like a date
func (p *Parser) isDateLike(s string) bool {
	if s == "*" {
		return false
	}
	return strings.Contains(s, "-") || strings.Contains(s, "/") ||
		strings.Contains(s, ":") || strings.Contains(s, " ") ||
		strings.Contains(s, "T")
}

// parseWildcard parses a wildcard pattern and returns a regex BSON query
func (p *Parser) parseWildcard(valueStr string) (bson.M, error) {
	pattern := strings.ReplaceAll(valueStr, "*", ".*")

	// Add proper anchoring based on wildcard position
	if p.isContainsPattern(valueStr) {
		// *J* - contains pattern
	} else if p.isEndsWithPattern(valueStr) {
		// *J - ends with pattern
		pattern = pattern + "$"
	} else if p.isStartsWithPattern(valueStr) {
		// J* - starts with pattern
		pattern = "^" + pattern
	} else {
		// J*K - starts and ends with specific patterns
		pattern = "^" + pattern + "$"
	}

	return bson.M{"$regex": pattern, "$options": "i"}, nil
}

// isContainsPattern checks if the pattern is a contains pattern (*J*)
func (p *Parser) isContainsPattern(valueStr string) bool {
	return strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*")
}

// isEndsWithPattern checks if the pattern is an ends with pattern (*J)
func (p *Parser) isEndsWithPattern(valueStr string) bool {
	return strings.HasPrefix(valueStr, "*") && !strings.HasSuffix(valueStr, "*")
}

// isStartsWithPattern checks if the pattern is a starts with pattern (J*)
func (p *Parser) isStartsWithPattern(valueStr string) bool {
	return !strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*")
}

// parseDateRange parses date range queries
func (p *Parser) parseDateRange(startStr, endStr string) (interface{}, error) {
	result := bson.M{}

	if startStr == "*" {
		if endStr == "*" {
			return nil, errors.New("invalid date range: both start and end cannot be wildcards")
		}
		endDate, err := p.parseDate(endStr)
		if err != nil {
			return nil, err
		}
		result["$lte"] = endDate
	} else {
		startDate, err := p.parseDate(startStr)
		if err != nil {
			return nil, err
		}
		result["$gte"] = startDate

		if endStr != "*" {
			endDate, err := p.parseDate(endStr)
			if err != nil {
				return nil, err
			}
			result["$lte"] = endDate
		}
	}

	return result, nil
}

// parseDate parses a date string in various formats
func (p *Parser) parseDate(dateStr string) (time.Time, error) {
	if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return date, nil
	}

	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"01/02/2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, errors.New("unable to parse date: " + dateStr)
}

// parseNumberRange parses number range queries
func (p *Parser) parseNumberRange(startStr, endStr string) (interface{}, error) {
	result := bson.M{}

	if startStr == "*" {
		if endStr == "*" {
			return nil, errors.New("invalid number range: both start and end cannot be wildcards")
		}
		endNum, err := strconv.ParseFloat(endStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid end number: %v", err)
		}
		result["$lte"] = endNum
	} else {
		startNum, err := strconv.ParseFloat(startStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start number: %v", err)
		}
		result["$gte"] = startNum

		if endStr != "*" {
			endNum, err := strconv.ParseFloat(endStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid end number: %v", err)
			}
			result["$lte"] = endNum
		}
	}

	return result, nil
}

// validateFieldQuery validates that a field query doesn't contain standalone text terms when text search is disabled
func (p *Parser) validateFieldQuery(query string) error {
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
