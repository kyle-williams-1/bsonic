# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.8.0-beta.1]

### Added

- **Default fields support** for free text queries without requiring text indexes
- `ParseWithDefaults()` function for specifying default fields per query
- `WithDefaultFields()` config method for parser-level default field configuration
- Wildcard and regex support in default field queries

### Removed

- **MongoDB text search support** - Removed `$text` operator functionality, replaced with default field searches for better Lucene compatibility and performance
- `WithEnableTextSearch()` config method

### Changed

- Free text queries with default fields use case-insensitive regex by default
- ParseWithDefaults takes priority over config-level default fields

## [v0.7.0-beta.1]

### Changed

- **BREAKING: Renamed BSON formatter to MongoDB formatter** (`config.FormatterBSON` → `config.FormatterMongo`)
- Moved formatter from `formatter/bson/` to `formatter/mongo/` for better clarity

### Added

- Enhanced developer documentation with architecture overview
- Reorganized test structure with language-formatter specific test directories

### Migration

Update your code to use the new formatter name:

```go
// Old
cfg := config.Default().WithFormatter(config.FormatterBSON)

// New
cfg := config.Default().WithFormatter(config.FormatterMongo)
```

## [v0.6.0-beta.1]

### Added

- **Regex pattern support** with Lucene-style syntax `field:/regex/`
- **MongoDB regex output** with case-sensitive matching (like Lucene default)
- **Regex with logical operators** for complex query combinations

### Features Implemented

- `name:/john/` - Basic regex patterns
- `email:/.*@example\\.com/` - Complex regex with escaped characters
- `phone:/\\d{3}-\\d{3}-\\d{4}/` - Regex with digit matching
- `status:/^(active|pending|inactive)$/` - Regex with alternation
- `name:/john/ OR email:/.*@example\\.com/` - Regex with logical operators

### Technical Improvements

- Added `Regex` field to `ParticipleValue` struct for regex pattern recognition
- Enhanced lexer with regex token pattern `/([^/\\]|\\.)*/`
- Added `tryParseRegex()` and `parseRegex()` methods to formatter
- Reordered parser priority to check regex before wildcard patterns
- Added comprehensive test coverage for regex functionality

### Documentation

- Updated README with regex examples and improved readability
- Streamlined documentation structure and removed redundancy
- Added regex feature to features list and query syntax sections

## [v0.5.0-beta.1]

### Added

- **Participle integration** for robust parsing and grammar handling
- **Developer documentation** with comprehensive architecture overview

### Changed

- **Refactored parsing engine** to use Participle for lexical analysis and AST generation
- **Improved error handling** with better syntax error messages from Participle

### Technical Improvements

- Migrated from custom lexer to Participle lexer for better token recognition
- Implemented proper grammar hierarchy with Participle struct tags
- Added comprehensive AST to BSON conversion pipeline
- Maintained backward compatibility with existing API

### Documentation

- Added `DEVELOPER_README.md` with detailed technical documentation
- Documented Participle integration patterns and custom implementation details
- Added visual flow diagrams and code distribution breakdown

## [v0.4.0-beta.1]

### Added

- **Number range queries** with Lucene-style syntax
- **Numeric comparison operators** (`>`, `<`, `>=`, `<=`)
- **Range syntax** with `[start TO end]` for numeric fields
- **Wildcard support** in number ranges (`[* TO end]`, `[start TO *]`)
- **Decimal number support** for price ranges and precise numeric queries
- **Complex number queries** combining ranges with logical operators

### Features Implemented

- `age:[18 TO 65]` - Number range queries using `$gte` and `$lte`
- `price:[10.50 TO 99.99]` - Decimal number ranges
- `score:>85` - Greater than comparisons using `$gt`
- `score:<60` - Less than comparisons using `$lt`
- `score:>=90` - Greater than or equal using `$gte`
- `score:<=50` - Less than or equal using `$lte`
- `age:[18 TO *]` - Open-ended ranges with wildcards
- `age:[* TO 65]` - Lower-bound ranges with wildcards
- `age:[18 TO 65] AND status:active` - Number ranges with field conditions
- `age:>18 OR score:<60` - Multiple numeric comparisons with OR
- `price:[0 TO 100] OR rating:[4 TO 5]` - Multiple number ranges with OR

### API Changes

- Enhanced `parseValue()` method to detect and parse number ranges
- Added `isNumberRange()` and `isNumberComparison()` detection methods
- Added `parseNumberRange()` and `parseNumberComparison()` parsing methods
- Improved date detection to distinguish between date and number ranges
- Enhanced type detection for better numeric vs date parsing

### Documentation

- Updated README.md with comprehensive number range examples
- Added number range features to feature list
- Updated integration tests with real MongoDB number range queries
- Enhanced examples with number range demonstrations

### Testing

- Added comprehensive unit tests for number range functionality
- Added integration tests for invalid number query handling
- All existing tests continue to pass (no regressions)

## [v0.3.0-beta.1]

### Added

- **Full-text search support** with MongoDB `$text` operator
- **Configurable search modes** (`SearchModeDisabled`, `SearchModeText`)
- **Mixed queries** combining text search with field searches
- **Text search examples** in documentation
- **MongoDB text index requirements** documentation

### Features Implemented

- `engineer software` - Pure text search using `$text` operator
- `engineer name:john` - Mixed queries combining text and field search
- `software engineer role:admin` - Multiple text terms with field filtering
- `designer role:user AND active:true` - Text search with complex field queries
- `devops role:admin OR name:charlie` - Text search with OR field queries
- `engineer (role:admin AND age:25)` - Text search with grouped field conditions

### API Changes

- Added `NewWithTextSearch()` constructor for text search enabled parser
- Added `SetSearchMode(mode SearchMode)` method for runtime configuration
- Added `SearchMode` enum with `SearchModeDisabled` and `SearchModeText` options
- Made `HandleTextSearchNode()` public for advanced usage

### Documentation

- Updated README with text search examples and configuration
- Added MongoDB text index setup instructions
- Added mixed query usage examples
- Enhanced API documentation with search mode examples

### Technical Improvements

- Refactored query parsing logic for better separation of concerns
- Improved cyclomatic complexity through helper methods and structs
- Enhanced tokenization for mixed query support
- Added comprehensive unit tests for text search functionality
- Improved error messages and validation

## [v0.2.0-beta.1]

### Added

- Initial implementation of Lucene-style syntax parser
- Basic field matching (exact matches and wildcards)
- Support for dot notation for nested fields
- AND operator support
- OR operator support
- NOT operator support
- Complex operator combinations (OR with AND and NOT)
- Parentheses support for query grouping and precedence control
- Quoted string value support
- Comprehensive test suite
- Example usage code
- Makefile for development tasks

### Features Implemented

- `name:john` - Exact field matching
- `name:jo*` - Wildcard matching with regex
- `name:"john doe"` - Quoted string values
- `name:john AND age:25` - AND operator
- `name:john OR name:jane` - OR operator
- `name:john AND NOT age:25` - NOT operator
- `NOT status:inactive` - NOT operator at beginning
- `name:jo* OR name:ja* AND NOT age:18` - Complex combinations
- `(name:john OR name:jane) AND age:25` - Parentheses grouping
- `NOT (name:john OR name:jane)` - NOT with grouped expressions
- `((name:john OR name:jane) AND age:25) OR status:active` - Nested parentheses
- `user.profile.email:john@example.com` - Dot notation for nested fields

### Planned Features

- Array search optimization
- Range queries (age:[18 TO 65])
- Fuzzy search
- Custom field mappings
- Query validation
- Performance optimizations
