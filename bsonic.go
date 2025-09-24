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
// The parsing process follows these steps:
// 1. Check if query should use text search or field searches
// 2. Tokenize the query string into tokens (field:value pairs, operators, parentheses)
// 3. Validate that parentheses are properly matched
// 4. Build an Abstract Syntax Tree (AST) from the tokens with proper operator precedence
// 5. Convert the AST to MongoDB BSON format
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
	return p.parseFieldQueryInternal(query)
}

// shouldUseTextSearch determines if a query should use text search instead of field searches.
// Text search is used when:
// 1. SearchMode is SearchModeText
// 2. The query contains no field:value pairs (no colons)
// 3. The query is not empty and contains search terms
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
// A mixed query contains both field:value pairs and standalone text terms.
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
// This parses the query to identify field:value pairs vs text terms and combines them.
func (p *Parser) parseMixedQuery(query string) (bson.M, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return bson.M{}, nil
	}

	// Simple approach: split the query into parts and separate field queries from text terms
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

	// Add field search conditions
	if len(fieldParts) > 0 {
		fieldQuery := strings.Join(fieldParts, " ")
		fieldBSON, err := p.parseFieldQuery(fieldQuery)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, fieldBSON)
	}

	// Add text search condition
	if len(textParts) > 0 {
		textQuery := strings.Join(textParts, " ")
		textBSON := bson.M{"$text": bson.M{"$search": textQuery}}
		conditions = append(conditions, textBSON)
	}

	// Combine conditions
	if len(conditions) == 0 {
		return bson.M{}, nil
	} else if len(conditions) == 1 {
		return conditions[0], nil
	} else {
		return bson.M{"$and": conditions}, nil
	}
}

// parseFieldQuery parses a field-only query (without text search terms).
func (p *Parser) parseFieldQuery(query string) (bson.M, error) {
	return p.parseFieldQueryInternal(query)
}

// parseFieldQueryInternal contains the core field parsing logic without SearchMode checks
func (p *Parser) parseFieldQueryInternal(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// If text search is disabled, validate that the query doesn't contain standalone text terms
	if p.SearchMode != SearchModeText {
		if err := p.validateFieldQuery(query); err != nil {
			return nil, err
		}
	}

	// Parse using Participle
	ast, err := participleParser.ParseString("", query)
	if err != nil {
		return nil, err
	}

	// Convert the Participle AST to MongoDB BSON format
	return p.participleASTToBSON(ast), nil
}

// Participle AST to BSON conversion methods

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

	// For multiple AND conditions, check if we have any complex expressions
	var andConditions []bson.M
	directFields := bson.M{}
	hasComplexExpressions := false

	for _, notExpr := range andExpr.And {
		childBSON := p.participleNotExpressionToBSON(notExpr)

		// Check if this is a complex expression (has $or, $and, etc.)
		if p.isComplexExpression(childBSON) {
			hasComplexExpressions = true
			andConditions = append(andConditions, childBSON)
		} else if p.isSimpleFieldValue(childBSON) {
			// Only merge simple field:value pairs if we don't have complex expressions
			if !hasComplexExpressions {
				// Check if we already have this field
				for k := range childBSON {
					if _, exists := directFields[k]; exists {
						// Field already exists, don't merge - add as separate condition
						andConditions = append(andConditions, childBSON)
						goto nextCondition
					}
				}
				// No conflict, merge the field
				for k, v := range childBSON {
					directFields[k] = v
				}
			} else {
				// If we have complex expressions, add simple fields as separate conditions
				andConditions = append(andConditions, childBSON)
			}
		} else {
			// Add as a separate condition
			andConditions = append(andConditions, childBSON)
		}
	nextCondition:
	}

	// Combine direct fields and other conditions
	if len(directFields) > 0 && len(andConditions) > 0 {
		andConditions = append(andConditions, directFields)
		return bson.M{"$and": andConditions}
	} else if len(andConditions) > 0 {
		return bson.M{"$and": andConditions}
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
	// Get the actual value string from the ParticipleValue
	var valueStr string
	if len(fv.Value.TextTerms) > 0 {
		// Join multiple text terms with spaces for values like "John Doe" or "San Francisco, CA"
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

	// Parse the value using the existing parseValue logic
	value, err := p.parseValue(valueStr)
	if err != nil {
		// If parsing fails, treat as string
		value = valueStr
	}
	return bson.M{fv.Field: value}
}

// negateBSON negates a BSON condition (used by Participle AST)
func (p *Parser) negateBSON(condition bson.M) bson.M {
	// Handle NOT with OR expressions using De Morgan's law
	if orClause, hasOr := condition["$or"]; hasOr {
		return p.negateOrExpression(orClause.([]bson.M))
	}

	// Handle NOT with AND expressions using De Morgan's law
	if andClause, hasAnd := condition["$and"]; hasAnd {
		return p.negateAndExpression(andClause.([]bson.M))
	}

	// For field:value pairs, negate each field
	return p.negateFieldValuePairs(condition)
}

// negateOrExpression negates an OR expression using De Morgan's law: NOT (A OR B) = (NOT A) AND (NOT B)
func (p *Parser) negateOrExpression(orConditions []bson.M) bson.M {
	return bson.M{"$and": p.negateConditions(orConditions)}
}

// negateAndExpression negates an AND expression using De Morgan's law: NOT (A AND B) = (NOT A) OR (NOT B)
func (p *Parser) negateAndExpression(andConditions []bson.M) bson.M {
	return bson.M{"$or": p.negateConditions(andConditions)}
}

// negateConditions negates a list of conditions by adding $ne operators to each field
func (p *Parser) negateConditions(conditions []bson.M) []bson.M {
	var negatedConditions []bson.M
	for _, condition := range conditions {
		negatedCondition := bson.M{}
		for k, v := range condition {
			negatedCondition[k] = bson.M{"$ne": v}
		}
		negatedConditions = append(negatedConditions, negatedCondition)
	}
	return negatedConditions
}

// negateFieldValuePairs negates field:value pairs by adding $ne operators
func (p *Parser) negateFieldValuePairs(childBSON bson.M) bson.M {
	result := bson.M{}
	for k, v := range childBSON {
		result[k] = bson.M{"$ne": v}
	}
	return result
}

// isSimpleFieldValue checks if a BSON condition is a simple field:value pair
func (p *Parser) isSimpleFieldValue(condition bson.M) bool {
	// Must have exactly one field
	if len(condition) != 1 {
		return false
	}

	// Check that the value is not a complex operator
	for _, v := range condition {
		if vMap, ok := v.(bson.M); ok {
			// If it's a map, check if it contains MongoDB operators
			for key := range vMap {
				// Allow $ne (NOT operations), $regex (wildcard operations), and range operators to be merged
				// but not other complex operators
				if key == "$or" || key == "$and" {
					return false
				}
			}
		}
	}

	return true
}

// isComplexExpression checks if a BSON condition is a complex expression (has $or, $and, etc.)
func (p *Parser) isComplexExpression(condition bson.M) bool {
	// Check if any field has complex operators
	for _, v := range condition {
		if vMap, ok := v.(bson.M); ok {
			// If it's a map, check if it contains complex operators
			for key := range vMap {
				if key == "$or" || key == "$and" {
					return true
				}
			}
		}
	}

	// Check if the condition itself has complex operators
	if _, hasOr := condition["$or"]; hasOr {
		return true
	}
	if _, hasAnd := condition["$and"]; hasAnd {
		return true
	}

	return false
}

// parseValue parses a value string, handling wildcards, dates, and special syntax
func (p *Parser) parseValue(valueStr string) (interface{}, error) {
	// Check for date range queries: [start TO end]
	if p.isDateRange(valueStr) {
		return p.parseDateRange(valueStr)
	}

	// Check for number range queries: [start TO end]
	if p.isNumberRange(valueStr) {
		return p.parseNumberRange(valueStr)
	}

	// Check for date comparison operators: >date, <date, >=date, <=date
	if p.isDateComparison(valueStr) {
		return p.parseDateComparison(valueStr)
	}

	// Check for number comparison operators: >5, <10, >=5, <=10
	if p.isNumberComparison(valueStr) {
		return p.parseNumberComparison(valueStr)
	}

	// Check for wildcards
	if strings.Contains(valueStr, "*") {
		return p.parseWildcard(valueStr)
	}

	// Try to parse as a date
	if date, err := p.parseDate(valueStr); err == nil {
		return date, nil
	}

	// Check if it's a number
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil
	}

	// Check if it's a boolean
	if p.isBoolean(valueStr) {
		return p.parseBoolean(valueStr)
	}

	// Default to string
	return valueStr, nil
}

// isDateRange checks if the value string is a date range query
func (p *Parser) isDateRange(valueStr string) bool {
	if !strings.HasPrefix(valueStr, "[") ||
		!strings.HasSuffix(valueStr, "]") ||
		!strings.Contains(strings.ToUpper(valueStr), " TO ") {
		return false
	}

	// Extract the range content and check if it contains date-like patterns
	rangeStr := strings.Trim(valueStr, "[]")
	parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
	if len(parts) != 2 {
		return false
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	// Check if either part looks like a date (contains dashes, slashes, or colons)
	// and is not a pure number
	hasDatePattern := func(s string) bool {
		if s == "*" {
			return false // wildcards don't indicate date type
		}
		// Check for date patterns: contains dashes, slashes, colons, or spaces
		return strings.Contains(s, "-") ||
			strings.Contains(s, "/") ||
			strings.Contains(s, ":") ||
			strings.Contains(s, " ") ||
			strings.Contains(s, "T") // ISO format
	}

	return hasDatePattern(startStr) || hasDatePattern(endStr)
}

// isDateComparison checks if the value string is a date comparison operator
func (p *Parser) isDateComparison(valueStr string) bool {
	if !strings.HasPrefix(valueStr, ">=") &&
		!strings.HasPrefix(valueStr, "<=") &&
		!strings.HasPrefix(valueStr, ">") &&
		!strings.HasPrefix(valueStr, "<") {
		return false
	}

	// Extract the value after the operator
	var dateStr string
	if strings.HasPrefix(valueStr, ">=") || strings.HasPrefix(valueStr, "<=") {
		dateStr = valueStr[2:]
	} else {
		dateStr = valueStr[1:]
	}
	dateStr = strings.TrimSpace(dateStr)

	// Check if the value looks like a date (contains dashes, slashes, or colons)
	return strings.Contains(dateStr, "-") ||
		strings.Contains(dateStr, "/") ||
		strings.Contains(dateStr, ":") ||
		strings.Contains(dateStr, " ") ||
		strings.Contains(dateStr, "T") // ISO format
}

// isNumberRange checks if the value string is a number range query
func (p *Parser) isNumberRange(valueStr string) bool {
	return strings.HasPrefix(valueStr, "[") &&
		strings.HasSuffix(valueStr, "]") &&
		strings.Contains(strings.ToUpper(valueStr), " TO ") &&
		!p.isDateRange(valueStr) // Make sure it's not a date range
}

// isNumberComparison checks if the value string is a number comparison operator
func (p *Parser) isNumberComparison(valueStr string) bool {
	return (strings.HasPrefix(valueStr, ">=") ||
		strings.HasPrefix(valueStr, "<=") ||
		strings.HasPrefix(valueStr, ">") ||
		strings.HasPrefix(valueStr, "<")) &&
		!p.isDateComparison(valueStr) // Make sure it's not a date comparison
}

// isBoolean checks if the value string is a boolean
func (p *Parser) isBoolean(valueStr string) bool {
	return valueStr == "true" || valueStr == "false"
}

// parseBoolean parses a boolean value
func (p *Parser) parseBoolean(valueStr string) (bool, error) {
	return valueStr == "true", nil
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

// parseDateRange parses date range queries like [2023-01-01 TO 2023-12-31] or [2023-01-01 TO *]
func (p *Parser) parseDateRange(valueStr string) (interface{}, error) {
	rangeStr := strings.Trim(valueStr, "[]")
	parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
	if len(parts) != 2 {
		return nil, errors.New("invalid date range format: expected [start TO end]")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	result := bson.M{}

	// Handle start date (or wildcard)
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

// parseDateComparison parses date comparison queries like >2024-01-01, <=2023-12-31
func (p *Parser) parseDateComparison(valueStr string) (interface{}, error) {
	var operator string
	var dateStr string

	if strings.HasPrefix(valueStr, ">=") {
		operator = "$gte"
		dateStr = valueStr[2:]
	} else if strings.HasPrefix(valueStr, "<=") {
		operator = "$lte"
		dateStr = valueStr[2:]
	} else if strings.HasPrefix(valueStr, ">") {
		operator = "$gt"
		dateStr = valueStr[1:]
	} else if strings.HasPrefix(valueStr, "<") {
		operator = "$lt"
		dateStr = valueStr[1:]
	} else {
		return nil, errors.New("invalid date comparison operator")
	}

	dateStr = strings.TrimSpace(dateStr)
	date, err := p.parseDate(dateStr)
	if err != nil {
		return nil, err
	}

	return bson.M{operator: date}, nil
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

// parseNumberRange parses number range queries like [1 TO 10] or [1 TO *]
func (p *Parser) parseNumberRange(valueStr string) (interface{}, error) {
	rangeStr := strings.Trim(valueStr, "[]")
	parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
	if len(parts) != 2 {
		return nil, errors.New("invalid number range format: expected [start TO end]")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	result := bson.M{}

	// Handle start number (or wildcard)
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

// parseNumberComparison parses number comparison queries like >5, <=10
func (p *Parser) parseNumberComparison(valueStr string) (interface{}, error) {
	var operator string
	var numStr string

	if strings.HasPrefix(valueStr, ">=") {
		operator = "$gte"
		numStr = valueStr[2:]
	} else if strings.HasPrefix(valueStr, "<=") {
		operator = "$lte"
		numStr = valueStr[2:]
	} else if strings.HasPrefix(valueStr, ">") {
		operator = "$gt"
		numStr = valueStr[1:]
	} else if strings.HasPrefix(valueStr, "<") {
		operator = "$lt"
		numStr = valueStr[1:]
	} else {
		return nil, errors.New("invalid number comparison operator")
	}

	numStr = strings.TrimSpace(numStr)
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %v", err)
	}

	return bson.M{operator: num}, nil
}

// validateFieldQuery validates that a field query doesn't contain standalone text terms when text search is disabled
func (p *Parser) validateFieldQuery(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	// Simple validation: if the query doesn't contain colons and doesn't look like operators/parentheses,
	// it's likely a standalone text term which should be rejected when text search is disabled
	if !strings.Contains(trimmed, ":") {
		// Check if it's just logical operators and parentheses
		words := strings.Fields(trimmed)
		for _, word := range words {
			if word != "AND" && word != "OR" && word != "NOT" && word != "(" && word != ")" {
				return fmt.Errorf("text search term '%s' found but text search is disabled", word)
			}
		}
	}

	return nil
}
