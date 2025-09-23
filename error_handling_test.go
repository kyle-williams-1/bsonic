package bsonic_test

import (
	"strings"
	"testing"

	"github.com/kyle-williams-1/bsonic"
	"go.mongodb.org/mongo-driver/bson"
)

func TestErrorHandling(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name        string
		query       string
		expectError bool
		errorText   string
	}{
		{
			name:        "invalid query format",
			query:       "invalid query format",
			expectError: true,
			errorText:   "text search queries are not supported when text search mode is disabled",
		},
		{
			name:        "malformed field query",
			query:       ":value",
			expectError: true,
			errorText:   "failed to parse query",
		},
		{
			name:        "malformed field query 2",
			query:       "field:",
			expectError: true,
			errorText:   "failed to parse query",
		},
		{
			name:        "valid query",
			query:       "name:john",
			expectError: false,
		},
		{
			name:        "empty query",
			query:       "",
			expectError: false,
		},
		{
			name:        "whitespace query",
			query:       "   ",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parser.Parse(test.query)

			if test.expectError {
				if err == nil {
					t.Fatalf("Expected error for query '%s', got none", test.query)
				}
				if test.errorText != "" && !strings.Contains(err.Error(), test.errorText) {
					t.Fatalf("Expected error to contain '%s', got '%s'", test.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error for query '%s', got: %v", test.query, err)
				}
			}
		})
	}
}

func TestTextSearchErrorHandling(t *testing.T) {
	parser := bsonic.New() // Text search disabled

	t.Run("text search when disabled", func(t *testing.T) {
		_, err := parser.Parse("search term")
		if err == nil {
			t.Fatal("Expected error for text search when disabled, got none")
		}
		expectedError := "text search queries are not supported when text search mode is disabled"
		if !strings.Contains(err.Error(), expectedError) {
			t.Fatalf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
		}
	})

	parserWithText := bsonic.NewWithTextSearch()
	t.Run("text search when enabled", func(t *testing.T) {
		result, err := parserWithText.Parse("search term")
		if err != nil {
			t.Fatalf("Expected no error for text search when enabled, got: %v", err)
		}
		expected := bson.M{
			"$text": bson.M{
				"$search": "search term",
			},
		}
		if !compareBSONValues(result, expected) {
			t.Fatalf("Expected %+v, got %+v", expected, result)
		}
	})
}

func TestComplexErrorScenarios(t *testing.T) {
	parser := bsonic.New()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "unclosed parentheses",
			query: "(name:john",
		},
		{
			name:  "unopened parentheses",
			query: "name:john)",
		},
		{
			name:  "invalid operators",
			query: "name:john AND AND age:25",
		},
		{
			name:  "empty parentheses",
			query: "()",
		},
		{
			name:  "nested empty parentheses",
			query: "((()))",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parser.Parse(test.query)
			if err == nil {
				t.Fatalf("Expected error for query '%s', got none", test.query)
			}
		})
	}
}
