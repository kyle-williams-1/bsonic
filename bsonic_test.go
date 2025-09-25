package bsonic_test

import (
	"testing"
	"time"

	"github.com/kyle-williams-1/bsonic"
	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/factory"
	"go.mongodb.org/mongo-driver/bson"
)

func TestNew(t *testing.T) {
	parser := bsonic.New()
	if parser == nil {
		t.Fatal("New() should return a non-nil parser")
	}
}

func TestParsePackageLevel(t *testing.T) {
	// Test the new package-level Parse function
	query, err := bsonic.Parse("name:john")
	if err != nil {
		t.Fatalf("Parse should not return error, got: %v", err)
	}

	expected := bson.M{"name": "john"}
	if !compareBSONValues(query, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, query)
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

	invalidQueries := []string{":value"}

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
			desc: "exact datetime with timezone",
		},
		{
			input: "created_at:2023-01-15T10:30:00",
			expected: bson.M{
				"created_at": time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			desc: "datetime without timezone",
		},
		{
			input: "created_at:2023-01-15 10:30:00",
			expected: bson.M{
				"created_at": time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			desc: "datetime with space separator",
		},
		{
			input: "created_at:01/15/2023",
			expected: bson.M{
				"created_at": time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			desc: "US date format",
		},
		{
			input: "created_at:2023/01/15",
			expected: bson.M{
				"created_at": time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			desc: "ISO date format with slashes",
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

func TestParseNumberRangeQueries(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "age:[18 TO 65]",
			expected: bson.M{
				"age": bson.M{
					"$gte": 18.0,
					"$lte": 65.0,
				},
			},
			desc: "number range",
		},
		{
			input: "price:[10.50 TO 99.99]",
			expected: bson.M{
				"price": bson.M{
					"$gte": 10.50,
					"$lte": 99.99,
				},
			},
			desc: "decimal number range",
		},
		{
			input: "age:[18 TO *]",
			expected: bson.M{
				"age": bson.M{
					"$gte": 18.0,
				},
			},
			desc: "number range with wildcard end",
		},
		{
			input: "age:[* TO 65]",
			expected: bson.M{
				"age": bson.M{
					"$lte": 65.0,
				},
			},
			desc: "number range with wildcard start",
		},
		{
			input: "score:>85",
			expected: bson.M{
				"score": bson.M{
					"$gt": 85.0,
				},
			},
			desc: "number greater than",
		},
		{
			input: "score:<60",
			expected: bson.M{
				"score": bson.M{
					"$lt": 60.0,
				},
			},
			desc: "number less than",
		},
		{
			input: "score:>=90",
			expected: bson.M{
				"score": bson.M{
					"$gte": 90.0,
				},
			},
			desc: "number greater than or equal",
		},
		{
			input: "score:<=50",
			expected: bson.M{
				"score": bson.M{
					"$lte": 50.0,
				},
			},
			desc: "number less than or equal",
		},
		{
			input: "age:[18 TO 65] AND status:active",
			expected: bson.M{
				"age": bson.M{
					"$gte": 18.0,
					"$lte": 65.0,
				},
				"status": "active",
			},
			desc: "number range with AND",
		},
		{
			input: "age:>18 OR score:<60",
			expected: bson.M{
				"$or": []bson.M{
					{"age": bson.M{"$gt": 18.0}},
					{"score": bson.M{"$lt": 60.0}},
				},
			},
			desc: "number comparisons with OR",
		},
		{
			input: "age:[18 TO 65] OR score:[80 TO 100]",
			expected: bson.M{
				"$or": []bson.M{
					{"age": bson.M{"$gte": 18.0, "$lte": 65.0}},
					{"score": bson.M{"$gte": 80.0, "$lte": 100.0}},
				},
			},
			desc: "number ranges with OR",
		},
		{
			input: "(age:[18 TO 65] OR score:[80 TO 100]) AND status:active",
			expected: bson.M{
				"$and": []bson.M{
					{
						"$or": []bson.M{
							{"age": bson.M{"$gte": 18.0, "$lte": 65.0}},
							{"score": bson.M{"$gte": 80.0, "$lte": 100.0}},
						},
					},
					{"status": "active"},
				},
			},
			desc: "number ranges with grouped status",
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

func TestParseNotWithAndExpressions(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "NOT (name:john AND age:25)",
			expected: bson.M{
				"name": bson.M{"$ne": "john"},
				"age":  bson.M{"$ne": 25.0},
			},
			desc: "NOT with grouped AND",
		},
		{
			input: "NOT (status:active AND role:admin AND age:30)",
			expected: bson.M{
				"status": bson.M{"$ne": "active"},
				"role":   bson.M{"$ne": "admin"},
				"age":    bson.M{"$ne": 30.0},
			},
			desc: "NOT with multiple AND conditions",
		},
		{
			input: "NOT (name:jo* AND age:25)",
			expected: bson.M{
				"name": bson.M{"$ne": bson.M{"$regex": "^jo.*", "$options": "i"}},
				"age":  bson.M{"$ne": 25.0},
			},
			desc: "NOT with wildcard AND condition",
		},
		{
			input: "NOT (created_at:>2024-01-01 AND status:active)",
			expected: bson.M{
				"created_at": bson.M{"$ne": bson.M{"$gt": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
				"status":     bson.M{"$ne": "active"},
			},
			desc: "NOT with date comparison AND condition",
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

func TestParseWildcardPatterns(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "name:*john*",
			expected: bson.M{
				"name": bson.M{
					"$regex":   ".*john.*",
					"$options": "i",
				},
			},
			desc: "contains pattern",
		},
		{
			input: "name:*john",
			expected: bson.M{
				"name": bson.M{
					"$regex":   ".*john$",
					"$options": "i",
				},
			},
			desc: "ends with pattern",
		},
		{
			input: "name:john*",
			expected: bson.M{
				"name": bson.M{
					"$regex":   "^john.*",
					"$options": "i",
				},
			},
			desc: "starts with pattern",
		},
		{
			input: "name:jo*n",
			expected: bson.M{
				"name": bson.M{
					"$regex":   "^jo.*n$",
					"$options": "i",
				},
			},
			desc: "starts and ends with specific patterns",
		},
		{
			input: "name:*",
			expected: bson.M{
				"name": bson.M{
					"$regex":   ".*",
					"$options": "i",
				},
			},
			desc: "wildcard only",
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

func TestParseWhitespaceHandling(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input:    "  name:john  ",
			expected: bson.M{"name": "john"},
			desc:     "leading and trailing whitespace",
		},
		{
			input:    "\tname:john\t",
			expected: bson.M{"name": "john"},
			desc:     "tab whitespace",
		},
		{
			input:    "\nname:john\n",
			expected: bson.M{"name": "john"},
			desc:     "newline whitespace",
		},
		{
			input:    "  name:john  AND  age:25  ",
			expected: bson.M{"name": "john", "age": 25.0},
			desc:     "whitespace around AND operator",
		},
		{
			input: "  name:john  OR  age:25  ",
			expected: bson.M{
				"$or": []bson.M{
					{"name": "john"},
					{"age": 25.0},
				},
			},
			desc: "whitespace around OR operator",
		},
		{
			input:    "  NOT  name:john  ",
			expected: bson.M{"name": bson.M{"$ne": "john"}},
			desc:     "whitespace around NOT operator",
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

func TestParseEdgeCases(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input:    "name:false",
			expected: bson.M{"name": false},
			desc:     "boolean false value",
		},
		{
			input:    "name:true",
			expected: bson.M{"name": true},
			desc:     "boolean true value",
		},
		{
			input:    "age:0",
			expected: bson.M{"age": 0.0},
			desc:     "zero numeric value",
		},
		{
			input:    "age:-1",
			expected: bson.M{"age": -1.0},
			desc:     "negative numeric value",
		},
		{
			input:    "age:3.14",
			expected: bson.M{"age": 3.14},
			desc:     "decimal numeric value",
		},
		{
			input:    "name:",
			expected: bson.M{},
			desc:     "empty value should error",
		},
		{
			input:    ":value",
			expected: bson.M{},
			desc:     "empty field should error",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result, err := parser.Parse(test.input)
			if test.desc == "empty value should error" || test.desc == "empty field should error" {
				if err == nil {
					t.Fatalf("Expected error for '%s', got none", test.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestParseErrorConditions(t *testing.T) {
	parser := bsonic.New()

	errorTests := []struct {
		input string
		desc  string
	}{
		{
			input: "name:john AND",
			desc:  "AND at end without right operand",
		},
		{
			input: "name:john OR",
			desc:  "OR at end without right operand",
		},
		{
			input: "NOT",
			desc:  "NOT without operand",
		},
		{
			input: "name:john AND NOT",
			desc:  "NOT at end without operand",
		},
	}

	for _, test := range errorTests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := parser.Parse(test.input)
			if err == nil {
				t.Fatalf("Expected error for '%s', got none", test.input)
			}
		})
	}
}

func TestParseComplexNestedQueries(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "((name:john OR name:jane) AND (age:25 OR age:30)) OR status:active",
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
							{
								"$or": []bson.M{
									{"age": 25.0},
									{"age": 30.0},
								},
							},
						},
					},
					{"status": "active"},
				},
			},
			desc: "complex nested parentheses",
		},
		{
			input: "NOT ((name:john OR name:jane) AND age:25)",
			expected: bson.M{
				"$or": []bson.M{
					{
						"$or": bson.M{
							"$ne": []bson.M{
								{"name": "john"},
								{"name": "jane"},
							},
						},
					},
					{"age": bson.M{"$ne": 25.0}},
				},
			},
			desc: "NOT with complex nested expression",
		},
		{
			input: "(name:jo* OR name:ja*) AND (age:18 AND age:65)",
			expected: bson.M{
				"$and": []bson.M{
					{
						"$or": []bson.M{
							{"name": bson.M{"$regex": "^jo.*", "$options": "i"}},
							{"name": bson.M{"$regex": "^ja.*", "$options": "i"}},
						},
					},
					{
						"$and": []bson.M{
							{"age": 65.0},
							{"age": 18.0},
						},
					},
				},
			},
			desc: "wildcards with numeric values",
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

func TestParseNotWithOrExpressions(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
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
			input: "NOT (status:active OR status:pending OR status:inactive)",
			expected: bson.M{
				"$and": []bson.M{
					{"status": bson.M{"$ne": "active"}},
					{"status": bson.M{"$ne": "pending"}},
					{"status": bson.M{"$ne": "inactive"}},
				},
			},
			desc: "NOT with multiple OR conditions",
		},
		{
			input: "NOT (name:jo* OR name:ja*)",
			expected: bson.M{
				"$and": []bson.M{
					{"name": bson.M{"$ne": bson.M{"$regex": "^jo.*", "$options": "i"}}},
					{"name": bson.M{"$ne": bson.M{"$regex": "^ja.*", "$options": "i"}}},
				},
			},
			desc: "NOT with wildcard OR conditions",
		},
		{
			input: "NOT (created_at:>2024-01-01 OR updated_at:<2023-01-01)",
			expected: bson.M{
				"$and": []bson.M{
					{"created_at": bson.M{"$ne": bson.M{"$gt": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}}},
					{"updated_at": bson.M{"$ne": bson.M{"$lt": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}}},
				},
			},
			desc: "NOT with date comparison OR conditions",
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

func compareBSONValues(actual, expected interface{}) bool {
	// Handle time.Time comparison
	if actualTime, ok := actual.(time.Time); ok {
		return compareTimeValues(actualTime, expected)
	}

	// Handle bson.M comparison
	if actualMap, ok := actual.(bson.M); ok {
		return compareBSONMaps(actualMap, expected)
	}

	// Handle []bson.M comparison
	if actualArray, ok := actual.([]bson.M); ok {
		return compareBSONArrays(actualArray, expected)
	}

	// Default comparison
	return actual == expected
}

// compareTimeValues compares time.Time values
func compareTimeValues(actualTime time.Time, expected interface{}) bool {
	expectedTime, ok := expected.(time.Time)
	return ok && actualTime.Equal(expectedTime)
}

// compareBSONMaps compares bson.M values
func compareBSONMaps(actualMap bson.M, expected interface{}) bool {
	expectedMap, ok := expected.(bson.M)
	if !ok {
		return false
	}

	if len(actualMap) != len(expectedMap) {
		return false
	}

	for key, expectedValue := range expectedMap {
		actualValue, exists := actualMap[key]
		if !exists || !compareBSONValues(actualValue, expectedValue) {
			return false
		}
	}
	return true
}

// compareBSONArrays compares []bson.M values
func compareBSONArrays(actualArray []bson.M, expected interface{}) bool {
	expectedArray, ok := expected.([]bson.M)
	if !ok {
		return false
	}

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

func TestNOTInParentheses(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
		desc     string
	}{
		{
			name:  "NOT in single parentheses",
			query: "(NOT role:admin)",
			expected: bson.M{
				"role": bson.M{"$ne": "admin"},
			},
			desc: "NOT operation inside single parentheses",
		},
		{
			name:  "NOT in multiple parentheses",
			query: "(NOT role:admin) AND (NOT name:Bob Johnson)",
			expected: bson.M{
				"role": bson.M{"$ne": "admin"},
				"name": bson.M{"$ne": "Bob Johnson"},
			},
			desc: "Multiple NOT operations in separate parentheses",
		},
		{
			name:  "NOT with complex field in parentheses",
			query: "(NOT name:jo*) AND (NOT status:active)",
			expected: bson.M{
				"name":   bson.M{"$ne": bson.M{"$regex": "^jo.*", "$options": "i"}},
				"status": bson.M{"$ne": "active"},
			},
			desc: "NOT with wildcard and simple field in parentheses",
		},
		{
			name:  "NOT with quoted value in parentheses",
			query: "(NOT name:\"john doe\") AND (NOT email:\"test@example.com\")",
			expected: bson.M{
				"name":  bson.M{"$ne": "john doe"},
				"email": bson.M{"$ne": "test@example.com"},
			},
			desc: "NOT with quoted values in parentheses",
		},
		{
			name:  "NOT with nested parentheses",
			query: "((NOT role:admin) AND (NOT name:Bob)) OR status:active",
			expected: bson.M{
				"$or": []bson.M{
					{
						"role": bson.M{"$ne": "admin"},
						"name": bson.M{"$ne": "Bob"},
					},
					{"status": "active"},
				},
			},
			desc: "NOT with nested parentheses and OR condition",
		},
		{
			name:  "NOT with date comparison in parentheses",
			query: "(NOT created_at:>2024-01-01) AND (NOT updated_at:<2023-01-01)",
			expected: bson.M{
				"created_at": bson.M{"$ne": bson.M{"$gt": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
				"updated_at": bson.M{"$ne": bson.M{"$lt": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}},
			},
			desc: "NOT with date comparisons in parentheses",
		},
		{
			name:  "NOT with whitespace in parentheses",
			query: "  (  NOT  role:admin  )  AND  (  NOT  name:Bob  )  ",
			expected: bson.M{
				"role": bson.M{"$ne": "admin"},
				"name": bson.M{"$ne": "Bob"},
			},
			desc: "NOT with extra whitespace in parentheses",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse failed for query %q: %v", test.query, err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Query: %s\nExpected: %+v\nGot: %+v", test.query, test.expected, result)
			}
		})
	}
}

// TestAdditionalEdgeCases tests additional edge cases through public API
func TestAdditionalEdgeCases(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name        string
		query       string
		expectError bool
		expected    bson.M
	}{
		{
			name:        "empty query",
			query:       "",
			expectError: false,
			expected:    bson.M{},
		},
		{
			name:        "whitespace only",
			query:       "   ",
			expectError: false,
			expected:    bson.M{},
		},
		{
			name:        "invalid date range both wildcards",
			query:       "created_at:[* TO *]",
			expectError: false,
			expected:    bson.M{"created_at": "[* TO *]"},
		},
		{
			name:        "invalid number range both wildcards",
			query:       "age:[* TO *]",
			expectError: false,
			expected:    bson.M{"age": "[* TO *]"},
		},
		{
			name:        "invalid date range with bad dates",
			query:       "created_at:[invalid TO 2023-12-31]",
			expectError: false,
			expected:    bson.M{"created_at": "[invalid TO 2023-12-31]"},
		},
		{
			name:        "invalid number range with bad numbers",
			query:       "age:[not-a-number TO 100]",
			expectError: false,
			expected:    bson.M{"age": "[not-a-number TO 100]"},
		},
		{
			name:        "valid date range with wildcard start",
			query:       "created_at:[* TO 2023-12-31]",
			expectError: false,
			expected:    bson.M{"created_at": bson.M{"$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)}},
		},
		{
			name:        "valid date range with wildcard end",
			query:       "created_at:[2023-01-01 TO *]",
			expectError: false,
			expected:    bson.M{"created_at": bson.M{"$gte": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}},
		},
		{
			name:        "valid number range with wildcard start",
			query:       "age:[* TO 65]",
			expectError: false,
			expected:    bson.M{"age": bson.M{"$lte": 65.0}},
		},
		{
			name:        "valid number range with wildcard end",
			query:       "age:[18 TO *]",
			expectError: false,
			expected:    bson.M{"age": bson.M{"$gte": 18.0}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if test.expectError {
				if err == nil {
					t.Fatalf("Expected error for query: %s", test.query)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for query: %s, got: %v", test.query, err)
				}
				if !compareBSONValues(result, test.expected) {
					t.Fatalf("Query: %s\nExpected: %+v\nGot: %+v", test.query, test.expected, result)
				}
			}
		})
	}
}

func TestComplexQueryEdgeCases(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name        string
		query       string
		expectError bool
		description string
	}{
		{
			name:        "nested parentheses with complex logic",
			query:       "((name:john OR name:jane) AND age:25) OR (status:active AND role:admin)",
			expectError: false,
			description: "complex nested parentheses",
		},
		{
			name:        "multiple NOT operations",
			query:       "NOT name:john AND NOT age:25",
			expectError: false,
			description: "multiple NOT operations",
		},
		{
			name:        "NOT with parentheses",
			query:       "NOT (name:john OR age:25)",
			expectError: false,
			description: "NOT with grouped OR",
		},
		{
			name:        "complex wildcard patterns",
			query:       "name:jo*n AND email:*@example.com",
			expectError: false,
			description: "complex wildcard patterns",
		},
		{
			name:        "mixed date and number ranges",
			query:       "created_at:[2023-01-01 TO 2023-12-31] AND age:[18 TO 65]",
			expectError: false,
			description: "mixed date and number ranges",
		},
		{
			name:        "complex comparison operators",
			query:       "score:>85 AND rating:>=4.5 AND price:<100",
			expectError: false,
			description: "complex comparison operators",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if test.expectError {
				if err == nil {
					t.Fatalf("Expected error for query: %s", test.query)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for query: %s, got: %v", test.query, err)
				}
				if result == nil {
					t.Fatalf("Expected non-nil result for query: %s", test.query)
				}
				// Just verify it parses without error and produces some BSON
				if len(result) == 0 && test.query != "" {
					t.Fatalf("Expected non-empty result for query: %s", test.query)
				}
			}
		})
	}
}

// TestNewWithConfig tests the NewWithConfig function
func TestNewWithConfig(t *testing.T) {
	// Test with valid config
	cfg := &config.Config{
		Language:  config.LanguageLucene,
		Formatter: config.FormatterBSON,
	}

	parser, err := bsonic.NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig should not return error, got: %v", err)
	}
	if parser == nil {
		t.Fatal("NewWithConfig should return a non-nil parser")
	}

	// Test with invalid language
	invalidCfg := &config.Config{
		Language:  "invalid",
		Formatter: config.FormatterBSON,
	}

	_, err = bsonic.NewWithConfig(invalidCfg)
	if err == nil {
		t.Fatal("NewWithConfig should return error for invalid language")
	}

	// Test with invalid formatter
	invalidCfg2 := &config.Config{
		Language:  config.LanguageLucene,
		Formatter: "invalid",
	}

	_, err = bsonic.NewWithConfig(invalidCfg2)
	if err == nil {
		t.Fatal("NewWithConfig should return error for invalid formatter")
	}
}

// TestConfigFunctions tests config package functions
func TestConfigFunctions(t *testing.T) {
	// Test Default config
	cfg := config.Default()
	if cfg == nil {
		t.Fatal("Default() should return a non-nil config")
	}

	// Test WithLanguage
	cfgWithLang := cfg.WithLanguage(config.LanguageLucene)
	if cfgWithLang.Language != config.LanguageLucene {
		t.Error("WithLanguage should set the language")
	}

	// Test WithFormatter
	cfgWithFmt := cfg.WithFormatter(config.FormatterBSON)
	if cfgWithFmt.Formatter != config.FormatterBSON {
		t.Error("WithFormatter should set the formatter")
	}
}

// TestFactoryFunctions tests factory package functions
func TestFactoryFunctions(t *testing.T) {
	// Test CreateParser
	parser, err := factory.CreateParser(config.LanguageLucene)
	if err != nil {
		t.Fatalf("CreateParser should not return error, got: %v", err)
	}
	if parser == nil {
		t.Fatal("CreateParser should return a non-nil parser")
	}

	// Test CreateParser with invalid language
	_, err = factory.CreateParser("invalid")
	if err == nil {
		t.Fatal("CreateParser should return error for invalid language")
	}

	// Test CreateFormatter
	formatter, err := factory.CreateFormatter(config.FormatterBSON)
	if err != nil {
		t.Fatalf("CreateFormatter should not return error, got: %v", err)
	}
	if formatter == nil {
		t.Fatal("CreateFormatter should return a non-nil formatter")
	}

	// Test CreateFormatter with invalid formatter
	_, err = factory.CreateFormatter("invalid")
	if err == nil {
		t.Fatal("CreateFormatter should return error for invalid formatter")
	}

	// Test CreateBSONFormatter
	bsonFormatter := factory.CreateBSONFormatter()
	if bsonFormatter == nil {
		t.Fatal("CreateBSONFormatter should return a non-nil formatter")
	}
}

func TestParseFreeTextSearch(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:  "Simple free text search",
			query: `"John Doe"`,
			expected: bson.M{
				"$text": bson.M{
					"$search": `"John Doe"`,
				},
			},
		},
		{
			name:  "Free text search with single quotes",
			query: `'John Doe'`,
			expected: bson.M{
				"$text": bson.M{
					"$search": `"John Doe"`,
				},
			},
		},
		{
			name:  "Free text search with field query",
			query: `"John Doe" AND active:true`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": `"John Doe"`,
						},
					},
					{
						"active": true,
					},
				},
			},
		},
		{
			name:  "Free text search with OR condition",
			query: `"John Doe" AND (active:true OR role:admin)`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": `"John Doe"`,
						},
					},
					{
						"$or": []bson.M{
							{"active": true},
							{"role": "admin"},
						},
					},
				},
			},
		},
		{
			name:  "Free text search with NOT condition",
			query: `"John Doe" AND NOT active:false`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": `"John Doe"`,
						},
					},
					{
						"active": bson.M{
							"$ne": false,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parser.Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse should not return error for %s, got: %v", tt.name, err)
			}

			if !compareBSONValues(query, tt.expected) {
				t.Fatalf("Test %s: Expected %+v, got %+v", tt.name, tt.expected, query)
			}
		})
	}
}
