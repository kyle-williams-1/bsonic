package bsonic

import (
	"fmt"
	"testing"
	"time"

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
		{"name:john", bson.M{"name": "john"}},
		{"age:25", bson.M{"age": 25.0}},
		{"active:true", bson.M{"active": true}},
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

	regexValue, ok := nameValue.(bson.M)
	if !ok {
		t.Fatalf("Expected regex object, got %T", nameValue)
	}

	if regexValue["$regex"] != "^jo.*" {
		t.Fatalf("Expected regex '^jo.*', got %v", regexValue["$regex"])
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

	expected := bson.M{"name": "john", "age": 25.0}
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

	invalidQueries := []string{"invalid", ":value"}

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
					{"age": 25.0},
					{"age": 30.0},
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
				"age":  bson.M{"$ne": 25.0},
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
					{"name": "jane", "age": 25.0},
				},
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
					{"name": bson.M{"$regex": "^jo.*", "$options": "i"}},
					{"name": bson.M{"$regex": "^ja.*", "$options": "i"}, "age": bson.M{"$ne": 18.0}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if len(result) < 1 {
				t.Fatalf("Expected at least 1 field, got %d", len(result))
			}

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

			for field, expectedValue := range test.expected {
				if field == "$or" {
					continue
				}

				if actualValue, exists := result[field]; !exists {
					t.Fatalf("Expected field %s not found", field)
				} else {
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

func TestParseDateQueries(t *testing.T) {
	parser := New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "created_at:2023-01-15",
			expected: bson.M{
				"created_at": time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			desc: "exact date",
		},
		{
			input: "created_at:2023-01-15T10:30:00Z",
			expected: bson.M{
				"created_at": time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			desc: "exact datetime",
		},
		{
			input: "created_at:[2023-01-01 TO 2023-12-31]",
			expected: bson.M{
				"created_at": bson.M{
					"$gte": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					"$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date range",
		},
		{
			input: "created_at:>2024-01-01",
			expected: bson.M{
				"created_at": bson.M{
					"$gt": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date greater than",
		},
		{
			input: "created_at:<2023-12-31",
			expected: bson.M{
				"created_at": bson.M{
					"$lt": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date less than",
		},
		{
			input: "created_at:>=2024-01-01",
			expected: bson.M{
				"created_at": bson.M{
					"$gte": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date greater than or equal",
		},
		{
			input: "created_at:<=2023-12-31",
			expected: bson.M{
				"created_at": bson.M{
					"$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date less than or equal",
		},
		{
			input: "created_at:[2023-01-01 TO *]",
			expected: bson.M{
				"created_at": bson.M{
					"$gte": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date range with wildcard end",
		},
		{
			input: "created_at:[* TO 2023-12-31]",
			expected: bson.M{
				"created_at": bson.M{
					"$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				},
			},
			desc: "date range with wildcard start",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 field, got %d", len(result))
			}

			field := "created_at"
			actualValue, exists := result[field]
			if !exists {
				t.Fatalf("Expected field %s not found", field)
			}

			if !compareBSONValues(actualValue, test.expected[field]) {
				t.Fatalf("Expected %s=%v, got %s=%v", field, test.expected[field], field, actualValue)
			}
		})
	}
}

func TestParseComplexDateQueries(t *testing.T) {
	parser := New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "created_at:[2023-01-01 TO 2023-12-31] AND status:active",
			expected: bson.M{
				"created_at": bson.M{
					"$gte": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					"$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				},
				"status": "active",
			},
			desc: "date range with AND",
		},
		{
			input: "created_at:>2024-01-01 OR updated_at:<2023-01-01",
			expected: bson.M{
				"$or": []bson.M{
					{"created_at": bson.M{"$gt": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
					{"updated_at": bson.M{"$lt": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}},
				},
			},
			desc: "date comparisons with OR",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if len(result) < 1 {
				t.Fatalf("Expected at least 1 field, got %d", len(result))
			}

			for field, expectedValue := range test.expected {
				if actualValue, exists := result[field]; !exists {
					t.Fatalf("Expected field %s not found", field)
				} else if !compareBSONValues(actualValue, expectedValue) {
					t.Fatalf("Expected %s=%v, got %s=%v", field, expectedValue, field, actualValue)
				}
			}
		})
	}
}

func TestParseInvalidDateQueries(t *testing.T) {
	parser := New()

	invalidQueries := []string{
		"created_at:[invalid TO 2023-12-31]",
		"created_at:[2023-01-01 TO invalid]",
		"created_at:>invalid",
		"created_at:[2023-01-01 TO 2023-12-31 TO 2024-01-01]",
		"created_at:[* TO *]",
	}

	for _, invalidQuery := range invalidQueries {
		t.Run(invalidQuery, func(t *testing.T) {
			_, err := parser.Parse(invalidQuery)
			if err == nil {
				t.Fatalf("Expected error for invalid date query '%s', got none", invalidQuery)
			}
		})
	}
}

func TestParseParenthesesQueries(t *testing.T) {
	parser := New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "(name:john OR name:jane) AND age:25",
			expected: bson.M{
				"$and": []bson.M{
					{
						"$or": []bson.M{
							{"name": "john"},
							{"name": "jane"},
						},
					},
					{"age": 25.0},
				},
			},
			desc: "grouped OR with AND",
		},
		{
			input: "name:john OR (name:jane AND age:25)",
			expected: bson.M{
				"$or": []bson.M{
					{"name": "john"},
					{"name": "jane", "age": 25.0},
				},
			},
			desc: "OR with grouped AND",
		},
		{
			input: "(name:john AND age:25) OR (name:jane AND age:30)",
			expected: bson.M{
				"$or": []bson.M{
					{"name": "john", "age": 25.0},
					{"name": "jane", "age": 30.0},
				},
			},
			desc: "grouped AND expressions with OR",
		},
		{
			input: "NOT (name:john OR name:jane)",
			expected: bson.M{
				"$and": []bson.M{
					{
						"$or": []bson.M{
							{"name": bson.M{"$ne": "john"}},
							{"name": bson.M{"$ne": "jane"}},
						},
					},
				},
			},
			desc: "NOT with grouped OR",
		},
		{
			input: "((name:john OR name:jane) AND age:25) OR status:active",
			expected: bson.M{
				"$or": []bson.M{
					{
						"$and": []bson.M{
							{
								"$or": []bson.M{
									{"name": "john"},
									{"name": "jane"},
								},
							},
							{"age": 25.0},
						},
					},
					{"status": "active"},
				},
			},
			desc: "nested parentheses",
		},
		{
			input: "(name:jo* OR name:ja*) AND (age:25 OR age:30)",
			expected: bson.M{
				"$and": []bson.M{
					{
						"$or": []bson.M{
							{"name": bson.M{"$regex": "^jo.*", "$options": "i"}},
							{"name": bson.M{"$regex": "^ja.*", "$options": "i"}},
						},
					},
					{
						"$or": []bson.M{
							{"age": 25.0},
							{"age": 30.0},
						},
					},
				},
			},
			desc: "grouped wildcards and numbers",
		},
		{
			input: "created_at:[2023-01-01 TO 2023-12-31] AND (status:active OR status:pending)",
			expected: bson.M{
				"$and": []bson.M{
					{
						"$or": []bson.M{
							{"status": "active"},
							{"status": "pending"},
						},
					},
					{
						"created_at": bson.M{
							"$gte": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							"$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
						},
					},
				},
			},
			desc: "date range with grouped status",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestDebugIncompleteExpression(t *testing.T) {
	parser := New()

	query := "(name:john AND)"
	fmt.Printf("Testing: %s\n", query)
	result, err := parser.Parse(query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %+v\n", result)
	}
}

func TestParseInvalidParenthesesQueries(t *testing.T) {
	parser := New()

	invalidQueries := []string{
		"(name:john OR name:jane",      // unmatched opening parenthesis
		"name:john OR name:jane)",      // unmatched closing parenthesis
		"((name:john OR name:jane)",    // unmatched nested parentheses
		"name:john OR name:jane))",     // extra closing parenthesis
		"()",                           // empty parentheses
		"(name:john AND)",              // incomplete expression in parentheses
		"AND (name:john OR name:jane)", // AND at start
		"(name:john OR name:jane) AND", // AND at end
	}

	for _, invalidQuery := range invalidQueries {
		t.Run(invalidQuery, func(t *testing.T) {
			_, err := parser.Parse(invalidQuery)
			if err == nil {
				t.Fatalf("Expected error for invalid parentheses query '%s', got none", invalidQuery)
			}
		})
	}
}

func compareBSONValues(actual, expected interface{}) bool {
	if actualTime, ok := actual.(time.Time); ok {
		if expectedTime, ok := expected.(time.Time); ok {
			return actualTime.Equal(expectedTime)
		}
		return false
	}

	if actualMap, ok := actual.(bson.M); ok {
		if expectedMap, ok := expected.(bson.M); ok {
			if len(actualMap) != len(expectedMap) {
				return false
			}
			for key, expectedValue := range expectedMap {
				actualValue, exists := actualMap[key]
				if !exists {
					return false
				}
				if !compareBSONValues(actualValue, expectedValue) {
					return false
				}
			}
			return true
		}
		return false
	}

	if actualArray, ok := actual.([]bson.M); ok {
		if expectedArray, ok := expected.([]bson.M); ok {
			if len(actualArray) != len(expectedArray) {
				return false
			}
			for i, expectedValue := range expectedArray {
				if !compareBSONValues(actualArray[i], expectedValue) {
					return false
				}
			}
			return true
		}
		return false
	}

	return actual == expected
}
