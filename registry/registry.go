// Package registry provides dynamic discovery and registration of languages and formatters.
package registry

import (
	"fmt"

	"github.com/kyle-williams-1/bsonic/config"
	"github.com/kyle-williams-1/bsonic/formatter"
	"github.com/kyle-williams-1/bsonic/language"
	"go.mongodb.org/mongo-driver/bson"
)

// LanguageFactory creates a new language parser instance.
type LanguageFactory func() language.Parser

// FormatterFactory creates a new formatter instance.
type FormatterFactory func() formatter.Formatter[bson.M]

// LanguageRegistry manages available language parsers.
type LanguageRegistry struct {
	languages map[config.LanguageType]LanguageFactory
}

// FormatterRegistry manages available formatters.
type FormatterRegistry struct {
	formatters map[config.FormatterType]FormatterFactory
}

// Registry combines language and formatter registries.
type Registry struct {
	Languages  *LanguageRegistry
	Formatters *FormatterRegistry
}

// New creates a new registry with default languages and formatters.
func New() *Registry {
	return &Registry{
		Languages:  NewLanguageRegistry(),
		Formatters: NewFormatterRegistry(),
	}
}

// NewLanguageRegistry creates a new language registry with default languages.
func NewLanguageRegistry() *LanguageRegistry {
	registry := &LanguageRegistry{
		languages: make(map[config.LanguageType]LanguageFactory),
	}

	// Register default languages
	// Note: This will be populated by the init() functions in language packages
	return registry
}

// NewFormatterRegistry creates a new formatter registry with default formatters.
func NewFormatterRegistry() *FormatterRegistry {
	registry := &FormatterRegistry{
		formatters: make(map[config.FormatterType]FormatterFactory),
	}

	// Register default formatters
	// Note: This will be populated by the init() functions in formatter packages
	return registry
}

// RegisterLanguage registers a language factory.
func (lr *LanguageRegistry) RegisterLanguage(langType config.LanguageType, factory LanguageFactory) {
	lr.languages[langType] = factory
}

// RegisterFormatter registers a formatter factory.
func (fr *FormatterRegistry) RegisterFormatter(formatterType config.FormatterType, factory FormatterFactory) {
	fr.formatters[formatterType] = factory
}

// GetLanguage creates a language parser instance.
func (lr *LanguageRegistry) GetLanguage(langType config.LanguageType) (language.Parser, error) {
	factory, exists := lr.languages[langType]
	if !exists {
		return nil, fmt.Errorf("unsupported language type: %s", langType)
	}
	return factory(), nil
}

// GetFormatter creates a formatter instance.
func (fr *FormatterRegistry) GetFormatter(formatterType config.FormatterType) (formatter.Formatter[bson.M], error) {
	factory, exists := fr.formatters[formatterType]
	if !exists {
		return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
	}
	return factory(), nil
}

// ListLanguages returns all registered language types.
func (lr *LanguageRegistry) ListLanguages() []config.LanguageType {
	var languages []config.LanguageType
	for langType := range lr.languages {
		languages = append(languages, langType)
	}
	return languages
}

// ListFormatters returns all registered formatter types.
func (fr *FormatterRegistry) ListFormatters() []config.FormatterType {
	var formatters []config.FormatterType
	for formatterType := range fr.formatters {
		formatters = append(formatters, formatterType)
	}
	return formatters
}

// ValidateConfig validates that a language-formatter combination is supported.
func (r *Registry) ValidateConfig(cfg *config.Config) error {
	_, err := r.Languages.GetLanguage(cfg.Language)
	if err != nil {
		return fmt.Errorf("invalid language: %w", err)
	}

	_, err = r.Formatters.GetFormatter(cfg.Formatter)
	if err != nil {
		return fmt.Errorf("invalid formatter: %w", err)
	}

	return nil
}

// Global registry instance
var DefaultRegistry = New()

// RegisterLanguage registers a language with the global registry.
func RegisterLanguage(langType config.LanguageType, factory LanguageFactory) {
	DefaultRegistry.Languages.RegisterLanguage(langType, factory)
}

// RegisterFormatter registers a formatter with the global registry.
func RegisterFormatter(formatterType config.FormatterType, factory FormatterFactory) {
	DefaultRegistry.Formatters.RegisterFormatter(formatterType, factory)
}
