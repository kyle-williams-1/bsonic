# BSON Integration Testing

This directory contains integration tests for the BSON library against a real MongoDB database. These tests validate that the library works correctly with actual MongoDB operations and data.

## Overview

The integration testing setup includes:

- **Docker Compose configuration** for MongoDB with persistent data
- **MongoDB Express** web interface for database inspection
- **Seeded test data** with realistic use cases
- **Comprehensive test suite** covering all library features
- **Performance testing** to ensure queries execute efficiently

## Quick Start

### 1. Start the MongoDB Container

```bash
# Start MongoDB and Mongo Express
docker-compose up -d

# Check if containers are running
docker-compose ps

# View logs
docker-compose logs -f mongodb
```

### 2. Run Integration Tests

```bash
# Run all integration tests
go test -tags=integration ./integration/...

# Run with verbose output
go test -tags=integration -v ./integration/...

# Run specific test
go test -tags=integration -v ./integration/... -run TestBasicQueries
```

### 3. Access Database

- **MongoDB**: `mongodb://admin:password@localhost:27017/bsonic_test`
- **Mongo Express**: http://localhost:8081 (admin/admin)

## Test Data

The integration tests use three collections with realistic data:

### Users Collection
- 5 users with various roles (admin, user, moderator)
- Nested profile data with bio, location, website
- Array fields for tags
- Different active/inactive states
- Various creation and login dates

### Products Collection
- 3 products with different categories
- Price, stock status, and specifications
- Array fields for tags and reviews
- Nested specification objects

### Orders Collection
- 2 orders with complex nested customer data
- Array of order items
- Different payment methods and statuses
- Nested address information

## Test Categories

### Basic Queries
- Exact field matching
- String, number, and boolean values
- Email and name searches

### Wildcard Queries
- Pattern matching with `*` wildcards
- Case-insensitive regex searches
- Partial string matching

### Dot Notation Queries
- Nested field access
- Profile data queries
- Complex object navigation

### Array Queries
- Tag-based searches
- Array element matching
- Multi-value field queries

### Logical Operators
- AND operations
- OR operations
- NOT operations
- Complex combinations

### Performance Tests
- Query execution time validation
- Large dataset handling
- Index utilization

## Development Workflow

### For Development
```bash
# Start containers and keep them running
docker-compose up -d

# Run tests during development
go test -tags=integration ./integration/...

# Stop containers when done
docker-compose down
```

### For CI/CD
```bash
# Run tests with fresh containers
docker-compose up -d --build
go test -tags=integration ./integration/...
docker-compose down
```

### Database Inspection
```bash
# Connect to MongoDB directly
docker exec -it bsonic-mongodb mongosh -u admin -p password --authenticationDatabase admin

# Use the bsonic_test database
use bsonic_test

# Query the data
db.users.find().pretty()
db.products.find().pretty()
db.orders.find().pretty()
```

## Environment Variables

- `MONGODB_URI`: Override default MongoDB connection string
- `TEST_TIMEOUT`: Set custom test timeout (default: 10s)

## Adding New Tests

1. Add test data to `init/01-seed-data.js`
2. Create test functions in `integration_test.go`
3. Follow the naming convention: `Test<FeatureName>`
4. Use the existing test structure with `tests` slice
5. Include both positive and negative test cases

## Troubleshooting

### Container Issues
```bash
# Check container status
docker-compose ps

# View logs
docker-compose logs mongodb

# Restart containers
docker-compose restart

# Rebuild and restart
docker-compose up -d --build
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
go test -tags=integration -v ./integration/... -run TestBasicQueries

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

When adding new features to the BSON library:

1. Add corresponding integration tests
2. Update test data if needed
3. Ensure tests pass with real MongoDB
4. Update this documentation
