package bsonic_test

import (
	"testing"
	"time"

	"github.com/grindlemire/go-lucene"
	"github.com/grindlemire/go-lucene/pkg/lucene/expr"
	"github.com/kyle-williams-1/bsonic"
	"go.mongodb.org/mongo-driver/bson"
)

func TestBSONDriverRangeQueries(t *testing.T) {
	driver := bsonic.NewBSONDriver(bsonic.SearchModeDisabled)

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "numeric range",
			query:    "age:[18 TO 65]",
			expected: bson.M{"age": bson.M{"$gte": 18.0, "$lte": 65.0}},
		},
		{
			name:     "date range",
			query:    "created_at:[2023-01-01 TO 2023-12-31]",
			expected: bson.M{"created_at": bson.M{"$gte": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), "$lte": time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)}},
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
			name:     "negative range",
			query:    "temperature:[-10 TO 25]",
			expected: bson.M{"temperature": bson.M{"$gte": -10.0, "$lte": 25.0}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the query using go-lucene
			expr, err := lucene.Parse(test.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			result, err := driver.RenderExpression(expr)
			if err != nil {
				t.Fatalf("Failed to render expression: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestBSONDriverNegation(t *testing.T) {
	driver := bsonic.NewBSONDriver(bsonic.SearchModeDisabled)

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "NOT simple field",
			query:    "NOT name:john",
			expected: bson.M{"name": bson.M{"$ne": "john"}},
		},
		{
			name:     "NOT OR expression",
			query:    "NOT (name:john OR name:jane)",
			expected: bson.M{"$and": []bson.M{{"name": bson.M{"$ne": "john"}}, {"name": bson.M{"$ne": "jane"}}}},
		},
		{
			name:     "NOT AND expression",
			query:    "NOT (name:john AND age:25)",
			expected: bson.M{"age": bson.M{"$ne": 25.0}, "name": bson.M{"$ne": "john"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the query using go-lucene
			expr, err := lucene.Parse(test.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			result, err := driver.RenderExpression(expr)
			if err != nil {
				t.Fatalf("Failed to render expression: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestBSONDriverEdgeCases(t *testing.T) {
	driver := bsonic.NewBSONDriver(bsonic.SearchModeDisabled)

	t.Run("nil expression", func(t *testing.T) {
		result, err := driver.RenderExpression(nil)
		if err != nil {
			t.Fatalf("Expected no error for nil expression, got: %v", err)
		}
		expected := bson.M{}
		if !compareBSONValues(result, expected) {
			t.Fatalf("Expected %+v, got %+v", expected, result)
		}
	})

	t.Run("unknown operation", func(t *testing.T) {
		// Create a mock expression with unknown operation
		expr := &expr.Expression{
			Op:    999, // Unknown operation
			Left:  "test",
			Right: "value",
		}

		result, err := driver.RenderExpression(expr)
		if err != nil {
			t.Fatalf("Expected no error for unknown operation, got: %v", err)
		}
		expected := bson.M{}
		if !compareBSONValues(result, expected) {
			t.Fatalf("Expected %+v, got %+v", expected, result)
		}
	})
}

func TestBSONDriverValueParsing(t *testing.T) {
	driver := bsonic.NewBSONDriver(bsonic.SearchModeDisabled)

	tests := []struct {
		name     string
		query    string
		expected bson.M
	}{
		{
			name:     "boolean true",
			query:    "active:true",
			expected: bson.M{"active": true},
		},
		{
			name:     "boolean false",
			query:    "active:false",
			expected: bson.M{"active": false},
		},
		{
			name:     "numeric integer",
			query:    "age:25",
			expected: bson.M{"age": 25.0},
		},
		{
			name:     "numeric float",
			query:    "price:19.99",
			expected: bson.M{"price": 19.99},
		},
		{
			name:     "date string",
			query:    "created_at:2023-01-01",
			expected: bson.M{"created_at": time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		{
			name:     "string value",
			query:    "name:john",
			expected: bson.M{"name": "john"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the query using go-lucene
			expr, err := lucene.Parse(test.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			result, err := driver.RenderExpression(expr)
			if err != nil {
				t.Fatalf("Failed to render expression: %v", err)
			}

			if !compareBSONValues(result, test.expected) {
				t.Fatalf("Expected %+v, got %+v", test.expected, result)
			}
		})
	}
}
