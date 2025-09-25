# BSONIC Developer Guide

This document provides a comprehensive technical overview of the BSONIC library's extensible architecture, Participle integration, and how to extend the system with new query languages and output formatters.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Extensible Design](#extensible-design)
- [Core Components](#core-components)
- [Query Processing Flow](#query-processing-flow)
- [Configuration System](#configuration-system)
- [Parser Modes](#parser-modes)
- [Value Parsing](#value-parsing)
- [Error Handling](#error-handling)
- [Performance](#performance)
- [Extension Guide](#extension-guide)
- [Architectural Benefits](#architectural-benefits)

## Architecture Overview

BSONIC uses a modular, extensible architecture with a two-phase parsing pipeline:

1. **Lexical Analysis & Parsing** (handled by configurable language parsers)
2. **AST to Output Conversion** (handled by configurable formatters)

### High-Level Flow

```
Input Query String
        ↓
    [Configuration] → [Language Parser] → [AST] → [Formatter] → [Output]
        ↓
    [Query Type Detection]
        ↓
    ┌─────────────────┬─────────────────┬─────────────────┐
    │   Field Query   │   Text Search   │   Mixed Query   │
    │                 │                 │                 │
    │ [Participle]    │ [Text Parser]   │ [Mixed Parser]  │
    │ Lexer → Parser  │ → Text Terms    │ → Field AST +   │
    │        ↓        │        ↓        │    Text Terms   │
    │ [AST Walker]    │ [Text Formatter]│        ↓        │
    │        ↓        │        ↓        │ [Mixed Formatter]│
    │ [Field Formatter]│ {"$text": {...}}│        ↓        │
    │        ↓        │                 │ [Combine BSON]  │
    └────────┴────────┴─────────────────┴────────┴────────┘
             ↓                                    ↓
    [BSON Document] ←─────────────────────────────┘
             ↓
    [MongoDB Filter]
```

## Extensible Design

BSONIC is built with extensibility as a core principle, enabling easy addition of new query languages and output formats.

### Package Structure

```
bsonic/
├── config/           # Configuration management
│   └── config.go     # Language and formatter types
├── factory/          # Component factory functions
│   └── factory.go    # Parser and formatter creation
├── language/         # Query language implementations
│   ├── interface.go  # Language parser interface
│   └── lucene/       # Lucene-style parser (Participle-based)
├── formatter/        # Output formatter implementations
│   ├── interface.go  # Formatter interface
│   └── bson/         # BSON output formatter
└── bsonic.go         # Main API and orchestration
```

### Interface-Based Architecture

**Base Language Parser Interface:**
```go
type Parser interface {
    Parse(query string) (AST, error)
}
```

**Text Search Parser Interface:**
```go
, textTerms string, err error)
    ParseFieldQuery(query string) (interface{}, error)
    // Validation
    ValidateFieldQuery(query string) error
}
```

**Base Formatter Interface:**
```go
type Formatter[T any] interface {
    Format(ast interface{}) (T, error)
}
```

**Text Search Formatter Interface:**
```go

```

### Configuration System

```go
type Config struct {
    Language  LanguageType
    Formatter FormatterType
}

// Builder pattern for configuration
cfg := config.Default().
    WithLanguage(config.LanguageLucene).
    WithFormatter(config.FormatterBSON)
```

## Core Components

### Main Parser (`bsonic.go`)
- **Generic Orchestration**: Pure coordination between language parsers and formatters
- **No Language-Specific Logic**: All parsing logic delegated to language parsers
- **No Output-Specific Logic**: All formatting logic delegated to formatters

### Language System (`language/`)
- **Base Parser Interface**: Defines contract for basic query language parsers
- **Text Search Parser Interface**: Extended interface for parsers supporting text search
- **Lucene Parser**: Implements both interfaces with Participle-based parsing
- **Query Type Detection**: Language-specific logic for identifying mixed queries and text search
- **Query Parsing**: Language-specific parsing of field queries, text search, and mixed queries
- **Participle Grammar**: Defines the Lucene-style query syntax structure
- **Lexer**: Tokenizes input strings into recognized tokens

### Formatter System (`formatter/`)
- **Base Formatter Interface**: Generic interface for output formatters
- **Text Search Formatter Interface**: Extended interface for formatters supporting text search
- **BSON Formatter**: Implements both interfaces for MongoDB BSON output
- **Text Search Formatting**: MongoDB-specific `$text` operator formatting
- **Mixed Query Formatting**: MongoDB-specific `$and` operator for combining field and text search
- **AST Walker**: Converts Participle AST nodes to MongoDB BSON operations
- **Value Parser**: Handles type detection and special syntax (wildcards, ranges, dates)

### Configuration System (`config/`)
- **Config Types**: Language and formatter type definitions
- **Builder Pattern**: Fluent configuration API
- **Default Values**: Sensible defaults for common use cases

### Factory System (`factory/`)
- **Component Creation**: Centralized creation of parsers and formatters
- **Error Handling**: Proper error propagation during component creation
- **Type Safety**: Compile-time validation of component types

## Query Processing Flow

### Entry Points

```go
// Package-level convenience function (uses default configuration)
func Parse(query string) (bson.M, error)

// Instance-based parsing with default configuration
func (p *Parser) Parse(query string) (bson.M, error)

// Custom configuration
parser, err := bsonic.NewWithConfig(cfg)
```

## Configuration System

BSONIC uses a configuration-driven approach for selecting language parsers and output formatters.

### Configuration Types

```go
type LanguageType string
const (
    LanguageLucene LanguageType = "lucene"
)

type FormatterType string
const (
    FormatterBSON FormatterType = "bson"
)
```

### Configuration API

```go
// Default configuration
cfg := config.Default()

// Custom configuration
cfg := config.Default().
    WithLanguage(config.LanguageLucene).
    WithFormatter(config.FormatterBSON)

// Create parser with custom configuration
parser, err := bsonic.NewWithConfig(cfg)
```

### Factory Pattern

```go
// Language parser creation
languageParser, err := factory.CreateParser(cfg.Language)

// Formatter creation
formatter, err := factory.CreateFormatter(cfg.Formatter)
```

### Query Type Detection

The language parser determines query type based on search mode and content:

- **Mixed Query**: `engineer role:admin AND active:true` (detected by `IsMixedQuery()` returning true)

### Processing Flow

### Processing Examples

**Simple Field Query:**
```
Input: "name:john AND age:25"
→ Participle parsing → AST → BSON conversion
→ Output: {"name": "john", "age": 25}
```

**Complex Query with Wildcards:**
```
Input: "name:jo* AND age:[18 TO 65] OR status:active"
→ Output: {"$or": [{"name": {"$regex": "^jo.*", "$options": "i"}, "age": {"$gte": 18, "$lte": 65}}, {"status": "active"}]}
```

**Text Search:**
```
→ Output: {"$text": {"$search": "software engineer"}}
```

## Parser Modes

### Search Modes

```go

```

### Optimizations

1. **Field Merging**: Simple field:value pairs merge directly instead of using `$and`
2. **Operator Simplification**: Single-element arrays are flattened
3. **Conflict Detection**: Prevents field conflicts during merging

### Memory Management

- Participle AST nodes are garbage collected after conversion
- BSON documents use MongoDB driver's efficient BSON types
- No intermediate string allocations during parsing

## Development Guidelines

### Adding New Query Features

1. **Grammar Changes**: Update Participle struct definitions
2. **Lexer Updates**: Add new token patterns if needed
3. **AST Conversion**: Implement corresponding BSON conversion logic
4. **Value Parsing**: Add type detection and parsing logic
5. **Tests**: Add comprehensive test coverage

### Testing

Run tests with:
```bash
go test ./...
go test -tags=integration ./integration/...
```

## Extension Guide

BSONIC's extensible architecture makes it easy to add new query languages and output formatters.

### Adding a New Query Language

1. **Create Language Package:**
```go
// language/sql/parser.go
package sql

import "github.com/kyle-williams-1/bsonic/language"

type Parser struct {
    // SQL-specific parser implementation
}

// Implement base Parser interface
func (p *Parser) Parse(query string) (language.AST, error) {
    // Parse SQL query and return AST
}

func (p *Parser) ParseFieldQuery(query string) (interface{}, error) {
    // Parse field-only SQL query
}

func (p *Parser) ValidateFieldQuery(query string) error {
    // Validate field query doesn't contain text terms when text search disabled
}
```

2. **Add Language Type:**
```go
// config/config.go
const (
    LanguageLucene LanguageType = "lucene"
    LanguageSQL    LanguageType = "sql"  // New language
)
```

3. **Update Factory:**
```go
// factory/factory.go
func CreateParser(langType config.LanguageType) (language.Parser, error) {
    switch langType {
    case config.LanguageLucene:
        return lucene.New(), nil
    case config.LanguageSQL:
        return sql.New(), nil  // New case
    default:
        return nil, fmt.Errorf("unsupported language type: %s", langType)
    }
}
```

### Adding a New Output Formatter

1. **Create Formatter Package:**
```go
// formatter/json/formatter.go
package json

import "github.com/kyle-williams-1/bsonic/formatter"

type Formatter struct {
    // JSON-specific formatter implementation
}

// Implement base Formatter interface
func (f *Formatter) Format(ast interface{}) (string, error) {
    // Convert AST to JSON string
}

func (f *Formatter) FormatMixedQuery(fieldResult string, textTerms string) (string, error) {
    // Combine field query result with text search terms
    // e.g., return `{"$and": [fieldResult, {"text_search": textTerms}]}`
}
```

2. **Add Formatter Type:**
```go
// config/config.go
const (
    FormatterBSON FormatterType = "bson"
    FormatterJSON FormatterType = "json"  // New formatter
)
```

3. **Update Factory:**
```go
// factory/factory.go
func CreateFormatter(formatterType config.FormatterType) (formatter.Formatter[string], error) {
    switch formatterType {
    case config.FormatterBSON:
        return bsonformatter.New(), nil
    case config.FormatterJSON:
        return jsonformatter.New(), nil  // New case
    default:
        return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
    }
}
```

### Usage with New Components

```go
// Create parser with new language and formatter
cfg := config.Default().
    WithLanguage(config.LanguageSQL).
    WithFormatter(config.FormatterJSON)

parser, err := bsonic.NewWithConfig(cfg)
if err != nil {
    log.Fatal(err)
}

// Parse SQL query to JSON
result, err := parser.Parse("SELECT * FROM users WHERE age > 25")
// result is now a JSON string
```

## Architectural Benefits

### True Separation of Concerns

The new architecture achieves complete separation between language parsing and output formatting:

- **Main Parser**: Pure orchestration with no language-specific or output-specific logic
- **Language Parsers**: Handle all query type detection and parsing logic
- **Formatters**: Handle all output format conversion logic
- **No Cross-Contamination**: Language parsers don't know about output formats, formatters don't know about query languages

### Extensibility Without Modification

### Clean Interface Design

- **Base Interfaces**: `Parser` and `Formatter` for basic functionality
- **Optional Features**: Languages and formatters can opt into text search support
- **Type Safety**: Compile-time validation of interface implementations

## Conclusion

BSONIC's extensible architecture combines the power of Participle for robust parsing with a flexible plugin system for languages and formatters. This design provides:

- **Robust parsing** through Participle's battle-tested lexer and grammar system
- **True extensibility** for new query languages and output formats without modifying core code
- **Type safety** through Go's generic type system and interface design
- **Clean separation of concerns** with modular package design and no cross-contamination
- **Easy configuration** through the builder pattern API
- **Future-proof design** that can accommodate new query types and output formats

The architecture is designed to grow with your needs, allowing you to add new query languages (SQL, GraphQL, etc.) and output formats (JSON, XML, etc.) without modifying the core parsing logic.