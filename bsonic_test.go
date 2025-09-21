package bsonic_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyle-williams-1/bsonic"
	"go.mongodb.org/mongo-driver/bson"
)

func TestNew(t *testing.T) {
	parser := bsonic.New()
	if parser == nil {
		t.Fatal("New() should return a non-nil parser")
	}
}

func TestParseEmptyQuery(t *testing.T) {
	parser := bsonic.New()

	query, err := parser.Parse("")
	if err != nil {
		t.Fatalf("Parse empty query should not return error, got: %v", err)
	}

	if len(query) != 0 {
		t.Fatalf("Empty query should return empty BSON, got: %+v", query)
	}
}

func TestParseWhitespaceQuery(t *testing.T) {
	parser := bsonic.New()

	query, err := parser.Parse("   ")
	if err != nil {
		t.Fatalf("Parse whitespace query should not return error, got: %v", err)
	}

	if len(query) != 0 {
		t.Fatalf("Whitespace query should return empty BSON, got: %+v", query)
	}
}

func TestParseSimpleFieldValue(t *testing.T) {
	parser := bsonic.New()

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

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestParseWildcardQuery(t *testing.T) {
	parser := bsonic.New()

	query, err := parser.Parse("name:jo*")
	if err != nil {
		t.Fatalf("Parse wildcard query should not return error, got: %v", err)
	}

	expected := bson.M{
		"name": bson.M{
			"$regex":   "^jo.*",
			"$options": "i",
		},
	}

	if !compareBSONValues(query, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, query)
	}
}

func TestParseQuotedValue(t *testing.T) {
	parser := bsonic.New()

	query, err := parser.Parse(`name:"john doe"`)
	if err != nil {
		t.Fatalf("Parse quoted value should not return error, got: %v", err)
	}

	expected := bson.M{"name": "john doe"}

	if !compareBSONValues(query, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, query)
	}
}

func TestParseAndOperator(t *testing.T) {
	parser := bsonic.New()

	query, err := parser.Parse("name:john AND age:25")
	if err != nil {
		t.Fatalf("Parse AND query should not return error, got: %v", err)
	}

	expected := bson.M{"name": "john", "age": 25.0}

	if !compareBSONValues(query, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, query)
	}
}

func TestParseInvalidQuery(t *testing.T) {
	parser := bsonic.New()

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
	parser := bsonic.New()

	query, err := parser.Parse("user.profile.email:john@example.com")
	if err != nil {
		t.Fatalf("Parse dot notation should not return error, got: %v", err)
	}

	expected := bson.M{"user.profile.email": "john@example.com"}

	if !compareBSONValues(query, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, query)
	}
}

func TestParseOrOperator(t *testing.T) {
	parser := bsonic.New()

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

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestParseNotOperator(t *testing.T) {
	parser := bsonic.New()

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

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestParseComplexOperators(t *testing.T) {
	parser := bsonic.New()

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

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestParseDateQueries(t *testing.T) {
	parser := bsonic.New()

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
	parser := bsonic.New()

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

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestParseInvalidDateQueries(t *testing.T) {
	parser := bsonic.New()

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
	parser := bsonic.New()

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
					{"name": bson.M{"$ne": "john"}},
					{"name": bson.M{"$ne": "jane"}},
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
	parser := bsonic.New()

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
	parser := bsonic.New()

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
