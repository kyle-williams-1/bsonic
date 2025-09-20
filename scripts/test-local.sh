#!/bin/bash

# Local testing script for bsonic
# This script runs the same checks that GitHub Actions will run

set -e

echo "🧪 Running local tests for bsonic..."

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Please run this script from the project root directory"
    exit 1
fi

echo "📦 Downloading dependencies..."
go mod download

echo "🔍 Verifying dependencies..."
go mod verify

echo "🎨 Checking code formatting..."
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo "❌ Code is not formatted. Please run: gofmt -s -w ."
    gofmt -s -l .
    exit 1
fi
echo "✅ Code is properly formatted"

echo "🔍 Running go vet..."
go vet ./...

echo "🧪 Running tests..."
go test -v ./...

echo "📊 Running tests with coverage..."
go test -v -coverprofile=coverage.out -covermode=atomic .

echo "📈 Coverage report:"
go tool cover -func=coverage.out | grep total

echo "🏗️ Building..."
go build -v ./...

echo "✅ All checks passed! Ready to commit."
