#!/bin/bash

# Database Test Runner for URL Shortener
# This script runs database-dependent tests locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Database configuration
TEST_DB_NAME="${TEST_DB_NAME:-urlshortener_test}"
TEST_DB_USER="${TEST_DB_USER:-postgres}"
TEST_DB_PASSWORD="${TEST_DB_PASSWORD:-postgres}"
TEST_DB_HOST="${TEST_DB_HOST:-localhost}"
TEST_DB_PORT="${TEST_DB_PORT:-5432}"
TEST_DB_SSLMODE="${TEST_DB_SSLMODE:-disable}"
E2E_TEST_DB_NAME="${E2E_TEST_DB_NAME:-urlshortener_e2e_test}"

# Export environment variables
export TEST_DB_NAME TEST_DB_USER TEST_DB_PASSWORD TEST_DB_HOST TEST_DB_PORT TEST_DB_SSLMODE

print_header "Database Test Runner"
print_info "This script runs database-dependent tests for comprehensive validation"
print_info "Test database: $TEST_DB_NAME"
print_info "E2E database: $E2E_TEST_DB_NAME"
print_info "Database host: $TEST_DB_HOST:$TEST_DB_PORT"
echo

# Check prerequisites
print_header "Checking Prerequisites"

if ! command -v go &> /dev/null; then
    print_error "Go is not installed"
    exit 1
fi
print_success "Go is installed ($(go version))"

if ! command -v psql &> /dev/null; then
    print_warning "PostgreSQL client (psql) not found"
    print_info "Database tests may fail if PostgreSQL is not accessible"
else
    print_success "PostgreSQL client is available"
fi

# Test database connection
if command -v psql &> /dev/null; then
    export PGPASSWORD="$TEST_DB_PASSWORD"
    if psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c '\q' &> /dev/null 2>&1; then
        print_success "Database server is accessible"
    else
        print_error "Cannot connect to database server"
        print_info "Please ensure PostgreSQL is running and credentials are correct"
        print_info "Connection string: host=$TEST_DB_HOST port=$TEST_DB_PORT user=$TEST_DB_USER"
        exit 1
    fi
    unset PGPASSWORD
fi

echo

# Setup test databases
print_header "Setting Up Test Databases"

export PGPASSWORD="$TEST_DB_PASSWORD"

# Create main test database
print_info "Creating test database: $TEST_DB_NAME"
psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "CREATE DATABASE $TEST_DB_NAME;" || {
    print_error "Failed to create test database"
    exit 1
}

# Create E2E test database
print_info "Creating E2E test database: $E2E_TEST_DB_NAME"
psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $E2E_TEST_DB_NAME;" 2>/dev/null || true
psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "CREATE DATABASE $E2E_TEST_DB_NAME;" || {
    print_error "Failed to create E2E test database"
    exit 1
}

# Create test table schema
print_info "Setting up test table schema"
for db in "$TEST_DB_NAME" "$E2E_TEST_DB_NAME"; do
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d "$db" << EOF
CREATE TABLE IF NOT EXISTS url_records (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    long_url TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
EOF
    if [ $? -ne 0 ]; then
        print_error "Failed to create schema in $db"
        exit 1
    fi
done

unset PGPASSWORD
print_success "Test databases created and configured"
echo

# Run database tests
print_header "Running Database-Dependent Tests"

print_info "Running repository tests..."
if go test -v ./internal/repo; then
    print_success "Repository tests passed"
else
    print_error "Repository tests failed"
    exit 1
fi

echo

print_info "Running HTTP server tests..."
if go test -v ./internal/http; then
    print_success "HTTP server tests passed"
else
    print_error "HTTP server tests failed"
    exit 1
fi

echo

print_info "Running end-to-end tests..."
export TEST_DB_NAME="$E2E_TEST_DB_NAME"
if go test -v ./e2e_test.go; then
    print_success "End-to-end tests passed"
else
    print_error "End-to-end tests failed"
    exit 1
fi

echo

print_header "Database Tests Completed Successfully"
print_success "All database-dependent tests passed! 🎉"
print_info "These tests complement the core CI pipeline"
print_info "Run this script to verify database functionality"

# Cleanup option
echo
read -p "Clean up test databases? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Cleaning up test databases..."
    export PGPASSWORD="$TEST_DB_PASSWORD"
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $E2E_TEST_DB_NAME;" 2>/dev/null || true
    unset PGPASSWORD
    print_success "Test databases cleaned up"
fi
