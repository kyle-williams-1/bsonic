# Dependencies

This document outlines the dependencies required to run the BSON library and its integration tests.

## Required Dependencies

### Go
- **Version**: Go 1.25 or later
- **Purpose**: Core language for the library
- **Installation**: [Download from golang.org](https://golang.org/dl/)

### MongoDB Go Driver
- **Package**: `go.mongodb.org/mongo-driver`
- **Version**: v1.15.0 (defined in go.mod)
- **Purpose**: MongoDB connectivity for integration tests
- **Installation**: Automatically installed with `go mod download`

## Integration Testing Dependencies

### Docker
- **Purpose**: Container runtime for MongoDB
- **Installation**: [Docker Desktop for Mac](https://www.docker.com/products/docker-desktop/)
- **Alternative**: [Docker Engine](https://docs.docker.com/engine/install/)

### Docker Compose
- **Purpose**: Multi-container orchestration
- **Installation**: 
  - **Modern**: Included with Docker Desktop (use `docker compose`)
  - **Legacy**: [Install separately](https://docs.docker.com/compose/install/) (use `docker-compose`)

### MongoDB
- **Version**: 7.0 (via Docker)
- **Purpose**: Real database for integration testing
- **Installation**: Automatically pulled via Docker Compose

### Mongo Express (Optional)
- **Purpose**: Web interface for database inspection
- **Installation**: Automatically included in Docker Compose setup

## Optional Dependencies

### timeout Command
- **Purpose**: Test execution timeout (prevents hanging tests)
- **Installation**:
  - **macOS**: `brew install coreutils` (provides `gtimeout`)
  - **Linux**: Usually pre-installed
  - **Fallback**: Script works without it (no timeout protection)

### golangci-lint (Optional)
- **Purpose**: Code linting and quality checks
- **Installation**: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

## Platform-Specific Notes

### macOS
- **Docker**: Install Docker Desktop from docker.com
- **timeout**: Install via `brew install coreutils` (optional)
- **Docker Compose**: Use `docker compose` (modern) or install separately

### Linux
- **Docker**: Install Docker Engine
- **timeout**: Usually pre-installed
- **Docker Compose**: Install separately or use `docker compose`

### Windows
- **Docker**: Install Docker Desktop
- **timeout**: Not available, script uses fallback
- **Docker Compose**: Included with Docker Desktop

## Quick Setup

### 1. Install Go
```bash
# Download from https://golang.org/dl/
# Or use package manager:
# macOS: brew install go
# Ubuntu: sudo apt install golang-go
```

### 2. Install Docker
```bash
# macOS: Download Docker Desktop from docker.com
# Linux: Follow instructions at https://docs.docker.com/engine/install/
```

### 3. Verify Installation
```bash
# Check Go
go version

# Check Docker
docker --version

# Check Docker Compose
docker compose version
# OR
docker-compose --version
```

### 4. Install Optional Dependencies
```bash
# timeout command (macOS)
brew install coreutils

# golangci-lint (optional)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Troubleshooting

### "timeout: command not found"
- **Cause**: timeout command not installed on macOS
- **Solution**: Install coreutils (`brew install coreutils`) or ignore (script has fallback)

### "docker-compose: command not found"
- **Cause**: Docker Compose not installed or using modern Docker
- **Solution**: Use `docker compose` instead of `docker-compose`

### "Docker is not running"
- **Cause**: Docker daemon not started
- **Solution**: Start Docker Desktop or Docker daemon

### "MongoDB connection failed"
- **Cause**: MongoDB container not running
- **Solution**: Run `make docker-up` to start containers

## Development Workflow

### Without timeout command (macOS default)
```bash
# Start MongoDB
make docker-up

# Run tests (no timeout protection)
make test-integration

# Stop MongoDB
make docker-down
```

### With timeout command (recommended)
```bash
# Install timeout
brew install coreutils

# Start MongoDB
make docker-up

# Run tests (with timeout protection)
make test-integration

# Stop MongoDB
make docker-down
```

## CI/CD Dependencies

The GitHub Actions workflow automatically installs:
- Go (via actions/setup-go)
- Docker (via GitHub-hosted runners)
- MongoDB (via Docker service)

No additional setup required for CI/CD.
