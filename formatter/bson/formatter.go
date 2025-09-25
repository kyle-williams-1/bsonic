// Package bson provides BSON formatting functionality for query results.
package bson

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kyle-williams-1/bsonic/formatter"
	"github.com/kyle-williams-1/bsonic/language/lucene"
	"go.mongodb.org/mongo-driver/bson"
)

// Formatter represents a BSON formatter for query results.
type Formatter struct{}

// New creates a new BSON formatter instance.
func New() *Formatter {
	return &Formatter{}
}

// Ensure Formatter implements the generic interface
var _ formatter.Formatter[bson.M] = (*Formatter)(nil)

// Format converts a parsed query AST into a BSON document.
func (f *Formatter) Format(ast interface{}) (bson.M, error) {
	// Type assert to the ParticipleQuery AST type from the Lucene parser
	participleQuery, ok := ast.(*lucene.ParticipleQuery)
	if !ok {
		return bson.M{}, fmt.Errorf("expected *lucene.ParticipleQuery AST, got %T", ast)
	}

	if participleQuery.Expression == nil {
		return bson.M{}, nil
	}
	return f.expressionToBSON(participleQuery.Expression), nil
}

// parseValue parses a value string, handling wildcards, dates, and special syntax
func (f *Formatter) parseValue(valueStr string) (interface{}, error) {
	// Create a chain of value parsers
	parsers := []func(string) (interface{}, error, bool){
		f.tryParseRange,
		f.tryParseComparison,
		f.tryParseWildcard,
		f.tryParseDate,
		f.tryParseNumber,
		f.tryParseBoolean,
	}

	for _, parser := range parsers {
		if result, err, handled := parser(valueStr); handled {
			return result, err
		}
	}

	// Default: return as string
	return valueStr, nil
}

// tryParseRange attempts to parse a range value
func (f *Formatter) tryParseRange(valueStr string) (interface{}, error, bool) {
	if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") && strings.Contains(strings.ToUpper(valueStr), " TO ") {
		result, err := f.parseRange(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseComparison attempts to parse a comparison value
func (f *Formatter) tryParseComparison(valueStr string) (interface{}, error, bool) {
	if strings.HasPrefix(valueStr, ">=") || strings.HasPrefix(valueStr, "<=") || strings.HasPrefix(valueStr, ">") || strings.HasPrefix(valueStr, "<") {
		result, err := f.parseComparison(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseWildcard attempts to parse a wildcard value
func (f *Formatter) tryParseWildcard(valueStr string) (interface{}, error, bool) {
	if strings.Contains(valueStr, "*") {
		result, err := f.parseWildcard(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseDate attempts to parse a date value
func (f *Formatter) tryParseDate(valueStr string) (interface{}, error, bool) {
	if date, err := f.parseDate(valueStr); err == nil {
		return date, nil, true
	}
	return nil, nil, false
}

// tryParseNumber attempts to parse a number value
func (f *Formatter) tryParseNumber(valueStr string) (interface{}, error, bool) {
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil, true
	}
	return nil, nil, false
}

// tryParseBoolean attempts to parse a boolean value
func (f *Formatter) tryParseBoolean(valueStr string) (interface{}, error, bool) {
	if valueStr == "true" || valueStr == "false" {
		return valueStr == "true", nil, true
	}
	return nil, nil, false
}

// parseRange parses range queries like [start TO end] for both dates and numbers
func (f *Formatter) parseRange(valueStr string) (interface{}, error) {
	rangeStr := strings.Trim(valueStr, "[]")
	parts := strings.Split(strings.ToUpper(rangeStr), " TO ")
	if len(parts) != 2 {
		return nil, errors.New("invalid range format: expected [start TO end]")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	if f.isDateLike(startStr) || f.isDateLike(endStr) {
		return f.parseDateRange(startStr, endStr)
	}

	return f.parseNumberRange(startStr, endStr)
}

// parseComparison parses comparison operators like >value, <value, >=value, <=value
func (f *Formatter) parseComparison(valueStr string) (interface{}, error) {
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

	if f.isDateLike(value) {
		date, err := f.parseDate(value)
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
func (f *Formatter) isDateLike(s string) bool {
	if s == "*" {
		return false
	}
	return strings.Contains(s, "-") || strings.Contains(s, "/") ||
		strings.Contains(s, ":") || strings.Contains(s, " ") ||
		strings.Contains(s, "T")
}

// parseWildcard parses a wildcard pattern and returns a regex BSON query
func (f *Formatter) parseWildcard(valueStr string) (bson.M, error) {
	pattern := strings.ReplaceAll(valueStr, "*", ".*")

	// Add proper anchoring based on wildcard position
	if f.isContainsPattern(valueStr) {
		// *J* - contains pattern
	} else if f.isEndsWithPattern(valueStr) {
		// *J - ends with pattern
		pattern = pattern + "$"
	} else if f.isStartsWithPattern(valueStr) {
		// J* - starts with pattern
		pattern = "^" + pattern
	} else {
		// J*K - starts and ends with specific patterns
		pattern = "^" + pattern + "$"
	}

	return bson.M{"$regex": pattern, "$options": "i"}, nil
}

// isContainsPattern checks if the pattern is a contains pattern (*J*)
func (f *Formatter) isContainsPattern(valueStr string) bool {
	return strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*")
}

// isEndsWithPattern checks if the pattern is an ends with pattern (*J)
func (f *Formatter) isEndsWithPattern(valueStr string) bool {
	return strings.HasPrefix(valueStr, "*") && !strings.HasSuffix(valueStr, "*")
}

// isStartsWithPattern checks if the pattern is a starts with pattern (J*)
func (f *Formatter) isStartsWithPattern(valueStr string) bool {
	return !strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*")
}

// parseDateRange parses date range queries
func (f *Formatter) parseDateRange(startStr, endStr string) (interface{}, error) {
	result := bson.M{}

	if startStr == "*" {
		if endStr == "*" {
			return nil, errors.New("invalid date range: both start and end cannot be wildcards")
		}
		endDate, err := f.parseDate(endStr)
		if err != nil {
			return nil, err
		}
		result["$lte"] = endDate
	} else {
		startDate, err := f.parseDate(startStr)
		if err != nil {
			return nil, err
		}
		result["$gte"] = startDate

		if endStr != "*" {
			endDate, err := f.parseDate(endStr)
			if err != nil {
				return nil, err
			}
			result["$lte"] = endDate
		}
	}

	return result, nil
}

// parseDate parses a date string in various formats
func (f *Formatter) parseDate(dateStr string) (time.Time, error) {
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
func (f *Formatter) parseNumberRange(startStr, endStr string) (interface{}, error) {
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

// expressionToBSON converts a ParticipleExpression to BSON
func (f *Formatter) expressionToBSON(expr *lucene.ParticipleExpression) bson.M {
	if len(expr.Or) == 0 {
		return bson.M{}
	}

	if len(expr.Or) == 1 {
		return f.andExpressionToBSON(expr.Or[0])
	}

	var conditions []bson.M
	for _, andExpr := range expr.Or {
		conditions = append(conditions, f.andExpressionToBSON(andExpr))
	}
	return bson.M{"$or": conditions}
}

// andExpressionToBSON converts a ParticipleAndExpression to BSON
func (f *Formatter) andExpressionToBSON(andExpr *lucene.ParticipleAndExpression) bson.M {
	if len(andExpr.And) == 0 {
		return bson.M{}
	}

	if len(andExpr.And) == 1 {
		return f.notExpressionToBSON(andExpr.And[0])
	}

	directFields, conditions := f.processAndExpressions(andExpr.And)
	return f.buildAndResult(directFields, conditions)
}

// processAndExpressions processes all AND expressions and separates simple fields from complex conditions
func (f *Formatter) processAndExpressions(expressions []*lucene.ParticipleNotExpression) (bson.M, []bson.M) {
	var conditions []bson.M
	directFields := bson.M{}
	hasComplexExpressions := false

	for _, notExpr := range expressions {
		childBSON := f.notExpressionToBSON(notExpr)

		if f.isSimpleFieldValue(childBSON) {
			if f.canMergeField(directFields, childBSON, hasComplexExpressions) {
				f.mergeField(directFields, childBSON)
			} else {
				conditions = append(conditions, childBSON)
			}
		} else {
			hasComplexExpressions = true
			conditions = append(conditions, childBSON)
		}
	}

	return directFields, conditions
}

// canMergeField checks if a field can be merged into directFields
func (f *Formatter) canMergeField(directFields bson.M, childBSON bson.M, hasComplexExpressions bool) bool {
	if hasComplexExpressions {
		return false
	}

	// Check for field conflicts
	for k := range childBSON {
		if _, exists := directFields[k]; exists {
			return false
		}
	}
	return true
}

// mergeField merges a simple field into directFields
func (f *Formatter) mergeField(directFields bson.M, childBSON bson.M) {
	for k, v := range childBSON {
		directFields[k] = v
	}
}

// buildAndResult builds the final result from directFields and conditions
func (f *Formatter) buildAndResult(directFields bson.M, conditions []bson.M) bson.M {
	if len(directFields) > 0 && len(conditions) > 0 {
		conditions = append(conditions, directFields)
		return bson.M{"$and": conditions}
	} else if len(conditions) > 0 {
		return bson.M{"$and": conditions}
	}
	return directFields
}

// notExpressionToBSON converts a ParticipleNotExpression to BSON
func (f *Formatter) notExpressionToBSON(notExpr *lucene.ParticipleNotExpression) bson.M {
	if notExpr.Not != nil {
		// Handle NOT operation
		childBSON := f.notExpressionToBSON(notExpr.Not)
		return f.negateBSON(childBSON)
	}

	return f.termToBSON(notExpr.Term)
}

// termToBSON converts a ParticipleTerm to BSON
func (f *Formatter) termToBSON(term *lucene.ParticipleTerm) bson.M {
	if term.FieldValue != nil {
		return f.fieldValueToBSON(term.FieldValue)
	}

	if term.FreeText != nil {
		return f.freeTextToBSON(term.FreeText)
	}

	if term.Group != nil {
		return f.expressionToBSON(term.Group.Expression)
	}

	return bson.M{}
}

// fieldValueToBSON converts a ParticipleFieldValue to BSON
func (f *Formatter) fieldValueToBSON(fv *lucene.ParticipleFieldValue) bson.M {
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

	value, err := f.parseValue(valueStr)
	if err != nil {
		value = valueStr
	}
	return bson.M{fv.Field: value}
}

// freeTextToBSON converts a ParticipleFreeText to BSON using MongoDB's $text search
func (f *Formatter) freeTextToBSON(ft *lucene.ParticipleFreeText) bson.M {
	var valueStr string
	if len(ft.Value.TextTerms) > 0 {
		valueStr = strings.Join(ft.Value.TextTerms, " ")
	} else if ft.Value.String != nil {
		valueStr = *ft.Value.String
	} else if ft.Value.SingleString != nil {
		valueStr = *ft.Value.SingleString
	} else if ft.Value.Bracketed != nil {
		valueStr = *ft.Value.Bracketed
	} else if ft.Value.DateTime != nil {
		valueStr = *ft.Value.DateTime
	} else if ft.Value.TimeString != nil {
		valueStr = *ft.Value.TimeString
	}

	// For free text search, we use MongoDB's $text operator
	// The value should be quoted for exact phrase matching
	return bson.M{"$text": bson.M{"$search": fmt.Sprintf("\"%s\"", valueStr)}}
}

// negateBSON negates a BSON condition using De Morgan's law
func (f *Formatter) negateBSON(condition bson.M) bson.M {
	if orClause, hasOr := condition["$or"]; hasOr {
		return bson.M{"$and": f.negateConditions(orClause.([]bson.M))}
	}

	if andClause, hasAnd := condition["$and"]; hasAnd {
		return bson.M{"$or": f.negateConditions(andClause.([]bson.M))}
	}

	result := bson.M{}
	for k, v := range condition {
		result[k] = bson.M{"$ne": v}
	}
	return result
}

// negateConditions negates a list of conditions by adding $ne operators
func (f *Formatter) negateConditions(conditions []bson.M) []bson.M {
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
func (f *Formatter) isSimpleFieldValue(condition bson.M) bool {
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

	// Check if this is a $text query (free text search)
	if _, hasText := condition["$text"]; hasText {
		return false
	}

	// Check if any field value contains complex operators
	for _, v := range condition {
		if vMap, ok := v.(bson.M); ok {
			for key := range vMap {
				if key == "$or" || key == "$and" || key == "$text" {
					return false
				}
			}
		}
	}
	return true
}
