.PHONY: test build run-example clean

# Run tests
test:
	go test -v ./...

# Build the library
build:
	go build ./...

# Run the example
run-example:
	go run examples/main.go

# Clean build artifacts
clean:
	go clean

# Install dependencies
deps:
	go mod tidy

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run all checks
check: fmt test lint
