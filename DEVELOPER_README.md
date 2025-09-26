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
    [Configuration] → [Language Parser] → [AST] → [Formatter] → [BSON Output]
        ↓                ↓                 ↓         ↓
    [Config]        [Lucene Parser]   [Participle]  [BSON Formatter]
    - Language:     - Participle      - AST         - Field Queries
      lucene          Lexer/Parser      Walker      - Text Search ($text)
    - Formatter:    - Grammar         - Expression  - Wildcards ($regex)
      bson            Rules             Processing  - Ranges ($gte/$lte)
                                                    - Comparisons ($gt/$lt)
                                                    - Boolean Logic ($and/$or)
                                                    - Negation ($ne)
```

## Extensible Design

BSONIC is built with extensibility as a core principle, enabling easy addition of new query languages and output formats. While BSONIC primarily outputs BSON, the architecture supports different BSON implementations and document database variants to accommodate various database systems and their specific BSON requirements.

### Package Structure

```
bsonic/
├── config/           # Configuration management
│   └── config.go     # Language and formatter types
├── language/         # Query language implementations
│   ├── interface.go  # Language parser interface
│   └── lucene/       # Lucene-style parser (Participle-based)
├── formatter/        # Output formatter implementations
│   ├── interface.go  # Formatter interface
│   └── bson/         # BSON output formatter
└── bsonic.go         # Main API, orchestration, and factory functions
```

### Interface-Based Architecture

**Base Language Parser Interface:**
```go
type Parser interface {
    Parse(query string) (AST, error)
}
```

**Base Formatter Interface:**
```go
type Formatter[T any] interface {
    Format(ast interface{}) (T, error)
}
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
Generic orchestration layer that coordinates between language parsers and formatters without containing any language or output-specific logic.

### Language System (`language/`)
- **Interfaces**: Base parser contract
- **Lucene Parser**: Participle-based implementation with grammar definition and lexer
- **Query Processing**: Parses field queries and text search

### Formatter System (`formatter/`)
- **Interfaces**: Generic formatter contract
- **BSON Formatter**: MongoDB output with `$text` and `$and` operators
- **AST Processing**: Converts parsed queries to BSON operations with value parsing

### Configuration System (`config/`)
Builder pattern API with type definitions and sensible defaults for language and formatter selection.

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

### Factory Functions

```go
// Language parser creation
languageParser, err := bsonic.NewParser(cfg.Language)

// Formatter creation
formatter, err := bsonic.NewFormatter(cfg.Formatter)
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

1. **Grammar Changes**: Update Participle struct definitions in `language/lucene/parser.go`
2. **Lexer Updates**: Add new token patterns to the lexer rules
3. **AST Conversion**: Implement corresponding BSON conversion logic in `formatter/bson/formatter.go`
4. **Value Parsing**: Add type detection and parsing logic to `parseValue()` method
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
func (p *Parser) Parse(query string) (interface{}, error) {
    // Parse SQL query and return AST
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

3. **Update Factory Functions:**
```go
// bsonic.go - Update NewParser function
func NewParser(langType config.LanguageType) (language.Parser, error) {
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

#### BSON Variants for Different Document Databases

BSONIC supports different BSON implementations for various document databases:

**MongoDB BSON (Default):**
```go
// formatter/bson/formatter.go - Current implementation
// Uses go.mongodb.org/mongo-driver/bson
// Supports MongoDB-specific operators: $text, $regex, $gte, $lte, etc.
```

**Custom Document Database BSON:**
```go
// formatter/customdb/formatter.go
package customdb

import "github.com/kyle-williams-1/bsonic/formatter"

type Formatter struct {
    // Custom database-specific BSON implementation
}

// Implement base Formatter interface
func (f *Formatter) Format(ast interface{}) (CustomBSON, error) {
    // Convert AST to custom database BSON format
    // May use different operators or data types
}

// Example: Different text search operator
func (f *Formatter) freeTextToBSON(ft *lucene.ParticipleFreeText) CustomBSON {
    // Use custom database's text search operator instead of $text
    return CustomBSON{"custom_text_search": searchTerms}
}
```

**Database-Specific Optimizations:**
- **CouchDB**: Use `$or` with `$regex` instead of `$text`
- **Amazon DocumentDB**: Adapt operators for DocumentDB compatibility
- **Azure Cosmos DB**: Use Cosmos DB-specific query operators
- **Custom BSON Libraries**: Support different BSON serialization libraries

#### Custom BSON Variants

1. **Create Custom BSON Formatter:**
```go
// formatter/customdb/formatter.go
package customdb

import "github.com/kyle-williams-1/bsonic/formatter"

type Formatter struct {
    // Custom database-specific BSON implementation
}

// Implement BSON formatter interface
func (f *Formatter) Format(ast interface{}) (bson.M, error) {
    // Convert AST to custom database BSON format
    // May use different operators or data types
}
```

2. **Add Formatter Type:**
```go
// config/config.go
const (
    FormatterBSON     FormatterType = "bson"      // MongoDB BSON
    FormatterCustomDB FormatterType = "customdb"  // Custom document DB
)
```

3. **Update Factory Functions:**
```go
// bsonic.go - Update NewFormatter function
func NewFormatter(formatterType config.FormatterType) (formatter.Formatter[bson.M], error) {
    switch formatterType {
    case config.FormatterBSON:
        return bsonformatter.New(), nil
    case config.FormatterCustomDB:
        return customdbformatter.New(), nil
    default:
        return nil, fmt.Errorf("unsupported formatter type: %s", formatterType)
    }
}
```

## Architectural Benefits

### Key Benefits

- **Separation of Concerns**: Language parsers handle parsing, formatters handle output conversion
- **Extensibility**: Easy to add new languages and formatters without modifying core code
- **Type Safety**: Generic interfaces with compile-time validation
- **Clean Architecture**: No cross-contamination between parsing and formatting logic

## Conclusion

BSONIC provides a clean, extensible architecture for query parsing with:

- **Robust parsing** via Participle's lexer and grammar system
- **Easy extensibility** for new languages and formatters
- **Type safety** through Go's generic interfaces
- **Simple configuration** with builder pattern API

Add new query languages (SQL, GraphQL) and output formats without modifying core logic.