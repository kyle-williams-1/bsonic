# Bsonic Test Organization

This directory contains tests organized by language-formatter combinations to support scalable growth of new query languages and output formatters.

## Directory Structure

```
tests/
├── lucene-mongo/           # Lucene language + MongoDB formatter
│   ├── unit_test.go        # Unit tests for lucene-mongo combination
│   ├── integration_test.go # Integration tests for lucene-mongo combination
│   └── benchmarks_test.go  # Performance tests (future)
├── lucene-elasticsearch/   # Future: Lucene language + Elasticsearch formatter
│   ├── unit_test.go
│   └── integration_test.go
├── sql-mongo/              # Future: SQL language + MongoDB formatter
│   ├── unit_test.go
│   └── integration_test.go
└── shared/                 # Shared test utilities
    ├── fixtures/           # Test data and fixtures
    └── helpers/            # Common test helper functions
```

## Test Naming Convention

Tests are organized by the specific language-formatter combination they test:

- `{language}-{formatter}/unit_test.go` - Unit tests for the specific combination
- `{language}-{formatter}/integration_test.go` - Integration tests for the specific combination
- `{language}-{formatter}/benchmarks_test.go` - Performance tests for the specific combination

## Adding New Language-Formatter Combinations

1. Create a new directory: `tests/{language}-{formatter}/`
2. Add the appropriate test files:
   - `unit_test.go` - Test individual functions and methods
   - `integration_test.go` - Test end-to-end functionality
   - `benchmarks_test.go` - Test performance characteristics
3. Use shared utilities from `tests/shared/` when possible

## Shared Test Utilities

The `tests/shared/` directory contains common utilities that can be used across different language-formatter combinations:

- `fixtures/` - Test data, sample queries, and expected outputs
- `helpers/` - Common test helper functions and utilities

## Running Tests

Run all tests:
```bash
go test ./tests/...
```

Run tests for a specific combination:
```bash
go test ./tests/lucene-mongo/...
```

Run only unit tests:
```bash
go test ./tests/... -run TestUnit
```

Run only integration tests:
```bash
go test ./tests/... -run TestIntegration
```
