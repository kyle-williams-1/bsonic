// Package bsonic provides a Lucene-style syntax parser for MongoDB BSON filters.
// It converts Lucene query strings into BSON documents that can be used with
// the MongoDB Go driver.
package bsonic

import (
	"errors"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// Parser represents a Lucene-style query parser for MongoDB BSON filters.
type Parser struct {
	// Future: configuration options for custom operators, field mappings, etc.
}

// New creates a new BSON parser instance.
func New() *Parser {
	return &Parser{}
}

// Parse converts a Lucene-style query string into a BSON document.
// Example: "name:john AND age:25" -> bson.M{"name": "john", "age": 25}
func (p *Parser) Parse(query string) (bson.M, error) {
	if strings.TrimSpace(query) == "" {
		return bson.M{}, nil
	}

	// For now, we'll implement a simple parser
	// This will be expanded to handle complex queries
	return p.parseSimpleQuery(query)
}

// parseSimpleQuery handles basic field:value queries with AND/OR/NOT operators
func (p *Parser) parseSimpleQuery(query string) (bson.M, error) {
	result := bson.M{}

	// Handle different query types
	if strings.Contains(strings.ToUpper(query), " OR ") {
		// Handle OR queries
		orConditions, err := p.extractOrConditions(query)
		if err != nil {
			return nil, err
		}
		if len(orConditions) > 0 {
			result["$or"] = orConditions
		}

		// Also extract AND conditions from the OR query
		andConditions, err := p.extractAndConditions(query)
		if err != nil {
			return nil, err
		}
		for field, value := range andConditions {
			result[field] = value
		}

		// Also extract NOT conditions from the OR query
		// Only extract NOT conditions that are not part of OR conditions
		notConditions, err := p.extractNotConditionsFromOrQuery(query)
		if err != nil {
			return nil, err
		}
		for field, value := range notConditions {
			result[field] = bson.M{"$ne": value}
		}
	} else if strings.Contains(strings.ToUpper(query), " NOT ") || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "NOT ") {
		// Handle NOT queries (including NOT at the beginning)
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
	// Handle both cases: NOT at beginning and NOT after AND
	re := regexp.MustCompile(`(?:^|\s+)NOT\s+(\w+(?:\.\w+)*):([^\s]+(?:"[^"]*"|[^\s]+)*)`)
	matches := re.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		field := match[1]
		valueStr := strings.TrimSpace(match[2])

		// Handle quoted values
		if len(valueStr) >= 2 && valueStr[0] == '"' && valueStr[len(valueStr)-1] == '"' {
			valueStr = valueStr[1 : len(valueStr)-1]
		}

		value, err := p.parseValue(valueStr)
		if err != nil {
			return "", nil, err
		}

		notConditions[field] = value
	}

	// Remove NOT conditions from the original query
	cleanedQuery := re.ReplaceAllString(query, "")

	// Clean up any remaining AND operators at the end
	cleanedQuery = strings.TrimSpace(cleanedQuery)
	cleanedQuery = strings.TrimSuffix(cleanedQuery, " AND")

	return cleanedQuery, notConditions, nil
}

// extractNotConditionsFromOrQuery extracts NOT conditions from OR queries
func (p *Parser) extractNotConditionsFromOrQuery(query string) (bson.M, error) {
	notConditions := bson.M{}

	// Find NOT patterns that are not part of OR conditions
	// Look for "AND NOT field:value" patterns
	re := regexp.MustCompile(`\s+AND\s+NOT\s+(\w+(?:\.\w+)*):([^\s]+(?:"[^"]*"|[^\s]+)*)`)
	matches := re.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		field := match[1]
		valueStr := strings.TrimSpace(match[2])

		// Handle quoted values
		if len(valueStr) >= 2 && valueStr[0] == '"' && valueStr[len(valueStr)-1] == '"' {
			valueStr = valueStr[1 : len(valueStr)-1]
		}

		value, err := p.parseValue(valueStr)
		if err != nil {
			return nil, err
		}

		notConditions[field] = value
	}

	return notConditions, nil
}

// extractOrConditions extracts OR conditions from the query
func (p *Parser) extractOrConditions(query string) ([]bson.M, error) {
	var orConditions []bson.M

	// Split by OR operators
	re := regexp.MustCompile(`\s+OR\s+`)
	parts := re.Split(query, -1)

	// If we have more than one part, it means there are OR conditions
	if len(parts) > 1 {
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Check if this part contains AND or NOT operators
			if strings.Contains(strings.ToUpper(part), " AND ") || strings.Contains(strings.ToUpper(part), " NOT ") {
				// This part has complex operators, extract only the field:value part before AND
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

			// Parse each part as a field:value pair
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

	// Split by AND operators
	re := regexp.MustCompile(`\s+AND\s+`)
	parts := re.Split(query, -1)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "AND" {
			continue
		}

		// Skip if this part contains OR (already handled)
		if strings.Contains(strings.ToUpper(part), " OR ") {
			continue
		}

		// Skip if this part contains NOT (will be handled by NOT extraction)
		if strings.Contains(strings.ToUpper(part), " NOT ") || strings.HasPrefix(strings.ToUpper(part), "NOT ") {
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
	// Look for field:value pattern
	colonIndex := strings.Index(part, ":")
	if colonIndex == -1 {
		return "", nil, errors.New("invalid query format: expected field:value")
	}

	field := strings.TrimSpace(part[:colonIndex])
	valueStr := strings.TrimSpace(part[colonIndex+1:])

	if field == "" {
		return "", nil, errors.New("field name cannot be empty")
	}

	if valueStr == "" {
		return "", nil, errors.New("value cannot be empty")
	}

	// Parse value (handle wildcards, quotes, etc.)
	value, err := p.parseValue(valueStr)
	if err != nil {
		return "", nil, err
	}

	return field, value, nil
}

// parseValue parses a value string, handling wildcards and special syntax
func (p *Parser) parseValue(valueStr string) (interface{}, error) {
	// Remove surrounding quotes if present
	if len(valueStr) >= 2 && valueStr[0] == '"' && valueStr[len(valueStr)-1] == '"' {
		valueStr = valueStr[1 : len(valueStr)-1]
	}

	// Check for wildcards
	if strings.Contains(valueStr, "*") {
		// Convert to regex pattern
		pattern := strings.ReplaceAll(valueStr, "*", ".*")
		regex := bson.M{"$regex": pattern, "$options": "i"}
		return regex, nil
	}

	// For now, return as string
	// Future: handle numbers, booleans, dates, etc.
	return valueStr, nil
}
