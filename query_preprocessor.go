package bsonic

import (
	"regexp"
	"strings"
)

// QueryPreprocessor handles query preprocessing to fix parsing issues
type QueryPreprocessor struct {
	// Regex patterns for different query types
	emailPattern       *regexp.Regexp
	dotNotationPattern *regexp.Regexp
	quotedValuePattern *regexp.Regexp
	mixedQueryPattern  *regexp.Regexp
}

// NewQueryPreprocessor creates a new query preprocessor
func NewQueryPreprocessor() *QueryPreprocessor {
	return &QueryPreprocessor{
		emailPattern:       regexp.MustCompile(`(\w+):([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`),
		dotNotationPattern: regexp.MustCompile(`(\w+(?:\.\w+)+):([^:]+?)(?:\s+(?:AND|OR|NOT)\s+|$)`),
		quotedValuePattern: regexp.MustCompile(`(\w+(?:\.\w+)*):"([^"]+)"`),
		mixedQueryPattern:  regexp.MustCompile(`^([a-zA-Z\s]+)\s*\(([^)]+)\)$`),
	}
}

// PreprocessQuery applies preprocessing fixes to a query string
func (qp *QueryPreprocessor) PreprocessQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}

	// 1. Fix email addresses - quote them to prevent dot parsing issues
	query = qp.fixEmailAddresses(query)

	// 2. Fix dot notation fields with spaces - quote the values
	query = qp.fixDotNotationFields(query)

	// 2.5. Fix parentheses and quoted values first
	query = qp.fixParenthesesAndQuotes(query)

	// 2.6. Fix regular fields with spaces - quote the values
	query = qp.fixRegularFieldsWithSpaces(query)

	// 3. Fix quoted values that might have been broken
	query = qp.fixQuotedValues(query)

	// 4. Fix mixed queries (text search + field queries)
	query = qp.fixMixedQueries(query)

	// 5. Fix NOT operators at the beginning of queries
	query = qp.fixNotOperators(query)

	return query
}

// fixEmailAddresses quotes email addresses to prevent dot parsing issues
func (qp *QueryPreprocessor) fixEmailAddresses(query string) string {
	return qp.emailPattern.ReplaceAllStringFunc(query, func(match string) string {
		parts := strings.Split(match, ":")
		if len(parts) == 2 {
			field := parts[0]
			value := parts[1]
			// Only quote if it looks like an email and isn't already quoted
			if strings.Contains(value, "@") && !strings.HasPrefix(value, "\"") {
				return field + ":\"" + value + "\""
			}
		}
		return match
	})
}

// fixDotNotationFields quotes values for dot notation fields that contain spaces
func (qp *QueryPreprocessor) fixDotNotationFields(query string) string {
	// More comprehensive approach: find all field:value patterns and fix them
	// Pattern to match field:value where field contains dots
	pattern := regexp.MustCompile(`(\w+(?:\.\w+)+):([^:\s]+(?:\s+[^:\s]+)*)`)

	return pattern.ReplaceAllStringFunc(query, func(match string) string {
		// Extract field and value
		colonIndex := strings.Index(match, ":")
		if colonIndex == -1 {
			return match
		}

		field := match[:colonIndex]
		value := match[colonIndex+1:]

		// Quote the value if it contains spaces and isn't already quoted
		if strings.Contains(value, " ") && !strings.HasPrefix(value, "\"") {
			return field + ":\"" + value + "\""
		}

		return match
	})
}

// fixParenthesesAndQuotes handles parentheses and quoted values properly
func (qp *QueryPreprocessor) fixParenthesesAndQuotes(query string) string {
	// Find all parenthesized expressions and quote values with spaces inside them
	parenPattern := regexp.MustCompile(`\(([^)]+)\)`)

	return parenPattern.ReplaceAllStringFunc(query, func(match string) string {
		// Extract the content inside parentheses
		content := match[1 : len(match)-1]

		// Process field:value pairs within the parentheses
		processedContent := qp.processFieldValuePairs(content)

		return "(" + processedContent + ")"
	})
}

// fixRegularFieldsWithSpaces quotes values for regular fields that contain spaces
func (qp *QueryPreprocessor) fixRegularFieldsWithSpaces(query string) string {
	// Split the query by logical operators to process each part separately
	parts := regexp.MustCompile(`\s+(AND|OR|NOT)\s+`).Split(query, -1)
	operators := regexp.MustCompile(`\s+(AND|OR|NOT)\s+`).FindAllString(query, -1)

	if len(parts) == 1 {
		// No logical operators, process the whole query
		return qp.processFieldValuePairs(parts[0])
	}

	// Process each part and reassemble
	result := qp.processFieldValuePairs(parts[0])
	for i, operator := range operators {
		if i+1 < len(parts) {
			result += " " + operator + " " + qp.processFieldValuePairs(parts[i+1])
		}
	}

	return result
}

// processFieldValuePairs processes field:value pairs within a single expression
func (qp *QueryPreprocessor) processFieldValuePairs(expr string) string {
	// If the expression contains logical operators, split and process each part
	if strings.Contains(strings.ToUpper(expr), " OR ") ||
		strings.Contains(strings.ToUpper(expr), " AND ") ||
		strings.Contains(strings.ToUpper(expr), " NOT ") {
		// Split by logical operators and process each part
		parts := regexp.MustCompile(`\s+(AND|OR|NOT)\s+`).Split(expr, -1)
		operators := regexp.MustCompile(`\s+(AND|OR|NOT)\s+`).FindAllString(expr, -1)

		result := qp.quoteFieldValues(parts[0])
		for i, operator := range operators {
			if i+1 < len(parts) {
				// Preserve the original operator spacing (operator already includes spaces)
				result += operator + qp.quoteFieldValues(parts[i+1])
			}
		}
		return result
	}

	// No logical operators, just quote field values
	return qp.quoteFieldValues(expr)
}

// quoteFieldValues quotes values for field:value pairs that contain spaces
func (qp *QueryPreprocessor) quoteFieldValues(expr string) string {
	// Pattern to match field:value where field doesn't contain dots and value contains spaces
	pattern := regexp.MustCompile(`(\w+):([^:\s]+(?:\s+[^:\s]+)+)`)

	return pattern.ReplaceAllStringFunc(expr, func(match string) string {
		// Extract field and value
		colonIndex := strings.Index(match, ":")
		if colonIndex == -1 {
			return match
		}

		field := match[:colonIndex]
		value := match[colonIndex+1:]

		// Don't quote range values (containing [ and ] or TO)
		if strings.Contains(value, "[") || strings.Contains(value, "]") || strings.Contains(strings.ToUpper(value), " TO ") {
			return match
		}

		// Only quote if the field doesn't contain dots and the value contains spaces
		// and isn't already quoted
		if !strings.Contains(field, ".") &&
			strings.Contains(value, " ") &&
			!strings.HasPrefix(value, "\"") {
			return field + ":\"" + value + "\""
		}

		return match
	})
}

// fixQuotedValues ensures quoted values are properly handled
func (qp *QueryPreprocessor) fixQuotedValues(query string) string {
	// This is a placeholder for more complex quoted value handling
	// For now, we'll rely on the other fixes
	return query
}

// fixMixedQueries handles mixed text search + field queries
func (qp *QueryPreprocessor) fixMixedQueries(query string) string {
	// Don't process NOT queries
	if strings.HasPrefix(strings.ToUpper(query), "NOT ") {
		return query
	}

	// Check if this looks like a mixed query: "text (field:value AND field:value)"
	matches := qp.mixedQueryPattern.FindStringSubmatch(query)
	if len(matches) == 3 {
		textPart := strings.TrimSpace(matches[1])
		fieldPart := strings.TrimSpace(matches[2])

		// If the text part doesn't contain colons, it's likely text search
		if !strings.Contains(textPart, ":") {
			// Convert to: text AND (field:value AND field:value)
			return textPart + " AND (" + fieldPart + ")"
		}
	}

	// Also handle cases like "engineer role:admin AND age:30" (without parentheses)
	// Pattern: word(s) followed by field:value patterns
	// But exclude queries that start with NOT, AND, OR
	if !strings.HasPrefix(strings.ToUpper(query), "NOT ") &&
		!strings.HasPrefix(strings.ToUpper(query), "AND ") &&
		!strings.HasPrefix(strings.ToUpper(query), "OR ") {
		wordFieldPattern := regexp.MustCompile(`^([a-zA-Z\s]+)\s+(\w+:[^:]+(?:\s+(?:AND|OR|NOT)\s+\w+:[^:]+)*)$`)
		wordMatches := wordFieldPattern.FindStringSubmatch(query)
		if len(wordMatches) == 3 {
			textPart := strings.TrimSpace(wordMatches[1])
			fieldPart := strings.TrimSpace(wordMatches[2])

			// If the text part doesn't contain colons, it's likely text search
			if !strings.Contains(textPart, ":") {
				// Convert to: text AND (field:value AND field:value)
				return textPart + " AND (" + fieldPart + ")"
			}
		}
	}

	return query
}

// fixNotOperators handles NOT operators at the beginning of queries
func (qp *QueryPreprocessor) fixNotOperators(query string) string {
	// Check if query starts with NOT followed by a field:value
	notPattern := regexp.MustCompile(`^NOT\s+(\w+(?:\.\w+)*):([^:\s]+(?:\s+[^:\s]+)*)$`)
	matches := notPattern.FindStringSubmatch(query)
	if len(matches) == 3 {
		field := matches[1]
		value := matches[2]

		// Quote the value if it contains spaces and isn't already quoted
		if strings.Contains(value, " ") && !strings.HasPrefix(value, "\"") {
			value = "\"" + value + "\""
		}

		return "NOT " + field + ":" + value
	}

	// Check if query starts with NOT followed by parentheses
	notParenPattern := regexp.MustCompile(`^NOT\s+\((.+)\)$`)
	parenMatches := notParenPattern.FindStringSubmatch(query)
	if len(parenMatches) == 2 {
		// Process the content inside parentheses but don't add AND
		content := parenMatches[1]
		processedContent := qp.processFieldValuePairs(content)
		return "NOT (" + processedContent + ")"
	}

	return query
}
