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
}

func TestBasicFieldQueries(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "simple field value",
			query:    "name:john",
			expected: bson.M{"name": "john"},
		},
		{
			name:     "numeric value",
			query:    "age:25",
			expected: bson.M{"age": 25.0},
		},
		{
			name:     "boolean value",
			query:    "active:true",
			expected: bson.M{"active": true},
		},
		{
			name:     "empty query",
			query:    "",
			expected: bson.M{},
		},
		{
			name:     "whitespace query",
			query:    "   ",
			expected: bson.M{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestBooleanOperators(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "AND operator",
			query:    "name:john AND age:25",
			expected: bson.M{"name": "john", "age": 25.0},
		},
		{
			name:     "OR operator",
			query:    "name:john OR name:jane",
			expected: bson.M{"$or": []bson.M{{"name": "john"}, {"name": "jane"}}},
		},
		{
			name:     "NOT operator",
			query:    "NOT name:john",
			expected: bson.M{"name": bson.M{"$ne": "john"}},
		},
		{
			name:     "complex AND OR",
			query:    "(name:john OR name:jane) AND age:25",
			expected: bson.M{"$and": []bson.M{{"$or": []bson.M{{"name": "john"}, {"name": "jane"}}}, {"age": 25.0}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestWildcardQueries(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "prefix wildcard",
			query:    "name:john*",
			expected: bson.M{"name": bson.M{"$regex": "^john.*", "$options": "i"}},
		},
		{
			name:     "suffix wildcard",
			query:    "name:*john",
			expected: bson.M{"name": bson.M{"$regex": ".*john$", "$options": "i"}},
		},
		{
			name:     "contains wildcard",
			query:    "name:*john*",
			expected: bson.M{"name": bson.M{"$regex": ".*john.*", "$options": "i"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestComparisonOperators(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "greater than",
			query:    "age:>25",
			expected: bson.M{"age": bson.M{"$gt": 25.0}},
		},
		{
			name:     "less than",
			query:    "age:<30",
			expected: bson.M{"age": bson.M{"$lt": 30.0}},
		},
		{
			name:     "greater than or equal",
			query:    "age:>=25",
			expected: bson.M{"age": bson.M{"$gte": 25.0}},
		},
		{
			name:     "less than or equal",
			query:    "age:<=30",
			expected: bson.M{"age": bson.M{"$lte": 30.0}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestTextSearch(t *testing.T) {
	parser := bsonic.NewWithTextSearch()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "simple text search",
			query:    "search term",
			expected: bson.M{"$text": bson.M{"$search": "search term"}},
		},
		{
			name:     "multiple words",
			query:    "search multiple terms",
			expected: bson.M{"$text": bson.M{"$search": "search multiple terms"}},
		},
		{
			name:     "empty text search",
			query:    "",
			expected: bson.M{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestDateParsing(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "ISO date",
			query:    "created_at:2024-01-01",
			expected: bson.M{"created_at": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

// Helper function to compare BSON values
func compareBSONValues(a, b bson.M) bool {
	// Use DeepEqual for comparison which handles field order differences
	return compareBSONMaps(a, b)
}

// compareBSONMaps compares two BSON maps recursively
func compareBSONMaps(a, b bson.M) bool {
	if len(a) != len(b) {
		return false
	}

	for key, valueA := range a {
		valueB, exists := b[key]
		if !exists {
			return false
		}

		if !compareBSONValue(valueA, valueB) {
			return false
		}
	}

	return true
}

// compareBSONValue compares any BSON values
func compareBSONValue(a, b interface{}) bool {
	// Handle bson.M comparison
	if mapA, ok := a.(bson.M); ok {
		if mapB, ok := b.(bson.M); ok {
			return compareBSONMaps(mapA, mapB)
		}
		return false
	}

	// Handle []bson.M comparison
	if sliceA, ok := a.([]bson.M); ok {
		if sliceB, ok := b.([]bson.M); ok {
			if len(sliceA) != len(sliceB) {
				return false
			}
			for i, itemA := range sliceA {
				if !compareBSONMaps(itemA, sliceB[i]) {
					return false
				}
			}
			return true
		}
		return false
	}

	// Handle time.Time comparison
	if timeA, ok := a.(time.Time); ok {
		if timeB, ok := b.(time.Time); ok {
			return timeA.Equal(timeB)
		}
		return false
	}

	// Default comparison
	return a == b
}

func TestNumericRangeQueries(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "integer range",
			query:    "age:[18 TO 65]",
			expected: bson.M{"age": bson.M{"$gte": 18.0, "$lte": 65.0}},
		},
		{
			name:     "decimal range",
			query:    "price:[10.50 TO 99.99]",
			expected: bson.M{"price": bson.M{"$gte": 10.50, "$lte": 99.99}},
		},
		{
			name:     "score range",
			query:    "score:[80 TO 100]",
			expected: bson.M{"score": bson.M{"$gte": 80.0, "$lte": 100.0}},
		},
		{
			name:     "open range start",
			query:    "age:[18 TO *]",
			expected: bson.M{"age": bson.M{"$gte": 18.0}},
		},
		{
			name:     "open range end",
			query:    "age:[* TO 65]",
			expected: bson.M{"age": bson.M{"$lte": 65.0}},
		},
		{
			name:     "negative range",
			query:    "temperature:[-10 TO 25]",
			expected: bson.M{"temperature": bson.M{"$gte": -10.0, "$lte": 25.0}},
		},
		{
			name:     "range with AND",
			query:    "age:[18 TO 65] AND status:active",
			expected: bson.M{"age": bson.M{"$gte": 18.0, "$lte": 65.0}, "status": "active"},
		},
		{
			name:  "range with OR",
			query: "age:[18 TO 30] OR age:[60 TO 65]",
			expected: bson.M{"$or": []bson.M{
				{"age": bson.M{"$gte": 18.0, "$lte": 30.0}},
				{"age": bson.M{"$gte": 60.0, "$lte": 65.0}},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parser.Parse(test.query)
			if err != nil {
				t.Fatalf("Parse should not return error, got: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}
