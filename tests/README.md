# Bsonic Test Suite

This directory contains comprehensive tests for Bsonic, organized by language-formatter combinations to support scalable growth of new query languages and output formatters.

## Directory Structure

```
tests/
├── lucene-mongo/           # Lucene language + MongoDB formatter
│   ├── unit_test.go        # Unit tests for lucene-mongo combination
│   ├── integration_test.go # Integration tests for lucene-mongo combination
│   ├── docker-compose.yml  # MongoDB Docker setup for integration tests
│   └── fixtures/           # Test data and fixtures
│       └── 01-seed-data.js # MongoDB seed data for integration tests
├── lucene-elasticsearch/   # Future: Lucene language + Elasticsearch formatter
│   ├── unit_test.go
│   ├── integration_test.go
│   └── docker-compose.yml  # Elasticsearch Docker setup
└── shared/                 # Shared test utilities
    ├── fixtures/           # Shared test data
    └── helpers/            # Common test helper functions
```

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./tests/...

# Run tests for a specific combination
go test ./tests/lucene-mongo/...

# Run with verbose output
go test -v ./tests/lucene-mongo/...
```

### Integration Tests
Integration tests require a running MongoDB instance. Use the provided Docker setup:

```bash
# Start MongoDB container
docker-compose up -d

# Run integration tests
go test -tags=integration ./tests/lucene-mongo/...

# Stop MongoDB container
docker-compose down
```

### Using Make Commands
```bash
# Run all tests (unit + integration)
make test-all

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Start/stop MongoDB for integration tests
make docker-up
make docker-down
```

## Integration Testing Setup

The integration testing setup includes:

- **Docker Compose configuration** for MongoDB with persistent data
- **MongoDB Express** web interface for database inspection
- **Seeded test data** with realistic use cases
- **Comprehensive test suite** covering all library features

### Quick Start

1. **Start MongoDB Container**:
   ```bash
   docker-compose up -d
   ```

2. **Run Integration Tests**:
   ```bash
   go test -tags=integration ./tests/lucene-mongo/...
   ```

3. **Access Database**:
   - **MongoDB**: `mongodb://admin:password@localhost:27017/bsonic_test`
   - **Mongo Express**: http://localhost:8081 (admin/admin)

### Test Data

The integration tests use three collections with realistic data:

- **Users Collection**: 5 users with various roles, nested profile data, and different states
- **Products Collection**: 3 products with different categories, prices, and specifications
- **Orders Collection**: 2 orders with complex nested customer data and payment information

### Test Categories

- **Basic Queries**: Exact field matching, string/number/boolean values
- **Wildcard Queries**: Pattern matching with `*` wildcards and regex
- **Dot Notation Queries**: Nested field access and complex object navigation
- **Array Queries**: Tag-based searches and multi-value field queries
- **Logical Operators**: AND, OR, NOT operations with complex combinations
- **Performance Tests**: Query execution time validation

## Adding New Language-Formatter Combinations

1. Create a new directory: `tests/{language}-{formatter}/`
2. Add the appropriate test files:
   - `unit_test.go` - Test individual functions and methods
   - `integration_test.go` - Test end-to-end functionality
   - `fixtures/` - Test data specific to this combination
3. Use shared utilities from `tests/shared/` when possible
4. Follow the existing test patterns and naming conventions

## Shared Test Utilities

The `tests/shared/` directory contains common utilities:

- `fixtures/` - Shared test data and sample queries
- `helpers/` - Common test helper functions (e.g., BSON comparison utilities)

## Troubleshooting

### Docker Issues
```bash
# Check container status
docker-compose ps

# View logs
docker-compose logs mongodb

# Restart containers
docker-compose restart
```

### Connection Issues
```bash
# Test MongoDB connection
docker exec -it bsonic-mongodb mongosh --eval "db.adminCommand('ping')"

# Check if port is available
netstat -an | grep 27017
```

### Test Failures
```bash
# Run with detailed output
go test -tags=integration -v ./tests/lucene-mongo/... -run TestBasicQueries

# Check test data
docker exec -it bsonic-mongodb mongosh -u admin -p password --authenticationDatabase admin bsonic_test --eval "db.users.countDocuments()"
```

## Best Practices

1. **Test Isolation**: Each test should be independent
2. **Data Cleanup**: Tests should not modify shared data
3. **Performance**: Keep test execution time reasonable
4. **Coverage**: Test both success and failure scenarios
5. **Documentation**: Document complex test scenarios

## Contributing

When adding new features to Bsonic:

1. Add corresponding unit and integration tests
2. Update test data if needed
3. Ensure tests pass with real MongoDB
4. Follow the established test organization patterns