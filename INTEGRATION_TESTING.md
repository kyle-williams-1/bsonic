# Integration Testing Guide

This document explains the integration testing approach for the BSON library.

## Why Integration Testing?

Integration testing against a real MongoDB database validates:
- BSON generation works with the MongoDB Go driver
- Queries execute correctly in a real database environment
- Data type handling with real MongoDB data types
- Performance with realistic data volumes

## Architecture

The integration tests use Docker to run a real MongoDB instance with seeded test data.

## Test Data

The integration tests use three collections:
- **Users**: 5 users with profiles, roles, and metadata
- **Products**: 3 products with specifications and reviews  
- **Orders**: 2 orders with customer and item information

## Test Categories

- **Basic Queries**: Field matching, exact matches, wildcards
- **Dot Notation**: Nested field access (`profile.location`)
- **Array Queries**: Array element matching and tag searches
- **Logical Operators**: AND, OR, NOT operations
- **Performance**: Query execution time validation

## Docker Setup

The integration tests use Docker Compose with:
- **MongoDB 7.0**: Primary database (admin/password)
- **Mongo Express**: Web interface (admin/admin)
- **Database**: bsonic_test
- **Ports**: 27017 (MongoDB), 8081 (Mongo Express)

## Development Workflow

```bash
# Start MongoDB
make docker-up

# Run integration tests
make test-integration

# Inspect database (optional)
# Visit http://localhost:8081 (admin/admin)

# Stop when done
make docker-down
```

## Benefits

Integration testing provides:
- Validation that the library works with real MongoDB
- Confidence in complex query scenarios
- Performance validation with real data
- Easy debugging with real database inspection
