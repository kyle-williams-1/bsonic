package config

import (
	"testing"
)

// TestLanguageTypeConstants tests that the LanguageType constants are defined correctly
func TestLanguageTypeConstants(t *testing.T) {
	if LanguageLucene != "lucene" {
		t.Errorf("Expected LanguageLucene to be 'lucene', got %q", LanguageLucene)
	}
}

// TestFormatterTypeConstants tests that the FormatterType constants are defined correctly
func TestFormatterTypeConstants(t *testing.T) {
	if FormatterMongo != "mongo" {
		t.Errorf("Expected FormatterMongo to be 'mongo', got %q", FormatterMongo)
	}
}

// TestDefault tests that the Default function returns the expected configuration
func TestDefault(t *testing.T) {
	config := Default()

	if config == nil {
		t.Fatal("Expected Default() to return a non-nil config")
	}

	if config.Language != LanguageLucene {
		t.Errorf("Expected default language to be %q, got %q", LanguageLucene, config.Language)
	}

	if config.Formatter != FormatterMongo {
		t.Errorf("Expected default formatter to be %q, got %q", FormatterMongo, config.Formatter)
	}
}

// TestConfigWithLanguage tests the WithLanguage fluent method
func TestConfigWithLanguage(t *testing.T) {
	config := &Config{
		Language:  LanguageLucene,
		Formatter: FormatterMongo,
	}

	// Test that WithLanguage returns the same config instance
	result := config.WithLanguage(LanguageLucene)
	if result != config {
		t.Error("Expected WithLanguage to return the same config instance")
	}

	// Test that the language was set correctly
	if config.Language != LanguageLucene {
		t.Errorf("Expected language to be %q, got %q", LanguageLucene, config.Language)
	}
}

// TestConfigWithFormatter tests the WithFormatter fluent method
func TestConfigWithFormatter(t *testing.T) {
	config := &Config{
		Language:  LanguageLucene,
		Formatter: FormatterMongo,
	}

	// Test that WithFormatter returns the same config instance
	result := config.WithFormatter(FormatterMongo)
	if result != config {
		t.Error("Expected WithFormatter to return the same config instance")
	}

	// Test that the formatter was set correctly
	if config.Formatter != FormatterMongo {
		t.Errorf("Expected formatter to be %q, got %q", FormatterMongo, config.Formatter)
	}
}

// TestConfigFluentChaining tests that fluent methods can be chained
func TestConfigFluentChaining(t *testing.T) {
	config := &Config{}

	result := config.WithLanguage(LanguageLucene).WithFormatter(FormatterMongo)

	if result != config {
		t.Error("Expected chained methods to return the same config instance")
	}

	if config.Language != LanguageLucene {
		t.Errorf("Expected chained language to be %q, got %q", LanguageLucene, config.Language)
	}

	if config.Formatter != FormatterMongo {
		t.Errorf("Expected chained formatter to be %q, got %q", FormatterMongo, config.Formatter)
	}
}

// TestConfigStructFields tests direct field access and modification
func TestConfigStructFields(t *testing.T) {
	config := &Config{}

	// Test setting and getting Language field
	config.Language = LanguageLucene
	if config.Language != LanguageLucene {
		t.Errorf("Expected Language field to be %q, got %q", LanguageLucene, config.Language)
	}

	// Test setting and getting Formatter field
	config.Formatter = FormatterMongo
	if config.Formatter != FormatterMongo {
		t.Errorf("Expected Formatter field to be %q, got %q", FormatterMongo, config.Formatter)
	}
}

// TestConfigZeroValue tests the zero value of Config struct
func TestConfigZeroValue(t *testing.T) {
	var config Config

	// Zero values should be empty strings
	if config.Language != "" {
		t.Errorf("Expected zero value Language to be empty string, got %q", config.Language)
	}

	if config.Formatter != "" {
		t.Errorf("Expected zero value Formatter to be empty string, got %q", config.Formatter)
	}
}

// TestConfigWithDefaultFields tests the WithDefaultFields fluent method
func TestConfigWithDefaultFields(t *testing.T) {
	config := &Config{}

	fields := []string{"name", "description"}
	result := config.WithDefaultFields(fields)

	if result != config {
		t.Error("Expected WithDefaultFields to return the same config instance")
	}

	if len(config.DefaultFields) != 2 {
		t.Errorf("Expected DefaultFields length 2, got %d", len(config.DefaultFields))
	}

	if config.DefaultFields[0] != "name" {
		t.Errorf("Expected first field 'name', got %q", config.DefaultFields[0])
	}

	if config.DefaultFields[1] != "description" {
		t.Errorf("Expected second field 'description', got %q", config.DefaultFields[1])
	}
}

// TestConfigWithReplaceIDWithMongoID tests the WithReplaceIDWithMongoID fluent method
func TestConfigWithReplaceIDWithMongoID(t *testing.T) {
	config := &Config{}

	result := config.WithReplaceIDWithMongoID(false)

	if result != config {
		t.Error("Expected WithReplaceIDWithMongoID to return the same config instance")
	}

	if config.ReplaceIDWithMongoID != false {
		t.Errorf("Expected ReplaceIDWithMongoID false, got %v", config.ReplaceIDWithMongoID)
	}

	// Test setting to true
	config.WithReplaceIDWithMongoID(true)
	if config.ReplaceIDWithMongoID != true {
		t.Errorf("Expected ReplaceIDWithMongoID true, got %v", config.ReplaceIDWithMongoID)
	}
}

// TestConfigWithAutoConvertIDToObjectID tests the WithAutoConvertIDToObjectID fluent method
func TestConfigWithAutoConvertIDToObjectID(t *testing.T) {
	config := &Config{}

	result := config.WithAutoConvertIDToObjectID(false)

	if result != config {
		t.Error("Expected WithAutoConvertIDToObjectID to return the same config instance")
	}

	if config.AutoConvertIDToObjectID != false {
		t.Errorf("Expected AutoConvertIDToObjectID false, got %v", config.AutoConvertIDToObjectID)
	}

	// Test setting to true
	config.WithAutoConvertIDToObjectID(true)
	if config.AutoConvertIDToObjectID != true {
		t.Errorf("Expected AutoConvertIDToObjectID true, got %v", config.AutoConvertIDToObjectID)
	}
}

// TestConfigDefaultValues tests that Default() sets the correct default values
func TestConfigDefaultValues(t *testing.T) {
	config := Default()

	if config.Language != LanguageLucene {
		t.Errorf("Expected default language %v, got %v", LanguageLucene, config.Language)
	}
	if config.Formatter != FormatterMongo {
		t.Errorf("Expected default formatter %v, got %v", FormatterMongo, config.Formatter)
	}
	if config.DefaultFields == nil {
		t.Error("Expected DefaultFields to be initialized, got nil")
	}
	if len(config.DefaultFields) != 0 {
		t.Errorf("Expected empty DefaultFields, got %v", config.DefaultFields)
	}
	if config.ReplaceIDWithMongoID != true {
		t.Errorf("Expected default ReplaceIDWithMongoID true, got %v", config.ReplaceIDWithMongoID)
	}
	if config.AutoConvertIDToObjectID != true {
		t.Errorf("Expected default AutoConvertIDToObjectID true, got %v", config.AutoConvertIDToObjectID)
	}
}
