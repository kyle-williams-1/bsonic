<p align="center">
  <img src="assets/logo.png" alt="bsonic logo" width="200"/>
</p>

<h1 align="center">BSONIC</h1>

<p align="center">
  Parse <b>Lucene-style query syntax</b> into <b>BSON filters</b> for MongoDB — fast, simple, and developer-friendly.
</p>

[![CI](https://github.com/kyle-williams-1/bsonic/actions/workflows/ci.yml/badge.svg)](https://github.com/kyle-williams-1/bsonic/actions/workflows/ci.yml)
[![Integration Tests](https://github.com/kyle-williams-1/bsonic/actions/workflows/integration.yml/badge.svg)](https://github.com/kyle-williams-1/bsonic/actions/workflows/integration.yml)
[![codecov](https://codecov.io/gh/kyle-williams-1/bsonic/branch/main/graph/badge.svg)](https://codecov.io/gh/kyle-williams-1/bsonic)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyle-williams-1/bsonic)](https://goreportcard.com/report/github.com/kyle-williams-1/bsonic)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Go Reference](https://pkg.go.dev/badge/github.com/kyle-williams-1/bsonic.svg)](https://pkg.go.dev/github.com/kyle-williams-1/bsonic)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Go library that provides Lucene-style syntax for MongoDB BSON filters. Convert human-readable query strings into MongoDB BSON documents that work seamlessly with the official MongoDB Go driver. Built with extensibility in mind, supporting multiple query languages and output formatters.

## Features

- **Lucene-style syntax**: Write queries in familiar Lucene format
- **Field matching**: Exact matches, wildcard patterns, and regex support
- **Default fields**: Search across multiple fields using regex patterns (recommended)
- **Mixed queries**: Combine free text search with field-specific queries
- **Nested data**: Dot notation for nested fields and array search
- **Logical operators**: AND, OR, NOT with parentheses grouping
- **Date & number queries**: Range queries and comparisons with type-aware parsing
- **MongoDB compatible**: Generates BSON for the official MongoDB Go driver
- **Extensible**: Easy to add new query languages and output formatters

## Installation

```bash
go get github.com/kyle-williams-1/bsonic
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/kyle-williams-1/bsonic"
)

func main() {
    // Parse a simple query using the package-level function
    query, err := bsonic.Parse("name:john AND age:25")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("BSON: %+v\n", query)
    // Output: BSON: {"age": 25, "name": "john"}
}
```

### Alternative API

Used when you need to reuse parser instances:

```go
parser := bsonic.New()
query, err := parser.Parse("name:john AND age:25")
```

## Extensible Architecture

Bsonic supports multiple query languages and output formatters through a modular design.

### Configuration

```go
import (
    "github.com/kyle-williams-1/bsonic"
    "github.com/kyle-williams-1/bsonic/config"
)

// Default (Lucene + BSON)
parser := bsonic.New()

// Custom configuration
cfg := config.Default().
    WithLanguage(config.LanguageLucene).
    WithFormatter(config.FormatterMongo)
parser, _ := bsonic.NewWithConfig(cfg)
```

### Package Structure

```
bsonic/
├── config/           # Configuration types
├── language/lucene/  # Lucene query parser
├── formatter/mongo/  # MongoDB BSON output formatter
└── bsonic.go         # Main API
```

**Adding New Languages/Formatters:** Implement the `language.Parser` or `formatter.Formatter` interfaces.

## Query Syntax

### Field Matching

```go
// Exact match
query, _ := bsonic.Parse("name:john")
// BSON: {"name": "john"}

// Wildcard patterns
query, _ := bsonic.Parse("name:jo*")
// BSON: {"name": {"$regex": "^jo.*", "$options": "i"}}

// Regex patterns (case-sensitive by default)
query, _ := bsonic.Parse("email:/.*@example\\.com/")
// BSON: {"email": {"$regex": ".*@example\\.com"}}

// Quoted values with spaces
query, _ := bsonic.Parse(`name:"john doe"`)
// BSON: {"name": "john doe"}

// Nested fields
query, _ := bsonic.Parse("user.profile.email:john@example.com")
// BSON: {"user.profile.email": "john@example.com"}

// Array fields
query, _ := bsonic.Parse("tags:mongodb")
// BSON: {"tags": "mongodb"}
```

### Default Fields (Recommended)

Bsonic supports default fields for free text queries, allowing you to search across specific fields without using MongoDB's text search operator. This provides more flexibility and doesn't require text indexes.

```go
// Simple default field search
query, _ := bsonic.ParseWithDefaults([]string{"name"}, "john")
// BSON: {"name": {"$regex": "john", "$options": "i"}}

// Search across multiple default fields
query, _ := bsonic.ParseWithDefaults([]string{"name", "description"}, "engineer")
// BSON: {"$or": [{"name": {"$regex": "engineer", "$options": "i"}}, {"description": {"$regex": "engineer", "$options": "i"}}]}

// Default fields with wildcards
query, _ := bsonic.ParseWithDefaults([]string{"name"}, "jo*")
// BSON: {"name": {"$regex": "^jo.*$", "$options": "i"}}

// Default fields with regex patterns (case-sensitive)
query, _ := bsonic.ParseWithDefaults([]string{"email"}, "/.*@example\\.com/")
// BSON: {"email": {"$regex": ".*@example\\.com"}}

// Mixed free text and field queries
query, _ := bsonic.ParseWithDefaults([]string{"name"}, "john AND age:25")
// BSON: {"age": 25, "name": {"$regex": "john", "$options": "i"}}

// With complex field conditions
query, _ := bsonic.ParseWithDefaults([]string{"name"}, "john AND (role:admin OR department:engineering)")
// BSON: {"$and": [{"$or": [{"role": "admin"}, {"department": "engineering"}]}, {"name": {"$regex": "john", "$options": "i"}}]}
```

#### Configuration-Based Default Fields

You can also configure default fields at the parser level:

```go
import (
    "github.com/kyle-williams-1/bsonic"
    "github.com/kyle-williams-1/bsonic/config"
)

// Configure parser with default fields
cfg := config.Default().
    WithDefaultFields([]string{"name", "description"}).

parser, _ := bsonic.NewWithConfig(cfg)
query, _ := parser.Parse("engineer")
// BSON: {"$or": [{"name": {"$regex": "engineer", "$options": "i"}}, {"description": {"$regex": "engineer", "$options": "i"}}]}
```

### Logical Operators & Grouping

**Operator Precedence:** `NOT` > `AND` > `OR`

```go
// Basic operations
query, _ := bsonic.Parse("name:john AND age:25")
// BSON: {"age": 25, "name": "john"}

query, _ := bsonic.Parse("name:john OR name:jane")
// BSON: {"$or": [{"name": "john"}, {"name": "jane"}]}

query, _ := bsonic.Parse("name:john AND NOT age:25")
// BSON: {"age": {"$ne": 25}, "name": "john"}

// Grouping with parentheses
query, _ := bsonic.Parse("(name:john OR name:jane) AND age:25")
// BSON: {"$and": [{"$or": [{"name": "john"}, {"name": "jane"}]}, {"age": 25}]}

query, _ := bsonic.Parse("NOT (name:john OR name:jane)")
// BSON: {"$and": [{"name": {"$ne": "john"}}, {"name": {"$ne": "jane"}}]}

// Complex combinations with regex
query, _ := bsonic.Parse("name:/john/ OR email:/.*@example\\.com/ AND NOT status:inactive")
// BSON: {"$or": [{"name": {"$regex": "john"}}, {"email": {"$regex": ".*@example\\.com"}, "status": {"$ne": "inactive"}}]}
```

### Date & Number Queries

```go
// Date ranges and comparisons
query, _ := bsonic.Parse("created_at:[2023-01-01 TO 2023-12-31]")
// BSON: {"created_at": {"$gte": "2023-01-01 00:00:00 +0000 UTC", "$lte": "2023-12-31 00:00:00 +0000 UTC"}}

query, _ := bsonic.Parse("created_at:>2024-01-01")
// BSON: {"created_at": {"$gt": "2024-01-01 00:00:00 +0000 UTC"}}

// Number ranges and comparisons
query, _ := bsonic.Parse("age:[18 TO 65]")
// BSON: {"age": {"$gte": 18, "$lte": 65}}

query, _ := bsonic.Parse("score:>85")
// BSON: {"score": {"$gt": 85}}

// Type-aware parsing (auto-detects booleans, numbers, dates)
query, _ := bsonic.Parse("active:true AND age:25 AND created_at:2023-01-15")
// BSON: {"active": true, "age": 25, "created_at": "2023-01-15 00:00:00 +0000 UTC"}
```

**Supported Date Formats:** `2023-01-15`, `2023-01-15T10:30:00Z`, `01/15/2023`, etc.

### Error Handling & Performance

```go
// Safe parsing with error handling
func parseQuerySafely(query string) (bson.M, error) {
    result, err := bsonic.Parse(query)
    if err != nil {
        return nil, fmt.Errorf("failed to parse query '%s': %w", query, err)
    }
    return result, nil
}

// For high-performance applications, reuse parser instances
var globalParser = bsonic.New()
```

## Examples & Testing

- [Examples](examples/) - Detailed usage examples
- [Integration Tests](integration/README.md) - MongoDB integration testing guide

## Contributing

Contributions welcome! See [DEPENDENCIES.md](DEPENDENCIES.md) for development setup.

**Requirements:** Go 1.25+, golangci-lint, Docker (for integration tests)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Links

- [Changelog](CHANGELOG.md) - Recent changes and features
- [Dependencies](DEPENDENCIES.md) - Required and optional dependencies
