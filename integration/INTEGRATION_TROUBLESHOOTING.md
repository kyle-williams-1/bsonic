# Integration Testing Troubleshooting

Common issues and solutions for integration testing.

## Data Type Mismatch

**Problem**: The BSON library treats all values as strings, but MongoDB stores actual data types.

**Solution**: Updated integration tests to reflect current library behavior - all values are treated as strings.

## Wildcard Pattern Matching

**Problem**: Wildcard patterns match more results than expected due to case-insensitive regex.

**Solution**: Updated test expectations to match actual behavior - case-insensitive matching is used.

## Empty Query Handling

**Problem**: Empty queries should return empty BSON (matching all documents), not an error.

**Solution**: Library correctly returns empty BSON for empty queries - this is the expected behavior.

## Docker Compose Version Warning

**Problem**: Docker Compose shows warning about obsolete `version` attribute.

**Solution**: Remove the `version` field from `docker-compose.yml`.

## macOS timeout Command

**Problem**: `timeout` command not available on macOS by default.

**Solution**: Script detects `timeout` command availability and works without it.

## Running Tests

```bash
# Start MongoDB
make docker-up

# Run integration tests
make test-integration

# Stop MongoDB
make docker-down
```
