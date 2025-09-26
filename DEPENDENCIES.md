# Dependencies

Required dependencies for the BSON library and integration tests.

## Required Dependencies

### Go
- **Version**: Go 1.25 or later
- **Installation**: [Download from golang.org](https://golang.org/dl/)

### MongoDB Go Driver
- **Package**: `go.mongodb.org/mongo-driver`
- **Version**: v1.17.4 (defined in go.mod)
- **Installation**: Automatically installed with `go mod download`

## Integration Testing Dependencies

### Docker
- **Installation**: [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### Docker Compose
- **Modern**: Included with Docker Desktop (use `docker compose`)
- **Legacy**: [Install separately](https://docs.docker.com/compose/install/) (use `docker-compose`)

### MongoDB
- **Version**: 7.0 (via Docker)
- **Installation**: Automatically pulled via Docker Compose

### Mongo Express (Optional)
- **Purpose**: Web interface for database inspection
- **Installation**: Automatically included in Docker Compose setup

## Optional Dependencies

### timeout Command
- **macOS**: `brew install coreutils`
- **Linux**: Usually pre-installed
- **Fallback**: Script works without it

### golangci-lint (Development)
- **Purpose**: Code linting and quality checks
- **Installation**: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- **Usage**: `make lint` (automatically runs if available)

## Quick Setup

1. **Install Go**: Download from [golang.org](https://golang.org/dl/)
2. **Install golangci-lint**: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
3. **Install Docker**: Download [Docker Desktop](https://www.docker.com/products/docker-desktop/)
4. **Verify Installation**:
   ```bash
   go version
   golangci-lint --version
   docker --version
   docker compose version
   ```

## Troubleshooting

- **"timeout: command not found"**: Install coreutils (`brew install coreutils`) or ignore
- **"docker-compose: command not found"**: Use `docker compose` instead
- **"Docker is not running"**: Start Docker Desktop
- **"MongoDB connection failed"**: Run `make docker-up`
