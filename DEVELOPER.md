# Developer Guide

This guide is for developers working on the bsonic library itself.

## Architecture Overview

Bsonic uses a modular architecture with clear separation of concerns:

```
bsonic/
├── bsonic.go              # Public API and main parser
├── bson_driver.go         # BSON conversion logic
├── query_preprocessor.go  # Query preprocessing
├── bsonic_test.go         # Core unit tests
├── bson_driver_test.go    # BSON driver tests
├── query_preprocessor_test.go # Preprocessor tests
├── error_handling_test.go # Error condition tests
└── integration/           # Integration tests
```

## Key Components

### 1. Parser (`bsonic.go`)
- **Purpose**: Public API and main entry point
- **Responsibilities**: 
  - Query preprocessing
  - Text search detection
  - Error handling
  - Public interface

### 2. BSON Driver (`bson_driver.go`)
- **Purpose**: Converts go-lucene AST to BSON
- **Responsibilities**:
  - AST traversal and conversion
  - BSON structure generation
  - Value type parsing
  - Complex query handling (NOT, OR, AND)

### 3. Query Preprocessor (`query_preprocessor.go`)
- **Purpose**: Fixes common parsing issues before go-lucene parsing
- **Responsibilities**:
  - Email address quoting
  - Dot notation field handling
  - Parentheses and quoted value processing
  - Mixed query detection

## External Dependencies

### go-lucene Library
- **Package**: `github.com/grindlemire/go-lucene`
- **Version**: v0.0.21
- **Purpose**: Lucene query parsing
- **Why**: Battle-tested, comprehensive Lucene syntax support

### MongoDB Go Driver
- **Package**: `go.mongodb.org/mongo-driver`
- **Version**: v1.17.4
- **Purpose**: BSON type definitions and MongoDB compatibility

## Development Workflow

### Running Tests
```bash
# Unit tests
go test ./...

# Unit tests with coverage
go test -cover ./...

# Integration tests (requires MongoDB)
go test -tags=integration ./integration/...

# All tests
make test
```

### Code Coverage
Current coverage: **74.9%**

**Well covered (90%+):**
- Basic field queries
- Logical operators
- Wildcard queries
- Comparison operators
- Text search functionality

**Needs improvement (50-89%):**
- Complex BSON merging logic
- Edge cases in preprocessing

**Low coverage (0-49%):**
- Range query parsing (complex, may need integration tests)
- Some helper functions

### Adding New Features

1. **Add tests first** (TDD approach)
2. **Update documentation** (README, CHANGELOG)
3. **Add integration tests** for complex features
4. **Update examples** if applicable

### Code Style

- Follow Go conventions
- Add comprehensive comments for public APIs
- Use descriptive variable and function names
- Keep functions focused and small
- Add tests for all public functions

## Testing Strategy

### Unit Tests
- **Location**: `*_test.go` files
- **Coverage**: Individual functions and methods
- **Mocking**: Use interfaces where appropriate

### Integration Tests
- **Location**: `integration/integration_test.go`
- **Coverage**: End-to-end functionality with real MongoDB
- **Setup**: Requires Docker and MongoDB instance

### Test Organization
- **Core tests**: `bsonic_test.go` - Public API
- **Driver tests**: `bson_driver_test.go` - BSON conversion
- **Preprocessor tests**: `query_preprocessor_test.go` - Query preprocessing
- **Error tests**: `error_handling_test.go` - Error conditions

## Performance Considerations

### Parsing Performance
- Uses external `go-lucene` library (optimized)
- Query preprocessing is lightweight
- BSON conversion is efficient

### Memory Usage
- Minimal allocations in hot paths
- Reuses parser instances when possible
- Efficient BSON structure generation

### Benchmarking
```bash
# Run benchmarks
go test -bench=.

# Profile CPU usage
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

## Error Handling

### Error Types
1. **Parsing errors**: Invalid query syntax
2. **Validation errors**: Text search when disabled
3. **Type errors**: Invalid value types

### Error Messages
- Clear and descriptive
- Include context about what went wrong
- Suggest fixes when possible

## Debugging

### Common Issues
1. **Query preprocessing**: Check `query_preprocessor.go`
2. **BSON conversion**: Check `bson_driver.go`
3. **Parsing errors**: Check go-lucene library compatibility

### Debug Tools
```bash
# Verbose test output
go test -v ./...

# Race condition detection
go test -race ./...

# Memory profiling
go test -memprofile=mem.prof ./...
```

## Contributing

1. **Fork the repository**
2. **Create a feature branch**
3. **Add tests for new functionality**
4. **Update documentation**
5. **Run all tests**
6. **Submit a pull request**

### Pull Request Checklist
- [ ] Tests pass
- [ ] Code coverage maintained or improved
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] No breaking changes (or clearly documented)

## Release Process

1. **Update version** in relevant files
2. **Update CHANGELOG.md** with new features
3. **Run full test suite**
4. **Create release tag**
5. **Publish to GitHub**

## Architecture Decisions

### Why go-lucene Library?
- **Battle-tested**: Mature, community-supported
- **Comprehensive**: Full Lucene syntax support
- **Maintainable**: Reduces custom parsing code
- **Performance**: Optimized parsing algorithms

### Why Modular Architecture?
- **Testability**: Each component can be tested independently
- **Maintainability**: Clear separation of concerns
- **Extensibility**: Easy to add new features
- **Debugging**: Easier to isolate issues

### Why Query Preprocessing?
- **Compatibility**: Fixes common parsing issues
- **User Experience**: Handles edge cases gracefully
- **Robustness**: Makes parsing more reliable
