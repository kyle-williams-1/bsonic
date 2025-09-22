# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
