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
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Go library that provides Lucene-style syntax for MongoDB BSON filters. Convert human-readable query strings into MongoDB BSON documents that work seamlessly with the official MongoDB Go driver.

## Features

- **Lucene-style syntax**: Write queries in familiar Lucene format
- **Field matching**: Support for exact matches and wildcard patterns
- **Dot notation**: Query nested data structures using dot notation
- **Array search**: Search within array fields
- **Logical operators**: Support for AND, OR, and NOT operations
- **Grouping logic**: Parentheses support for complex query grouping and precedence control
- **Date queries**: Full support for date range and comparison queries
- **Number ranges**: Support for numeric range queries and comparisons
- **Type-aware parsing**: Automatically detects and parses booleans, numbers, and dates
- **Full-text search**: Support for MongoDB text search with configurable search modes
- **MongoDB compatible**: Generates BSON that works with the latest MongoDB Go driver

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

## Query Syntax

### Basic Field Matching

```go
// Exact match
query, _ := bsonic.Parse("name:john")
// BSON: {"name": "john"}

// Wildcard patterns
query, _ := bsonic.Parse("name:jo*")
// BSON: {"name": {"$regex": "jo.*", "$options": "i"}}

// Quoted values with spaces
query, _ := bsonic.Parse(`name:"john doe"`)
// BSON: {"name": "john doe"}
```

### Dot Notation for Nested Fields

```go
query, _ := bsonic.Parse("user.profile.email:john@example.com")
// BSON: {"user.profile.email": "john@example.com"}
```

### Array Field Queries

```go
query, _ := bsonic.Parse("tags:mongodb")
// BSON: {"tags": "mongodb"}
```

### Logical Operators

```go
// AND operation
query, _ := bsonic.Parse("name:john AND age:25")
// BSON: {"age": 25, "name": "john"}

// OR operation
query, _ := bsonic.Parse("name:john OR name:jane")
// BSON: {"$or": [{"name": "john"}, {"name": "jane"}]}

// NOT operation
query, _ := bsonic.Parse("name:john AND NOT age:25")
// BSON: {"age": {"$ne": 25}, "name": "john"}

// Complex combinations
query, _ := bsonic.Parse("name:jo* OR name:ja* AND NOT age:18")
// BSON: {"$or": [{"name": {"$regex": "jo.*", "$options": "i"}}, {"name": {"$regex": "ja.*", "$options": "i"}}], "age": {"$ne": 18}}
```

### Grouping Logic with Parentheses

Bsonic supports parentheses for grouping expressions and controlling operator precedence.

**Operator Precedence (without parentheses):**
1. `NOT` (highest precedence)
2. `AND` 
3. `OR` (lowest precedence)

```go
// Grouped OR with AND
query, _ := bsonic.Parse("(name:john OR name:jane) AND age:25")
// BSON: {"$and": [{"$or": [{"name": "john"}, {"name": "jane"}]}, {"age": 25}]}

// OR with grouped AND
query, _ := bsonic.Parse("name:john OR (name:jane AND age:25)")
// BSON: {"$or": [{"name": "john"}, {"name": "jane", "age": 25}]}

// Grouped AND expressions with OR
query, _ := bsonic.Parse("(name:john AND age:25) OR (name:jane AND age:30)")
// BSON: {"$or": [{"name": "john", "age": 25}, {"name": "jane", "age": 30}]}

// NOT with grouped OR
query, _ := bsonic.Parse("NOT (name:john OR name:jane)")
// BSON: {"$or": [{"name": {"$ne": "john"}}, {"name": {"$ne": "jane"}}]}

// Nested parentheses
query, _ := bsonic.Parse("((name:john OR name:jane) AND age:25) OR status:active")
// BSON: {"$or": [{"$and": [{"$or": [{"name": "john"}, {"name": "jane"}]}, {"age": 25}]}, {"status": "active"}]}

// Grouped wildcards and numbers
query, _ := bsonic.Parse("(name:jo* OR name:ja*) AND (age:25 OR age:30)")
// BSON: {"$and": [{"$or": [{"name": {"$regex": "jo.*", "$options": "i"}}, {"name": {"$regex": "ja.*", "$options": "i"}}]}, {"$or": [{"age": 25}, {"age": 30}]}]}

// Date range with grouped status
query, _ := bsonic.Parse("created_at:[2023-01-01 TO 2023-12-31] AND (status:active OR status:pending)")
// BSON: {"$and": [{"$or": [{"status": "active"}, {"status": "pending"}]}, {"created_at": {"$gte": "2023-01-01 00:00:00 +0000 UTC", "$lte": "2023-12-31 00:00:00 +0000 UTC"}}]}
```

**Note:** Parentheses must be properly matched.

### Date Queries

```go
// Exact date
query, _ := bsonic.Parse("created_at:2023-01-15")
// BSON: {"created_at": "2023-01-15 00:00:00 +0000 UTC"}

// Date range
query, _ := bsonic.Parse("created_at:[2023-01-01 TO 2023-12-31]")
// BSON: {"created_at": {"$gte": "2023-01-01 00:00:00 +0000 UTC", "$lte": "2023-12-31 00:00:00 +0000 UTC"}}

// Date range with wildcards
query, _ := bsonic.Parse("created_at:[2023-01-01 TO *]")
// BSON: {"created_at": {"$gte": "2023-01-01 00:00:00 +0000 UTC"}}

query, _ := bsonic.Parse("created_at:[* TO 2023-12-31]")
// BSON: {"created_at": {"$lte": "2023-12-31 00:00:00 +0000 UTC"}}

// Date comparisons
query, _ := bsonic.Parse("created_at:>2024-01-01")
// BSON: {"created_at": {"$gt": "2024-01-01 00:00:00 +0000 UTC"}}

query, _ := bsonic.Parse("created_at:<2023-12-31")
// BSON: {"created_at": {"$lt": "2023-12-31 00:00:00 +0000 UTC"}}

query, _ := bsonic.Parse("created_at:>=2024-01-01")
// BSON: {"created_at": {"$gte": "2024-01-01 00:00:00 +0000 UTC"}}

query, _ := bsonic.Parse("created_at:<=2023-12-31")
// BSON: {"created_at": {"$lte": "2023-12-31 00:00:00 +0000 UTC"}}

// Complex date queries
query, _ := bsonic.Parse("created_at:[2023-01-01 TO 2023-12-31] AND status:active")
// BSON: {"created_at": {"$gte": "2023-01-01 00:00:00 +0000 UTC", "$lte": "2023-12-31 00:00:00 +0000 UTC"}, "status": "active"}

query, _ := bsonic.Parse("created_at:>2024-01-01 OR updated_at:<2023-01-01")
// BSON: {"$or": [{"created_at": {"$gt": "2024-01-01 00:00:00 +0000 UTC"}}, {"updated_at": {"$lt": "2023-01-01 00:00:00 +0000 UTC"}}]}
```

### Number Range Queries

```go
// Number ranges
query, _ := bsonic.Parse("age:[18 TO 65]")
// BSON: {"age": {"$gte": 18, "$lte": 65}}

// Decimal number ranges
query, _ := bsonic.Parse("price:[10.50 TO 99.99]")
// BSON: {"price": {"$gte": 10.5, "$lte": 99.99}}

// Number ranges with wildcards
query, _ := bsonic.Parse("age:[18 TO *]")
// BSON: {"age": {"$gte": 18}}

query, _ := bsonic.Parse("age:[* TO 65]")
// BSON: {"age": {"$lte": 65}}

// Number comparisons
query, _ := bsonic.Parse("score:>85")
// BSON: {"score": {"$gt": 85}}

query, _ := bsonic.Parse("score:<60")
// BSON: {"score": {"$lt": 60}}

query, _ := bsonic.Parse("score:>=90")
// BSON: {"score": {"$gte": 90}}

query, _ := bsonic.Parse("score:<=50")
// BSON: {"score": {"$lte": 50}}

// Complex number queries
query, _ := bsonic.Parse("age:[18 TO 65] AND status:active")
// BSON: {"age": {"$gte": 18, "$lte": 65}, "status": "active"}

query, _ := bsonic.Parse("age:>18 OR score:<60")
// BSON: {"$or": [{"age": {"$gt": 18}}, {"score": {"$lt": 60}}]}

query, _ := bsonic.Parse("price:[0 TO 100] OR rating:[4 TO 5]")
// BSON: {"$or": [{"price": {"$gte": 0, "$lte": 100}}, {"rating": {"$gte": 4, "$lte": 5}}]}
```

### Supported Date Formats

The library automatically detects and parses various date formats:

- `2023-01-15` (ISO 8601 date)
- `2023-01-15T10:30:00Z` (ISO 8601 datetime)
- `2023-01-15T10:30:00` (ISO 8601 datetime without timezone)
- `2023-01-15 10:30:00` (space-separated datetime)
- `01/15/2023` (US format)
- `2023/01/15` (ISO format)

### Type-Aware Parsing

The library automatically detects and parses different data types:

```go
// Boolean values
query, _ := parser.Parse("active:true")
// BSON: {"active": true}

// Numeric values
query, _ := parser.Parse("age:25")
// BSON: {"age": 25}

// Date values
query, _ := parser.Parse("created_at:2023-01-15")
// BSON: {"created_at": "2023-01-15 00:00:00 +0000 UTC"}

// String values (default)
query, _ := parser.Parse("name:john")
// BSON: {"name": "john"}
```

### Full-Text Search

Bsonic supports MongoDB's full-text search functionality, allowing you to search across multiple fields simultaneously.

#### Enabling Text Search

```go
// Create a parser with text search enabled
parser := bsonic.NewWithTextSearch()

// Or enable text search on an existing parser
parser := bsonic.New()
parser.SetSearchMode(bsonic.SearchModeText)
```

#### Basic Text Search

```go
// Simple text search
query, _ := parser.Parse("search terms")
// BSON: {"$text": {"$search": "search terms"}}

// Multiple words
query, _ := parser.Parse("multiple words search")
// BSON: {"$text": {"$search": "multiple words search"}}
```

#### Text Search vs Field Search

When text search is enabled, bsonic automatically determines whether to use text search or field search:

- **Text search**: Used when the query contains no field:value pairs (no colons)
- **Field search**: Used when the query contains field:value pairs

```go
// This will use text search
query, _ := parser.Parse("engineer software")
// BSON: {"$text": {"$search": "engineer software"}}

// This will use field search
query, _ := parser.Parse("name:john")
// BSON: {"name": "john"}}
```

#### MongoDB Text Index Requirements

**Important**: To use text search, you must create text indexes on your MongoDB collections. Text indexes must be created on all fields you want to be searchable.

```javascript
// Example: Create a text index on multiple fields
db.users.createIndex({ 
  "name": "text", 
  "email": "text", 
  "profile.bio": "text",
  "tags": "text"
});

// Example: Create a text index on product fields
db.products.createIndex({ 
  "name": "text", 
  "category": "text",
  "tags": "text",
  "description": "text"
});
```

#### Mixed Queries

Mixed queries combine text search with field searches, allowing you to search for text terms while filtering by specific field values:

```go
// Text search combined with field search
query, _ := parser.Parse("engineer active:true")
// BSON: {"$and": [{"$text": {"$search": "engineer"}}, {"active": true}]}

// Multiple text terms with field search
query, _ := parser.Parse("software engineer role:admin")
// BSON: {"$and": [{"$text": {"$search": "software engineer"}}, {"role": "admin"}]}

// Text search with complex field queries
query, _ := parser.Parse("designer role:user AND active:true")
// BSON: {"$and": [{"$text": {"$search": "designer"}}, {"role": "user", "active": true}]}

// Text search with OR field queries
query, _ := parser.Parse("devops role:admin OR name:charlie")
// BSON: {"$and": [{"$text": {"$search": "devops"}}, {"$or": [{"role": "admin"}, {"name": "charlie"}]}]}
```

**How it works:**
- Text terms (without colons) are combined into a single `$text` search
- Field:value pairs are parsed as regular field searches
- Both are combined with `$and` operator for precise filtering

## Integration Testing

This library includes comprehensive integration tests that run against a real MongoDB instance.

- [Integration Testing Guide](integration/README.md) - Complete setup and testing guide
- [Integration Troubleshooting](integration/INTEGRATION_TROUBLESHOOTING.md) - Common issues and solutions

## Examples

Check out the [examples](examples/) directory for more detailed usage examples.

## Contributing

Contributions are welcome! Please open an issue or pull request on GitHub.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Dependencies

See [DEPENDENCIES.md](DEPENDENCIES.md) for information about required and optional dependencies.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a list of changes and new features.