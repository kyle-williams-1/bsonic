// Package mongo provides MongoDB BSON formatting functionality for query results.
package mongo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kyle-williams-1/bsonic/language/lucene"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoFormatter represents a MongoDB BSON formatter for query results.
type MongoFormatter struct{}

// New creates a new MongoDB BSON formatter instance.
func New() *MongoFormatter {
	return &MongoFormatter{}
}

// Format converts a parsed query AST into a BSON document.
// This method handles structured queries only.
func (f *MongoFormatter) Format(ast interface{}) (bson.M, error) {
	// Type assert to the ParticipleQuery AST type from the Lucene parser
	participleQuery, ok := ast.(*lucene.ParticipleQuery)
	if !ok {
		return bson.M{}, fmt.Errorf("expected *lucene.ParticipleQuery AST, got %T", ast)
	}

	if participleQuery.Expression == nil {
		return bson.M{}, nil
	}
	return f.expressionToBSON(participleQuery.Expression, nil), nil
}

// Format converts a parsed query AST into a BSON document.
// This method handles both structured queries (field:value pairs) and unstructured queries (free text).
// For unstructured queries, the free text is searched across all provided defaultFields using regex.
func (f *MongoFormatter) FormatWithDefaults(ast interface{}, defaultFields []string) (bson.M, error) {
	// Type assert to the ParticipleQuery AST type from the Lucene parser
	participleQuery, ok := ast.(*lucene.ParticipleQuery)
	if !ok {
		return bson.M{}, fmt.Errorf("expected *lucene.ParticipleQuery AST, got %T", ast)
	}

	if participleQuery.Expression == nil {
		return bson.M{}, nil
	}
	return f.expressionToBSON(participleQuery.Expression, defaultFields), nil
}

// parseValue parses a value string, handling wildcards, dates, and special syntax
func (f *MongoFormatter) parseValue(valueStr string) (interface{}, error) {
	// Create a chain of value parsers
	parsers := []func(string) (interface{}, error, bool){
		f.tryParseRange,
		f.tryParseComparison,
		f.tryParseRegex,
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
func (f *MongoFormatter) tryParseRange(valueStr string) (interface{}, error, bool) {
	if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") && strings.Contains(strings.ToUpper(valueStr), " TO ") {
		result, err := f.parseRange(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseComparison attempts to parse a comparison value
func (f *MongoFormatter) tryParseComparison(valueStr string) (interface{}, error, bool) {
	if strings.HasPrefix(valueStr, ">=") || strings.HasPrefix(valueStr, "<=") || strings.HasPrefix(valueStr, ">") || strings.HasPrefix(valueStr, "<") {
		result, err := f.parseComparison(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseWildcard attempts to parse a wildcard value
func (f *MongoFormatter) tryParseWildcard(valueStr string) (interface{}, error, bool) {
	if strings.Contains(valueStr, "*") {
		result, err := f.parseWildcard(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseRegex attempts to parse a regex value
func (f *MongoFormatter) tryParseRegex(valueStr string) (interface{}, error, bool) {
	if strings.HasPrefix(valueStr, "/") && strings.HasSuffix(valueStr, "/") && len(valueStr) > 2 {
		result, err := f.parseRegex(valueStr)
		return result, err, true
	}
	return nil, nil, false
}

// tryParseDate attempts to parse a date value
func (f *MongoFormatter) tryParseDate(valueStr string) (interface{}, error, bool) {
	if date, err := f.parseDate(valueStr); err == nil {
		return date, nil, true
	}
	return nil, nil, false
}

// tryParseNumber attempts to parse a number value
func (f *MongoFormatter) tryParseNumber(valueStr string) (interface{}, error, bool) {
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil, true
	}
	return nil, nil, false
}

// tryParseBoolean attempts to parse a boolean value
func (f *MongoFormatter) tryParseBoolean(valueStr string) (interface{}, error, bool) {
	if valueStr == "true" || valueStr == "false" {
		return valueStr == "true", nil, true
	}
	return nil, nil, false
}

// parseRange parses range queries like [start TO end] for both dates and numbers
func (f *MongoFormatter) parseRange(valueStr string) (interface{}, error) {
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
func (f *MongoFormatter) parseComparison(valueStr string) (interface{}, error) {
	operator, value, err := f.extractOperatorAndValue(valueStr)
	if err != nil {
		return nil, err
	}

	value = strings.TrimSpace(value)

	if f.isDateLike(value) {
		return f.parseDateComparison(operator, value)
	}

	return f.parseNumberComparison(operator, value)
}

// extractOperatorAndValue extracts the operator and value from a comparison string
func (f *MongoFormatter) extractOperatorAndValue(valueStr string) (string, string, error) {
	comparisonOperators := []struct {
		prefix   string
		operator string
	}{
		{">=", "$gte"},
		{"<=", "$lte"},
		{">", "$gt"},
		{"<", "$lt"},
	}

	for _, op := range comparisonOperators {
		if strings.HasPrefix(valueStr, op.prefix) {
			return op.operator, valueStr[len(op.prefix):], nil
		}
	}

	return "", "", errors.New("invalid comparison operator")
}

// parseDateComparison parses a date comparison
func (f *MongoFormatter) parseDateComparison(operator, value string) (interface{}, error) {
	date, err := f.parseDate(value)
	if err != nil {
		return nil, err
	}
	return bson.M{operator: date}, nil
}

// parseNumberComparison parses a number comparison
func (f *MongoFormatter) parseNumberComparison(operator, value string) (interface{}, error) {
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %v", err)
	}
	return bson.M{operator: num}, nil
}

// isDateLike checks if a string looks like a date
func (f *MongoFormatter) isDateLike(s string) bool {
	if s == "*" {
		return false
	}
	return strings.Contains(s, "-") || strings.Contains(s, "/") ||
		strings.Contains(s, ":") || strings.Contains(s, " ") ||
		strings.Contains(s, "T")
}

// parseWildcard parses a wildcard pattern and returns a regex BSON query
func (f *MongoFormatter) parseWildcard(valueStr string) (bson.M, error) {
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

	return bson.M{"$regex": pattern}, nil
}

// isContainsPattern checks if the pattern is a contains pattern (*J*)
func (f *MongoFormatter) isContainsPattern(valueStr string) bool {
	return strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*")
}

// isEndsWithPattern checks if the pattern is an ends with pattern (*J)
func (f *MongoFormatter) isEndsWithPattern(valueStr string) bool {
	return strings.HasPrefix(valueStr, "*") && !strings.HasSuffix(valueStr, "*")
}

// isStartsWithPattern checks if the pattern is a starts with pattern (J*)
func (f *MongoFormatter) isStartsWithPattern(valueStr string) bool {
	return !strings.HasPrefix(valueStr, "*") && strings.HasSuffix(valueStr, "*")
}

// parseRegex parses a regex pattern and returns a regex BSON query
func (f *MongoFormatter) parseRegex(valueStr string) (bson.M, error) {
	// Remove the leading and trailing slashes
	pattern := valueStr[1 : len(valueStr)-1]

	// Add anchors for exact match if not already present
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}

	// Return as MongoDB regex query (case-sensitive by default)
	return bson.M{"$regex": pattern}, nil
}

// parseDateRange parses date range queries
func (f *MongoFormatter) parseDateRange(startStr, endStr string) (interface{}, error) {
	if err := f.validateDateRange(startStr, endStr); err != nil {
		return nil, err
	}

	if startStr == "*" {
		return f.parseDateRangeWithWildcardStart(endStr)
	}

	return f.parseDateRangeWithStart(startStr, endStr)
}

// validateDateRange validates that the date range is valid
func (f *MongoFormatter) validateDateRange(startStr, endStr string) error {
	if startStr == "*" && endStr == "*" {
		return errors.New("invalid date range: both start and end cannot be wildcards")
	}
	return nil
}

// parseDateRangeWithWildcardStart parses a date range with wildcard start
func (f *MongoFormatter) parseDateRangeWithWildcardStart(endStr string) (interface{}, error) {
	endDate, err := f.parseDate(endStr)
	if err != nil {
		return nil, err
	}
	return bson.M{"$lte": endDate}, nil
}

// parseDateRangeWithStart parses a date range with a start value
func (f *MongoFormatter) parseDateRangeWithStart(startStr, endStr string) (interface{}, error) {
	startDate, err := f.parseDate(startStr)
	if err != nil {
		return nil, err
	}

	result := bson.M{"$gte": startDate}

	if endStr != "*" {
		endDate, err := f.parseDate(endStr)
		if err != nil {
			return nil, err
		}
		result["$lte"] = endDate
	}

	return result, nil
}

// parseDate parses a date string in various formats
func (f *MongoFormatter) parseDate(dateStr string) (time.Time, error) {
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
func (f *MongoFormatter) parseNumberRange(startStr, endStr string) (interface{}, error) {
	if err := f.validateNumberRange(startStr, endStr); err != nil {
		return nil, err
	}

	if startStr == "*" {
		return f.parseNumberRangeWithWildcardStart(endStr)
	}

	return f.parseNumberRangeWithStart(startStr, endStr)
}

// validateNumberRange validates that the number range is valid
func (f *MongoFormatter) validateNumberRange(startStr, endStr string) error {
	if startStr == "*" && endStr == "*" {
		return errors.New("invalid number range: both start and end cannot be wildcards")
	}
	return nil
}

// parseNumberRangeWithWildcardStart parses a number range with wildcard start
func (f *MongoFormatter) parseNumberRangeWithWildcardStart(endStr string) (interface{}, error) {
	endNum, err := strconv.ParseFloat(endStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid end number: %v", err)
	}
	return bson.M{"$lte": endNum}, nil
}

// parseNumberRangeWithStart parses a number range with a start value
func (f *MongoFormatter) parseNumberRangeWithStart(startStr, endStr string) (interface{}, error) {
	startNum, err := strconv.ParseFloat(startStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid start number: %v", err)
	}

	result := bson.M{"$gte": startNum}

	if endStr != "*" {
		endNum, err := strconv.ParseFloat(endStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid end number: %v", err)
		}
		result["$lte"] = endNum
	}

	return result, nil
}

// expressionToBSON converts a ParticipleExpression to BSON, handling both structured and unstructured queries
func (f *MongoFormatter) expressionToBSON(expr *lucene.ParticipleExpression, defaultFields []string) bson.M {
	if len(expr.Or) == 0 {
		return bson.M{}
	}

	if len(expr.Or) == 1 {
		return f.andExpressionToBSON(expr.Or[0], defaultFields)
	}

	var conditions []bson.M
	for _, andExpr := range expr.Or {
		conditions = append(conditions, f.andExpressionToBSON(andExpr, defaultFields))
	}
	return bson.M{"$or": conditions}
}

// andExpressionToBSON converts AND expressions to BSON, handling both structured and unstructured queries
func (f *MongoFormatter) andExpressionToBSON(andExpr *lucene.ParticipleAndExpression, defaultFields []string) bson.M {
	if len(andExpr.And) == 0 {
		return bson.M{}
	}

	if len(andExpr.And) == 1 {
		return f.operandToBSON(andExpr.And[0], defaultFields)
	}

	directFields, conditions := f.processAndExpressions(andExpr.And, defaultFields)
	return f.buildAndResult(directFields, conditions)
}

// canMergeField checks if a field can be merged into directFields
func (f *MongoFormatter) canMergeField(directFields bson.M, childBSON bson.M, hasComplexExpressions bool) bool {
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
func (f *MongoFormatter) mergeField(directFields bson.M, childBSON bson.M) {
	for k, v := range childBSON {
		directFields[k] = v
	}
}

// buildAndResult builds the final result from directFields and conditions
func (f *MongoFormatter) buildAndResult(directFields bson.M, conditions []bson.M) bson.M {
	if len(directFields) > 0 && len(conditions) > 0 {
		conditions = append(conditions, directFields)
		return bson.M{"$and": conditions}
	} else if len(conditions) > 0 {
		return bson.M{"$and": conditions}
	}
	return directFields
}

// operandToBSON converts operands to BSON, handling both structured and unstructured queries
func (f *MongoFormatter) operandToBSON(operand *lucene.ParticipleOperand, defaultFields []string) bson.M {
	return f.operandToBSONWithContext(operand, defaultFields, false)
}

// operandToBSONWithContext converts operands to BSON with negation context
func (f *MongoFormatter) operandToBSONWithContext(operand *lucene.ParticipleOperand, defaultFields []string, inNotContext bool) bson.M {
	if operand.Not != nil {
		// Handle NOT operation
		childBSON := f.operandToBSONWithContext(operand.Not, defaultFields, true)
		return f.negateBSON(childBSON)
	}

	return f.termToBSONWithContext(operand.Term, defaultFields, inNotContext)
}

// termToBSONWithContext converts terms (field values, free text, groups) to BSON with negation context
func (f *MongoFormatter) termToBSONWithContext(term *lucene.ParticipleTerm, defaultFields []string, inNotContext bool) bson.M {
	if term.FieldValue != nil {
		return f.fieldValueToBSONWithContext(term.FieldValue, defaultFields, inNotContext)
	}

	if term.FreeText != nil {
		if defaultFields == nil {
			// Unstructured queries require default fields - this should not happen in the regular Format method
			// Return empty BSON since this should not occur in the regular Format method
			return bson.M{}
		}
		return f.freeTextToBSONUnstructured(term.FreeText, defaultFields)
	}

	if term.Group != nil {
		return f.expressionToBSON(term.Group.Expression, defaultFields)
	}

	return bson.M{}
}

// fieldValueToBSONWithContext converts field:value pairs to BSON with negation context
func (f *MongoFormatter) fieldValueToBSONWithContext(fv *lucene.ParticipleFieldValue, defaultFields []string, inNotContext bool) bson.M {
	// Check if this field value should be split into field:value + free text
	if fieldValue, freeText := fv.SplitIntoFieldAndText(); fieldValue != nil {
		// Convert field value to BSON
		fieldBSON := f.fieldValueToBSONWithContext(fieldValue, defaultFields, inNotContext)

		if defaultFields == nil {
			// Unstructured queries require default fields - this should not happen in the regular Format method
			// Return the field BSON only since this should not occur in the regular Format method
			return fieldBSON
		}

		// Convert free text to BSON using default fields
		freeTextBSON := f.freeTextToBSONUnstructured(freeText, defaultFields)

		// Return as $or with field:value and free text search (default behavior for mixed queries)
		return bson.M{
			"$or": []bson.M{
				fieldBSON,
				freeTextBSON,
			},
		}
	}

	// Single term or other value type - handle normally
	valueStr := f.extractValueString(fv.Value)
	value, err := f.parseValue(valueStr)
	if err != nil {
		value = valueStr
	}
	return bson.M{fv.Field: value}
}

// extractValueString extracts the string value from a ParticipleValue
func (f *MongoFormatter) extractValueString(value *lucene.ParticipleValue) string {
	// Try each field in order of preference
	if len(value.TextTerms) > 0 {
		return strings.Join(value.TextTerms, " ")
	}
	if value.String != nil {
		return *value.String
	}
	if value.SingleString != nil {
		return *value.SingleString
	}
	if value.Bracketed != nil {
		return *value.Bracketed
	}
	if value.DateTime != nil {
		return *value.DateTime
	}
	if value.TimeString != nil {
		return *value.TimeString
	}
	if value.Regex != nil {
		return *value.Regex
	}
	return ""
}

// freeTextToBSONUnstructured converts a ParticipleFreeText to BSON using default fields for unstructured queries
func (f *MongoFormatter) freeTextToBSONUnstructured(ft *lucene.ParticipleFreeText, defaultFields []string) bson.M {
	if ft.QuotedValue != nil {
		// Handle quoted values as a single term
		var valueStr string
		if ft.QuotedValue.String != nil {
			valueStr = *ft.QuotedValue.String
		} else if ft.QuotedValue.SingleString != nil {
			valueStr = *ft.QuotedValue.SingleString
		}
		return f.createDefaultFieldSearch(valueStr, defaultFields)
	} else if ft.UnquotedValue != nil {
		// Handle unquoted values - each word searches default fields with OR
		words := ft.UnquotedValue.TextTerms
		if len(words) == 1 {
			// Single word - direct search
			return f.createDefaultFieldSearch(words[0], defaultFields)
		}
		// Multiple words - each word searches default fields, all ORed together
		return f.createMultiWordDefaultFieldSearch(words, defaultFields)
	} else if ft.RegexValue != nil {
		// Handle regex values - strip the leading and trailing slashes and anchor
		pattern := (*ft.RegexValue)[1 : len(*ft.RegexValue)-1]
		// Add anchors for exact match if not already present
		if !strings.HasPrefix(pattern, "^") {
			pattern = "^" + pattern
		}
		if !strings.HasSuffix(pattern, "$") {
			pattern = pattern + "$"
		}
		return f.createDefaultFieldRegexSearch(pattern, defaultFields)
	}

	return bson.M{}
}

// negateBSON negates a BSON condition using De Morgan's law
func (f *MongoFormatter) negateBSON(condition bson.M) bson.M {
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
func (f *MongoFormatter) negateConditions(conditions []bson.M) []bson.M {
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
func (f *MongoFormatter) isSimpleFieldValue(condition bson.M) bool {
	if len(condition) != 1 {
		return false
	}

	// Check if the condition itself has complex operators
	if f.hasComplexOperators(condition) {
		return false
	}

	// Check if any field value contains complex operators
	return !f.hasComplexFieldValues(condition)
}

// hasComplexOperators checks if a BSON condition contains complex operators
func (f *MongoFormatter) hasComplexOperators(condition bson.M) bool {
	complexOperators := []string{"$or", "$and", "$text"}
	for _, op := range complexOperators {
		if _, hasOp := condition[op]; hasOp {
			return true
		}
	}
	return false
}

// hasComplexFieldValues checks if any field value contains complex operators
func (f *MongoFormatter) hasComplexFieldValues(condition bson.M) bool {
	complexOperators := []string{"$or", "$and", "$text"}
	for _, v := range condition {
		if vMap, ok := v.(bson.M); ok {
			for key := range vMap {
				for _, op := range complexOperators {
					if key == op {
						return true
					}
				}
			}
		}
	}
	return false
}

// createDefaultFieldSearch creates a BSON query that searches for the value in all default fields
func (f *MongoFormatter) createDefaultFieldSearch(valueStr string, defaultFields []string) bson.M {
	if len(defaultFields) == 0 {
		return bson.M{}
	}

	if len(defaultFields) == 1 {
		// Single field - create direct regex search
		return f.createFieldRegexSearch(defaultFields[0], valueStr)
	}

	// Multiple fields - create OR query
	var conditions []bson.M
	for _, field := range defaultFields {
		conditions = append(conditions, f.createFieldRegexSearch(field, valueStr))
	}
	return bson.M{"$or": conditions}
}

// createMultiWordDefaultFieldSearch creates a BSON query for multiple words across default fields
// Each word is searched against all default fields, all ORed together
func (f *MongoFormatter) createMultiWordDefaultFieldSearch(words []string, defaultFields []string) bson.M {
	if len(defaultFields) == 0 || len(words) == 0 {
		return bson.M{}
	}

	var conditions []bson.M
	for _, word := range words {
		for _, field := range defaultFields {
			conditions = append(conditions, f.createFieldRegexSearch(field, word))
		}
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return bson.M{"$or": conditions}
}

// createDefaultFieldRegexSearch creates a BSON query that searches for the regex pattern in all default fields
func (f *MongoFormatter) createDefaultFieldRegexSearch(pattern string, defaultFields []string) bson.M {
	if len(defaultFields) == 0 {
		return bson.M{}
	}

	if len(defaultFields) == 1 {
		// Single field - create direct regex search
		return bson.M{defaultFields[0]: bson.M{"$regex": pattern}}
	}

	// Multiple fields - create OR query
	var conditions []bson.M
	for _, field := range defaultFields {
		conditions = append(conditions, bson.M{field: bson.M{"$regex": pattern}})
	}
	return bson.M{"$or": conditions}
}

// createFieldRegexSearch creates a regex search for a specific field
func (f *MongoFormatter) createFieldRegexSearch(field, valueStr string) bson.M {
	regexBSON, err := f.parseValueToRegex(valueStr)
	if err != nil {
		// Fallback to plain text with regex escaping
		escapedValue := f.escapeRegex(valueStr)
		regexBSON = bson.M{"$regex": escapedValue, "$options": "i"}
	}

	// Apply the regex to the specific field
	return bson.M{field: regexBSON}
}

// parseValueToRegex parses a value string and returns a regex BSON query
func (f *MongoFormatter) parseValueToRegex(valueStr string) (bson.M, error) {
	// Check if the value contains wildcards
	if strings.Contains(valueStr, "*") {
		return f.parseWildcard(valueStr)
	}

	// Check if the value is a regex pattern
	if strings.HasPrefix(valueStr, "/") && strings.HasSuffix(valueStr, "/") && len(valueStr) > 2 {
		return f.parseRegex(valueStr)
	}

	// For plain text, we need to escape it and make it case-insensitive with exact match
	escapedValue := f.escapeRegex(valueStr)
	return bson.M{"$regex": "^" + escapedValue + "$", "$options": "i"}, nil
}

// escapeRegex escapes special regex characters in a string
func (f *MongoFormatter) escapeRegex(s string) string {
	// Escape special regex characters
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "^", "\\^")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, ".", "\\.")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")
	escaped = strings.ReplaceAll(escaped, "?", "\\?")
	escaped = strings.ReplaceAll(escaped, "*", "\\*")
	escaped = strings.ReplaceAll(escaped, "+", "\\+")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "{", "\\{")
	escaped = strings.ReplaceAll(escaped, "}", "\\}")
	return escaped
}

// processAndExpressions processes AND expressions, separating simple fields from complex conditions
func (f *MongoFormatter) processAndExpressions(expressions []*lucene.ParticipleOperand, defaultFields []string) (bson.M, []bson.M) {
	var conditions []bson.M
	directFields := bson.M{}
	hasComplexExpressions := false

	for _, operand := range expressions {
		childBSON := f.operandToBSON(operand, defaultFields)

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
