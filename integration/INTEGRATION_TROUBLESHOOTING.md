# Integration Testing Troubleshooting

Common issues and solutions for integration testing.

## Docker Compose Compatibility

**Problem**: Different systems may have different Docker Compose commands available.

**Solution**: The test script automatically detects and uses the available command:
- `docker-compose` (legacy)
- `docker compose` (newer Docker versions)

## macOS timeout Command

**Problem**: `timeout` command not available on macOS by default.

**Solution**: The test script detects `timeout` command availability and works without it, running tests directly if timeout is not available.

## MongoDB Connection Issues

**Problem**: Tests fail with connection errors.

**Solution**: Ensure MongoDB container is running and healthy:
```bash
# Check container status
make docker-logs

# Restart if needed
make docker-down
make docker-up
```

## Running Tests

```bash
# Start MongoDB and run tests
make test-integration

# Or run step by step
make docker-up
make test-integration
make docker-down
```

## Debugging Failed Tests

1. **Check MongoDB logs**: `make docker-logs`
2. **Verify container status**: `docker ps`
3. **Test connection manually**: Use Mongo Express at http://localhost:8081
4. **Run specific test**: `go test -tags=integration -run TestName ./integration/... -v`
