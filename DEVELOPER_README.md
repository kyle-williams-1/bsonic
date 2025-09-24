# BSONIC Developer Guide

This document provides a comprehensive technical overview of the BSONIC library architecture, implementation details, and how it leverages the Participle parsing library.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Call Structure & Order of Operations](#call-structure--order-of-operations)
- [Participle Integration](#participle-integration)
- [Custom Implementation Details](#custom-implementation-details)
- [Parser Modes & Query Types](#parser-modes--query-types)
- [AST to BSON Conversion](#ast-to-bson-conversion)
- [Value Parsing & Type Detection](#value-parsing--type-detection)
- [Error Handling](#error-handling)
- [Performance Considerations](#performance-considerations)

## Architecture Overview

BSONIC is built around a two-phase parsing architecture:

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

### Detailed Parsing Flow

```
"name:john AND age:25"
        ↓
    [Participle Lexer]
    Tokens: [TextTerm:"name", Colon, TextTerm:"john", AND, TextTerm:"age", Colon, TextTerm:"25"]
        ↓
    [Participle Parser]
    AST: ParticipleQuery {
        Expression: {
            Or: [{
                And: [{
                    Not: {
                        Term: {
                            FieldValue: {Field: "name", Value: {TextTerms: ["john"]}}
                        }
                    }
                }, {
                    Not: {
                        Term: {
                            FieldValue: {Field: "age", Value: {TextTerms: ["25"]}}
                        }
                    }
                }]
            }]
        }
    }
        ↓
    [Custom AST Walker]
    BSON: {"name": "john", "age": 25}
```

### Core Components

- **Parser**: Main entry point with configurable search modes
- **Participle Grammar**: Defines the Lucene-style query syntax structure
- **Lexer**: Tokenizes input strings into recognized tokens
- **AST Walker**: Converts Participle AST nodes to MongoDB BSON operations
- **Value Parser**: Handles type detection and special syntax (wildcards, ranges, dates)

## Call Structure & Order of Operations

### 1. Entry Points

```go
// Package-level convenience function
func Parse(query string) (bson.M, error)

// Instance-based parsing
func (p *Parser) Parse(query string) (bson.M, error)
```

### 2. Query Type Detection

The parser first determines the query type based on the search mode and query content:

```go
func (p *Parser) Parse(query string) (bson.M, error) {
    if p.SearchMode == SearchModeText {
        if p.isMixedQuery(query) {
            return p.parseMixedQuery(query)      // Field + text search
        }
        if p.shouldUseTextSearch(query) {
            return p.parseTextSearch(query)      // Pure text search
        }
        return p.parseFieldQuery(query)          // Pure field search
    }
    return p.parseFieldQuery(query)              // Default field search
}
```

### 3. Field Query Processing

For field-based queries, the flow is:

```
Query String
    ↓
[Query Validation] (if text search disabled)
    ↓
[Participle Parser] → ParticipleQuery AST
    ↓
[AST Walker] → BSON Document
```

### 4. Text Search Processing

For text search queries:

```
Query String
    ↓
[Text Search Detection]
    ↓
[Direct BSON Generation] → {"$text": {"$search": "query"}}
```

### 5. Detailed Order of Operations

#### Example 1: Simple Field Query
```
Input: "name:john AND age:25"

1. Parse() entry point
2. SearchMode check (SearchModeDisabled)
3. parseFieldQuery() called
4. validateFieldQuery() - validates no standalone text terms
5. participleParser.ParseString() - Participle parsing
   ├── Lexical analysis: [TextTerm:"name", Colon, TextTerm:"john", AND, TextTerm:"age", Colon, TextTerm:"25"]
   └── Grammar parsing: ParticipleQuery AST
6. participleASTToBSON() - Custom AST walker
   ├── participleExpressionToBSON() - Handle OR operations
   ├── participleAndExpressionToBSON() - Handle AND operations
   │   ├── Field merging optimization
   │   └── Merge {"name": "john"} and {"age": 25}
   └── Return: {"name": "john", "age": 25}
```

#### Example 2: Complex Query with Wildcards
```
Input: "name:jo* AND age:[18 TO 65] OR status:active"

1. Parse() entry point
2. SearchMode check (SearchModeDisabled)
3. parseFieldQuery() called
4. validateFieldQuery() - validates no standalone text terms
5. participleParser.ParseString() - Participle parsing
   ├── Lexical analysis: [TextTerm:"name", Colon, TextTerm:"jo*", AND, TextTerm:"age", Colon, Bracketed:"[18 TO 65]", OR, TextTerm:"status", Colon, TextTerm:"active"]
   └── Grammar parsing: ParticipleQuery AST with OR at root
6. participleASTToBSON() - Custom AST walker
   ├── participleExpressionToBSON() - Detect OR operation
   ├── Left side: participleAndExpressionToBSON()
   │   ├── Parse "name:jo*" → parseValue() → parseWildcard() → {"$regex": "^jo.*", "$options": "i"}
   │   ├── Parse "age:[18 TO 65]" → parseValue() → parseRange() → {"$gte": 18, "$lte": 65}
   │   └── Field merging: {"name": {"$regex": "^jo.*", "$options": "i"}, "age": {"$gte": 18, "$lte": 65}}
   ├── Right side: participleAndExpressionToBSON()
   │   └── Parse "status:active" → {"status": "active"}
   └── Return: {"$or": [{"name": {"$regex": "^jo.*", "$options": "i"}, "age": {"$gte": 18, "$lte": 65}}, {"status": "active"}]}
```

#### Example 3: Text Search Query
```
Input: "software engineer" (with SearchModeText)

1. Parse() entry point
2. SearchMode check (SearchModeText)
3. shouldUseTextSearch() - no colons, no logical operators → true
4. parseTextSearch() called
5. Direct BSON generation
   └── Return: {"$text": {"$search": "software engineer"}}
```

#### Example 4: Mixed Query
```
Input: "engineer role:admin AND active:true" (with SearchModeText)

1. Parse() entry point
2. SearchMode check (SearchModeText)
3. isMixedQuery() - has colons AND text terms → true
4. parseMixedQuery() called
5. Query splitting:
   ├── Field parts: ["role:admin", "AND", "active:true"]
   └── Text parts: ["engineer"]
6. Field query parsing:
   ├── parseFieldQuery("role:admin AND active:true")
   └── Result: {"role": "admin", "active": true}
7. Text search generation:
   └── {"$text": {"$search": "engineer"}}
8. Combine with $and:
   └── Return: {"$and": [{"$text": {"$search": "engineer"}}, {"role": "admin", "active": true}]}
```

## Participle Integration

### Grammar Definition

BSONIC defines a complete grammar hierarchy using Participle struct tags:

```go
// Root query structure
type ParticipleQuery struct {
    Expression *ParticipleExpression `@@`
}

// Expression handles OR operations (lowest precedence)
type ParticipleExpression struct {
    Or []*ParticipleAndExpression `@@ ( "OR" @@ )*`
}

// AndExpression handles AND operations (higher precedence)
type ParticipleAndExpression struct {
    And []*ParticipleNotExpression `@@ ( "AND" @@ )*`
}

// NotExpression handles NOT operations (highest precedence)
type ParticipleNotExpression struct {
    Not  *ParticipleNotExpression `"NOT" @@`
    Term *ParticipleTerm          `| @@`
}

// Term represents individual query elements
type ParticipleTerm struct {
    FieldValue *ParticipleFieldValue `@@`
    Group      *ParticipleGroup      `| @@`
    TextSearch *string               `| @TextTerm`
}
```

### Lexer Configuration

The lexer is configured with specific rules for Lucene-style syntax:

```go
var luceneLexer = lexer.MustSimple([]lexer.SimpleRule{
    {Name: "Whitespace", Pattern: `\s+`},
    {Name: "AND", Pattern: `AND`},
    {Name: "OR", Pattern: `OR`},
    {Name: "NOT", Pattern: `NOT`},
    {Name: "LParen", Pattern: `\(`},
    {Name: "RParen", Pattern: `\)`},
    {Name: "String", Pattern: `"([^"\\]|\\.)*"`},
    {Name: "SingleString", Pattern: `'([^'\\]|\\.)*'`},
    {Name: "Bracketed", Pattern: `\[[^\]]+\]`},
    {Name: "DateTime", Pattern: `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`},
    {Name: "TimeString", Pattern: `\d{2}:\d{2}:\d{2}(\.\d+)?`},
    {Name: "Colon", Pattern: `:`},
    {Name: "TextTerm", Pattern: `[^:\s\[\]()]+`},
})
```

### Parser Configuration

```go
var participleParser = participle.MustBuild[ParticipleQuery](
    participle.Lexer(luceneLexer),
    participle.Unquote("String", "SingleString"),
    participle.UseLookahead(2),
    participle.Elide("Whitespace"),
)
```

## Custom Implementation Details

### What's Custom vs Participle

| Component | Implementation | Details |
|-----------|---------------|---------|
| **Lexical Analysis** | Participle | Token recognition and classification |
| **Grammar Definition** | Participle | AST structure and precedence rules |
| **Parsing Logic** | Participle | String-to-AST conversion |
| **AST to BSON** | Custom | MongoDB-specific BSON generation |
| **Value Parsing** | Custom | Type detection, wildcards, ranges, dates |
| **Query Type Detection** | Custom | Text vs field search logic |
| **Mixed Query Handling** | Custom | Combining text and field searches |
| **BSON Optimization** | Custom | Field merging, operator simplification |

### Implementation Breakdown

```
┌─────────────────────────────────────────────────────────────────┐
│                        BSONIC Architecture                      │
├─────────────────────────────────────────────────────────────────┤
│  Input: "name:john* AND age:[18 TO 65] OR status:active"       │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                    PARTICIPLE LAYER (70%)                       │
├─────────────────────────────────────────────────────────────────┤
│  • Lexical Analysis (Token Recognition)                         │
│  • Grammar Parsing (AST Generation)                             │
│  • Syntax Validation                                            │
│  • Operator Precedence Handling                                 │
│  • Parentheses Grouping                                         │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                    CUSTOM LAYER (30%)                           │
├─────────────────────────────────────────────────────────────────┤
│  • Query Type Detection (Text vs Field vs Mixed)                │
│  • AST to BSON Conversion                                       │
│  • Value Type Detection (String, Number, Date, Boolean)         │
│  • Wildcard Pattern Processing                                  │
│  • Range Query Processing ([start TO end])                      │
│  • Comparison Operator Processing (>, <, >=, <=)                │
│  • Field Merging Optimization                                   │
│  • MongoDB-specific BSON Generation                             │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│  Output: {"$or": [{"name": {"$regex": "^john.*", "$options":    │
│  "i"}, "age": {"$gte": 18, "$lte": 65}}, {"status": "active"}]} │
└─────────────────────────────────────────────────────────────────┘
```

### Code Distribution

```
bsonic.go (736 lines)
├── Participle Grammar Definitions (150 lines, ~20%)
│   ├── ParticipleQuery, ParticipleExpression, etc.
│   └── Lexer configuration
├── Parser Entry Points (50 lines, ~7%)
│   ├── New(), Parse(), SetSearchMode()
│   └── Query type detection
├── AST to BSON Conversion (200 lines, ~27%)
│   ├── participleASTToBSON()
│   ├── participleExpressionToBSON()
│   └── Field merging logic
├── Value Parsing & Type Detection (200 lines, ~27%)
│   ├── parseValue(), parseWildcard()
│   ├── parseRange(), parseDate()
│   └── Type detection logic
├── Text Search Implementation (100 lines, ~14%)
│   ├── parseTextSearch(), parseMixedQuery()
│   └── Query splitting logic
└── Utility Functions (36 lines, ~5%)
    ├── negateBSON(), isSimpleFieldValue()
    └── Validation helpers
```

### AST to BSON Conversion

The conversion process walks the Participle AST and generates MongoDB BSON:

```go
func (p *Parser) participleASTToBSON(query *ParticipleQuery) bson.M {
    if query.Expression == nil {
        return bson.M{}
    }
    return p.participleExpressionToBSON(query.Expression)
}
```

Each AST node type has a corresponding conversion method:

- `participleExpressionToBSON()` - Handles OR operations
- `participleAndExpressionToBSON()` - Handles AND operations with field merging
- `participleNotExpressionToBSON()` - Handles NOT operations using De Morgan's law
- `participleTermToBSON()` - Handles individual terms
- `participleFieldValueToBSON()` - Converts field:value pairs

### Field Merging Optimization

The parser includes intelligent field merging to optimize BSON output:

```go
// Simple field:value pairs are merged directly when possible
if p.isSimpleFieldValue(childBSON) && !hasConflict && !hasComplexExpressions {
    for k, v := range childBSON {
        directFields[k] = v
    }
} else {
    conditions = append(conditions, childBSON)
}
```

This converts:
```go
// Input: "name:john AND age:25"
// Output: {"name": "john", "age": 25}

// Instead of: {"$and": [{"name": "john"}, {"age": 25}]}
```

## Parser Modes & Query Types

### Search Modes

```go
type SearchMode int

const (
    SearchModeDisabled SearchMode = iota  // Default: field queries only
    SearchModeText                        // Enables text search + field queries
)
```

### Query Type Detection Logic

```go
func (p *Parser) shouldUseTextSearch(query string) bool {
    // Must have text search enabled
    if p.SearchMode != SearchModeText {
        return false
    }
    
    // Must not contain field:value pairs
    if strings.Contains(trimmed, ":") {
        return false
    }
    
    // Must not contain logical operators without field pairs
    // (would be a mixed query)
    parts := strings.Fields(trimmed)
    for _, part := range parts {
        if part == "AND" || part == "OR" || part == "NOT" {
            return false
        }
    }
    
    return true
}
```

### Mixed Query Handling

Mixed queries combine text search with field searches:

```go
func (p *Parser) parseMixedQuery(query string) (bson.M, error) {
    // Split query into field parts and text parts
    parts := strings.Fields(trimmed)
    var fieldParts []string
    var textParts []string
    
    for _, part := range parts {
        if strings.Contains(part, ":") || isLogicalOperator(part) {
            fieldParts = append(fieldParts, part)
        } else {
            textParts = append(textParts, part)
        }
    }
    
    // Parse field query and combine with text search
    var conditions []bson.M
    
    if len(fieldParts) > 0 {
        fieldBSON, _ := p.parseFieldQuery(strings.Join(fieldParts, " "))
        conditions = append(conditions, fieldBSON)
    }
    
    if len(textParts) > 0 {
        conditions = append(conditions, bson.M{"$text": bson.M{"$search": strings.Join(textParts, " ")}})
    }
    
    return bson.M{"$and": conditions}, nil
}
```

## Value Parsing & Type Detection

### Type Detection Pipeline

```go
func (p *Parser) parseValue(valueStr string) (interface{}, error) {
    // 1. Range queries: [start TO end]
    if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
        return p.parseRange(valueStr)
    }
    
    // 2. Comparison operators: >value, <value, >=value, <=value
    if strings.HasPrefix(valueStr, ">=") || strings.HasPrefix(valueStr, "<=") {
        return p.parseComparison(valueStr)
    }
    
    // 3. Wildcard patterns: *pattern*, pattern*, *pattern
    if strings.Contains(valueStr, "*") {
        return p.parseWildcard(valueStr)
    }
    
    // 4. Date parsing
    if date, err := p.parseDate(valueStr); err == nil {
        return date, nil
    }
    
    // 5. Number parsing
    if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
        return num, nil
    }
    
    // 6. Boolean parsing
    if valueStr == "true" || valueStr == "false" {
        return valueStr == "true", nil
    }
    
    // 7. Default: string
    return valueStr, nil
}
```

### Wildcard Pattern Processing

```go
func (p *Parser) parseWildcard(valueStr string) (bson.M, error) {
    pattern := strings.ReplaceAll(valueStr, "*", ".*")
    
    // Add proper anchoring based on wildcard position
    if p.isContainsPattern(valueStr) {
        // *J* - contains pattern (no anchoring)
    } else if p.isEndsWithPattern(valueStr) {
        // *J - ends with pattern
        pattern = pattern + "$"
    } else if p.isStartsWithPattern(valueStr) {
        // J* - starts with pattern
        pattern = "^" + pattern
    } else {
        // J*K - starts and ends with specific patterns
        pattern = "^" + pattern + "$"
    }
    
    return bson.M{"$regex": pattern, "$options": "i"}, nil
}
```

### Date Range Processing

```go
func (p *Parser) parseDateRange(startStr, endStr string) (interface{}, error) {
    result := bson.M{}
    
    if startStr == "*" {
        if endStr == "*" {
            return nil, errors.New("invalid date range: both start and end cannot be wildcards")
        }
        endDate, _ := p.parseDate(endStr)
        result["$lte"] = endDate
    } else {
        startDate, _ := p.parseDate(startStr)
        result["$gte"] = startDate
        
        if endStr != "*" {
            endDate, _ := p.parseDate(endStr)
            result["$lte"] = endDate
        }
    }
    
    return result, nil
}
```

## Error Handling

### Validation Errors

```go
func (p *Parser) validateFieldQuery(query string) error {
    if !strings.Contains(trimmed, ":") {
        words := strings.Fields(trimmed)
        for _, word := range words {
            if word != "AND" && word != "OR" && word != "NOT" && word != "(" && word != ")" {
                return fmt.Errorf("text search term '%s' found but text search is disabled", word)
            }
        }
    }
    return nil
}
```

### Parsing Errors

Participle handles syntax errors and provides detailed error messages:

```go
ast, err := participleParser.ParseString("", query)
if err != nil {
    return nil, err  // Participle provides detailed syntax error info
}
```

## Performance Considerations

### Parser Reuse

```go
// Good: Reuse parser instance
parser := bsonic.New()
for _, query := range queries {
    result, _ := parser.Parse(query)
}

// Less efficient: Create new parser each time
for _, query := range queries {
    result, _ := bsonic.Parse(query)  // Creates new parser internally
}
```

### AST Optimization

The parser includes several optimizations:

1. **Field Merging**: Simple field:value pairs are merged directly instead of using `$and`
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

### Debugging

Enable debug logging to trace the parsing process:

```go
// The parser includes detailed error messages from Participle
ast, err := participleParser.ParseString("", query)
if err != nil {
    log.Printf("Parse error: %v", err)
}
```

### Testing

The library includes comprehensive test coverage:

- Unit tests for individual parsing functions
- Integration tests against real MongoDB
- Edge case testing for complex queries
- Performance benchmarks

Run tests with:

```bash
go test ./...
go test -tags=integration ./integration/...
```

## Conclusion

BSONIC leverages Participle for the heavy lifting of lexical analysis and grammar parsing, while implementing custom logic for MongoDB-specific BSON generation and query optimization. This hybrid approach provides the benefits of a robust parsing library while maintaining full control over the output format and performance characteristics.

The architecture is designed to be extensible, allowing new query features to be added by extending the grammar definition and implementing corresponding BSON conversion logic.
