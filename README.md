<p align="center">
  <img src="assets/logo.png" alt="bsonic logo" width="200"/>
</p>

<h1 align="center">BSONIC</h1>

<p align="center">
  Parse <b>Lucene-style query syntax</b> into <b>BSON filters</b> for MongoDB — fast, simple, and developer-friendly.
</p>

[![CI](https://github.com/kyle-williams-1/bsonic/actions/workflows/ci.yml/badge.svg)](https://github.com/kyle-williams-1/bsonic/actions/workflows/ci.yml)
[![Integration Tests](https://github.com/kyle-williams-1/bsonic/actions/workflows/integration.yml/badge.svg)](https://github.com/kyle-williams-1/bsonic/actions/workflows/integration.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyle-williams-1/bsonic)](https://goreportcard.com/report/github.com/kyle-williams-1/bsonic)
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
- **Type-aware parsing**: Automatically detects and parses booleans, numbers, and dates
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
    parser := bsonic.New()
    
    // Parse a simple query
    query, err := parser.Parse("name:john AND age:25")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("BSON: %+v\n", query)
    // Output: BSON: map[age:25 name:john]
}
```

## Query Syntax

### Basic Field Matching

```go
// Exact match
query, _ := parser.Parse("name:john")
// BSON: map[name:john]

// Wildcard patterns
query, _ := parser.Parse("name:jo*")
// BSON: map[name:map[$regex:jo.* $options:i]]

// Quoted values with spaces
query, _ := parser.Parse(`name:"john doe"`)
// BSON: map[name:john doe]
```

### Dot Notation for Nested Fields

```go
query, _ := parser.Parse("user.profile.email:john@example.com")
// BSON: map[user.profile.email:john@example.com]
```

### Array Field Queries

```go
query, _ := parser.Parse("tags:mongodb")
// BSON: map[tags:mongodb]
```

### Logical Operators

```go
// AND operation
query, _ := parser.Parse("name:john AND age:25")
// BSON: map[age:25 name:john]

// OR operation
query, _ := parser.Parse("name:john OR name:jane")
// BSON: map[$or:[map[name:john] map[name:jane]]]

// NOT operation
query, _ := parser.Parse("name:john AND NOT age:25")
// BSON: map[age:map[$ne:25] name:john]

// Complex combinations
query, _ := parser.Parse("name:jo* OR name:ja* AND NOT age:18")
// BSON: map[$or:[map[name:map[$regex:jo.* $options:i]] map[name:map[$regex:ja.* $options:i]]] age:map[$ne:18]]
```

### Grouping Logic with Parentheses

Bsonic supports parentheses for grouping expressions and controlling operator precedence.

**Operator Precedence (without parentheses):**
1. `NOT` (highest precedence)
2. `AND` 
3. `OR` (lowest precedence)

```go
// Grouped OR with AND
query, _ := parser.Parse("(name:john OR name:jane) AND age:25")
// BSON: map[$and:[map[$or:[map[name:john] map[name:jane]]] map[age:25]]]

// OR with grouped AND
query, _ := parser.Parse("name:john OR (name:jane AND age:25)")
// BSON: map[$or:[map[name:john] map[name:jane age:25]]]

// Grouped AND expressions with OR
query, _ := parser.Parse("(name:john AND age:25) OR (name:jane AND age:30)")
// BSON: map[$or:[map[name:john age:25] map[name:jane age:30]]]

// NOT with grouped OR
query, _ := parser.Parse("NOT (name:john OR name:jane)")
// BSON: map[$or:[map[name:map[$ne:john]] map[name:map[$ne:jane]]]]

// Nested parentheses
query, _ := parser.Parse("((name:john OR name:jane) AND age:25) OR status:active")
// BSON: map[$or:[map[$and:[map[$or:[map[name:john] map[name:jane]]] map[age:25]]] map[status:active]]]

// Grouped wildcards and numbers
query, _ := parser.Parse("(name:jo* OR name:ja*) AND (age:25 OR age:30)")
// BSON: map[$and:[map[$or:[map[name:map[$regex:jo.* $options:i]] map[name:map[$regex:ja.* $options:i]]]] map[$or:[map[age:25] map[age:30]]]]]

// Date range with grouped status
query, _ := parser.Parse("created_at:[2023-01-01 TO 2023-12-31] AND (status:active OR status:pending)")
// BSON: map[$and:[map[$or:[map[status:active] map[status:pending]]] map[created_at:map[$gte:2023-01-01 00:00:00 +0000 UTC $lte:2023-12-31 00:00:00 +0000 UTC]]]]
```

**Note:** Parentheses must be properly matched.

### Date Queries

```go
// Exact date
query, _ := parser.Parse("created_at:2023-01-15")
// BSON: map[created_at:2023-01-15 00:00:00 +0000 UTC]

// Date range
query, _ := parser.Parse("created_at:[2023-01-01 TO 2023-12-31]")
// BSON: map[created_at:map[$gte:2023-01-01 00:00:00 +0000 UTC $lte:2023-12-31 00:00:00 +0000 UTC]]

// Date range with wildcards
query, _ := parser.Parse("created_at:[2023-01-01 TO *]")
// BSON: map[created_at:map[$gte:2023-01-01 00:00:00 +0000 UTC]]

query, _ := parser.Parse("created_at:[* TO 2023-12-31]")
// BSON: map[created_at:map[$lte:2023-12-31 00:00:00 +0000 UTC]]

// Date comparisons
query, _ := parser.Parse("created_at:>2024-01-01")
// BSON: map[created_at:map[$gt:2024-01-01 00:00:00 +0000 UTC]]

query, _ := parser.Parse("created_at:<2023-12-31")
// BSON: map[created_at:map[$lt:2023-12-31 00:00:00 +0000 UTC]]

query, _ := parser.Parse("created_at:>=2024-01-01")
// BSON: map[created_at:map[$gte:2024-01-01 00:00:00 +0000 UTC]]

query, _ := parser.Parse("created_at:<=2023-12-31")
// BSON: map[created_at:map[$lte:2023-12-31 00:00:00 +0000 UTC]]

// Complex date queries
query, _ := parser.Parse("created_at:[2023-01-01 TO 2023-12-31] AND status:active")
// BSON: map[created_at:map[$gte:2023-01-01 00:00:00 +0000 UTC $lte:2023-12-31 00:00:00 +0000 UTC] status:active]

query, _ := parser.Parse("created_at:>2024-01-01 OR updated_at:<2023-01-01")
// BSON: map[$or:[map[created_at:map[$gt:2024-01-01 00:00:00 +0000 UTC]] map[updated_at:map[$lt:2023-01-01 00:00:00 +0000 UTC]]]]
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
// BSON: map[active:true]

// Numeric values
query, _ := parser.Parse("age:25")
// BSON: map[age:25]

// Date values
query, _ := parser.Parse("created_at:2023-01-15")
// BSON: map[created_at:2023-01-15 00:00:00 +0000 UTC]

// String values (default)
query, _ := parser.Parse("name:john")
// BSON: map[name:john]
```

## Integration Testing

This library includes comprehensive integration tests that run against a real MongoDB instance. See [INTEGRATION_TESTING.md](INTEGRATION_TESTING.md) for details on how to run integration tests locally.

- [Integration Testing Guide](INTEGRATION_TESTING.md) - Setup and run integration tests
- [Integration Troubleshooting](INTEGRATION_TROUBLESHOOTING.md) - Common issues and solutions
- [Detailed Integration Guide](integration/README.md) - Advanced integration testing

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