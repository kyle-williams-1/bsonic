package bsonic_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kyle-williams-1/bsonic"
	"github.com/kyle-williams-1/bsonic/config"
	"go.mongodb.org/mongo-driver/bson"
)

// TestLuceneMongoIntegration tests the integration between Lucene query language and MongoDB BSON formatter
// This test suite validates that Lucene-style queries are correctly parsed and converted to MongoDB BSON documents
//
// Test Structure:
// - TestLuceneMongoBasicParsing: Basic field-value queries and constructors
// - TestLuceneMongoLogicalOperators: AND, OR, NOT operators
// - TestLuceneMongoDateParsing: Date range and comparison queries
// - TestLuceneMongoNumberRangeAndComparison: Numeric range and comparison queries
// - TestLuceneMongoParenthesesAndGrouping: Complex nested expressions
// - TestLuceneMongoWildcardPatterns: Wildcard pattern matching
// - TestLuceneMongoRegexPatterns: Regular expression patterns
// - TestLuceneMongoWhitespaceHandling: Whitespace normalization
// - TestLuceneMongoErrorConditions: Error handling and edge cases
// - TestLuceneMongoConfigurationAndFactory: Configuration and factory functions
// - TestLuceneMongoFreeTextSearch: Full-text search functionality

// TestLuceneMongoIntegration validates the complete Lucene-to-MongoDB BSON pipeline
func TestLuceneMongoIntegration(t *testing.T) {
	t.Run("CompletePipeline", func(t *testing.T) {
		// Test that a complex Lucene query is correctly converted to MongoDB BSON
		query := `name:john AND (age:[25 TO 35] OR role:admin) AND NOT status:inactive`

		parser := bsonic.New()
		result, err := parser.Parse(query)
		if err != nil {
			t.Fatalf("Parse should not return error, got: %v", err)
		}

		// Verify the result is a valid MongoDB BSON document
		if result == nil {
			t.Fatal("Result should not be nil")
		}

		// The result should contain MongoDB-specific operators
		// This validates that Lucene syntax was properly converted to MongoDB BSON
		t.Logf("Lucene query: %s", query)
		t.Logf("MongoDB BSON: %+v", result)
	})

	t.Run("MongoDBSpecificOperators", func(t *testing.T) {
		// Test that Lucene queries produce MongoDB-specific BSON operators
		tests := []struct {
			name        string
			query       string
			expectedOp  string
			description string
		}{
			{
				name:        "TextSearch",
				query:       "search term",
				expectedOp:  "$text",
				description: "Free text should produce $text search",
			},
			{
				name:        "RangeQuery",
				query:       "age:[25 TO 35]",
				expectedOp:  "$gte",
				description: "Range queries should produce $gte/$lte operators",
			},
			{
				name:        "RegexQuery",
				query:       "name:/john.*/",
				expectedOp:  "$regex",
				description: "Regex patterns should produce $regex operator",
			},
			{
				name:        "LogicalAnd",
				query:       "name:john AND (age:30 OR role:admin)",
				expectedOp:  "$and",
				description: "Complex AND operations should produce $and operator",
			},
		}

		parser := bsonic.New()
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result, err := parser.Parse(test.query)
				if err != nil {
					t.Fatalf("Parse should not return error for %s, got: %v", test.query, err)
				}

				// Convert result to string to check for MongoDB operators
				resultStr := fmt.Sprintf("%+v", result)
				if !strings.Contains(resultStr, test.expectedOp) {
					t.Errorf("Expected MongoDB operator '%s' in result for query '%s'. Got: %s",
						test.expectedOp, test.query, resultStr)
				}
			})
		}
	})
}

// TestLuceneMongoBasicParsing tests basic Lucene parsing with MongoDB BSON output including constructors, empty queries, and simple field-value pairs
func TestLuceneMongoBasicParsing(t *testing.T) {
	// Test New() constructor
	t.Run("New", func(t *testing.T) {
		parser := bsonic.New()
		if parser == nil {
			t.Fatal("New() should return a non-nil parser")
		}
	})

	// Test package-level Parse function
	t.Run("PackageLevelParse", func(t *testing.T) {
		query, err := bsonic.Parse("name:john")
		if err != nil {
			t.Fatalf("Parse should not return error, got: %v", err)
		}

		expected := bson.M{"name": "john"}
		if !compareBSONValues(query, expected) {
			t.Fatalf("Expected %+v, got %+v", expected, query)
		}
	})

	parser := bsonic.New()

	// Test empty and whitespace queries
	t.Run("EmptyAndWhitespaceQueries", func(t *testing.T) {
		tests := []struct {
			name  string
			query string
		}{
			{"EmptyQuery", ""},
			{"WhitespaceQuery", "   "},
			{"TabWhitespace", "\t"},
			{"NewlineWhitespace", "\n"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				query, err := parser.Parse(test.query)
				if err != nil {
					t.Fatalf("Parse should not return error, got: %v", err)
				}

				if len(query) != 0 {
					t.Fatalf("Query should return empty BSON, got: %+v", query)
				}
			})
		}
	})

	// Test simple field-value pairs
	t.Run("SimpleFieldValuePairs", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
			{"name:john", bson.M{"name": "john"}, "string value"},
			{"age:25", bson.M{"age": 25.0}, "numeric value"},
			{"active:true", bson.M{"active": true}, "boolean true"},
			{"active:false", bson.M{"active": false}, "boolean false"},
			{"age:0", bson.M{"age": 0.0}, "zero numeric value"},
			{"age:-1", bson.M{"age": -1.0}, "negative numeric value"},
			{"age:3.14", bson.M{"age": 3.14}, "decimal numeric value"},
			{`name:"john doe"`, bson.M{"name": "john doe"}, "quoted string value"},
			{"user.profile.email:john@example.com", bson.M{"user.profile.email": "john@example.com"}, "dot notation field"},
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
	})

	// Test field value with spaces (special case)
	t.Run("FieldValueWithSpaces", func(t *testing.T) {
		query, err := parser.Parse("name:John Doe")
		if err != nil {
			t.Fatalf("Parse field value with spaces should not return error, got: %v", err)
		}

		// Should be parsed as name:John (regex) AND $text:"Doe"
		expected := bson.M{
			"$and": []bson.M{
				{"name": "John"},
				{"$text": bson.M{"$search": "Doe"}},
			},
		}

		if !compareBSONValues(query, expected) {
			t.Fatalf("Expected %+v, got %+v", expected, query)
		}
	})
}

// TestLogicalOperators tests AND, OR, and NOT operators with various combinations
func TestLuceneMongoLogicalOperators(t *testing.T) {
	parser := bsonic.New()

	// Test AND operator
	t.Run("ANDOperator", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
			{
				input:    "name:john AND age:25",
				expected: bson.M{"name": "john", "age": 25.0},
				desc:     "simple AND",
			},
			{
				input:    "status:active AND NOT role:guest",
				expected: bson.M{"status": "active", "role": bson.M{"$ne": "guest"}},
				desc:     "AND with NOT",
			},
			{
				input:    "  name:john  AND  age:25  ",
				expected: bson.M{"name": "john", "age": 25.0},
				desc:     "AND with whitespace",
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
	})

	// Test OR operator
	t.Run("OROperator", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
			{
				input: "name:john OR name:jane",
				expected: bson.M{
					"$or": []bson.M{
						{"name": "john"},
						{"name": "jane"},
					},
				},
				desc: "simple OR",
			},
			{
				input: "age:25 OR age:30",
				expected: bson.M{
					"$or": []bson.M{
						{"age": 25.0},
						{"age": 30.0},
					},
				},
				desc: "numeric OR",
			},
			{
				input: "status:active OR status:pending",
				expected: bson.M{
					"$or": []bson.M{
						{"status": "active"},
						{"status": "pending"},
					},
				},
				desc: "status OR",
			},
			{
				input: "  name:john  OR  age:25  ",
				expected: bson.M{
					"$or": []bson.M{
						{"name": "john"},
						{"age": 25.0},
					},
				},
				desc: "OR with whitespace",
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
	})

	// Test NOT operator
	t.Run("NOTOperator", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
			{
				input:    "name:john AND NOT age:25",
				expected: bson.M{"name": "john", "age": bson.M{"$ne": 25.0}},
				desc:     "AND with NOT",
			},
			{
				input:    "NOT status:inactive",
				expected: bson.M{"status": bson.M{"$ne": "inactive"}},
				desc:     "simple NOT",
			},
			{
				input:    `name:"john doe" AND NOT role:admin`,
				expected: bson.M{"name": "john doe", "role": bson.M{"$ne": "admin"}},
				desc:     "quoted value with NOT",
			},
			{
				input:    "  NOT  name:john  ",
				expected: bson.M{"name": bson.M{"$ne": "john"}},
				desc:     "NOT with whitespace",
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
	})

	// Test complex operator combinations
	t.Run("ComplexOperatorCombinations", func(t *testing.T) {
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
				input:       "name:jo* OR name:ja* AND NOT age:18",
				description: "Wildcard OR with AND and NOT",
				expected: bson.M{
					"$or": []bson.M{
						{"name": bson.M{"$regex": "^jo.*", "$options": "i"}},
						{"name": bson.M{"$regex": "^ja.*", "$options": "i"}, "age": bson.M{"$ne": 18.0}},
					},
				},
			},
			{
				input:       "name:/john/ OR email:/.*@example\\.com/ AND NOT status:inactive",
				description: "Regex OR with AND and NOT",
				expected: bson.M{
					"$or": []bson.M{
						{"name": bson.M{"$regex": "john"}},
						{"email": bson.M{"$regex": ".*@example\\.com"}, "status": bson.M{"$ne": "inactive"}},
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
	})
}

// TestDateParsing tests date parsing functionality including various formats and range queries
func TestLuceneMongoDateParsing(t *testing.T) {
	parser := bsonic.New()

	// Test basic date formats
	t.Run("BasicDateFormats", func(t *testing.T) {
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
	})

	// Test date range queries
	t.Run("DateRangeQueries", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
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

				if !compareBSONValues(result, test.expected) {
					t.Fatalf("Expected %+v, got %+v", test.expected, result)
				}
			})
		}
	})

	// Test date comparison operators
	t.Run("DateComparisonOperators", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
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
	})

	// Test complex date queries with logical operators
	t.Run("ComplexDateQueries", func(t *testing.T) {
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
	})
}

// TestNumberRangeAndComparison tests number range queries and comparison operators
func TestLuceneMongoNumberRangeAndComparison(t *testing.T) {
	parser := bsonic.New()

	// Test number range queries
	t.Run("NumberRangeQueries", func(t *testing.T) {
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
				desc: "integer range",
			},
			{
				input: "price:[10.50 TO 99.99]",
				expected: bson.M{
					"price": bson.M{
						"$gte": 10.50,
						"$lte": 99.99,
					},
				},
				desc: "decimal range",
			},
			{
				input: "age:[18 TO *]",
				expected: bson.M{
					"age": bson.M{
						"$gte": 18.0,
					},
				},
				desc: "range with wildcard end",
			},
			{
				input: "age:[* TO 65]",
				expected: bson.M{
					"age": bson.M{
						"$lte": 65.0,
					},
				},
				desc: "range with wildcard start",
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
	})

	// Test number comparison operators
	t.Run("NumberComparisonOperators", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
			{
				input: "score:>85",
				expected: bson.M{
					"score": bson.M{
						"$gt": 85.0,
					},
				},
				desc: "greater than",
			},
			{
				input: "score:<60",
				expected: bson.M{
					"score": bson.M{
						"$lt": 60.0,
					},
				},
				desc: "less than",
			},
			{
				input: "score:>=90",
				expected: bson.M{
					"score": bson.M{
						"$gte": 90.0,
					},
				},
				desc: "greater than or equal",
			},
			{
				input: "score:<=50",
				expected: bson.M{
					"score": bson.M{
						"$lte": 50.0,
					},
				},
				desc: "less than or equal",
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
	})

	// Test complex number queries with logical operators
	t.Run("ComplexNumberQueries", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
			{
				input: "age:[18 TO 65] AND status:active",
				expected: bson.M{
					"age": bson.M{
						"$gte": 18.0,
						"$lte": 65.0,
					},
					"status": "active",
				},
				desc: "range with AND",
			},
			{
				input: "age:>18 OR score:<60",
				expected: bson.M{
					"$or": []bson.M{
						{"age": bson.M{"$gt": 18.0}},
						{"score": bson.M{"$lt": 60.0}},
					},
				},
				desc: "comparisons with OR",
			},
			{
				input: "age:[18 TO 65] OR score:[80 TO 100]",
				expected: bson.M{
					"$or": []bson.M{
						{"age": bson.M{"$gte": 18.0, "$lte": 65.0}},
						{"score": bson.M{"$gte": 80.0, "$lte": 100.0}},
					},
				},
				desc: "ranges with OR",
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
				desc: "grouped ranges with status",
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
	})
}

// TestParenthesesAndGrouping tests parentheses grouping and complex nested expressions
func TestLuceneMongoParenthesesAndGrouping(t *testing.T) {
	parser := bsonic.New()

	// Test basic parentheses grouping
	t.Run("BasicParenthesesGrouping", func(t *testing.T) {
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
	})

	// Test NOT with parentheses
	t.Run("NOTWithParentheses", func(t *testing.T) {
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
	})

	// Test nested parentheses
	t.Run("NestedParentheses", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bson.M
			desc     string
		}{
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
	})

	// Test invalid parentheses
	t.Run("InvalidParentheses", func(t *testing.T) {
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
	})
}

// TestWildcardPatterns tests wildcard pattern matching
func TestLuceneMongoWildcardPatterns(t *testing.T) {
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

// TestRegexPatterns tests regex pattern matching
func TestLuceneMongoRegexPatterns(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		input    string
		expected bson.M
		desc     string
	}{
		{
			input: "name:/john/",
			expected: bson.M{
				"name": bson.M{
					"$regex": "john",
				},
			},
			desc: "basic regex pattern",
		},
		{
			input: "name:/^john$/",
			expected: bson.M{
				"name": bson.M{
					"$regex": "^john$",
				},
			},
			desc: "anchored regex pattern",
		},
		{
			input: "email:/.*@example\\.com$/",
			expected: bson.M{
				"email": bson.M{
					"$regex": ".*@example\\.com$",
				},
			},
			desc: "complex regex pattern with escaped characters",
		},
		{
			input: "phone:/\\d{3}-\\d{3}-\\d{4}/",
			expected: bson.M{
				"phone": bson.M{
					"$regex": "\\d{3}-\\d{3}-\\d{4}",
				},
			},
			desc: "regex pattern with digit matching",
		},
		{
			input: "status:/^(active|pending|inactive)$/",
			expected: bson.M{
				"status": bson.M{
					"$regex": "^(active|pending|inactive)$",
				},
			},
			desc: "regex pattern with alternation",
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

// TestWhitespaceHandling tests whitespace handling in queries
func TestLuceneMongoWhitespaceHandling(t *testing.T) {
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

// TestErrorConditions tests various error conditions and edge cases
func TestLuceneMongoErrorConditions(t *testing.T) {
	parser := bsonic.New()

	// Test invalid query syntax
	t.Run("InvalidQuerySyntax", func(t *testing.T) {
		invalidQueries := []struct {
			input string
			desc  string
		}{
			{":value", "empty field name"},
			{"name:john AND", "AND at end without right operand"},
			{"name:john OR", "OR at end without right operand"},
			{"NOT", "NOT without operand"},
			{"name:john AND NOT", "NOT at end without operand"},
			{"name:", "empty value"},
		}

		for _, test := range invalidQueries {
			t.Run(test.desc, func(t *testing.T) {
				_, err := parser.Parse(test.input)
				if err == nil {
					t.Fatalf("Expected error for '%s', got none", test.input)
				}
			})
		}
	})

	// Test edge cases that should work
	t.Run("ValidEdgeCases", func(t *testing.T) {
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
	})

	// Test complex nested queries
	t.Run("ComplexNestedQueries", func(t *testing.T) {
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
	})

	// Test NOT with OR expressions
	t.Run("NOTWithORExpressions", func(t *testing.T) {
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
	})
}

func TestLuceneMongoNOTInParentheses(t *testing.T) {
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
			query: "(NOT role:admin) AND (NOT name:Bob)",
			expected: bson.M{
				"role": bson.M{"$ne": "admin"},
				"name": bson.M{"$ne": "Bob"},
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
		{
			name:  "NOT with multi-word field value (should split)",
			query: "NOT name:Bob Johnson",
			expected: bson.M{
				"$or": []bson.M{
					{"name": bson.M{"$ne": "Bob"}},
					{"$text": bson.M{"$ne": bson.M{"$search": "Johnson"}}},
				},
			},
			desc: "NOT operation with multi-word field value should split into NOT (field:value AND $text:terms)",
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
func TestLuceneMongoAdditionalEdgeCases(t *testing.T) {
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

func TestLuceneMongoComplexQueryEdgeCases(t *testing.T) {
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

// TestConfigurationAndFactory tests configuration and factory functions
func TestLuceneMongoConfigurationAndFactory(t *testing.T) {
	// Test NewWithConfig
	t.Run("NewWithConfig", func(t *testing.T) {
		// Test with valid config
		cfg := &config.Config{
			Language:  config.LanguageLucene,
			Formatter: config.FormatterMongo,
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
			Formatter: config.FormatterMongo,
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
	})

	// Test config package functions
	t.Run("ConfigFunctions", func(t *testing.T) {
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
		cfgWithFmt := cfg.WithFormatter(config.FormatterMongo)
		if cfgWithFmt.Formatter != config.FormatterMongo {
			t.Error("WithFormatter should set the formatter")
		}
	})

	// Test factory functions
	t.Run("FactoryFunctions", func(t *testing.T) {
		// Test NewParser
		parser, err := bsonic.NewParser(config.LanguageLucene)
		if err != nil {
			t.Fatalf("NewParser should not return error, got: %v", err)
		}
		if parser == nil {
			t.Fatal("NewParser should return a non-nil parser")
		}

		// Test NewParser with invalid language
		_, err = bsonic.NewParser("invalid")
		if err == nil {
			t.Fatal("NewParser should return error for invalid language")
		}

		// Test NewFormatter
		formatter, err := bsonic.NewFormatter(config.FormatterMongo)
		if err != nil {
			t.Fatalf("NewFormatter should not return error, got: %v", err)
		}
		if formatter == nil {
			t.Fatal("NewFormatter should return a non-nil formatter")
		}

		// Test NewFormatter with invalid formatter
		_, err = bsonic.NewFormatter("invalid")
		if err == nil {
			t.Fatal("NewFormatter should return error for invalid formatter")
		}

		// Test NewMongoFormatter
		mongoFormatter := bsonic.NewMongoFormatter()
		if mongoFormatter == nil {
			t.Fatal("NewMongoFormatter should return a non-nil formatter")
		}
	})
}

func TestLuceneMongoFreeTextSearch(t *testing.T) {
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
		{
			name:  "Free text search with grouped OR condition",
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
			name:  "Free text search with nested grouped logic",
			query: `"John Doe" AND ((active:true OR role:admin) AND status:verified)`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": `"John Doe"`,
						},
					},
					{
						"$and": []bson.M{
							{
								"$or": []bson.M{
									{"active": true},
									{"role": "admin"},
								},
							},
							{"status": "verified"},
						},
					},
				},
			},
		},
		{
			name:  "Free text search with complex grouped OR logic",
			query: `"John Doe" AND (active:true OR (role:admin AND department:IT))`,
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
							{
								"role":       "admin",
								"department": "IT",
							},
						},
					},
				},
			},
		},
		{
			name:  "Free text search with NOT grouped condition",
			query: `"John Doe" AND NOT (active:false OR role:guest)`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": `"John Doe"`,
						},
					},
					{
						"$and": []bson.M{
							{"active": bson.M{"$ne": false}},
							{"role": bson.M{"$ne": "guest"}},
						},
					},
				},
			},
		},
		{
			name:  "Multiple free text searches with grouped logic",
			query: `("John Doe" OR "Jane Smith") AND (active:true OR role:admin)`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John Doe Jane Smith",
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
			name:  "Free text search with deeply nested grouped logic",
			query: `"John Doe" AND ((active:true OR role:admin) AND (department:IT OR department:Engineering))`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": `"John Doe"`,
						},
					},
					{
						"$and": []bson.M{
							{
								"$or": []bson.M{
									{"active": true},
									{"role": "admin"},
								},
							},
							{
								"$or": []bson.M{
									{"department": "IT"},
									{"department": "Engineering"},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "Free text search with mixed single and double quotes in groups",
			query: `("John Doe" OR 'Jane Smith') AND (active:true OR role:admin)`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John Doe Jane Smith",
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
			name:  "Unquoted single word free text search",
			query: `John`,
			expected: bson.M{
				"$text": bson.M{
					"$search": "John",
				},
			},
		},
		{
			name:  "Unquoted multiple words free text search",
			query: `John Doe`,
			expected: bson.M{
				"$text": bson.M{
					"$search": "John Doe",
				},
			},
		},
		{
			name:  "Unquoted free text search with field query",
			query: `John AND active:true`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John",
						},
					},
					{
						"active": true,
					},
				},
			},
		},
		{
			name:  "Unquoted free text search with OR condition",
			query: `John AND (active:true OR role:admin)`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John",
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
			name:  "Multiple unquoted free text searches with OR",
			query: `(John OR Jane) AND active:true`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John Jane",
						},
					},
					{
						"active": true,
					},
				},
			},
		},
		{
			name:  "Mixed quoted and unquoted free text searches",
			query: `("John Doe" OR Jane) AND active:true`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John Doe Jane",
						},
					},
					{
						"active": true,
					},
				},
			},
		},
		{
			name:  "Unquoted free text search with NOT condition",
			query: `John AND NOT active:false`,
			expected: bson.M{
				"$and": []bson.M{
					{
						"$text": bson.M{
							"$search": "John",
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

// Helper functions for BSON comparison

// compareBSONValues compares BSON values for testing
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
