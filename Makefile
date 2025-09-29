# BSON Library Makefile

.PHONY: help test test-integration test-all build clean docker-up docker-down docker-logs coverage lint fmt vet

# Default target
help:
	@echo "BSON Library - Available Commands:"
	@echo ""
	@echo "Testing:"
	@echo "  test              Run unit tests"
	@echo "  test-integration  Run integration tests (requires Docker)"
	@echo "  test-all          Run all tests (unit + integration)"
	@echo "  coverage          Generate unit test coverage report"
	@echo "  coverage-integration Generate integration test coverage report"
	@echo "  coverage-all      Generate all coverage reports"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up         Start MongoDB container for integration tests"
	@echo "  docker-down       Stop MongoDB container"
	@echo "  docker-logs       Show MongoDB container logs"
	@echo "  docker-clean      Stop containers and remove volumes"
	@echo ""
	@echo "Development:"
	@echo "  build             Build the library"
	@echo "  lint              Run linter"
	@echo "  fmt               Format code"
	@echo "  vet               Run go vet"
	@echo "  clean             Clean build artifacts"
	@echo ""
	@echo "Integration Testing:"
	@echo "  integration-start    Start MongoDB and keep running"
	@echo "  integration-test     Run integration tests"
	@echo "  integration-test-cov Run integration tests with coverage"
	@echo "  integration-stop     Stop MongoDB container"
	@echo "  integration-clean    Clean up integration environment"

# Testing
test:
	@echo "Running unit tests..."
	go test -v ./...

test-integration:
	@echo "Running integration tests..."
	@./scripts/test-integration.sh test

test-all: test test-integration
	@echo "All tests completed!"

coverage:
	@echo "Generating test coverage report..."
	@echo "Running unit tests with coverage for all packages..."
	go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./tests/lucene-mongo/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

coverage-integration:
	@echo "Generating integration test coverage report..."
	@echo "Running integration tests with coverage for all packages..."
	go test -tags=integration -coverprofile=integration_coverage.out -covermode=atomic -coverpkg=./... ./tests/lucene-mongo/...
	go tool cover -html=integration_coverage.out -o integration_coverage.html
	@echo "Integration coverage report generated: integration_coverage.html"

coverage-all: coverage coverage-integration
	@echo "All coverage reports generated!"

# Docker commands
docker-up:
	@echo "Starting MongoDB container..."
	@if command -v docker-compose > /dev/null 2>&1; then \
		docker-compose up -d; \
	elif command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then \
		docker compose up -d; \
	else \
		echo "Error: Neither docker-compose nor 'docker compose' command found"; \
		echo "Please install Docker Desktop or docker-compose"; \
		exit 1; \
	fi
	@echo "MongoDB container started. Access at:"
	@echo "  MongoDB: mongodb://admin:password@localhost:27017/bsonic_test?authSource=admin"
	@echo "  Mongo Express: http://localhost:8081 (admin/admin)"

docker-down:
	@echo "Stopping MongoDB container..."
	@if command -v docker-compose > /dev/null 2>&1; then \
		docker-compose down; \
	elif command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then \
		docker compose down; \
	else \
		echo "Error: Neither docker-compose nor 'docker compose' command found"; \
		exit 1; \
	fi

docker-logs:
	@echo "Showing MongoDB logs..."
	@if command -v docker-compose > /dev/null 2>&1; then \
		docker-compose logs -f mongodb; \
	elif command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then \
		docker compose logs -f mongodb; \
	else \
		echo "Error: Neither docker-compose nor 'docker compose' command found"; \
		exit 1; \
	fi

docker-clean:
	@echo "Cleaning up Docker environment..."
	@if command -v docker-compose > /dev/null 2>&1; then \
		docker-compose down -v; \
	elif command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then \
		docker compose down -v; \
	else \
		echo "Error: Neither docker-compose nor 'docker compose' command found"; \
		exit 1; \
	fi
	@echo "Docker environment cleaned up"

# Integration testing shortcuts
integration-start:
	@./scripts/test-integration.sh start

integration-test:
	@./scripts/test-integration.sh test

integration-test-cov:
	@./scripts/test-integration.sh test-cov

integration-stop:
	@./scripts/test-integration.sh stop

integration-clean:
	@./scripts/test-integration.sh cleanup

# Development
build:
	@echo "Building BSON library..."
	go build ./...

lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping linting"; \
	fi

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -f coverage.out coverage.html integration_coverage.out integration_coverage.html coverage_full.out coverage_full.html integration_coverage_full.out integration_coverage_full.html
	@echo "Build artifacts cleaned"

# CI/CD targets
ci-test: test
	@echo "CI unit tests completed"

ci-test-integration: test-integration
	@echo "CI integration tests completed"

ci-all: ci-test ci-test-integration
	@echo "CI all tests completed"

# Development workflow
dev-setup: docker-up
	@echo "Development environment ready!"
	@echo "Run 'make integration-test' to test your changes"

dev-test: fmt vet test integration-test
	@echo "Development tests completed!"

# Default target
.DEFAULT_GOAL := help