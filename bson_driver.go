package bsonic

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grindlemire/go-lucene"
	"github.com/grindlemire/go-lucene/pkg/lucene/expr"
	"go.mongodb.org/mongo-driver/bson"
)

// BSONDriver is a custom driver that converts go-lucene expressions directly to BSON
// This approach bypasses the string rendering and directly builds BSON structures
type BSONDriver struct {
	searchMode SearchMode
}

// NewBSONDriver creates a new BSON driver
func NewBSONDriver(searchMode SearchMode) *BSONDriver {
	return &BSONDriver{
		searchMode: searchMode,
	}
}

// SetSearchMode sets the search mode for the driver
func (d *BSONDriver) SetSearchMode(mode SearchMode) {
	d.searchMode = mode
}

// RenderExpression renders a go-lucene expression directly to BSON
func (d *BSONDriver) RenderExpression(expr *expr.Expression) (bson.M, error) {
	if expr == nil {
		return bson.M{}, nil
	}

	// Handle text search mode for literal expressions
	if d.searchMode == SearchModeText && expr.Op == 11 { // Literal
		value := d.getValue(expr.Left)
		return bson.M{"$text": bson.M{"$search": value}}, nil
	}

	// Convert the expression directly to BSON
	return d.expressionToBSON(expr), nil
}

// expressionToBSON converts a go-lucene Expression to BSON
func (d *BSONDriver) expressionToBSON(expr *expr.Expression) bson.M {
	if expr == nil {
		return bson.M{}
	}

	switch expr.Op {
	case 1: // And
		return d.handleAndExpression(expr)
	case 2: // Or
		return d.handleOrExpression(expr)
	case 5: // Not
		return d.handleNotExpression(expr)
	case 3: // Equals
		return d.handleEqualsExpression(expr)
	case 4: // Like
		return d.handleLikeExpression(expr)
	case 6: // Range
		return d.handleRangeExpression(expr)
	case 14: // Greater
		return d.handleGreaterExpression(expr)
	case 15: // Less
		return d.handleLessExpression(expr)
	case 16: // GreaterEq
		return d.handleGreaterEqExpression(expr)
	case 17: // LessEq
		return d.handleLessEqExpression(expr)
	case 11: // Literal
		// Handle text search when search mode is enabled
		if d.searchMode == SearchModeText {
			return bson.M{"$text": bson.M{"$search": d.getValue(expr.Left)}}
		}
		return bson.M{}
	default:
		return bson.M{}
	}
}

// handleAndExpression processes AND operations
func (d *BSONDriver) handleAndExpression(expr *expr.Expression) bson.M {
	leftBSON := d.expressionToBSON(d.getExpression(expr.Left))
	rightBSON := d.expressionToBSON(d.getExpression(expr.Right))

	// Merge the BSON documents
	return d.mergeAndConditions(leftBSON, rightBSON)
}

// handleOrExpression processes OR operations
func (d *BSONDriver) handleOrExpression(expr *expr.Expression) bson.M {
	leftBSON := d.expressionToBSON(d.getExpression(expr.Left))
	rightBSON := d.expressionToBSON(d.getExpression(expr.Right))

	var conditions []bson.M
	conditions = append(conditions, leftBSON)
	conditions = append(conditions, rightBSON)

	return bson.M{"$or": conditions}
}

// handleNotExpression processes NOT operations
func (d *BSONDriver) handleNotExpression(expr *expr.Expression) bson.M {
	// Check if Left is a string that contains logical operators
	if leftStr, ok := expr.Left.(string); ok {
		// If the string contains logical operators, parse it as a separate expression
		if strings.Contains(strings.ToUpper(leftStr), " OR ") ||
			strings.Contains(strings.ToUpper(leftStr), " AND ") ||
			strings.Contains(strings.ToUpper(leftStr), " NOT ") {
			// Parse the string as a new expression
			childExpr, err := lucene.Parse(leftStr)
			if err == nil {
				childBSON := d.expressionToBSON(childExpr)
				return d.negateBSON(childBSON)
			}
		}
	}

	childBSON := d.expressionToBSON(d.getExpression(expr.Left))
	return d.negateBSON(childBSON)
}

// handleEqualsExpression processes field:value pairs
func (d *BSONDriver) handleEqualsExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)
	value := d.parseValue(d.getValue(expr.Right))

	return bson.M{field: value}
}

// handleLikeExpression processes wildcard queries
func (d *BSONDriver) handleLikeExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)
	pattern := d.getValue(expr.Right)

	// Convert wildcard pattern to regex
	regexPattern := d.wildcardToRegex(pattern)

	return bson.M{
		field: bson.M{
			"$regex":   regexPattern,
			"$options": "i",
		},
	}
}

// handleRangeExpression processes range queries [start TO end]
func (d *BSONDriver) handleRangeExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)

	// Check if this is a range boundary by examining the Right field
	// Since we can't access RangeBoundary directly, we'll check the structure
	if expr.Right != nil {
		// Try to extract range information from the expression
		rangeInfo := d.extractRangeInfo(expr.Right)
		if rangeInfo != nil {
			return d.handleRangeInfo(field, rangeInfo)
		}
	}

	// Fallback for string-based ranges
	rangeStr := d.getValue(expr.Right)
	return d.parseRangeString(field, rangeStr)
}

// RangeInfo represents range boundary information
type RangeInfo struct {
	Min       string
	Max       string
	Inclusive bool
}

// extractRangeInfo extracts range information from the expression right field
func (d *BSONDriver) extractRangeInfo(right any) *RangeInfo {
	// Try to handle RangeBoundary struct directly using reflection
	// Since we can't import the RangeBoundary type directly, we'll use reflection
	rightStr := fmt.Sprintf("%v", right)

	// Check if it looks like a RangeBoundary: &{value value bool}
	if strings.HasPrefix(rightStr, "&{") && strings.HasSuffix(rightStr, "}") {
		// Parse the RangeBoundary struct string representation
		// Format: &{2023-01-15 2023-01-16 true}
		rightStr = strings.Trim(rightStr, "&{}")
		parts := strings.Split(rightStr, " ")

		if len(parts) >= 3 {
			min := parts[0]
			max := parts[1]
			inclusive := parts[2] == "true"

			return &RangeInfo{
				Min:       min,
				Max:       max,
				Inclusive: inclusive,
			}
		}
	}

	// Fallback: try to parse as string format
	rightStr = d.getValue(right)

	// Check if it looks like a range: [min TO max] or (min TO max)
	if strings.HasPrefix(rightStr, "[") && strings.HasSuffix(rightStr, "]") {
		// Inclusive range
		rangeStr := strings.Trim(rightStr, "[]")
		parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
		if len(parts) == 2 {
			return &RangeInfo{
				Min:       strings.TrimSpace(parts[0]),
				Max:       strings.TrimSpace(parts[1]),
				Inclusive: true,
			}
		}
	} else if strings.HasPrefix(rightStr, "(") && strings.HasSuffix(rightStr, ")") {
		// Exclusive range
		rangeStr := strings.Trim(rightStr, "()")
		parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
		if len(parts) == 2 {
			return &RangeInfo{
				Min:       strings.TrimSpace(parts[0]),
				Max:       strings.TrimSpace(parts[1]),
				Inclusive: false,
			}
		}
	}

	return nil
}

// handleRangeInfo processes range information
func (d *BSONDriver) handleRangeInfo(field string, rangeInfo *RangeInfo) bson.M {
	result := bson.M{}

	// Handle minimum value
	if rangeInfo.Min != "" && rangeInfo.Min != "*" {
		minValue := d.parseValue(rangeInfo.Min)
		if rangeInfo.Inclusive {
			result["$gte"] = minValue
		} else {
			result["$gt"] = minValue
		}
	}

	// Handle maximum value
	if rangeInfo.Max != "" && rangeInfo.Max != "*" {
		maxValue := d.parseValue(rangeInfo.Max)
		if rangeInfo.Inclusive {
			result["$lte"] = maxValue
		} else {
			result["$lt"] = maxValue
		}
	}

	// If we have no conditions, return exists check
	if len(result) == 0 {
		return bson.M{field: bson.M{"$exists": true}}
	}

	return bson.M{field: result}
}

// parseRangeString parses string-based range queries like "[18 TO 65]"
func (d *BSONDriver) parseRangeString(field, rangeStr string) bson.M {
	// Remove brackets
	rangeStr = strings.Trim(rangeStr, "[]")

	// Split on " TO " (case insensitive)
	parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
	if len(parts) != 2 {
		return bson.M{field: bson.M{"$exists": true}}
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	result := bson.M{}

	// Handle start value
	if startStr != "*" {
		startValue := d.parseValue(startStr)
		result["$gte"] = startValue
	}

	// Handle end value
	if endStr != "*" {
		endValue := d.parseValue(endStr)
		result["$lte"] = endValue
	}

	// If we have no conditions, return exists check
	if len(result) == 0 {
		return bson.M{field: bson.M{"$exists": true}}
	}

	return bson.M{field: result}
}

// handleGreaterExpression processes > queries
func (d *BSONDriver) handleGreaterExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)
	value := d.parseValue(d.getValue(expr.Right))

	return bson.M{field: bson.M{"$gt": value}}
}

// handleLessExpression processes < queries
func (d *BSONDriver) handleLessExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)
	value := d.parseValue(d.getValue(expr.Right))

	return bson.M{field: bson.M{"$lt": value}}
}

// handleGreaterEqExpression processes >= queries
func (d *BSONDriver) handleGreaterEqExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)
	value := d.parseValue(d.getValue(expr.Right))

	return bson.M{field: bson.M{"$gte": value}}
}

// handleLessEqExpression processes <= queries
func (d *BSONDriver) handleLessEqExpression(expr *expr.Expression) bson.M {
	field := d.getValue(expr.Left)
	value := d.parseValue(d.getValue(expr.Right))

	return bson.M{field: bson.M{"$lte": value}}
}

// getExpression safely extracts an Expression from an interface{}
func (d *BSONDriver) getExpression(operand any) *expr.Expression {
	if expr, ok := operand.(*expr.Expression); ok {
		return expr
	}
	return nil
}

// getValue extracts the string value from an expression operand
func (d *BSONDriver) getValue(operand any) string {
	switch v := operand.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case *expr.Expression:
		return d.getValue(v.Left)
	case expr.Column:
		return string(v)
	default:
		// For RangeBoundary and other complex types, use reflection or string conversion
		return fmt.Sprintf("%v", v)
	}
}

// parseValue parses a string value to the appropriate type
func (d *BSONDriver) parseValue(valueStr string) interface{} {
	// Try to parse as a date
	if date, err := d.parseDate(valueStr); err == nil {
		return date
	}

	// Try to parse as a number
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num
	}

	// Try to parse as a boolean
	if valueStr == "true" {
		return true
	}
	if valueStr == "false" {
		return false
	}

	// Default to string
	return valueStr
}

// parseDate parses a date string in various formats
func (d *BSONDriver) parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
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

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// wildcardToRegex converts a wildcard pattern to a regex pattern
func (d *BSONDriver) wildcardToRegex(pattern string) string {
	// Escape special regex characters except * and ?
	escaped := regexp.QuoteMeta(pattern)

	// Replace escaped * with .*
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")

	// Replace escaped ? with .
	escaped = strings.ReplaceAll(escaped, "\\?", ".")

	// Add anchoring based on pattern
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// Contains pattern: *text* -> .*text.*
		return escaped
	} else if strings.HasPrefix(pattern, "*") {
		// Ends with pattern: *text -> .*text$
		return escaped + "$"
	} else if strings.HasSuffix(pattern, "*") {
		// Starts with pattern: text* -> ^text.*
		return "^" + escaped
	} else {
		// Exact match: text -> ^text$
		return "^" + escaped + "$"
	}
}

// mergeAndConditions merges two BSON conditions for AND operations
func (d *BSONDriver) mergeAndConditions(left, right bson.M) bson.M {
	var andConditions []bson.M
	directFields := bson.M{}

	// Process left condition
	if orClause, hasOr := left["$or"]; hasOr {
		andConditions = append(andConditions, bson.M{"$or": orClause})
	} else if andClause, hasAnd := left["$and"]; hasAnd {
		andConditions = append(andConditions, bson.M{"$and": andClause})
	} else if d.hasConflictingOperators(left, directFields) {
		andConditions = append(andConditions, left)
	} else {
		for k, v := range left {
			directFields[k] = v
		}
	}

	// Process right condition
	if orClause, hasOr := right["$or"]; hasOr {
		andConditions = append(andConditions, bson.M{"$or": orClause})
	} else if andClause, hasAnd := right["$and"]; hasAnd {
		andConditions = append(andConditions, bson.M{"$and": andClause})
	} else if d.hasConflictingOperators(right, directFields) {
		andConditions = append(andConditions, right)
	} else {
		for k, v := range right {
			directFields[k] = v
		}
	}

	// Combine conditions
	if len(directFields) > 0 && len(andConditions) > 0 {
		andConditions = append(andConditions, directFields)
		return bson.M{"$and": andConditions}
	} else if len(andConditions) > 0 {
		return bson.M{"$and": andConditions}
	} else {
		return directFields
	}
}

// hasConflictingOperators checks if a BSON condition has operators that would conflict with direct field merging
func (d *BSONDriver) hasConflictingOperators(condition bson.M, directFields bson.M) bool {
	for field, v := range condition {
		if vMap, ok := v.(bson.M); ok {
			for key := range vMap {
				if key == "$or" || key == "$and" {
					return true
				}
			}
		}
		if _, exists := directFields[field]; exists {
			return true
		}
	}
	return false
}

// negateBSON negates a BSON condition
func (d *BSONDriver) negateBSON(condition bson.M) bson.M {
	result := bson.M{}
	for k, v := range condition {
		// Handle special operators at the top level
		if k == "$or" {
			// NOT (A OR B) = (NOT A) AND (NOT B)
			if orClause, ok := v.([]bson.M); ok {
				var negatedConditions []bson.M
				for _, orCondition := range orClause {
					negatedConditions = append(negatedConditions, d.negateBSON(orCondition))
				}
				result["$and"] = negatedConditions
			}
		} else if k == "$and" {
			// NOT (A AND B) = (NOT A) OR (NOT B)
			if andClause, ok := v.([]bson.M); ok {
				var negatedConditions []bson.M
				for _, andCondition := range andClause {
					negatedConditions = append(negatedConditions, d.negateBSON(andCondition))
				}
				result["$or"] = negatedConditions
			}
		} else if vMap, ok := v.(bson.M); ok {
			if orClause, hasOr := vMap["$or"]; hasOr {
				// NOT (A OR B) = (NOT A) AND (NOT B)
				var negatedConditions []bson.M
				for _, orCondition := range orClause.([]bson.M) {
					negatedConditions = append(negatedConditions, d.negateBSON(orCondition))
				}
				result["$and"] = negatedConditions
			} else if andClause, hasAnd := vMap["$and"]; hasAnd {
				// NOT (A AND B) = (NOT A) OR (NOT B)
				var negatedConditions []bson.M
				for _, andCondition := range andClause.([]bson.M) {
					negatedConditions = append(negatedConditions, d.negateBSON(andCondition))
				}
				result["$or"] = negatedConditions
			} else {
				result[k] = bson.M{"$ne": v}
			}
		} else {
			result[k] = bson.M{"$ne": v}
		}
	}
	return result
}
