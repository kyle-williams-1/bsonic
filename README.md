# Bsonic

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
├── parser.go          # Query parsing logic
├── operators.go       # Logical operators (AND, OR, NOT)
├── types.go          # Type definitions and utilities
├── go.mod            # Go module definition
├── README.md         # This file
└── examples/         # Usage examples
    └── main.go
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

### Testing

```bash
go test ./...
```

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
