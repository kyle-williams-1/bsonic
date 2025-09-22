// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenField
	TokenValue
	TokenAND
	TokenOR
	TokenNOT
	TokenLParen
	TokenRParen
	TokenTextSearch
)

// Logical operator definitions for tokenization
type operatorDef struct {
	pattern   string
	tokenType TokenType
}

// All logical operators (AND, OR, NOT) are treated uniformly during tokenization
// The distinction between binary/unary is handled during AST construction
var operators = []operatorDef{
	{" AND ", TokenAND}, // Binary: combines two conditions
	{" OR ", TokenOR},   // Binary: combines two conditions
	{" NOT ", TokenNOT}, // Unary: negates a condition (in middle of expression)
	{"NOT ", TokenNOT},  // Unary: negates a condition (at start of expression)
}

// Regex patterns for tokenization
var (
	// Pattern to match operators with spaces, end of string, or closing parentheses
	operatorRegex = regexp.MustCompile(`\s+(AND|OR|NOT)(?:\s+|$|\))`)
)

// Token represents a parsed token
type Token struct {
	Type  TokenType
	Value string
}

func (p *Parser) createToken(tokenType TokenType, value string) Token {
	return Token{Type: tokenType, Value: value}
}

func (p *Parser) hasOperators(part string) bool {
	upperPart := strings.ToUpper(part)
	trimmed := strings.TrimSpace(part)
	upperTrimmed := strings.ToUpper(trimmed)

	// Check for operators with spaces on both sides
	for _, op := range operators {
		if strings.Contains(upperPart, op.pattern) {
			return true
		}
	}

	// Check for operators at the end of strings
	operatorSuffixes := []string{" OR", " AND", " NOT"}
	for _, suffix := range operatorSuffixes {
		if strings.HasSuffix(upperTrimmed, suffix) {
			return true
		}
	}

	// Check for operators before closing parentheses (e.g., "(name:john AND)")
	// These are malformed queries that should be parsed as field queries, not text search
	operatorWithParen := []string{" OR)", " AND)", " NOT)"}
	for _, pattern := range operatorWithParen {
		if strings.Contains(upperTrimmed, pattern) {
			return true
		}
	}

	return false
}

func (p *Parser) findOperatorAtPosition(query string, pos int) (operatorDef, int) {
	remaining := query[pos:]
	upperRemaining := strings.ToUpper(remaining)

	for _, op := range operators {
		if strings.HasPrefix(upperRemaining, op.pattern) {
			return op, pos + len(op.pattern)
		}
	}

	return operatorDef{}, pos
}

// AST (Abstract Syntax Tree) node types
// An AST is a tree representation of the syntactic structure of the query.
// Each node represents a different type of operation or value in the query.
type NodeType int

const (
	NodeFieldValue NodeType = iota // Represents a field:value pair (e.g., "name:john")
	NodeAND                        // Represents an AND operation between multiple conditions
	NodeOR                         // Represents an OR operation between multiple conditions
	NodeNOT                        // Represents a NOT operation (negation)
	NodeGroup                      // Represents a parenthesized group of conditions
	NodeTextSearch                 // Represents a text search query (e.g., "search terms")
)

// AST node for query representation
// The AST allows us to represent complex nested queries in a tree structure
// that can be easily traversed and converted to MongoDB BSON format.
//
// Example: For the query "(name:john OR name:jane) AND age:25", the AST would be:
//
//	NodeAND
//	├── NodeGroup
//	│   └── NodeOR
//	│       ├── NodeFieldValue{Field: "name", Value: "john"}
//	│       └── NodeFieldValue{Field: "name", Value: "jane"}
//	└── NodeFieldValue{Field: "age", Value: 25}
type Node struct {
	Type     NodeType    // The type of operation this node represents
	Field    string      // The field name (only used for NodeFieldValue)
	Value    interface{} // The value to match (only used for NodeFieldValue)
	Children []*Node     // Child nodes (used for operations like AND, OR, NOT, Group)
}

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
	// First, validate that this looks like a proper field query
	if err := p.validateFieldQuery(query); err != nil {
		return nil, err
	}

	// Tokenize the query
	tokens, err := p.Tokenize(query)
	if err != nil {
		return nil, err
	}

	// Validate parentheses matching
	if err := p.validateParentheses(tokens); err != nil {
		return nil, err
	}

	// Parse tokens into an Abstract Syntax Tree (AST)
	// The AST represents the query structure as a tree of nodes,
	// making it easier to handle operator precedence and nested expressions.
	ast, _, err := p.parseExpression(tokens, 0)
	if err != nil {
		return nil, err
	}

	// Convert the AST to MongoDB BSON format
	// This traverses the tree and generates the appropriate BSON operators
	return p.astToBSON(ast), nil
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
	if p.hasOperators(trimmed) {
		return false
	}

	// If we get here, it's a simple text search query
	return true
}

// validateFieldQuery validates that a query looks like a proper field query when text search is disabled
func (p *Parser) validateFieldQuery(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	// Check if this looks like a text search query (no valid field:value pairs)
	// If so, suggest enabling text search
	if !p.HasValidFieldPairs(trimmed) {
		return fmt.Errorf("query '%s' appears to be a text search query but text search is disabled. Consider using NewWithTextSearch() or SetSearchMode(SearchModeText) to enable text search", trimmed)
	}

	return nil
}

// HasValidFieldPairs checks if a query contains valid field:value pairs and operators
func (p *Parser) HasValidFieldPairs(query string) bool {
	// Use the existing tokenization logic to properly parse the query
	tokens, err := p.Tokenize(query)
	if err != nil {
		return false
	}

	// Check if all tokens are valid (either field:value pairs or operators)
	for _, token := range tokens {
		if token.Type == TokenField {
			// Validate that this is actually a proper field:value pair
			if !p.IsValidFieldValuePair(token.Value) {
				return false
			}
		} else if token.Type != TokenAND && token.Type != TokenOR && token.Type != TokenNOT &&
			token.Type != TokenLParen && token.Type != TokenRParen {
			// Only allow field:value pairs, operators, and parentheses
			return false
		}
	}

	return len(tokens) > 0
}

// IsValidFieldValuePair checks if a string is a valid field:value pair
func (p *Parser) IsValidFieldValuePair(value string) bool {
	// Find the first colon (field:value separator)
	colonIndex := strings.Index(value, ":")
	if colonIndex == -1 {
		return false
	}

	field := strings.TrimSpace(value[:colonIndex])
	val := strings.TrimSpace(value[colonIndex+1:])

	// Both field and value must be non-empty
	if field == "" || val == "" {
		return false
	}

	// Field name must not contain spaces (it should be a single word)
	if strings.Contains(field, " ") {
		return false
	}

	// Value can contain colons (for dates/times) but should not start with a colon
	if strings.HasPrefix(val, ":") {
		return false
	}

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

	// Use the same tokenization logic as parseMixedQuery to properly detect mixed queries
	tokens, err := p.tokenizeMixedQuery(trimmed)
	if err != nil {
		return false
	}

	hasFieldTokens := false
	hasTextTokens := false

	for _, token := range tokens {
		switch token.Type {
		case TokenField:
			hasFieldTokens = true
		case TokenTextSearch:
			hasTextTokens = true
		}
	}

	return hasFieldTokens && hasTextTokens
}

// parseMixedQuery parses a mixed query containing both field searches and text search.
// This parses the query to identify field:value pairs vs text terms and combines them.
func (p *Parser) parseMixedQuery(query string) (bson.M, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return bson.M{}, nil
	}

	// Parse the query to separate field searches from text search terms
	fieldQueries, textTerms, err := p.separateFieldAndTextTerms(trimmed)
	if err != nil {
		return nil, err
	}

	var conditions []bson.M

	// Add field search conditions
	if len(fieldQueries) > 0 {
		fieldQuery := strings.Join(fieldQueries, " ")
		fieldBSON, err := p.parseFieldQuery(fieldQuery)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, fieldBSON)
	}

	// Add text search condition
	if len(textTerms) > 0 {
		textQuery := strings.Join(textTerms, " ")
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

// separateFieldAndTextTerms separates field:value pairs from standalone text terms in a mixed query.
func (p *Parser) separateFieldAndTextTerms(query string) ([]string, []string, error) {
	tokens, err := p.tokenizeMixedQuery(query)
	if err != nil {
		return nil, nil, err
	}

	separator := &querySeparator{
		fieldQueries: []string{},
		textTerms:    []string{},
	}

	separator.processTokens(tokens)
	return separator.fieldQueries, separator.textTerms, nil
}

// querySeparator helps separate field queries from text terms
type querySeparator struct {
	fieldQueries      []string
	textTerms         []string
	currentFieldQuery []string
	currentTextTerms  []string
}

// processTokens processes all tokens to separate field and text terms
func (qs *querySeparator) processTokens(tokens []Token) {
	for i, token := range tokens {
		qs.processToken(token)
		qs.flushIfLastToken(i, len(tokens))
	}
}

// processToken processes a single token
func (qs *querySeparator) processToken(token Token) {
	switch token.Type {
	case TokenField:
		qs.handleFieldToken(token)
	case TokenAND, TokenOR, TokenNOT:
		qs.handleOperatorToken(token)
	case TokenLParen, TokenRParen:
		qs.handleParenthesesToken(token)
	case TokenTextSearch:
		qs.handleTextSearchToken(token)
	}
}

// handleFieldToken processes a field:value token
func (qs *querySeparator) handleFieldToken(token Token) {
	qs.flushTextTerms()
	qs.currentFieldQuery = append(qs.currentFieldQuery, token.Value)
}

// handleOperatorToken processes logical operator tokens
func (qs *querySeparator) handleOperatorToken(token Token) {
	if len(qs.currentFieldQuery) > 0 {
		qs.currentFieldQuery = append(qs.currentFieldQuery, token.Value)
	} else if len(qs.currentTextTerms) > 0 {
		qs.currentTextTerms = append(qs.currentTextTerms, token.Value)
	}
}

// handleParenthesesToken processes parentheses tokens
func (qs *querySeparator) handleParenthesesToken(token Token) {
	if len(qs.currentFieldQuery) > 0 {
		qs.currentFieldQuery = append(qs.currentFieldQuery, token.Value)
	} else if len(qs.currentTextTerms) > 0 {
		qs.currentTextTerms = append(qs.currentTextTerms, token.Value)
	}
}

// handleTextSearchToken processes text search tokens
func (qs *querySeparator) handleTextSearchToken(token Token) {
	qs.flushFieldQuery()
	qs.currentTextTerms = append(qs.currentTextTerms, token.Value)
}

// flushTextTerms flushes accumulated text terms
func (qs *querySeparator) flushTextTerms() {
	if len(qs.currentTextTerms) > 0 {
		qs.textTerms = append(qs.textTerms, strings.Join(qs.currentTextTerms, " "))
		qs.currentTextTerms = nil
	}
}

// flushFieldQuery flushes accumulated field query
func (qs *querySeparator) flushFieldQuery() {
	if len(qs.currentFieldQuery) > 0 {
		qs.fieldQueries = append(qs.fieldQueries, strings.Join(qs.currentFieldQuery, " "))
		qs.currentFieldQuery = nil
	}
}

// flushIfLastToken flushes accumulated terms if this is the last token
func (qs *querySeparator) flushIfLastToken(index, totalTokens int) {
	if index == totalTokens-1 {
		qs.flushFieldQuery()
		qs.flushTextTerms()
	}
}

// tokenizeMixedQuery tokenizes a mixed query, identifying field:value pairs and text terms.
func (p *Parser) tokenizeMixedQuery(query string) ([]Token, error) {
	var tokens []Token
	query = strings.TrimSpace(query)

	// For mixed queries, we need to split by spaces first, then identify field:value pairs
	// This is different from regular field queries which need to preserve operator precedence
	parts := strings.Fields(query)

	for _, part := range parts {
		// Check if this part is an operator
		switch part {
		case "AND":
			tokens = append(tokens, Token{Type: TokenAND, Value: "AND"})
		case "OR":
			tokens = append(tokens, Token{Type: TokenOR, Value: "OR"})
		case "NOT":
			tokens = append(tokens, Token{Type: TokenNOT, Value: "NOT"})
		case "(":
			tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
		case ")":
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
		default:
			// This is either a field:value pair or a text term
			if strings.Contains(part, ":") {
				// Field:value pair
				colonIndex := strings.Index(part, ":")
				field := strings.TrimSpace(part[:colonIndex])
				value := strings.TrimSpace(part[colonIndex+1:])

				if field == "" || value == "" {
					return nil, errors.New("invalid field:value format in mixed query")
				}

				tokens = append(tokens, Token{Type: TokenField, Value: field + ":" + value})
			} else {
				// Text search term
				tokens = append(tokens, Token{Type: TokenTextSearch, Value: part})
			}
		}
	}

	return tokens, nil
}

// parseFieldQuery parses a field-only query (without text search terms).
func (p *Parser) parseFieldQuery(query string) (bson.M, error) {
	// Create a temporary parser with disabled text search to parse the field query
	tempParser := &Parser{SearchMode: SearchModeDisabled}
	return tempParser.Parse(query)
}

// validateParentheses checks if parentheses are properly matched
func (p *Parser) validateParentheses(tokens []Token) error {
	parenDepth := 0
	for _, token := range tokens {
		switch token.Type {
		case TokenLParen:
			parenDepth++
		case TokenRParen:
			parenDepth--
			if parenDepth < 0 {
				return errors.New("unmatched closing parenthesis")
			}
		}
	}
	if parenDepth > 0 {
		return errors.New("unmatched opening parenthesis")
	}
	return nil
}

// tokenize converts a query string into tokens
func (p *Parser) Tokenize(query string) ([]Token, error) {
	query = strings.TrimSpace(query)
	operatorPositions := p.findOperatorPositions(query)
	return p.buildTokensFromPositions(query, operatorPositions)
}

// findOperatorPositions finds all operator positions not inside parentheses
func (p *Parser) findOperatorPositions(query string) []int {
	var positions []int
	parenDepth := 0

	for i := 0; i < len(query); i++ {
		parenDepth = p.updateParenDepth(query[i], parenDepth)
		if parenDepth == 0 {
			if op, newPos := p.findOperatorAtPosition(query, i); op.pattern != "" {
				positions = append(positions, i)
				i = newPos - 1 // -1 because the loop will increment
			}
		}
	}
	return positions
}

// updateParenDepth updates parentheses depth based on character
func (p *Parser) updateParenDepth(char byte, currentDepth int) int {
	switch char {
	case '(':
		return currentDepth + 1
	case ')':
		return currentDepth - 1
	default:
		return currentDepth
	}
}

// buildTokensFromPositions builds tokens from query and operator positions
func (p *Parser) buildTokensFromPositions(query string, operatorPositions []int) ([]Token, error) {
	var tokens []Token
	lastPos := 0

	for _, pos := range operatorPositions {
		partTokens, err := p.processQueryPart(query[lastPos:pos])
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, partTokens...)

		// Add the operator
		op, _ := p.findOperatorAtPosition(query, pos)
		tokens = append(tokens, p.createToken(op.tokenType, strings.TrimSpace(op.pattern)))
		lastPos = pos + len(op.pattern)
	}

	// Add the last part
	partTokens, err := p.processQueryPart(query[lastPos:])
	if err != nil {
		return nil, err
	}
	tokens = append(tokens, partTokens...)

	return tokens, nil
}

// processQueryPart processes a part of the query and returns tokens
func (p *Parser) processQueryPart(part string) ([]Token, error) {
	part = strings.TrimSpace(part)
	if part == "" {
		return []Token{}, nil
	}
	return p.parsePart(part)
}

// parsePart parses a part of the query for field:value pairs and parentheses
func (p *Parser) parsePart(part string) ([]Token, error) {
	var tokens []Token

	// Check if this part contains operators (for nested parsing)
	if p.hasOperators(part) {
		// This part contains operators, so we need to parse it with a different approach
		// Split on operators and handle each part
		subParts := operatorRegex.Split(part, -1)
		subOperators := operatorRegex.FindAllString(part, -1)

		for i, subPart := range subParts {
			subPart = strings.TrimSpace(subPart)
			if subPart != "" {
				subTokens, err := p.parseSimplePart(subPart)
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, subTokens...)
			}

			// Add operator if present
			if i < len(subOperators) {
				op := strings.TrimSpace(subOperators[i])
				tokenType := p.getTokenTypeFromString(strings.ToUpper(op))
				tokens = append(tokens, p.createToken(tokenType, op))
			}
		}

		return tokens, nil
	}

	return p.parseSimplePart(part)
}

// getTokenTypeFromString converts operator string to token type
func (p *Parser) getTokenTypeFromString(op string) TokenType {
	switch op {
	case "AND":
		return TokenAND
	case "OR":
		return TokenOR
	case "NOT":
		return TokenNOT
	default:
		return TokenEOF
	}
}

// parseSimplePart parses a simple part without operators
func (p *Parser) parseSimplePart(part string) ([]Token, error) {
	var tokens []Token

	i := 0
	for i < len(part) {
		char := part[i]

		switch char {
		case ' ', '\t', '\n':
			i++
		case '(':
			tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
			i++
		case ')':
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
			i++
		default:
			// Check if this starts with NOT
			if p.isNotOperation(part[i:]) {
				tokens = append(tokens, Token{Type: TokenNOT, Value: "NOT"})
				i += 4 // Skip "NOT "
				continue
			}

			// Parse field:value pair
			field, value, newPos, err := p.parseFieldValuePair(part, i)
			if err != nil {
				return nil, err
			}

			tokens = append(tokens, Token{Type: TokenField, Value: field + ":" + value})
			i = newPos
		}
	}

	return tokens, nil
}

// isNotOperation checks if the remaining string starts with "NOT "
func (p *Parser) isNotOperation(remaining string) bool {
	return strings.HasPrefix(strings.ToUpper(remaining), "NOT ")
}

// parseFieldValuePair parses a field:value pair from the given position
func (p *Parser) parseFieldValuePair(part string, start int) (string, string, int, error) {
	// Find the first colon that separates field from value
	colonIndex := strings.Index(part[start:], ":")
	if colonIndex == -1 {
		return "", "", 0, errors.New("invalid query format: expected field:value")
	}

	colonIndex += start
	field := strings.TrimSpace(part[start:colonIndex])
	valueStart := colonIndex + 1

	// Validate field name (cannot be empty)
	if field == "" {
		return "", "", 0, errors.New("invalid query format: field name cannot be empty")
	}

	// Find the end of the value
	valueEnd, err := p.findValueEnd(part, valueStart)
	if err != nil {
		return "", "", 0, err
	}

	// Validate value (cannot be empty)
	value := strings.TrimSpace(part[valueStart:valueEnd])
	if value == "" {
		return "", "", 0, errors.New("invalid query format: value cannot be empty")
	}

	// Parse the value to validate it
	_, err = p.parseValue(p.unquote(value))
	if err != nil {
		return "", "", 0, err
	}

	return field, value, valueEnd, nil
}

// findValueEnd finds the end position of a value, handling quotes and brackets
func (p *Parser) findValueEnd(part string, valueStart int) (int, error) {
	valueEnd := valueStart
	inQuotes := false
	quoteChar := byte(0)
	inBrackets := false

	for valueEnd < len(part) {
		char := part[valueEnd]

		if p.shouldStartQuote(char, inQuotes, inBrackets) {
			inQuotes = true
			quoteChar = char
		} else if p.shouldEndQuote(char, inQuotes, quoteChar) {
			inQuotes = false
		} else if p.shouldStartBrackets(char, inQuotes) {
			inBrackets = true
		} else if p.shouldEndBrackets(char, inQuotes) {
			inBrackets = false
		} else if p.shouldEndValue(char, inQuotes, inBrackets) {
			// Only break on closing parentheses, not on spaces or opening parentheses
			// Spaces are allowed in values
			break
		}
		valueEnd++
	}

	return valueEnd, nil
}

// shouldStartQuote determines if a character should start a quoted string
func (p *Parser) shouldStartQuote(char byte, inQuotes, inBrackets bool) bool {
	return !inQuotes && !inBrackets && (char == '"' || char == '\'')
}

// shouldEndQuote determines if a character should end a quoted string
func (p *Parser) shouldEndQuote(char byte, inQuotes bool, quoteChar byte) bool {
	return inQuotes && char == quoteChar
}

// shouldStartBrackets determines if a character should start brackets
func (p *Parser) shouldStartBrackets(char byte, inQuotes bool) bool {
	return !inQuotes && char == '['
}

// shouldEndBrackets determines if a character should end brackets
func (p *Parser) shouldEndBrackets(char byte, inQuotes bool) bool {
	return !inQuotes && char == ']'
}

// shouldEndValue determines if a character should end the value parsing
func (p *Parser) shouldEndValue(char byte, inQuotes, inBrackets bool) bool {
	return !inQuotes && !inBrackets && char == ')'
}

// parseExpression parses tokens into an AST with proper operator precedence
// This is the entry point for building the Abstract Syntax Tree from tokens.
// The AST construction follows operator precedence: OR < AND < NOT
func (p *Parser) parseExpression(tokens []Token, start int) (*Node, int, error) {
	// Parse OR expressions (lowest precedence)
	return p.parseOrExpression(tokens, start)
}

// parseOrExpression handles OR operations in the AST
// OR has the lowest precedence, so it's parsed first
func (p *Parser) parseOrExpression(tokens []Token, start int) (*Node, int, error) {
	left, pos, err := p.parseAndExpression(tokens, start)
	if err != nil {
		return nil, 0, err
	}

	for pos < len(tokens) && tokens[pos].Type == TokenOR {
		pos++ // consume OR

		// Check if there's a valid right operand
		if pos >= len(tokens) {
			return nil, 0, errors.New("incomplete expression: OR operator missing right operand")
		}

		right, newPos, err := p.parseAndExpression(tokens, pos)
		if err != nil {
			return nil, 0, err
		}

		left = &Node{
			Type:     NodeOR,
			Children: []*Node{left, right},
		}
		pos = newPos
	}

	return left, pos, nil
}

// parseAndExpression handles AND operations in the AST
// AND has higher precedence than OR
func (p *Parser) parseAndExpression(tokens []Token, start int) (*Node, int, error) {
	left, pos, err := p.parseNotExpression(tokens, start)
	if err != nil {
		return nil, 0, err
	}

	for pos < len(tokens) && tokens[pos].Type == TokenAND {
		pos++ // consume AND

		// Check if there's a valid right operand
		if pos >= len(tokens) {
			return nil, 0, errors.New("incomplete expression: AND operator missing right operand")
		}

		right, newPos, err := p.parseNotExpression(tokens, pos)
		if err != nil {
			return nil, 0, err
		}

		left = &Node{
			Type:     NodeAND,
			Children: []*Node{left, right},
		}
		pos = newPos
	}

	return left, pos, nil
}

// parseNotExpression handles NOT operations in the AST
// NOT has the highest precedence
func (p *Parser) parseNotExpression(tokens []Token, start int) (*Node, int, error) {
	if start < len(tokens) && tokens[start].Type == TokenNOT {
		start++ // consume NOT
		expr, pos, err := p.parsePrimaryExpression(tokens, start)
		if err != nil {
			return nil, 0, err
		}

		return &Node{
			Type:     NodeNOT,
			Children: []*Node{expr},
		}, pos, nil
	}

	return p.parsePrimaryExpression(tokens, start)
}

// parsePrimaryExpression handles field:value pairs and parentheses in the AST
// This is the base case for parsing - handles individual field:value pairs
// and parenthesized groups that need to be parsed recursively
func (p *Parser) parsePrimaryExpression(tokens []Token, start int) (*Node, int, error) {
	if start >= len(tokens) {
		return nil, 0, errors.New("unexpected end of query")
	}

	token := tokens[start]

	if token.Type == TokenLParen {
		// Parse grouped expression
		expr, pos, err := p.parseExpression(tokens, start+1)
		if err != nil {
			return nil, 0, err
		}

		if pos >= len(tokens) || tokens[pos].Type != TokenRParen {
			return nil, 0, errors.New("unmatched opening parenthesis")
		}

		return &Node{
			Type:     NodeGroup,
			Children: []*Node{expr},
		}, pos + 1, nil
	}

	if token.Type == TokenField {
		// Parse field:value
		// Find the first colon to separate field from value
		colonIndex := strings.Index(token.Value, ":")
		if colonIndex == -1 {
			return nil, 0, errors.New("invalid field:value format")
		}

		field := token.Value[:colonIndex]
		valueStr := token.Value[colonIndex+1:]

		value, err := p.parseValue(p.unquote(valueStr))
		if err != nil {
			return nil, 0, err
		}

		return &Node{
			Type:  NodeFieldValue,
			Field: field,
			Value: value,
		}, start + 1, nil
	}

	return nil, 0, errors.New("unexpected token: " + token.Value)
}

// astToBSON converts an AST node to MongoDB BSON format
// This function recursively traverses the AST and generates the appropriate
// BSON operators ($and, $or, $ne, etc.) based on the node type
func (p *Parser) astToBSON(node *Node) bson.M {
	switch node.Type {
	case NodeFieldValue:
		return bson.M{node.Field: node.Value}
	case NodeAND:
		return p.handleAndNode(node)
	case NodeOR:
		return p.handleOrNode(node)
	case NodeNOT:
		return p.handleNotNode(node)
	case NodeGroup:
		return p.handleGroupNode(node)
	case NodeTextSearch:
		return p.HandleTextSearchNode(node)
	default:
		return bson.M{}
	}
}

// handleAndNode processes AND operations in the AST
func (p *Parser) handleAndNode(node *Node) bson.M {
	var andConditions []bson.M
	directFields := bson.M{}

	for _, child := range node.Children {
		childBSON := p.astToBSON(child)

		// Handle different types of child conditions
		if orClause, hasOr := childBSON["$or"]; hasOr {
			andConditions = append(andConditions, bson.M{"$or": orClause})
		} else if andClause, hasAnd := childBSON["$and"]; hasAnd {
			andConditions = append(andConditions, bson.M{"$and": andClause})
		} else if p.hasConflictingOperators(childBSON, directFields) {
			// This child has operators or field conflicts, add it as a separate condition
			andConditions = append(andConditions, childBSON)
		} else {
			// Merge direct fields
			for k, v := range childBSON {
				directFields[k] = v
			}
		}
	}

	return p.combineAndConditions(andConditions, directFields)
}

// handleOrNode processes OR operations in the AST
func (p *Parser) handleOrNode(node *Node) bson.M {
	var conditions []bson.M
	for _, child := range node.Children {
		conditions = append(conditions, p.astToBSON(child))
	}
	return bson.M{"$or": conditions}
}

// handleNotNode processes NOT operations in the AST
func (p *Parser) handleNotNode(node *Node) bson.M {
	if len(node.Children) != 1 {
		return bson.M{}
	}
	childBSON := p.astToBSON(node.Children[0])

	// Handle NOT with OR expressions using De Morgan's law
	if orClause, hasOr := childBSON["$or"]; hasOr {
		return p.negateOrExpression(orClause.([]bson.M))
	}

	// Handle NOT with AND expressions using De Morgan's law
	if andClause, hasAnd := childBSON["$and"]; hasAnd {
		return p.negateAndExpression(andClause.([]bson.M))
	}

	// For field:value pairs, negate each field
	return p.negateFieldValuePairs(childBSON)
}

// handleGroupNode processes parenthesized groups in the AST
func (p *Parser) handleGroupNode(node *Node) bson.M {
	if len(node.Children) != 1 {
		return bson.M{}
	}
	return p.astToBSON(node.Children[0])
}

// HandleTextSearchNode processes text search nodes in the AST
func (p *Parser) HandleTextSearchNode(node *Node) bson.M {
	if node.Value == nil {
		return bson.M{}
	}

	if p.SearchMode != SearchModeText {
		return bson.M{}
	}

	// Convert value to string for text search
	var searchTerm string
	switch v := node.Value.(type) {
	case string:
		searchTerm = v
	case int, int32, int64:
		searchTerm = fmt.Sprintf("%d", v)
	case float32, float64:
		searchTerm = fmt.Sprintf("%g", v)
	case bool:
		searchTerm = fmt.Sprintf("%t", v)
	default:
		searchTerm = fmt.Sprintf("%v", v)
	}

	return bson.M{"$text": bson.M{"$search": searchTerm}}
}

// hasConflictingOperators checks if a BSON condition has operators that would conflict with direct field merging
func (p *Parser) hasConflictingOperators(childBSON bson.M, directFields bson.M) bool {
	for field, v := range childBSON {
		if vMap, ok := v.(bson.M); ok {
			// Check for MongoDB query operators that would conflict with direct field merging
			// Only $or and $and are conflicting - $ne, $gt, $lt, etc. can be merged directly
			for key := range vMap {
				if key == "$or" || key == "$and" {
					return true
				}
			}
		}
		// Check if this field already exists in directFields
		if _, exists := directFields[field]; exists {
			return true
		}
	}
	return false
}

// combineAndConditions combines direct fields and other conditions for AND operations
func (p *Parser) combineAndConditions(andConditions []bson.M, directFields bson.M) bson.M {
	if len(directFields) > 0 && len(andConditions) > 0 {
		andConditions = append(andConditions, directFields)
		return bson.M{"$and": andConditions}
	} else if len(andConditions) > 0 {
		return bson.M{"$and": andConditions}
	} else {
		return directFields
	}
}

// negateOrExpression negates an OR expression using De Morgan's law: NOT (A OR B) = (NOT A) AND (NOT B)
func (p *Parser) negateOrExpression(orConditions []bson.M) bson.M {
	var negatedConditions []bson.M
	for _, condition := range orConditions {
		negatedCondition := bson.M{}
		for k, v := range condition {
			negatedCondition[k] = bson.M{"$ne": v}
		}
		negatedConditions = append(negatedConditions, negatedCondition)
	}
	return bson.M{"$and": negatedConditions}
}

// negateAndExpression negates an AND expression using De Morgan's law: NOT (A AND B) = (NOT A) OR (NOT B)
func (p *Parser) negateAndExpression(andConditions []bson.M) bson.M {
	var negatedConditions []bson.M
	for _, condition := range andConditions {
		negatedCondition := bson.M{}
		for k, v := range condition {
			negatedCondition[k] = bson.M{"$ne": v}
		}
		negatedConditions = append(negatedConditions, negatedCondition)
	}
	return bson.M{"$or": negatedConditions}
}

// negateFieldValuePairs negates field:value pairs by adding $ne operators
func (p *Parser) negateFieldValuePairs(childBSON bson.M) bson.M {
	result := bson.M{}
	for k, v := range childBSON {
		result[k] = bson.M{"$ne": v}
	}
	return result
}

// parseValue parses a value string, handling wildcards, dates, and special syntax
func (p *Parser) parseValue(valueStr string) (interface{}, error) {
	// Check for date range queries: [start TO end]
	if p.isDateRange(valueStr) {
		return p.parseDateRange(valueStr)
	}

	// Check for date comparison operators: >date, <date, >=date, <=date
	if p.isDateComparison(valueStr) {
		return p.parseDateComparison(valueStr)
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
	return strings.HasPrefix(valueStr, "[") &&
		strings.HasSuffix(valueStr, "]") &&
		strings.Contains(strings.ToUpper(valueStr), " TO ")
}

// isDateComparison checks if the value string is a date comparison operator
func (p *Parser) isDateComparison(valueStr string) bool {
	return strings.HasPrefix(valueStr, ">=") ||
		strings.HasPrefix(valueStr, "<=") ||
		strings.HasPrefix(valueStr, ">") ||
		strings.HasPrefix(valueStr, "<")
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

// unquote removes surrounding quotes if present
func (p *Parser) unquote(valueStr string) string {
	if len(valueStr) >= 2 && valueStr[0] == '"' && valueStr[len(valueStr)-1] == '"' {
		return valueStr[1 : len(valueStr)-1]
	}
	return valueStr
}
