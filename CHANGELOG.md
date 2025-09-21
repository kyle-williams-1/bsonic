# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
