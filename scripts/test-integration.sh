#!/bin/bash

# Integration Test Script for BSON Library
# This script manages the MongoDB container and runs integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="tests/lucene-mongo/docker-compose.yml"
TEST_TIMEOUT=${TEST_TIMEOUT:-"30s"}
MONGODB_URI=${MONGODB_URI:-"mongodb://admin:password@localhost:27017/bsonic_test?authSource=admin"}

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Check if docker-compose is available
check_docker_compose() {
    if command -v docker-compose > /dev/null 2>&1; then
        COMPOSE_CMD="docker-compose"
    elif command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    else
        log_error "Neither docker-compose nor 'docker compose' command found."
        log_error "Please install Docker Desktop or docker-compose and try again."
        exit 1
    fi
    log_info "Using Docker Compose command: $COMPOSE_CMD"
}

# Start MongoDB container
start_mongodb() {
    log_info "Starting MongoDB container..."
    
    # Check if containers are already running
    if $COMPOSE_CMD -f $COMPOSE_FILE ps | grep -q "Up"; then
        log_warning "MongoDB container is already running"
        return 0
    fi
    
    # Start containers
    $COMPOSE_CMD -f $COMPOSE_FILE up -d
    
    # Wait for MongoDB to be ready
    log_info "Waiting for MongoDB to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if docker exec bsonic-mongodb mongosh --eval "db.adminCommand('ping')" > /dev/null 2>&1; then
            log_success "MongoDB is ready!"
            return 0
        fi
        
        log_info "Attempt $attempt/$max_attempts - MongoDB not ready yet, waiting..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_error "MongoDB failed to start within expected time"
    return 1
}

# Stop MongoDB container
stop_mongodb() {
    log_info "Stopping MongoDB container..."
    $COMPOSE_CMD -f $COMPOSE_FILE down
    log_success "MongoDB container stopped"
}

# Run integration tests
run_tests() {
    log_info "Running integration tests..."
    
    # Set environment variables
    export MONGODB_URI="$MONGODB_URI"
    
    # Run tests with timeout (macOS compatible)
    if command -v timeout > /dev/null 2>&1; then
        # Use timeout command if available
        if timeout $TEST_TIMEOUT go test -tags=integration -v ./tests/lucene-mongo/...; then
            log_success "All integration tests passed!"
            return 0
        else
            log_error "Integration tests failed!"
            return 1
        fi
    else
        # Fallback for macOS without timeout command
        log_warning "timeout command not found, running tests without timeout"
        if go test -tags=integration -v ./tests/lucene-mongo/...; then
            log_success "All integration tests passed!"
            return 0
        else
            log_error "Integration tests failed!"
            return 1
        fi
    fi
}

# Run tests with coverage
run_tests_with_coverage() {
    log_info "Running integration tests with coverage..."
    
    # Set environment variables
    export MONGODB_URI="$MONGODB_URI"
    
    # Run tests with coverage (macOS compatible)
    if command -v timeout > /dev/null 2>&1; then
        # Use timeout command if available
        if timeout $TEST_TIMEOUT go test -tags=integration -v -coverprofile=integration_coverage.out ./integration/...; then
            log_success "Integration tests with coverage completed!"
            
            # Generate coverage report
            if command -v go > /dev/null 2>&1; then
                go tool cover -html=integration_coverage.out -o integration_coverage.html
                log_info "Coverage report generated: integration_coverage.html"
            fi
            
            return 0
        else
            log_error "Integration tests with coverage failed!"
            return 1
        fi
    else
        # Fallback for macOS without timeout command
        log_warning "timeout command not found, running tests without timeout"
        if go test -tags=integration -v -coverprofile=integration_coverage.out ./integration/...; then
            log_success "Integration tests with coverage completed!"
            
            # Generate coverage report
            if command -v go > /dev/null 2>&1; then
                go tool cover -html=integration_coverage.out -o integration_coverage.html
                log_info "Coverage report generated: integration_coverage.html"
            fi
            
            return 0
        else
            log_error "Integration tests with coverage failed!"
            return 1
        fi
    fi
}

# Show container status
show_status() {
    log_info "Container status:"
    $COMPOSE_CMD -f $COMPOSE_FILE ps
    
    echo ""
    log_info "MongoDB connection info:"
    echo "  URI: $MONGODB_URI"
    echo "  Mongo Express: http://localhost:8081 (admin/admin)"
}

# Show logs
show_logs() {
    log_info "Showing MongoDB logs:"
    $COMPOSE_CMD -f $COMPOSE_FILE logs -f mongodb
}

# Clean up
cleanup() {
    log_info "Cleaning up..."
    $COMPOSE_CMD -f $COMPOSE_FILE down -v
    log_success "Cleanup completed"
}

# Show help
show_help() {
    echo "BSON Integration Test Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start       Start MongoDB container"
    echo "  stop        Stop MongoDB container"
    echo "  test        Run integration tests"
    echo "  test-cov    Run integration tests with coverage"
    echo "  status      Show container status"
    echo "  logs        Show MongoDB logs"
    echo "  cleanup     Stop containers and remove volumes"
    echo "  help        Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  MONGODB_URI    MongoDB connection string (default: mongodb://admin:password@localhost:27017/bsonic_test?authSource=admin)"
    echo "  TEST_TIMEOUT   Test timeout (default: 30s)"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # Start MongoDB container"
    echo "  $0 test                     # Run integration tests"
    echo "  $0 test-cov                 # Run tests with coverage"
    echo "  $0 stop                     # Stop MongoDB container"
    echo "  $0 cleanup                  # Clean up everything"
}

# Main script logic
main() {
    # Check prerequisites
    check_docker
    check_docker_compose
    
    # Parse command
    case "${1:-help}" in
        start)
            start_mongodb
            show_status
            ;;
        stop)
            stop_mongodb
            ;;
        test)
            start_mongodb
            run_tests
            ;;
        test-cov)
            start_mongodb
            run_tests_with_coverage
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
