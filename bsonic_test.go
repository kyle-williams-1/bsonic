package bsonic

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestNew(t *testing.T) {
	parser := New()
	if parser == nil {
		t.Fatal("New() should return a non-nil parser")
	}
}

func TestParseEmptyQuery(t *testing.T) {
	parser := New()

	query, err := parser.Parse("")
	if err != nil {
		t.Fatalf("Parse empty query should not return error, got: %v", err)
	}

	if len(query) != 0 {
		t.Fatalf("Empty query should return empty BSON, got: %+v", query)
	}
}

func TestParseWhitespaceQuery(t *testing.T) {
	parser := New()

	query, err := parser.Parse("   ")
	if err != nil {
		t.Fatalf("Parse whitespace query should not return error, got: %v", err)
	}

	if len(query) != 0 {
		t.Fatalf("Whitespace query should return empty BSON, got: %+v", query)
	}
}

func TestParseSimpleFieldValue(t *testing.T) {
	parser := New()

	tests := []struct {
		input    string
		expected bson.M
	}{
		{
			input:    "name:john",
			expected: bson.M{"name": "john"},
		},
		{
			input:    "age:25",
			expected: bson.M{"age": "25"},
		},
		{
			input:    "active:true",
			expected: bson.M{"active": "true"},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if len(result) != len(test.expected) {
				t.Fatalf("Expected %d fields, got %d", len(test.expected), len(result))
			}

			for key, expectedValue := range test.expected {
				if actualValue, exists := result[key]; !exists {
					t.Fatalf("Expected field %s not found", key)
				} else if actualValue != expectedValue {
					t.Fatalf("Expected %s=%v, got %s=%v", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestParseWildcardQuery(t *testing.T) {
	parser := New()

	query, err := parser.Parse("name:jo*")
	if err != nil {
		t.Fatalf("Parse wildcard query should not return error, got: %v", err)
	}

	if len(query) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(query))
	}

	nameValue, exists := query["name"]
	if !exists {
		t.Fatal("Expected 'name' field not found")
	}

	// Check if it's a regex object
	regexValue, ok := nameValue.(bson.M)
	if !ok {
		t.Fatalf("Expected regex object, got %T", nameValue)
	}

	if regexValue["$regex"] != "jo.*" {
		t.Fatalf("Expected regex 'jo.*', got %v", regexValue["$regex"])
	}

	if regexValue["$options"] != "i" {
		t.Fatalf("Expected case-insensitive option, got %v", regexValue["$options"])
	}
}

func TestParseQuotedValue(t *testing.T) {
	parser := New()

	query, err := parser.Parse(`name:"john doe"`)
	if err != nil {
		t.Fatalf("Parse quoted value should not return error, got: %v", err)
	}

	expected := bson.M{"name": "john doe"}
	if len(query) != len(expected) {
		t.Fatalf("Expected %d fields, got %d", len(expected), len(query))
	}

	if query["name"] != "john doe" {
		t.Fatalf("Expected name='john doe', got name='%v'", query["name"])
	}
}

func TestParseAndOperator(t *testing.T) {
	parser := New()

	query, err := parser.Parse("name:john AND age:25")
	if err != nil {
		t.Fatalf("Parse AND query should not return error, got: %v", err)
	}

	expected := bson.M{"name": "john", "age": "25"}
	if len(query) != len(expected) {
		t.Fatalf("Expected %d fields, got %d", len(expected), len(query))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := query[key]; !exists {
			t.Fatalf("Expected field %s not found", key)
		} else if actualValue != expectedValue {
			t.Fatalf("Expected %s=%v, got %s=%v", key, expectedValue, key, actualValue)
		}
	}
}

func TestParseInvalidQuery(t *testing.T) {
	parser := New()

	invalidQueries := []string{
		"invalid",
		":value",
	}

	for _, invalidQuery := range invalidQueries {
		t.Run(invalidQuery, func(t *testing.T) {
			_, err := parser.Parse(invalidQuery)
			if err == nil {
				t.Fatalf("Expected error for invalid query '%s', got none", invalidQuery)
			}
		})
	}
}

func TestParseDotNotation(t *testing.T) {
	parser := New()

	query, err := parser.Parse("user.profile.email:john@example.com")
	if err != nil {
		t.Fatalf("Parse dot notation should not return error, got: %v", err)
	}

	expected := bson.M{"user.profile.email": "john@example.com"}
	if len(query) != len(expected) {
		t.Fatalf("Expected %d fields, got %d", len(expected), len(query))
	}

	if query["user.profile.email"] != "john@example.com" {
		t.Fatalf("Expected user.profile.email='john@example.com', got %v", query["user.profile.email"])
	}
}

func TestParseOrOperator(t *testing.T) {
	parser := New()

	tests := []struct {
		input    string
		expected bson.M
	}{
		{
			input: "name:john OR name:jane",
			expected: bson.M{
				"$or": []bson.M{
					{"name": "john"},
					{"name": "jane"},
				},
			},
		},
		{
			input: "age:25 OR age:30",
			expected: bson.M{
				"$or": []bson.M{
					{"age": "25"},
					{"age": "30"},
				},
			},
		},
		{
			input: "status:active OR status:pending",
			expected: bson.M{
				"$or": []bson.M{
					{"status": "active"},
					{"status": "pending"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			// Check $or field exists
			orValue, exists := result["$or"]
			if !exists {
				t.Fatalf("Expected $or field not found")
			}

			orArray, ok := orValue.([]bson.M)
			if !ok {
				t.Fatalf("Expected $or to be []bson.M, got %T", orValue)
			}

			expectedOr := test.expected["$or"].([]bson.M)
			if len(orArray) != len(expectedOr) {
				t.Fatalf("Expected %d OR conditions, got %d", len(expectedOr), len(orArray))
			}

			// Check each OR condition
			for i, expectedCondition := range expectedOr {
				if i >= len(orArray) {
					t.Fatalf("Missing OR condition at index %d", i)
				}

				actualCondition := orArray[i]
				if len(actualCondition) != len(expectedCondition) {
					t.Fatalf("Expected condition %d to have %d fields, got %d", i, len(expectedCondition), len(actualCondition))
				}

				for field, expectedValue := range expectedCondition {
					if actualValue, exists := actualCondition[field]; !exists {
						t.Fatalf("Expected field %s not found in condition %d", field, i)
					} else if actualValue != expectedValue {
						t.Fatalf("Expected %s=%v in condition %d, got %s=%v", field, expectedValue, i, field, actualValue)
					}
				}
			}
		})
	}
}

func TestParseNotOperator(t *testing.T) {
	parser := New()

	tests := []struct {
		input    string
		expected bson.M
	}{
		{
			input: "name:john AND NOT age:25",
			expected: bson.M{
				"name": "john",
				"age":  bson.M{"$ne": "25"},
			},
		},
		{
			input: "NOT status:inactive",
			expected: bson.M{
				"status": bson.M{"$ne": "inactive"},
			},
		},
		{
			input: `name:"john doe" AND NOT role:admin`,
			expected: bson.M{
				"name": "john doe",
				"role": bson.M{"$ne": "admin"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if len(result) != len(test.expected) {
				t.Fatalf("Expected %d fields, got %d", len(test.expected), len(result))
			}

			for field, expectedValue := range test.expected {
				if actualValue, exists := result[field]; !exists {
					t.Fatalf("Expected field %s not found", field)
				} else {
					// Handle $ne comparison
					if expectedMap, ok := expectedValue.(bson.M); ok {
						if actualMap, ok := actualValue.(bson.M); ok {
							if expectedMap["$ne"] != actualMap["$ne"] {
								t.Fatalf("Expected %s=%v, got %s=%v", field, expectedValue, field, actualValue)
							}
						} else {
							t.Fatalf("Expected %s to be bson.M, got %T", field, actualValue)
						}
					} else if actualValue != expectedValue {
						t.Fatalf("Expected %s=%v, got %s=%v", field, expectedValue, field, actualValue)
					}
				}
			}
		})
	}
}

func TestParseComplexOperators(t *testing.T) {
	parser := New()

	tests := []struct {
		input       string
		expected    bson.M
		description string
	}{
		{
			input:       "name:john OR name:jane AND age:25",
			description: "OR with AND",
			expected: bson.M{
				"$or": []bson.M{
					{"name": "john"},
					{"name": "jane"},
				},
				"age": "25",
			},
		},
		{
			input:       "status:active AND NOT role:guest",
			description: "AND with NOT",
			expected: bson.M{
				"status": "active",
				"role":   bson.M{"$ne": "guest"},
			},
		},
		{
			input:       "name:jo* OR name:ja* AND NOT age:18",
			description: "Wildcard OR with AND and NOT",
			expected: bson.M{
				"$or": []bson.M{
					{"name": bson.M{"$regex": "jo.*", "$options": "i"}},
					{"name": bson.M{"$regex": "ja.*", "$options": "i"}},
				},
				"age": bson.M{"$ne": "18"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			// Basic structure check
			if len(result) < 1 {
				t.Fatalf("Expected at least 1 field, got %d", len(result))
			}

			// Check for $or if expected
			if expectedOr, hasOr := test.expected["$or"]; hasOr {
				if actualOr, exists := result["$or"]; !exists {
					t.Fatalf("Expected $or field not found")
				} else {
					expectedOrArray := expectedOr.([]bson.M)
					actualOrArray := actualOr.([]bson.M)
					if len(actualOrArray) != len(expectedOrArray) {
						t.Fatalf("Expected %d OR conditions, got %d", len(expectedOrArray), len(actualOrArray))
					}
				}
			}

			// Check for $ne (NOT) conditions
			for field, expectedValue := range test.expected {
				if field == "$or" {
					continue // Already checked above
				}

				if actualValue, exists := result[field]; !exists {
					t.Fatalf("Expected field %s not found", field)
				} else {
					// Handle $ne comparison
					if expectedMap, ok := expectedValue.(bson.M); ok {
						if actualMap, ok := actualValue.(bson.M); ok {
							if expectedMap["$ne"] != actualMap["$ne"] {
								t.Fatalf("Expected %s=%v, got %s=%v", field, expectedValue, field, actualValue)
							}
						} else {
							t.Fatalf("Expected %s to be bson.M, got %T", field, actualValue)
						}
					} else if actualValue != expectedValue {
						t.Fatalf("Expected %s=%v, got %s=%v", field, expectedValue, field, actualValue)
					}
				}
			}
		})
	}
}
