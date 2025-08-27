#!/bin/bash

# URL Shortener Test Runner
# This script runs all tests for the URL shortener project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_DB_NAME="${TEST_DB_NAME:-urlshortener_test}"
TEST_DB_USER="${TEST_DB_USER:-test_user}"
TEST_DB_PASSWORD="${TEST_DB_PASSWORD:-test_password}"
TEST_DB_HOST="${TEST_DB_HOST:-localhost}"
TEST_DB_PORT="${TEST_DB_PORT:-5432}"
TEST_DB_SSLMODE="${TEST_DB_SSLMODE:-disable}"

# Export test database environment variables
export TEST_DB_NAME
export TEST_DB_USER
export TEST_DB_PASSWORD
export TEST_DB_HOST
export TEST_DB_PORT
export TEST_DB_SSLMODE

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

check_prerequisites() {
    print_header "Checking Prerequisites"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    print_success "Go is installed ($(go version))"

    # Check if PostgreSQL client is available
    if ! command -v psql &> /dev/null; then
        print_warning "PostgreSQL client (psql) not found - database tests may fail"
    else
        print_success "PostgreSQL client is available"
    fi

    # Check if test database is accessible
    if command -v psql &> /dev/null; then
        export PGPASSWORD="$TEST_DB_PASSWORD"
        if psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c '\q' &> /dev/null; then
            print_success "Database server is accessible"
        else
            print_warning "Cannot connect to database server - integration tests may fail"
            print_info "Make sure PostgreSQL is running and credentials are correct"
        fi
        unset PGPASSWORD
    fi

    echo
}

setup_test_database() {
    print_header "Setting Up Test Database"

    if ! command -v psql &> /dev/null; then
        print_warning "PostgreSQL client not available - skipping database setup"
        return
    fi

    export PGPASSWORD="$TEST_DB_PASSWORD"

    # Create test database if it doesn't exist
    if ! psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "SELECT 1 FROM pg_database WHERE datname='$TEST_DB_NAME'" | grep -q 1; then
        print_info "Creating test database: $TEST_DB_NAME"
        psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d postgres -c "CREATE DATABASE $TEST_DB_NAME;" || {
            print_error "Failed to create test database"
            unset PGPASSWORD
            return 1
        }
        print_success "Test database created"
    else
        print_success "Test database already exists"
    fi

    # Create test table
    print_info "Setting up test table"
    psql -h "$TEST_DB_HOST" -p "$TEST_DB_PORT" -U "$TEST_DB_USER" -d "$TEST_DB_NAME" << EOF
CREATE TABLE IF NOT EXISTS url_records (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    long_url TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
EOF

    if [ $? -eq 0 ]; then
        print_success "Test table setup completed"
    else
        print_error "Failed to setup test table"
        unset PGPASSWORD
        return 1
    fi

    unset PGPASSWORD
    echo
}

run_unit_tests() {
    print_header "Running Unit Tests"

    # Run tests for each package
    packages=(
        "./internal/util"
        "./internal/config"
        "./internal/service"
        "./internal/handler"
    )

    for package in "${packages[@]}"; do
        print_info "Testing package: $package"
        if go test -v "$package"; then
            print_success "Unit tests passed for $package"
        else
            print_error "Unit tests failed for $package"
            return 1
        fi
        echo
    done
}

run_integration_tests() {
    print_header "Running Integration Tests"

    # Integration test packages
    packages=(
        "./internal/repo"
        "./internal/http"
    )

    for package in "${packages[@]}"; do
        print_info "Testing package: $package"
        if go test -v "$package"; then
            print_success "Integration tests passed for $package"
        else
            print_error "Integration tests failed for $package"
            return 1
        fi
        echo
    done
}

run_benchmarks() {
    print_header "Running Benchmarks"

    # Run benchmarks for all packages
    print_info "Running benchmarks..."
    go test -bench=. -benchmem ./internal/... || {
        print_warning "Some benchmarks may have failed"
    }
    echo
}

run_race_tests() {
    print_header "Running Race Detection Tests"

    print_info "Running tests with race detection..."
    go test -race ./internal/... || {
        print_error "Race conditions detected"
        return 1
    }
    print_success "No race conditions detected"
    echo
}

run_coverage_tests() {
    print_header "Running Coverage Analysis"

    print_info "Generating coverage report..."
    go test -coverprofile=coverage.out ./internal/... || {
        print_error "Coverage analysis failed"
        return 1
    }

    if command -v go &> /dev/null; then
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        print_success "Total coverage: $coverage"

        # Generate HTML coverage report
        go tool cover -html=coverage.out -o coverage.html
        print_success "HTML coverage report generated: coverage.html"
    fi
    echo
}

cleanup() {
    print_header "Cleaning Up"

    # Remove coverage files
    if [ -f "coverage.out" ]; then
        rm coverage.out
        print_success "Removed coverage.out"
    fi

    # Clean test cache
    go clean -testcache
    print_success "Cleaned test cache"

    echo
}

print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help          Show this help message"
    echo "  -u, --unit          Run only unit tests"
    echo "  -i, --integration   Run only integration tests"
    echo "  -b, --bench         Run benchmarks"
    echo "  -r, --race          Run race detection tests"
    echo "  -c, --coverage      Run coverage analysis"
    echo "  --setup-db          Setup test database only"
    echo "  --cleanup          Cleanup only"
    echo "  --skip-db-setup    Skip database setup"
    echo ""
    echo "Environment Variables:"
    echo "  TEST_DB_NAME        Test database name (default: urlshortener_test)"
    echo "  TEST_DB_USER        Test database user (default: postgres)"
    echo "  TEST_DB_PASSWORD    Test database password (default: postgres)"
    echo "  TEST_DB_HOST        Test database host (default: localhost)"
    echo "  TEST_DB_PORT        Test database port (default: 5432)"
    echo "  TEST_DB_SSLMODE     Test database SSL mode (default: disable)"
}

main() {
    local run_unit=false
    local run_integration=false
    local run_bench=false
    local run_race=false
    local run_coverage=false
    local setup_db_only=false
    local cleanup_only=false
    local skip_db_setup=false
    local run_all=true

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -u|--unit)
                run_unit=true
                run_all=false
                shift
                ;;
            -i|--integration)
                run_integration=true
                run_all=false
                shift
                ;;
            -b|--bench)
                run_bench=true
                run_all=false
                shift
                ;;
            -r|--race)
                run_race=true
                run_all=false
                shift
                ;;
            -c|--coverage)
                run_coverage=true
                run_all=false
                shift
                ;;
            --setup-db)
                setup_db_only=true
                run_all=false
                shift
                ;;
            --cleanup)
                cleanup_only=true
                run_all=false
                shift
                ;;
            --skip-db-setup)
                skip_db_setup=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    # Change to project directory
    cd "$(dirname "$0")"

    print_header "URL Shortener Test Suite"
    print_info "Test database: $TEST_DB_NAME"
    print_info "Database host: $TEST_DB_HOST:$TEST_DB_PORT"
    echo

    # Handle specific actions
    if [ "$cleanup_only" = true ]; then
        cleanup
        exit 0
    fi

    if [ "$setup_db_only" = true ]; then
        check_prerequisites
        setup_test_database
        exit 0
    fi

    # Check prerequisites
    check_prerequisites

    # Setup database unless skipped
    if [ "$skip_db_setup" = false ]; then
        setup_test_database || {
            print_error "Database setup failed"
            exit 1
        }
    fi

    # Run tests based on options
    if [ "$run_all" = true ]; then
        run_unit_tests || exit 1
        run_integration_tests || exit 1
        run_race_tests || exit 1
        run_coverage_tests || exit 1
        run_benchmarks
    else
        [ "$run_unit" = true ] && { run_unit_tests || exit 1; }
        [ "$run_integration" = true ] && { run_integration_tests || exit 1; }
        [ "$run_race" = true ] && { run_race_tests || exit 1; }
        [ "$run_coverage" = true ] && { run_coverage_tests || exit 1; }
        [ "$run_bench" = true ] && run_benchmarks
    fi

    # Cleanup
    cleanup

    print_header "Test Suite Completed Successfully"
    print_success "All tests passed! 🎉"
}

# Handle script interruption
trap 'echo -e "\n${RED}Test suite interrupted${NC}"; cleanup; exit 1' INT TERM

# Run main function
main "$@"
