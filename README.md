# Bsonic

[![CI](https://github.com/kyle-williams-1/bsonic/actions/workflows/ci.yml/badge.svg)](https://github.com/kyle-williams-1/bsonic/actions/workflows/ci.yml)
[![Test](https://github.com/kyle-williams-1/bsonic/actions/workflows/test.yml/badge.svg)](https://github.com/kyle-williams-1/bsonic/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyle-williams-1/bsonic)](https://goreportcard.com/report/github.com/kyle-williams-1/bsonic)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Go library that provides Lucene-style syntax for MongoDB BSON filters. Convert human-readable query strings into MongoDB BSON documents that work seamlessly with the official MongoDB Go driver.

## Features

- **Lucene-style syntax**: Write queries in familiar Lucene format
- **Field matching**: Support for exact matches and wildcard patterns
- **Dot notation**: Query nested data structures using dot notation
- **Array search**: Search within array fields
- **Logical operators**: Support for AND, OR, and NOT operations
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

## Usage Examples

### Basic Field Matching

```go
parser := bsonic.New()

// Exact match
query, _ := parser.Parse("name:john")
// Generates: bson.M{"name": "john"}

// Wildcard search
query, _ := parser.Parse("name:jo*")
// Generates: bson.M{"name": bson.M{"$regex": "jo.*", "$options": "i"}}
```

### Nested Fields (Dot Notation)

```go
// Query nested fields
query, _ := parser.Parse("user.profile.email:john@example.com")
// Generates: bson.M{"user.profile.email": "john@example.com"}
```

### Array Search

```go
// Search in arrays
query, _ := parser.Parse("tags:mongodb")
// Generates: bson.M{"tags": "mongodb"} (uses $in for array fields)
```

### Logical Operators

```go
// AND operations
query, _ := parser.Parse("name:john AND age:25")
// Generates: bson.M{"name": "john", "age": 25}

// OR operations (coming soon)
query, _ := parser.Parse("name:john OR name:jane")
// Will generate: bson.M{"$or": [{"name": "john"}, {"name": "jane"}]}

// NOT operations (coming soon)
query, _ := parser.Parse("name:john AND NOT age:25")
// Will generate: bson.M{"name": "john", "age": bson.M{"$ne": 25}}
```

## Development

### Project Structure

```
bsonic/
├── bsonic.go          # Main library implementation
├── bsonic_test.go     # Unit tests
├── go.mod             # Go module definition
├── go.sum             # Go module checksums
├── README.md          # This file
├── CHANGELOG.md       # Version history
├── LICENSE            # Apache 2.0 license
├── Makefile           # Build and test commands
├── docker-compose.yml # MongoDB integration testing
├── examples/          # Usage examples
│   └── main.go
├── integration/       # Integration tests
│   ├── integration_test.go
│   ├── README.md
│   └── init/
│       └── 01-seed-data.js
└── scripts/           # Helper scripts
    └── test-integration.sh
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

### Testing

#### Unit Tests
```bash
go test ./...
```

#### Integration Tests
Integration tests run against a real MongoDB database using Docker:

```bash
# Start MongoDB container
make docker-up

# Run integration tests
make test-integration

# Stop MongoDB container
make docker-down
```

For more detailed integration testing options, see the [Integration Testing Guide](integration/README.md).

#### Dependencies
See [DEPENDENCIES.md](DEPENDENCIES.md) for a complete list of required and optional dependencies.

### Roadmap

- [x] Basic field matching
- [x] Wildcard support
- [x] Dot notation for nested fields
- [x] OR operator support
- [x] NOT operator support
- [x] Complex operator combinations (OR with AND and NOT)
- [ ] Array search optimization
- [ ] Range queries (age:[18 TO 65])
- [ ] Fuzzy search
- [ ] Custom field mappings
- [ ] Query validation
- [ ] Performance optimizations

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by Lucene query syntax and designed for seamless integration with MongoDB's Go driver.
