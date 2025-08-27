# URL Shortener - Test Suite Documentation

This document provides comprehensive information about the test suite for the URL Shortener project.

## Overview

The test suite includes:
- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test components working together with real database
- **HTTP Tests**: Test the REST API endpoints
- **Benchmark Tests**: Performance testing
- **Race Detection**: Concurrency safety testing
- **Coverage Analysis**: Code coverage reporting

## Test Structure

```
urlshortener/
├── internal/
│   ├── config/
│   │   └── config_test.go          # Configuration loading tests
│   ├── handler/
│   │   └── shorten_test.go         # HTTP handler tests
│   ├── http/
│   │   └── server_test.go          # Server integration tests
│   ├── repo/
│   │   └── urlrepo_test.go         # Database repository tests
│   ├── service/
│   │   └── shortener_test.go       # Business logic tests
│   └── util/
│       └── codegen_test.go         # Utility function tests
├── test.sh                         # Test runner script
└── TEST_README.md                  # This file
```

## Running Tests

### Prerequisites

1. **Go 1.23+**: Ensure Go is installed and available in your PATH
2. **PostgreSQL**: Database server running for integration tests
3. **Environment Variables**: Configure test database connection

### Environment Variables

Set these environment variables for database tests:

```bash
export TEST_DB_NAME="urlshortener_test"
export TEST_DB_USER="postgres"
export TEST_DB_PASSWORD="postgres"
export TEST_DB_HOST="localhost"
export TEST_DB_PORT="5432"
export TEST_DB_SSLMODE="disable"
```

### Using the Test Runner Script

The `test.sh` script provides a comprehensive way to run all tests:

```bash
# Run all tests (default)
./test.sh

# Run specific test types
./test.sh --unit              # Unit tests only
./test.sh --integration       # Integration tests only
./test.sh --bench             # Benchmarks only
./test.sh --race              # Race detection tests
./test.sh --coverage          # Coverage analysis

# Utility commands
./test.sh --setup-db          # Setup test database only
./test.sh --cleanup           # Cleanup test artifacts
./test.sh --skip-db-setup     # Skip database setup
./test.sh --help              # Show help
```

### Manual Test Execution

You can also run tests manually using Go's test command:

```bash
# Run all tests
go test ./internal/...

# Run tests for specific package
go test ./internal/service
go test ./internal/handler

# Run with verbose output
go test -v ./internal/...

# Run benchmarks
go test -bench=. ./internal/...

# Run with race detection
go test -race ./internal/...

# Generate coverage report
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html
```

## Test Categories

### 1. Unit Tests

**Location**: Most `*_test.go` files
**Purpose**: Test individual functions and methods in isolation
**Dependencies**: Minimal (use mocks where needed)

#### Code Generation Tests (`internal/util/codegen_test.go`)
- Tests the random code generation function
- Verifies code length and character set
- Checks for reasonable uniqueness
- Performance benchmarks

#### Configuration Tests (`internal/config/config_test.go`)
- Tests environment variable loading
- Validates configuration parsing
- Tests URL normalization (trailing slash)
- Tests DSN and bind address generation

#### Service Tests (`internal/service/shortener_test.go`)
- Tests business logic with mock repository
- Code collision handling
- URL deduplication logic
- Error scenarios and retries
- Mock-based testing patterns

#### Handler Tests (`internal/handler/shorten_test.go`)
- HTTP request/response handling
- Input validation
- Error response formatting
- URL parsing and validation
- JSON marshaling/unmarshaling

### 2. Integration Tests

**Location**: `internal/repo/urlrepo_test.go`, `internal/http/server_test.go`
**Purpose**: Test components with real dependencies
**Dependencies**: Requires PostgreSQL database

#### Repository Tests (`internal/repo/urlrepo_test.go`)
- Database CRUD operations
- Constraint handling (unique keys)
- Transaction behavior
- Connection error handling
- Performance benchmarks with real DB

#### HTTP Server Tests (`internal/http/server_test.go`)
- Full HTTP request/response cycle
- End-to-end URL shortening flow
- Concurrent request handling
- Database persistence verification
- Real database interactions

### 3. Performance Tests

**Benchmarks**: All test files include benchmark functions
**Naming**: Functions starting with `Benchmark`

#### What's Benchmarked
- Code generation performance
- Database operations (insert, select)
- HTTP request handling
- Configuration loading
- Service layer operations

#### Running Benchmarks
```bash
# All benchmarks
go test -bench=. ./internal/...

# Specific package benchmarks
go test -bench=. ./internal/util

# With memory allocation stats
go test -bench=. -benchmem ./internal/...
```

## Test Database Setup

### Automatic Setup
The `test.sh` script automatically:
1. Creates the test database if it doesn't exist
2. Creates the required table schema
3. Handles connection errors gracefully

### Manual Setup
```sql
-- Create test database
CREATE DATABASE urlshortener_test;

-- Connect to test database
\c urlshortener_test;

-- Create table
CREATE TABLE url_records (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    long_url TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Database Cleanup
Tests automatically clean up after themselves:
- Each test starts with `DELETE FROM url_records`
- No persistent test data between runs

## Mock Testing Patterns

### Repository Mocking
The service tests use a mock repository pattern:

```go
type mockURLRepo struct {
    urls           map[string]model.URLRecord
    codes          map[string]model.URLRecord
    insertError    error
    getByLongError error
    getByCodeError error
}
```

Benefits:
- Tests run without database dependencies
- Controlled error scenarios
- Fast execution
- Predictable behavior

### Service Mocking
HTTP handler tests mock the service layer:

```go
type mockShortener struct {
    shortenFunc func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error)
    resolveFunc func(ctx context.Context, code string) (string, error)
}
```

## Coverage Goals

### Current Coverage Targets
- **Overall**: >80% line coverage
- **Critical paths**: >90% coverage
- **Error handling**: >85% coverage

### Viewing Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./internal/...

# View in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

## Common Test Scenarios

### URL Validation Tests
- Valid HTTP/HTTPS URLs
- Invalid URLs (malformed, unsupported protocols)
- Edge cases (empty strings, very long URLs)
- URL normalization

### Code Generation Tests
- Uniqueness verification
- Character set validation
- Length requirements
- Performance characteristics

### Database Constraint Tests
- Unique code violations
- Unique URL violations
- Concurrent insertion races
- Transaction isolation

### Error Handling Tests
- Database connection failures
- Invalid input handling
- Service layer errors
- HTTP error responses

## Continuous Integration

### GitHub Actions Integration
The test suite is designed to work with CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run Tests
  run: |
    export TEST_DB_NAME=urlshortener_test
    export TEST_DB_USER=postgres
    export TEST_DB_PASSWORD=postgres
    ./test.sh
  env:
    PGPASSWORD: postgres
```

### Docker Testing
Tests can run in Docker containers:

```dockerfile
# Test stage in Dockerfile
FROM golang:1.23 AS test
WORKDIR /app
COPY . .
RUN go mod download
RUN ./test.sh --skip-db-setup  # DB runs in separate container
```

## Troubleshooting

### Common Issues

#### Database Connection Errors
```
Error: failed to ping database
```
**Solution**: Verify PostgreSQL is running and credentials are correct

#### Permission Errors
```
Error: permission denied for database creation
```
**Solution**: Ensure test user has CREATE DATABASE privileges

#### Test Timeouts
```
Error: test timeout exceeded
```
**Solution**: Check database performance or increase test timeout

#### Import Cycle Errors
```
Error: import cycle not allowed
```
**Solution**: Verify test packages don't create circular dependencies

### Debugging Tests

#### Verbose Output
```bash
go test -v ./internal/service
```

#### Specific Test Function
```bash
go test -v -run TestShortener_Shorten_NewURL ./internal/service
```

#### Debug with Delve
```bash
dlv test ./internal/service -- -test.run TestShortener_Shorten_NewURL
```
