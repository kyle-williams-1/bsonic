# Integration Testing Guide

This document explains the integration testing approach for the BSON library and why it's important for database libraries.

## Why Integration Testing?

Integration testing against a real MongoDB database is crucial for several reasons:

### 1. **Real-World Validation**
- Tests the library against actual MongoDB behavior
- Validates BSON generation works with the MongoDB Go driver
- Ensures queries execute correctly in a real database environment

### 2. **Data Type Validation**
- Tests with real MongoDB data types (ObjectId, Date, etc.)
- Validates nested document queries work correctly
- Ensures array queries function as expected

### 3. **Performance Validation**
- Tests query performance with real data
- Validates index usage and query optimization
- Ensures the library doesn't create inefficient queries

### 4. **Edge Case Discovery**
- Discovers issues that unit tests might miss
- Tests with realistic data volumes
- Validates behavior with complex nested structures

## Architecture Overview

```
┌─────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   Unit Tests    │    │ Integration Tests │    │   Real MongoDB    │
│                 │    │                   │    │                   │
│ - Mock data     │    │ - Real data       │    │ - Seeded data     │
│ - Fast execution│    │ - Docker container│    │ - Real indexes    │
│ - Isolated      │    │ - Real queries    │    │ - Real performance│
└─────────────────┘    └───────────────────┘    └───────────────────┘
```

## Test Data Design

The integration tests use three collections with realistic, diverse data:

### Users Collection
- **Purpose**: Test basic field matching, nested queries, and array operations
- **Data**: 5 users with varying roles, profiles, and metadata
- **Key Features**:
  - String fields (name, email)
  - Boolean fields (active)
  - Nested objects (profile with bio, location, website)
  - Array fields (tags)
  - Date fields (created_at, last_login)

### Products Collection
- **Purpose**: Test e-commerce style queries and complex nested data
- **Data**: 3 products with specifications and reviews
- **Key Features**:
  - Numeric fields (price)
  - Boolean fields (in_stock)
  - Nested objects (specifications)
  - Array fields (tags, reviews)
  - Mixed data types

### Orders Collection
- **Purpose**: Test complex nested queries and real-world data structures
- **Data**: 2 orders with customer and item information
- **Key Features**:
  - Deeply nested objects (customer.address)
  - Array of objects (items)
  - Mixed field types
  - Real-world data relationships

## Test Categories

### 1. Basic Queries
Tests fundamental field matching:
- Exact string matches
- Numeric value matches
- Boolean value matches
- Email and name searches

### 2. Wildcard Queries
Tests pattern matching:
- Prefix wildcards (`name:J*`)
- Suffix wildcards (`email:*example.com`)
- Contains wildcards (`name:*o*`)

### 3. Dot Notation Queries
Tests nested field access:
- Single level nesting (`profile.location`)
- Deep nesting (`customer.address.street`)
- Mixed nesting levels

### 4. Array Queries
Tests array field operations:
- Array element matching
- Tag-based searches
- Multi-value field queries

### 5. Logical Operators
Tests complex query combinations:
- AND operations
- OR operations
- NOT operations
- Mixed operator combinations

### 6. Performance Tests
Tests query efficiency:
- Execution time validation
- Large dataset handling
- Index utilization

## Docker Setup

The integration tests use Docker Compose for consistent, isolated testing:

### Services
- **MongoDB 7.0**: Primary database with authentication
- **Mongo Express**: Web interface for database inspection
- **Persistent Storage**: Data persists between container restarts

### Configuration
- **Authentication**: admin/password
- **Database**: bsonic_test
- **Ports**: 27017 (MongoDB), 8081 (Mongo Express)
- **Health Checks**: Automatic container health monitoring

## Development Workflow

### 1. **Local Development**
```bash
# Start MongoDB for development
make docker-up

# Run integration tests
make test-integration

# Inspect database
# Visit http://localhost:8081 (admin/admin)

# Stop when done
make docker-down
```

### 2. **CI/CD Pipeline**
```bash
# Automated testing in CI
make ci-test-integration
```

### 3. **Debugging**
```bash
# View MongoDB logs
make docker-logs

# Connect to MongoDB directly
docker exec -it bsonic-mongodb mongosh -u admin -p password --authenticationDatabase admin
```

## Best Practices

### 1. **Test Isolation**
- Each test is independent
- Tests don't modify shared data
- Clean state between test runs

### 2. **Realistic Data**
- Use real-world data structures
- Include edge cases and variations
- Test with different data types

### 3. **Performance Monitoring**
- Set reasonable timeout limits
- Monitor query execution times
- Validate index usage

### 4. **Error Handling**
- Test both success and failure scenarios
- Validate error messages
- Test malformed queries

## Industry Standards

This integration testing approach follows industry best practices:

### 1. **Database Library Testing**
- Real database validation
- Comprehensive test coverage
- Performance testing
- Error scenario testing

### 2. **Docker-Based Testing**
- Consistent environments
- Easy setup and teardown
- Isolated test execution
- CI/CD compatibility

### 3. **Test Data Management**
- Realistic, diverse test data
- Seeded data for consistency
- Multiple data scenarios
- Edge case coverage

### 4. **Documentation**
- Clear setup instructions
- Comprehensive test documentation
- Troubleshooting guides
- Best practice examples

## Benefits for Developers

### 1. **Confidence**
- Know the library works with real MongoDB
- Validate complex query scenarios
- Ensure performance is acceptable

### 2. **Development Speed**
- Easy setup with Docker
- Quick feedback on changes
- Comprehensive test coverage

### 3. **Debugging**
- Real database inspection
- Query execution analysis
- Performance monitoring

### 4. **Documentation**
- Live examples with real data
- Clear usage patterns
- Troubleshooting guides

## Conclusion

Integration testing against a real MongoDB database is essential for a BSON library. It provides:

- **Validation** that the library works with real MongoDB
- **Confidence** in complex query scenarios
- **Performance** validation with real data
- **Debugging** capabilities for development
- **Documentation** through live examples

This approach follows industry best practices and provides developers with a robust, reliable testing environment that ensures the BSON library works correctly in real-world scenarios.
