package bsonic

import (
	"strings"
	"testing"

	"github.com/kyle-williams-1/bsonic/config"
)

// This file contains minimal entry point API tests for bsonic.go.
// It only tests that the main functions don't error - detailed parsing
// behavior is tested in the comprehensive test suites in tests/*.

// TestParse tests the main entry point Parse() function
func TestParse(t *testing.T) {
	t.Run("ValidQuery", func(t *testing.T) {
		result, err := ParseWithDefaults([]string{"name"}, "test")
		if err != nil {
			t.Fatalf("ParseWithDefaults() should not return error, got: %v", err)
		}
		if result == nil {
			t.Fatal("ParseWithDefaults() should return a non-nil result")
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		result, err := ParseWithDefaults([]string{"name"}, "")
		if err != nil {
			t.Fatalf("ParseWithDefaults() should not return error with empty query, got: %v", err)
		}
		if result == nil {
			t.Fatal("ParseWithDefaults() should return a non-nil result")
		}
	})

	t.Run("WhitespaceQuery", func(t *testing.T) {
		result, err := ParseWithDefaults([]string{"name"}, "   ")
		if err != nil {
			t.Fatalf("ParseWithDefaults() should not return error with whitespace query, got: %v", err)
		}
		if result == nil {
			t.Fatal("ParseWithDefaults() should return a non-nil result")
		}
	})

	t.Run("ParseWithoutDefaultFields", func(t *testing.T) {
		// Test that Parse() without default fields returns an error
		_, err := Parse("test")
		if err == nil {
			t.Fatal("Parse() should return error when no default fields are configured")
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

// TestPackageLevelFunctions tests package-level functions
func TestPackageLevelFunctions(t *testing.T) {
	// Test NewMongoFormatter() constructor
	t.Run("NewMongoFormatter", func(t *testing.T) {
		formatter := NewMongoFormatter()
		if formatter == nil {
			t.Fatal("NewMongoFormatter() should return a non-nil formatter")
		}
	})

	// Test package-level Parse() function
	t.Run("PackageLevelParse", func(t *testing.T) {
		// This should fail because no default fields are configured
		_, err := Parse("name:john")
		if err == nil {
			t.Fatal("Parse() should return error when no default fields are configured")
		}
		if !strings.Contains(err.Error(), "no default fields are configured") {
			t.Fatalf("Expected error about default fields, got: %v", err)
		}
	})
}

// TestConfigMethods tests config methods
func TestConfigMethods(t *testing.T) {
	// Test WithLanguage method
	t.Run("WithLanguage", func(t *testing.T) {
		cfg := config.Default().WithLanguage(config.LanguageLucene)
		if cfg.Language != config.LanguageLucene {
			t.Fatalf("Expected language %s, got %s", config.LanguageLucene, cfg.Language)
		}
	})

	// Test WithFormatter method
	t.Run("WithFormatter", func(t *testing.T) {
		cfg := config.Default().WithFormatter(config.FormatterMongo)
		if cfg.Formatter != config.FormatterMongo {
			t.Fatalf("Expected formatter %s, got %s", config.FormatterMongo, cfg.Formatter)
		}
	})
}

// TestErrorHandling tests additional error handling scenarios
func TestErrorHandling(t *testing.T) {
	// Test NewParser with unsupported language
	t.Run("NewParserUnsupportedLanguage", func(t *testing.T) {
		_, err := NewParser("unsupported")
		if err == nil {
			t.Fatal("NewParser should return error for unsupported language")
		}
		if !strings.Contains(err.Error(), "unsupported language type") {
			t.Fatalf("Expected error about unsupported language, got: %v", err)
		}
	})

	// Test NewFormatter with unsupported formatter
	t.Run("NewFormatterUnsupportedFormatter", func(t *testing.T) {
		_, err := NewFormatter("unsupported")
		if err == nil {
			t.Fatal("NewFormatter should return error for unsupported formatter")
		}
		if !strings.Contains(err.Error(), "unsupported formatter type") {
			t.Fatalf("Expected error about unsupported formatter, got: %v", err)
		}
	})

	// Test NewWithConfig with invalid config
	t.Run("NewWithConfigInvalidConfig", func(t *testing.T) {
		cfg := config.Default().WithLanguage("unsupported")
		_, err := NewWithConfig(cfg)
		if err == nil {
			t.Fatal("NewWithConfig should return error for invalid config")
		}
		if !strings.Contains(err.Error(), "unsupported language type") {
			t.Fatalf("Expected error about unsupported language, got: %v", err)
		}
	})
}

// TestEdgeCases tests additional edge cases for better coverage
func TestEdgeCases(t *testing.T) {
	// Test parser with no default fields configured
	t.Run("ParserWithoutDefaultFields", func(t *testing.T) {
		cfg := config.Default() // No default fields
		parser, err := NewWithConfig(cfg)
		if err != nil {
			t.Fatalf("NewWithConfig should not return error, got: %v", err)
		}

		_, err = parser.Parse("john")
		if err == nil {
			t.Fatal("Parse should return error when no default fields are configured")
		}
		if !strings.Contains(err.Error(), "no default fields are configured") {
			t.Fatalf("Expected error about default fields, got: %v", err)
		}
	})

	// Test ParseWithDefaults with empty default fields
	t.Run("ParseWithDefaultsEmptyFields", func(t *testing.T) {
		_, err := ParseWithDefaults([]string{}, "john")
		if err == nil {
			t.Fatal("ParseWithDefaults should return error for empty default fields")
		}
		if !strings.Contains(err.Error(), "default fields cannot be empty") {
			t.Fatalf("Expected error about empty default fields, got: %v", err)
		}
	})

	// Test ParseWithDefaults with nil default fields
	t.Run("ParseWithDefaultsNilFields", func(t *testing.T) {
		_, err := ParseWithDefaults(nil, "john")
		if err == nil {
			t.Fatal("ParseWithDefaults should return error for nil default fields")
		}
		if !strings.Contains(err.Error(), "default fields cannot be empty") {
			t.Fatalf("Expected error about empty default fields, got: %v", err)
		}
	})

	// Test parser ParseWithDefaults with empty default fields
	t.Run("ParserParseWithDefaultsEmptyFields", func(t *testing.T) {
		parser := New()
		_, err := parser.ParseWithDefaults([]string{}, "john")
		if err == nil {
			t.Fatal("ParseWithDefaults should return error for empty default fields")
		}
		if !strings.Contains(err.Error(), "default fields cannot be empty") {
			t.Fatalf("Expected error about empty default fields, got: %v", err)
		}
	})
}
