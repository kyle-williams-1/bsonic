package bsonic_test

import (
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

func TestTextSearchConfiguration(t *testing.T) {
	// Test default parser configuration
	parser := bsonic.New()
	if parser.SearchMode != bsonic.SearchModeDisabled {
		t.Error("Default parser should use SearchModeDisabled")
	}

	// Test parser with text search enabled
	parserWithText := bsonic.NewWithTextSearch()
	if parserWithText.SearchMode != bsonic.SearchModeText {
		t.Error("Parser with text search should use SearchModeText")
	}

	// Test setting search modes
	parser.SetSearchMode(bsonic.SearchModeText)
	if parser.SearchMode != bsonic.SearchModeText {
		t.Error("Search mode should be SearchModeText after setting it")
	}

	parser.SetSearchMode(bsonic.SearchModeDisabled)
	if parser.SearchMode != bsonic.SearchModeDisabled {
		t.Error("Search mode should be SearchModeDisabled after setting it")
	}
}

func TestTextSearchQueries(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "search terms",
			expected: bson.M{
				"$text": bson.M{"$search": "search terms"},
			},
			desc: "simple text search",
		},
		{
			input: "multiple words search",
			expected: bson.M{
				"$text": bson.M{"$search": "multiple words search"},
			},
			desc: "text search with multiple words",
		},
		{
			input: "  whitespace  around  ",
			expected: bson.M{
				"$text": bson.M{"$search": "whitespace  around"},
			},
			desc: "text search with whitespace",
		},
		{
			input: "special-chars!@#$%",
			expected: bson.M{
				"$text": bson.M{"$search": "special-chars!@#$%"},
			},
			desc: "text search with special characters",
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

func TestTextSearchDisabled(t *testing.T) {
	parser := bsonic.New() // Default parser with text search disabled

	// Text search queries should be treated as field searches when disabled
	_, err := parser.Parse("search terms")
	if err == nil {
		t.Error("Expected error for text search query when text search is disabled")
	}
}

func TestFieldSearchTakesPrecedence(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	// Queries with field:value pairs should use field search even when text search is enabled
	query, err := parser.Parse("name:john")
	if err != nil {
		t.Fatalf("Parse should not return error, got: %v", err)
	}

	expected := bson.M{"name": "john"}
	if !compareBSONValues(query, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, query)
	}
}

func TestMixedQueries(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "engineer active:true",
			expected: bson.M{
				"$and": []bson.M{
					{"active": true},
					{"$text": bson.M{"$search": "engineer"}},
				},
			},
			desc: "text search with field search",
		},
		{
			input: "software name:john",
			expected: bson.M{
				"$and": []bson.M{
					{"name": "john"},
					{"$text": bson.M{"$search": "software"}},
				},
			},
			desc: "text search with name field",
		},
		{
			input: "designer role:user AND active:true",
			expected: bson.M{
				"$and": []bson.M{
					{"role": "user", "active": true},
					{"$text": bson.M{"$search": "designer"}},
				},
			},
			desc: "text search with complex field query",
		},
		{
			input: "devops role:admin OR name:charlie",
			expected: bson.M{
				"$and": []bson.M{
					{"$or": []bson.M{
						{"role": "admin"},
						{"name": "charlie"},
					}},
					{"$text": bson.M{"$search": "devops"}},
				},
			},
			desc: "text search with OR field query",
		},
		{
			input: "multiple text terms active:true",
			expected: bson.M{
				"$and": []bson.M{
					{"active": true},
					{"$text": bson.M{"$search": "multiple text terms"}},
				},
			},
			desc: "multiple text terms with field search",
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

func TestEmptyTextSearchQuery(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	// Empty queries should return empty BSON
	query, err := parser.Parse("")
	if err != nil {
		t.Fatalf("Empty query should not return error, got: %v", err)
	}

	if len(query) != 0 {
		t.Fatalf("Empty query should return empty BSON, got: %+v", query)
	}
}

func TestWhitespaceOnlyTextSearchQuery(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	// Whitespace-only queries should return empty BSON
	query, err := parser.Parse("   ")
	if err != nil {
		t.Fatalf("Whitespace query should not return error, got: %v", err)
	}

	if len(query) != 0 {
		t.Fatalf("Whitespace query should return empty BSON, got: %+v", query)
	}
}

func TestTextSearchNodeHandling(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	// Test with valid text search node
	node := &bsonic.Node{
		Type:  bsonic.NodeTextSearch,
		Value: "search term",
	}

	result := parser.HandleTextSearchNode(node)
	expected := bson.M{"$text": bson.M{"$search": "search term"}}
	if !compareBSONValues(result, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, result)
	}

	// Test with nil value
	node.Value = nil
	result = parser.HandleTextSearchNode(node)
	expected = bson.M{}
	if !compareBSONValues(result, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, result)
	}

	// Test with integer value (should convert to string)
	node.Value = 123
	result = parser.HandleTextSearchNode(node)
	expected = bson.M{"$text": bson.M{"$search": "123"}}
	if !compareBSONValues(result, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, result)
	}

	// Test with float value (should convert to string)
	node.Value = 45.67
	result = parser.HandleTextSearchNode(node)
	expected = bson.M{"$text": bson.M{"$search": "45.67"}}
	if !compareBSONValues(result, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, result)
	}

	// Test with boolean value (should convert to string)
	node.Value = true
	result = parser.HandleTextSearchNode(node)
	expected = bson.M{"$text": bson.M{"$search": "true"}}
	if !compareBSONValues(result, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, result)
	}

	// Test with disabled search mode
	parser.SetSearchMode(bsonic.SearchModeDisabled)
	node.Value = "search term"
	result = parser.HandleTextSearchNode(node)
	expected = bson.M{}
	if !compareBSONValues(result, expected) {
		t.Fatalf("Expected %+v, got %+v", expected, result)
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
					{
						"$or": bson.M{
							"$ne": []bson.M{
								{"status": "active"},
								{"status": "pending"},
							},
						},
					},
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
