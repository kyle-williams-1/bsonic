// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
package bsonic

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Token types for parsing
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
)

// Token represents a parsed token
type Token struct {
	Type  TokenType
	Value string
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

// Parser represents a Lucene-style query parser for MongoDB BSON filters.
type Parser struct{}

// New creates a new BSON parser instance.
func New() *Parser {
	return &Parser{}
}

// Parse converts a Lucene-style query string into a BSON document.
// The parsing process follows these steps:
// 1. Tokenize the query string into tokens (field:value pairs, operators, parentheses)
// 2. Validate that parentheses are properly matched
// 3. Build an Abstract Syntax Tree (AST) from the tokens with proper operator precedence
// 4. Convert the AST to MongoDB BSON format
func (p *Parser) Parse(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// Tokenize the query
	tokens, err := p.tokenize(query)
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
func (p *Parser) tokenize(query string) ([]Token, error) {
	var tokens []Token
	query = strings.TrimSpace(query)

	// Use a more sophisticated approach to handle parentheses
	// First, find all operators that are not inside parentheses
	var operatorPositions []int
	parenDepth := 0

	for i := 0; i < len(query); i++ {
		char := query[i]
		if char == '(' {
			parenDepth++
		} else if char == ')' {
			parenDepth--
		} else if parenDepth == 0 {
			// Check for operators at this position
			remaining := query[i:]
			if strings.HasPrefix(strings.ToUpper(remaining), " AND ") {
				operatorPositions = append(operatorPositions, i)
				i += 4 // Skip " AND"
			} else if strings.HasPrefix(strings.ToUpper(remaining), " OR ") {
				operatorPositions = append(operatorPositions, i)
				i += 3 // Skip " OR"
			} else if strings.HasPrefix(strings.ToUpper(remaining), " NOT ") {
				operatorPositions = append(operatorPositions, i)
				i += 4 // Skip " NOT"
			} else if strings.HasPrefix(strings.ToUpper(remaining), "NOT ") {
				operatorPositions = append(operatorPositions, i)
				i += 3 // Skip "NOT"
			}
		}
	}

	// Split the query at operator positions
	lastPos := 0
	for _, pos := range operatorPositions {
		part := strings.TrimSpace(query[lastPos:pos])
		if part != "" {
			partTokens, err := p.parsePart(part)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, partTokens...)
		}

		// Add the operator
		if strings.HasPrefix(strings.ToUpper(query[pos:]), " AND ") {
			tokens = append(tokens, Token{Type: TokenAND, Value: "AND"})
			lastPos = pos + 5
		} else if strings.HasPrefix(strings.ToUpper(query[pos:]), " OR ") {
			tokens = append(tokens, Token{Type: TokenOR, Value: "OR"})
			lastPos = pos + 4
		} else if strings.HasPrefix(strings.ToUpper(query[pos:]), " NOT ") {
			tokens = append(tokens, Token{Type: TokenNOT, Value: "NOT"})
			lastPos = pos + 5
		} else if strings.HasPrefix(strings.ToUpper(query[pos:]), "NOT ") {
			tokens = append(tokens, Token{Type: TokenNOT, Value: "NOT"})
			lastPos = pos + 4
		}
	}

	// Add the last part
	if lastPos < len(query) {
		part := strings.TrimSpace(query[lastPos:])
		if part != "" {
			partTokens, err := p.parsePart(part)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, partTokens...)
		}
	}

	return tokens, nil
}

// parsePart parses a part of the query for field:value pairs and parentheses
func (p *Parser) parsePart(part string) ([]Token, error) {
	var tokens []Token

	// Check if this part contains operators (for nested parsing)
	// We need to handle operators inside parentheses differently
	// Also check for operators at the end of strings or before closing parentheses
	trimmed := strings.TrimSpace(part)
	upperTrimmed := strings.ToUpper(trimmed)
	if strings.Contains(strings.ToUpper(part), " OR ") || strings.Contains(strings.ToUpper(part), " AND ") || strings.Contains(strings.ToUpper(part), " NOT ") ||
		strings.HasSuffix(upperTrimmed, " OR") || strings.HasSuffix(upperTrimmed, " AND") || strings.HasSuffix(upperTrimmed, " NOT") ||
		strings.Contains(upperTrimmed, " OR)") || strings.Contains(upperTrimmed, " AND)") || strings.Contains(upperTrimmed, " NOT)") {
		// This part contains operators, so we need to parse it with a different approach
		// Split on operators and handle each part
		// Updated regex to handle operators at the end of strings or before closing parentheses
		re := regexp.MustCompile(`\s+(AND|OR|NOT)(?:\s+|$|\))`)
		subParts := re.Split(part, -1)
		subOperators := re.FindAllString(part, -1)

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
				switch strings.ToUpper(op) {
				case "AND":
					tokens = append(tokens, Token{Type: TokenAND, Value: "AND"})
				case "OR":
					tokens = append(tokens, Token{Type: TokenOR, Value: "OR"})
				case "NOT":
					tokens = append(tokens, Token{Type: TokenNOT, Value: "NOT"})
				}
			}
		}

		return tokens, nil
	}

	return p.parseSimplePart(part)
}

// parseSimplePart parses a simple part without operators
func (p *Parser) parseSimplePart(part string) ([]Token, error) {
	var tokens []Token

	i := 0
	for i < len(part) {
		char := part[i]

		switch {
		case char == ' ' || char == '\t' || char == '\n':
			i++
		case char == '(':
			tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
			i++
		case char == ')':
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
			i++
		default:
			// Parse field:value
			// Find the first colon that separates field from value
			// The field name should not contain colons, but the value can
			colonIndex := strings.Index(part[i:], ":")
			if colonIndex == -1 {
				return nil, errors.New("invalid query format: expected field:value")
			}

			colonIndex += i
			field := strings.TrimSpace(part[i:colonIndex])
			valueStart := colonIndex + 1

			// Validate field name (cannot be empty)
			if field == "" {
				return nil, errors.New("invalid query format: field name cannot be empty")
			}

			// Find the end of the value
			valueEnd := valueStart
			inQuotes := false
			quoteChar := byte(0)
			inBrackets := false

			for valueEnd < len(part) {
				char := part[valueEnd]

				if !inQuotes && !inBrackets && (char == '"' || char == '\'') {
					inQuotes = true
					quoteChar = char
				} else if inQuotes && char == quoteChar {
					inQuotes = false
				} else if !inQuotes && char == '[' {
					inBrackets = true
				} else if !inQuotes && char == ']' {
					inBrackets = false
				} else if !inQuotes && !inBrackets && (char == '(' || char == ')') {
					// Only break on parentheses, not on spaces
					// Spaces are allowed in values
					break
				}
				valueEnd++
			}

			// Validate value (cannot be empty)
			value := strings.TrimSpace(part[valueStart:valueEnd])
			if value == "" {
				return nil, errors.New("invalid query format: value cannot be empty")
			}

			// Parse the value to validate it
			_, err := p.parseValue(p.unquote(value))
			if err != nil {
				return nil, err
			}

			tokens = append(tokens, Token{Type: TokenField, Value: field + ":" + value})
			i = valueEnd
		}
	}

	return tokens, nil
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
		var andConditions []bson.M
		directFields := bson.M{}

		for _, child := range node.Children {
			childBSON := p.astToBSON(child)

			// If child has $or, add it to andConditions
			if orClause, hasOr := childBSON["$or"]; hasOr {
				andConditions = append(andConditions, bson.M{"$or": orClause})
			} else {
				// Merge direct fields
				for k, v := range childBSON {
					directFields[k] = v
				}
			}
		}

		// If we have both direct fields and $or conditions, combine them
		if len(directFields) > 0 && len(andConditions) > 0 {
			andConditions = append(andConditions, directFields)
			return bson.M{"$and": andConditions}
		} else if len(andConditions) > 0 {
			return bson.M{"$and": andConditions}
		} else {
			return directFields
		}
	case NodeOR:
		var conditions []bson.M
		for _, child := range node.Children {
			conditions = append(conditions, p.astToBSON(child))
		}
		return bson.M{"$or": conditions}
	case NodeNOT:
		if len(node.Children) != 1 {
			return bson.M{}
		}
		childBSON := p.astToBSON(node.Children[0])

		// If the child is an OR expression, we need to negate each condition
		if orClause, hasOr := childBSON["$or"]; hasOr {
			var negatedConditions []bson.M
			for _, condition := range orClause.([]bson.M) {
				negatedCondition := bson.M{}
				for k, v := range condition {
					negatedCondition[k] = bson.M{"$ne": v}
				}
				negatedConditions = append(negatedConditions, negatedCondition)
			}
			return bson.M{"$and": []bson.M{{"$or": negatedConditions}}}
		}

		// For other cases, negate each field
		result := bson.M{}
		for k, v := range childBSON {
			result[k] = bson.M{"$ne": v}
		}
		return result
	case NodeGroup:
		if len(node.Children) != 1 {
			return bson.M{}
		}
		return p.astToBSON(node.Children[0])
	default:
		return bson.M{}
	}
}

// parseQuery handles field:value queries with AND/OR/NOT operators (legacy)
func (p *Parser) parseQuery(query string) (bson.M, error) {
	result := bson.M{}

	// Handle OR queries
	if strings.Contains(strings.ToUpper(query), " OR ") {
		orConditions, err := p.extractOrConditions(query)
		if err != nil {
			return nil, err
		}
		if len(orConditions) > 0 {
			result["$or"] = orConditions
		}

		// Extract AND conditions from OR query
		andConditions, err := p.extractAndConditions(query)
		if err != nil {
			return nil, err
		}
		for field, value := range andConditions {
			result[field] = value
		}

		// Extract NOT conditions from OR query
		_, notConditions, err := p.extractNotConditions(query)
		if err != nil {
			return nil, err
		}
		for field, value := range notConditions {
			result[field] = bson.M{"$ne": value}
		}
	} else if strings.Contains(strings.ToUpper(query), " NOT ") || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "NOT ") {
		// Handle NOT queries
		query, notConditions, err := p.extractNotConditions(query)
		if err != nil {
			return nil, err
		}

		// Add NOT conditions
		for field, value := range notConditions {
			result[field] = bson.M{"$ne": value}
		}

		// Parse remaining query for AND conditions
		if strings.TrimSpace(query) != "" {
			andConditions, err := p.extractAndConditions(query)
			if err != nil {
				return nil, err
			}
			for field, value := range andConditions {
				result[field] = value
			}
		}
	} else {
		// Handle simple AND queries or single field:value
		andConditions, err := p.extractAndConditions(query)
		if err != nil {
			return nil, err
		}
		for field, value := range andConditions {
			result[field] = value
		}
	}

	return result, nil
}

// extractNotConditions extracts NOT conditions from the query
func (p *Parser) extractNotConditions(query string) (string, bson.M, error) {
	notConditions := bson.M{}

	// Find NOT patterns: "NOT field:value" or "field:value AND NOT field2:value2"
	re := regexp.MustCompile(`(?:^|\s+)NOT\s+(\w+(?:\.\w+)*):([^\s]+(?:"[^"]*"|[^\s]+)*)`)
	matches := re.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		field := match[1]
		valueStr := strings.TrimSpace(match[2])

		value, err := p.parseValue(p.unquote(valueStr))
		if err != nil {
			return "", nil, err
		}

		notConditions[field] = value
	}

	// Remove NOT conditions from the original query
	cleanedQuery := re.ReplaceAllString(query, "")
	cleanedQuery = strings.TrimSpace(cleanedQuery)
	cleanedQuery = strings.TrimSuffix(cleanedQuery, " AND")

	return cleanedQuery, notConditions, nil
}

// extractOrConditions extracts OR conditions from the query
func (p *Parser) extractOrConditions(query string) ([]bson.M, error) {
	var orConditions []bson.M

	re := regexp.MustCompile(`\s+OR\s+`)
	parts := re.Split(query, -1)

	if len(parts) > 1 {
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Check if this part contains AND or NOT operators
			if strings.Contains(strings.ToUpper(part), " AND ") || strings.Contains(strings.ToUpper(part), " NOT ") {
				andIndex := strings.Index(strings.ToUpper(part), " AND ")
				if andIndex > 0 {
					fieldPart := strings.TrimSpace(part[:andIndex])
					field, value, err := p.parseFieldValue(fieldPart)
					if err == nil {
						orConditions = append(orConditions, bson.M{field: value})
					}
				}
				continue
			}

			field, value, err := p.parseFieldValue(part)
			if err != nil {
				return nil, err
			}

			orConditions = append(orConditions, bson.M{field: value})
		}
	}

	return orConditions, nil
}

// extractAndConditions extracts AND conditions and simple field:value pairs
func (p *Parser) extractAndConditions(query string) (bson.M, error) {
	result := bson.M{}

	re := regexp.MustCompile(`\s+AND\s+`)
	parts := re.Split(query, -1)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "AND" {
			continue
		}

		// Skip if this part contains OR or NOT
		if strings.Contains(strings.ToUpper(part), " OR ") || strings.Contains(strings.ToUpper(part), " NOT ") || strings.HasPrefix(strings.ToUpper(part), "NOT ") {
			continue
		}

		field, value, err := p.parseFieldValue(part)
		if err != nil {
			return nil, err
		}

		result[field] = value
	}

	return result, nil
}

// parseFieldValue parses a field:value pair
func (p *Parser) parseFieldValue(part string) (string, interface{}, error) {
	colonIndex := strings.Index(part, ":")
	if colonIndex == -1 {
		return "", nil, errors.New("invalid query format: expected field:value")
	}

	field := strings.TrimSpace(part[:colonIndex])
	valueStr := strings.TrimSpace(part[colonIndex+1:])

	if field == "" || valueStr == "" {
		return "", nil, errors.New("field and value cannot be empty")
	}

	value, err := p.parseValue(p.unquote(valueStr))
	if err != nil {
		return "", nil, err
	}

	return field, value, nil
}

// parseValue parses a value string, handling wildcards, dates, and special syntax
func (p *Parser) parseValue(valueStr string) (interface{}, error) {
	// Check for date range queries: [start TO end]
	if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") && strings.Contains(strings.ToUpper(valueStr), " TO ") {
		return p.parseDateRange(valueStr)
	}

	// Check for date comparison operators: >date, <date, >=date, <=date
	if strings.HasPrefix(valueStr, ">=") || strings.HasPrefix(valueStr, "<=") || strings.HasPrefix(valueStr, ">") || strings.HasPrefix(valueStr, "<") {
		return p.parseDateComparison(valueStr)
	}

	// Check for wildcards
	if strings.Contains(valueStr, "*") {
		pattern := strings.ReplaceAll(valueStr, "*", ".*")

		// Add proper anchoring based on wildcard position
		if strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*") {
			// *J* - contains pattern
		} else if strings.HasPrefix(valueStr, "*") {
			// *J - ends with pattern
			pattern = pattern + "$"
		} else if strings.HasSuffix(valueStr, "*") {
			// J* - starts with pattern
			pattern = "^" + pattern
		} else {
			// J*K - starts and ends with specific patterns
			pattern = "^" + pattern + "$"
		}

		return bson.M{"$regex": pattern, "$options": "i"}, nil
	}

	// Check if it's a date string
	if p.isDateString(valueStr) {
		date, err := p.parseDate(valueStr)
		if err != nil {
			return valueStr, nil
		}
		return date, nil
	}

	// Check if it's a number
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil
	}

	// Check if it's a boolean
	if valueStr == "true" {
		return true, nil
	}
	if valueStr == "false" {
		return false, nil
	}

	// Default to string
	return valueStr, nil
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

// isDateString checks if a string looks like a date
func (p *Parser) isDateString(valueStr string) bool {
	datePatterns := []string{
		`^\d{4}-\d{2}-\d{2}$`,
		`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`,
		`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$`,
		`^\d{4}/\d{2}/\d{2}$`,
		`^\d{2}/\d{2}/\d{4}$`,
	}

	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, valueStr); matched {
			return true
		}
	}

	return false
}

// unquote removes surrounding quotes if present
func (p *Parser) unquote(valueStr string) string {
	if len(valueStr) >= 2 && valueStr[0] == '"' && valueStr[len(valueStr)-1] == '"' {
		return valueStr[1 : len(valueStr)-1]
	}
	return valueStr
}
