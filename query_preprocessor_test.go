package bsonic_test

import (
	"testing"

	"github.com/kyle-williams-1/bsonic"
)

func TestQueryPreprocessor(t *testing.T) {
	preprocessor := bsonic.NewQueryPreprocessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "email address",
			input:    "email:john@example.com",
			expected: "email:\"john@example.com\"",
		},
		{
			name:     "dot notation with spaces",
			input:    "profile.name:john doe",
			expected: "profile.name:\"john doe\"",
		},
		{
			name:     "parentheses with spaces",
			input:    "(name:john doe OR name:jane smith)",
			expected: "(name:\"john doe\"  OR  name:\"jane smith\")",
		},
		{
			name:     "regular field with spaces",
			input:    "name:john doe",
			expected: "name:\"john doe\"",
		},
		{
			name:     "mixed query",
			input:    "engineer role:admin",
			expected: "engineer AND (role:admin)",
		},
		{
			name:     "NOT with parentheses",
			input:    "NOT (role:admin OR role:moderator)",
			expected: "NOT (role:admin  OR  role:moderator)",
		},
		{
			name:     "range query should not be quoted",
			input:    "age:[18 TO 65]",
			expected: "age:[18 TO 65]",
		},
		{
			name:     "empty query",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace query",
			input:    "   ",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := preprocessor.PreprocessQuery(test.input)
			if result != test.expected {
				t.Fatalf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestQueryPreprocessorEdgeCases(t *testing.T) {
	preprocessor := bsonic.NewQueryPreprocessor()

	t.Run("already quoted values", func(t *testing.T) {
		input := "name:\"john doe\""
		result := preprocessor.PreprocessQuery(input)
		expected := "name:\"john doe\""
		if result != expected {
			t.Fatalf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("complex mixed query", func(t *testing.T) {
		input := "software engineer (role:admin AND age:25)"
		result := preprocessor.PreprocessQuery(input)
		expected := "software engineer AND (role:admin  AND  age:25)"
		if result != expected {
			t.Fatalf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("NOT at beginning", func(t *testing.T) {
		input := "NOT name:john doe"
		result := preprocessor.PreprocessQuery(input)
		expected := "NOT name:\"john doe\""
		if result != expected {
			t.Fatalf("Expected '%s', got '%s'", expected, result)
		}
	})
}
