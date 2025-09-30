package bsonic

import (
	"testing"
)

// This file contains minimal entry point API tests for bsonic.go.
// It only tests that the main functions don't error - detailed parsing
// behavior is tested in the comprehensive test suites in tests/*.

// TestParse tests the main entry point Parse() function
func TestParse(t *testing.T) {
	t.Run("ValidQuery", func(t *testing.T) {
		result, err := Parse("test")
		if err != nil {
			t.Fatalf("Parse() should not return error, got: %v", err)
		}
		if result == nil {
			t.Fatal("Parse() should return a non-nil result")
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		result, err := Parse("")
		if err != nil {
			t.Fatalf("Parse() should not return error with empty query, got: %v", err)
		}
		if result == nil {
			t.Fatal("Parse() should return a non-nil result")
		}
	})

	t.Run("WhitespaceQuery", func(t *testing.T) {
		result, err := Parse("   ")
		if err != nil {
			t.Fatalf("Parse() should not return error with whitespace query, got: %v", err)
		}
		if result == nil {
			t.Fatal("Parse() should return a non-nil result")
		}
	})

}

// TestNew tests the basic constructor
func TestNew(t *testing.T) {
	parser := New()
	if parser == nil {
		t.Fatal("New() should return a non-nil parser")
	}
	if parser.Config == nil {
		t.Fatal("New() should return a parser with non-nil config")
	}
}
