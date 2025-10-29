// Package config provides configuration for language and formatter selection.
package config

// LanguageType represents the type of query language to use.
type LanguageType string

const (
	// LanguageLucene represents Lucene-style query syntax
	LanguageLucene LanguageType = "lucene"
)

// FormatterType represents the type of output formatter to use.
type FormatterType string

const (
	// FormatterMongo represents MongoDB BSON output format
	FormatterMongo FormatterType = "mongo"
)

// Config represents the configuration for a parser.
type Config struct {
	Language                LanguageType
	Formatter               FormatterType
	DefaultFields           []string
	ReplaceIDWithMongoID    bool
	AutoConvertIDToObjectID bool
}

// Default returns the default configuration with Lucene language and MongoDB formatter.
func Default() *Config {
	return &Config{
		Language:                LanguageLucene,
		Formatter:               FormatterMongo,
		DefaultFields:           []string{},
		ReplaceIDWithMongoID:    true,
		AutoConvertIDToObjectID: true,
	}
}

// WithLanguage sets the language type and returns the config.
func (c *Config) WithLanguage(lang LanguageType) *Config {
	c.Language = lang
	return c
}

// WithFormatter sets the formatter type and returns the config.
func (c *Config) WithFormatter(formatter FormatterType) *Config {
	c.Formatter = formatter
	return c
}

// WithDefaultFields sets the default fields for unstructured queries and returns the config.
func (c *Config) WithDefaultFields(fields []string) *Config {
	c.DefaultFields = fields
	return c
}

// WithReplaceIDWithMongoID sets whether to replace "id" field names with "_id" and returns the config.
func (c *Config) WithReplaceIDWithMongoID(enabled bool) *Config {
	c.ReplaceIDWithMongoID = enabled
	return c
}

// WithAutoConvertIDToObjectID sets whether to automatically convert string values to primitive.ObjectID for "_id" fields and returns the config.
func (c *Config) WithAutoConvertIDToObjectID(enabled bool) *Config {
	c.AutoConvertIDToObjectID = enabled
	return c
}
