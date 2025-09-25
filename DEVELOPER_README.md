# BSONIC Developer Guide

This document provides a high-level technical overview of the BSONIC library architecture and how it leverages the Participle parsing library.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Core Components](#core-components)
- [Query Processing Flow](#query-processing-flow)
- [Parser Modes](#parser-modes)
- [Value Parsing](#value-parsing)
- [Error Handling](#error-handling)
- [Performance](#performance)

## Architecture Overview

BSONIC uses a two-phase parsing architecture:

1. **Lexical Analysis & Parsing** (handled by Participle)
2. **AST to BSON Conversion** (custom implementation)

### High-Level Flow

```
Input Query String
        ↓
    [Query Type Detection]
        ↓
    ┌─────────────────┬─────────────────┬─────────────────┐
    │   Field Query   │   Text Search   │   Mixed Query   │
    │                 │                 │                 │
    │ [Participle]    │ [Direct BSON]   │ [Split & Parse] │
    │ Lexer → Parser  │ Generation      │ Both Methods    │
    │        ↓        │        ↓        │        ↓        │
    │ [AST Walker]    │ {"$text": {...}}│ [Combine BSON]  │
    │        ↓        │                 │        ↓        │
    └────────┴────────┴─────────────────┴────────┴────────┘
             ↓                                    ↓
    [BSON Document] ←─────────────────────────────┘
             ↓
    [MongoDB Filter]
```

## Core Components

- **Parser**: Main entry point with configurable search modes
- **Participle Grammar**: Defines the Lucene-style query syntax structure
- **Lexer**: Tokenizes input strings into recognized tokens
- **AST Walker**: Converts Participle AST nodes to MongoDB BSON operations
- **Value Parser**: Handles type detection and special syntax (wildcards, ranges, dates)

## Query Processing Flow

### Entry Points

```go
// Package-level convenience function
func Parse(query string) (bson.M, error)

// Instance-based parsing
func (p *Parser) Parse(query string) (bson.M, error)
```

### Query Type Detection

The parser determines query type based on search mode and content:

- **Field Query**: `name:john AND age:25`
- **Text Search**: `software engineer` (with SearchModeText)
- **Mixed Query**: `engineer role:admin AND active:true` (combines both)

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
Input: "software engineer" (with SearchModeText)
→ Output: {"$text": {"$search": "software engineer"}}
```

## Parser Modes

### Search Modes

```go
type SearchMode int

const (
    SearchModeDisabled SearchMode = iota  // Default: field queries only
    SearchModeText                        // Enables text search + field queries
)
```

### Query Type Detection Logic

The parser uses `shouldUseTextSearch()` to determine if a query should use text search:
- Must have text search enabled
- Must not contain field:value pairs (`:`)
- Must not contain logical operators without field pairs

### Mixed Query Handling

Mixed queries combine text search with field searches using `parseMixedQuery()`:
- Splits query into field parts and text parts
- Parses field query normally
- Generates text search BSON
- Combines with `$and` operator

## Value Parsing

### Type Detection Pipeline

The `parseValue()` function handles different value types in order:

1. **Range queries**: `[start TO end]` → `{"$gte": start, "$lte": end}`
2. **Comparison operators**: `>value`, `<value`, `>=value`, `<=value`
3. **Wildcard patterns**: `*pattern*`, `pattern*`, `*pattern` → regex
4. **Date parsing**: ISO dates and time strings
5. **Number parsing**: integers and floats
6. **Boolean parsing**: `true`/`false`
7. **Default**: string values

### Wildcard Pattern Processing

`parseWildcard()` converts wildcards to MongoDB regex:
- `*J*` → contains pattern (no anchoring)
- `*J` → ends with pattern (`$`)
- `J*` → starts with pattern (`^`)
- `J*K` → starts and ends with patterns (`^` and `$`)

## Error Handling

### Validation Errors

`validateFieldQuery()` ensures field queries don't contain standalone text terms when text search is disabled.

### Parsing Errors

Participle provides detailed syntax error messages for malformed queries.

## Performance

### Parser Reuse

```go
// Good: Reuse parser instance
parser := bsonic.New()
for _, query := range queries {
    result, _ := parser.Parse(query)
}
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

## Conclusion

BSONIC leverages Participle for lexical analysis and grammar parsing, while implementing custom logic for MongoDB-specific BSON generation and query optimization. This hybrid approach provides robust parsing while maintaining full control over output format and performance.

The architecture is designed to be extensible, allowing new query features to be added by extending the grammar definition and implementing corresponding BSON conversion logic.