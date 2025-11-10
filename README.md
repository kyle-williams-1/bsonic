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
[![Code scanning alerts](https://github.com/kyle-williams-1/bsonic/workflows/CodeQL/badge.svg)](https://github.com/kyle-williams-1/bsonic/security/code-scanning)
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
- **ID field conversion**: Automatic `id` to `_id` field conversion with ObjectID support
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
    "encoding/json"
    "fmt"
    "log"

    "github.com/kyle-williams-1/bsonic"
)

func main() {
    query, err := bsonic.Parse("name:john AND age:25")
    if err != nil {
        log.Fatal(err)
    }

    jsonBytes, _ := json.MarshalIndent(query, "", "  ")
    fmt.Println(string(jsonBytes))
}
```

**Output:**
```json
{
  "age": 25,
  "name": "john"
}
```

For applications that need to reuse parser instances:

```go
parser := bsonic.New()
query, err := parser.Parse("name:john AND age:25")
```

## Configuration

Bsonic provides flexible configuration options to customize parser behavior. Use configuration when you need to:
- Set default fields for free text searches
- Control ID field conversion behavior
- Customize language and formatter selection

**Why use configuration?** Default fields enable powerful free-text search without requiring MongoDB text indexes. ID conversion simplifies working with MongoDB's `_id` convention.

```go
import (
    "github.com/kyle-williams-1/bsonic"
    "github.com/kyle-williams-1/bsonic/config"
)

// Default configuration (Lucene + MongoDB formatter)
parser := bsonic.New()

// Custom configuration with default fields
cfg := config.Default().
    WithLanguage(config.LanguageLucene).
    WithFormatter(config.FormatterMongo).
    WithDefaultFields([]string{"name", "description", "title"}).
    WithReplaceIDWithMongoID(true).      // Convert "id" to "_id" (default: true)
    WithAutoConvertIDToObjectID(true)     // Convert string to ObjectID (default: true)

parser, _ := bsonic.NewWithConfig(cfg)

// Now free text queries search across name, description, and title
query, _ := parser.Parse("engineer")
```

**Configuration Options:**
- `WithDefaultFields([]string)`: Fields to search for free text queries
- `WithReplaceIDWithMongoID(bool)`: Convert `id` field names to `_id` (default: `true`)
- `WithAutoConvertIDToObjectID(bool)`: Convert string values to `primitive.ObjectID` (default: `true`)

## Query Syntax

### String Search

Exact string matching with support for quoted phrases. Field queries are case-sensitive.

```go
// Exact match
query, _ := bsonic.Parse("name:john")
// Output:
{
  "name": "john"
}

// Quoted values with spaces
query, _ := bsonic.Parse(`name:"john doe"`)
// Output:
{
  "name": "john doe"
}
```

**Note:** For case-insensitive searches, use default fields with free text (see Default Fields section).

### Wildcard Patterns

Use `*` to match any sequence of characters. Wildcards are case-sensitive.

```go
// Starts with pattern
query, _ := bsonic.Parse("name:Jo*")
// Output:
{
  "name": {
    "$regex": "^Jo.*"
  }
}
```

### Regex Patterns

Wrap patterns in forward slashes `/pattern/`. Bsonic automatically adds anchors for exact matching unless already present.

```go
// Basic regex pattern
query, _ := bsonic.Parse("email:/.*@example\\.com/")
// Output:
{
  "email": {
    "$regex": "^.*@example\\.com$"
  }
}

// Complex pattern
query, _ := bsonic.Parse("phone:/^\\+?[1-9]\\d{1,14}$/")
// Output:
{
  "phone": {
    "$regex": "^\\+?[1-9]\\d{1,14}$"
  }
}
```

**Note:** Regex patterns are case-sensitive. Anchors (`^` and `$`) are automatically added if not present.

### Primitive ID Conversion

Bsonic automatically detects fields ending with `_id` (including `id` which converts to `_id`) and converts valid 24-character hex strings to `primitive.ObjectID`. Invalid ObjectIDs fall back to string matching. All query patterns (regex, wildcards, ranges) work on ID fields when ObjectID conversion isn't applicable.

```go
// ObjectID conversion
query, _ := bsonic.Parse("id:507f1f77bcf86cd799439011")
// Output:
{
  "_id": ObjectID("507f1f77bcf86cd799439011")
}

// Fields ending with _id are automatically detected
query, _ := bsonic.Parse("user_id:507f1f77bcf86cd799439011")
// Output:
{
  "user_id": ObjectID("507f1f77bcf86cd799439011")
}

// Invalid ObjectID falls back to string search
query, _ := bsonic.Parse("id:invalid-hex")
// Output:
{
  "_id": "invalid-hex"
}
```

**Configuration:** ID field name conversion (`id` → `_id`) and ObjectID conversion are configurable via `WithReplaceIDWithMongoID()` and `WithAutoConvertIDToObjectID()`.

### Date Queries & Ranges

Bsonic automatically detects and parses dates in various formats. Use range syntax `[start TO end]` and comparison operators `>`, `<`, `>=`, `<=`.

**Supported Date Formats:** `2023-01-15`, `2023-01-15T10:30:00Z`, `2023-01-15T10:30:00`, `2023-01-15 10:30:00`, `01/15/2023`, `2023/01/15`

```go
// Single date
query, _ := bsonic.Parse("created_at:2023-01-15")
// Output:
{
  "created_at": "2023-01-15 00:00:00 +0000 UTC"
}

// Date range
query, _ := bsonic.Parse("created_at:[2023-01-01 TO 2023-12-31]")
// Output:
{
  "created_at": {
    "$gte": "2023-01-01 00:00:00 +0000 UTC",
    "$lte": "2023-12-31 00:00:00 +0000 UTC"
  }
}

// Comparison operators
query, _ := bsonic.Parse("created_at:>2024-01-01")
// Output:
{
  "created_at": {
    "$gt": "2024-01-01 00:00:00 +0000 UTC"
  }
}

// Open-ended range
query, _ := bsonic.Parse("created_at:[* TO 2023-12-31]")
// Output:
{
  "created_at": {
    "$lte": "2023-12-31 00:00:00 +0000 UTC"
  }
}
```

### Number Queries & Ranges

Numbers are automatically detected and parsed. Supports integers, floats, ranges, and comparisons.

```go
// Integer
query, _ := bsonic.Parse("age:25")
// Output:
{
  "age": 25
}

// Number range
query, _ := bsonic.Parse("age:[18 TO 65]")
// Output:
{
  "age": {
    "$gte": 18,
    "$lte": 65
  }
}

// Comparison operators
query, _ := bsonic.Parse("score:>85")
// Output:
{
  "score": {
    "$gt": 85
  }
}

// Float range
query, _ := bsonic.Parse("price:[10.50 TO 99.99]")
// Output:
{
  "price": {
    "$gte": 10.5,
    "$lte": 99.99
  }
}
```

### Boolean Queries

Boolean values are automatically detected and converted to Go boolean types.

```go
query, _ := bsonic.Parse("active:true")
// Output:
{
  "active": true
}
```

### Nested Data Search

Use dot notation to query nested fields. Works with all query types.

```go
// Nested field
query, _ := bsonic.Parse("user.profile.email:john@example.com")
// Output:
{
  "user.profile.email": "john@example.com"
}

// Nested with range
query, _ := bsonic.Parse("user.profile.age:[18 TO 65]")
// Output:
{
  "user.profile.age": {
    "$gte": 18,
    "$lte": 65
  }
}
```

### Array Searches

Query array fields like any other field. MongoDB automatically matches array elements.

```go
// Array field search
query, _ := bsonic.Parse("tags:mongodb")
// Output:
{
  "tags": "mongodb"
}

// Array with multiple values
query, _ := bsonic.Parse("tags:mongodb OR tags:go")
// Output:
{
  "$or": [
    {
      "tags": "mongodb"
    },
    {
      "tags": "go"
    }
  ]
}
```

### Logical Operators

Combine conditions using `AND` and `OR` operators. **Operator Precedence:** `NOT` > `AND` > `OR`

```go
// AND operator
query, _ := bsonic.Parse("name:john AND age:25")
// Output:
{
  "age": 25,
  "name": "john"
}

// OR operator
query, _ := bsonic.Parse("name:john OR name:jane")
// Output:
{
  "$or": [
    {
      "name": "john"
    },
    {
      "name": "jane"
    }
  ]
}

// Complex AND/OR combination
query, _ := bsonic.Parse("name:john AND (age:25 OR age:30)")
// Output:
{
  "$and": [
    {
      "name": "john"
    },
    {
      "$or": [
        {
          "age": 25
        },
        {
          "age": 30
        }
      ]
    }
  ]
}
```

### NOT Operator

Negate conditions using the `NOT` operator. Bsonic applies De Morgan's law for complex negations.

```go
// Simple NOT
query, _ := bsonic.Parse("NOT status:inactive")
// Output:
{
  "status": {
    "$ne": "inactive"
  }
}

// NOT with OR (applies De Morgan's law)
query, _ := bsonic.Parse("NOT (name:john OR name:jane)")
// Output:
{
  "$and": [
    {
      "name": {
        "$ne": "john"
      }
    },
    {
      "name": {
        "$ne": "jane"
      }
    }
  ]
}

// NOT with regex (uses $not operator)
query, _ := bsonic.Parse("NOT name:/john/")
// Output:
{
  "name": {
    "$not": {
      "$regex": "^john$"
    }
  }
}
```

### Grouping with Parentheses

Use parentheses to control operator precedence. Nested parentheses are supported.

```go
// Basic grouping
query, _ := bsonic.Parse("(name:john OR name:jane) AND age:25")
// Output:
{
  "$and": [
    {
      "$or": [
        {
          "name": "john"
        },
        {
          "name": "jane"
        }
      ]
    },
    {
      "age": 25
    }
  ]
}

// Complex nested query
query, _ := bsonic.Parse("(name:jo* OR name:ja*) AND (age:[18 TO 65] OR role:admin)")
// Output:
{
  "$and": [
    {
      "$or": [
        {
          "name": {
            "$regex": "^jo.*"
          }
        },
        {
          "name": {
            "$regex": "^ja.*"
          }
        }
      ]
    },
    {
      "$or": [
        {
          "age": {
            "$gte": 18,
            "$lte": 65
          }
        },
        {
          "role": "admin"
        }
      ]
    }
  ]
}
```

### Default Fields

Default fields enable free-text search across multiple fields without requiring MongoDB text indexes. Free text searches are case-insensitive by default, unless regex or wildcards are used.

```go
// Multiple default fields
query, _ := bsonic.ParseWithDefaults([]string{"name", "description"}, "engineer")
// Output:
{
  "$or": [
    {
      "name": {
        "$regex": "^engineer$",
        "$options": "i"
      }
    },
    {
      "description": {
        "$regex": "^engineer$",
        "$options": "i"
      }
    }
  ]
}

// Multiple words (each word searches all default fields with OR)
query, _ := bsonic.ParseWithDefaults([]string{"name", "title"}, "software engineer")
// Output:
{
  "$or": [
    {
      "name": {
        "$regex": "^software$",
        "$options": "i"
      }
    },
    {
      "title": {
        "$regex": "^software$",
        "$options": "i"
      }
    },
    {
      "name": {
        "$regex": "^engineer$",
        "$options": "i"
      }
    },
    {
      "title": {
        "$regex": "^engineer$",
        "$options": "i"
      }
    }
  ]
}

// Quoted phrase (treated as single term)
query, _ := bsonic.ParseWithDefaults([]string{"name"}, `"john doe"`)
// Output:
{
  "name": {
    "$regex": "^john doe$",
    "$options": "i"
  }
}

// Configuration-based default fields
cfg := config.Default().
    WithDefaultFields([]string{"name", "description", "title"})
parser, _ := bsonic.NewWithConfig(cfg)
query, _ := parser.Parse("engineer")
```

### Mixed Default Field and Structured Queries

Combine free text search with structured field queries. By default, they are combined with OR unless explicit operators are used.

```go
// Mixed query (defaults to OR)
query, _ := bsonic.ParseWithDefaults([]string{"role"}, "name:john admin")
// Output:
{
  "$or": [
    {
      "name": "john"
    },
    {
      "role": {
        "$regex": "^admin$",
        "$options": "i"
      }
    }
  ]
}

// Mixed query with explicit AND
query, _ := bsonic.ParseWithDefaults([]string{"name"}, "john AND role:admin")
// Output:
{
  "$and": [
    {
      "name": {
        "$regex": "^john$",
        "$options": "i"
      }
    },
    {
      "role": "admin"
    }
  ]
}
```

## Extensible Architecture

Bsonic supports multiple query languages and output formatters through a modular design.

### Package Structure

```
bsonic/
├── config/           # Configuration types
├── language/lucene/  # Lucene query parser
├── formatter/mongo/  # MongoDB BSON output formatter
└── bsonic.go         # Main API
```

**Adding New Languages/Formatters:** Implement the `language.Parser` or `formatter.Formatter` interfaces.

## Error Handling & Performance

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

func handleRequest(queryStr string) (bson.M, error) {
    return globalParser.Parse(queryStr)
}
```

## Examples & Testing

- [Examples](examples/) - Detailed usage examples
- [Integration Tests](tests/README.md) - MongoDB integration testing guide

## Contributing

Contributions welcome! See [DEPENDENCIES.md](DEPENDENCIES.md) for development setup.

**Requirements:** Go 1.25+, golangci-lint, Docker (for integration tests)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Links

- [Changelog](CHANGELOG.md) - Recent changes and features
- [Dependencies](DEPENDENCIES.md) - Required and optional dependencies
