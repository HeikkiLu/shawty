# URL Shortener - Makefile
# Development and testing commands

.PHONY: help build test test-unit test-integration test-e2e test-race test-coverage test-bench clean lint fmt vet deps setup-db run dev docker-build docker-run

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := urlshortener
CMD_DIR := ./cmd/api
BIN_DIR := ./bin
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# ---- defaults (used only if not set in .env or shell) ----
TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 5432
TEST_DB_USER ?= test_user
TEST_DB_PASSWORD ?= test_password
TEST_DB_NAME ?= urlshortener_test
TEST_DB_SSLMODE ?= disable

# ---- load .env if it exists and export its vars ----
ifneq (,$(wildcard .env))
include .env
# export only valid VAR= lines from .env
export $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env)
endif

# explicitly export the DB vars (after include)
export TEST_DB_HOST TEST_DB_PORT TEST_DB_USER TEST_DB_PASSWORD TEST_DB_NAME TEST_DB_SSLMODE

help: ## Display this help message
	@echo "URL Shortener - Development Commands"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -v -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Built $(BIN_DIR)/$(APP_NAME)"

build-race: ## Build the application with race detection
	@echo "Building $(APP_NAME) with race detection..."
	@mkdir -p $(BIN_DIR)
	@go build -race -v -o $(BIN_DIR)/$(APP_NAME)-race $(CMD_DIR)
	@echo "Built $(BIN_DIR)/$(APP_NAME)-race"

run: build ## Build and run the application
	@echo "Running $(APP_NAME)..."
	@$(BIN_DIR)/$(APP_NAME)

dev: ## Run the application in development mode (with live reload if available)
	@echo "Running $(APP_NAME) in development mode..."
	@go run $(CMD_DIR)/main.go

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@go mod tidy

deps-update: ## Update dependencies to latest versions
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

test: ## Run all tests
	@echo "Running all tests..."
	@chmod +x test.sh
	@./test.sh

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@chmod +x test.sh
	@./test.sh --unit

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	@chmod +x test.sh
	@./test.sh --integration

test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	@TEST_DB_NAME=$(E2E_TEST_DB_NAME) go test -v ./e2e_test.go

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@chmod +x test.sh
	@./test.sh --race

test-coverage: ## Run tests with coverage analysis
	@echo "Running tests with coverage analysis..."
	@chmod +x test.sh
	@./test.sh --coverage
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	@chmod +x test.sh
	@./test.sh --bench

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@go test -v ./internal/...

test-short: ## Run tests in short mode (skip slow tests)
	@echo "Running tests in short mode..."
	@go test -short ./internal/...

test-timeout: ## Run tests with custom timeout
	@echo "Running tests with timeout..."
	@go test -timeout 30s ./internal/...

setup-db: ## Setup test database
	@echo "Setting up test database..."
	@chmod +x test.sh
	@./test.sh --setup-db

clean: ## Clean build artifacts and test files
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@go clean -cache -testcache -modcache
	@echo "Clean complete"

clean-db: ## Clean test databases
	@echo "Cleaning test databases..."
	@-psql -h $(TEST_DB_HOST) -p $(TEST_DB_PORT) -U $(TEST_DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(TEST_DB_NAME);"
	@-psql -h $(TEST_DB_HOST) -p $(TEST_DB_PORT) -U $(TEST_DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(E2E_TEST_DB_NAME);"
	@echo "Test databases cleaned"

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@gofumpt -l -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

check: fmt vet  ## Run all code quality checks

pre-commit: check test ## Run checks and tests before committing
	@echo "Pre-commit checks complete ✅"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/air-verse/air@latest
	@echo "Development tools installed"

# docker-build: ## Build Docker image
# 	@echo "Building Docker image..."
# 	@docker build -t $(APP_NAME):latest .

# docker-run: docker-build ## Build and run Docker container
# 	@echo "Running Docker container..."
# 	@docker run --rm -p 8080:8080 $(APP_NAME):latest

# docker-test: ## Run tests in Docker container
# 	@echo "Running tests in Docker..."
# 	@docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
# 	@docker-compose -f docker-compose.test.yml down

migrate-up: ## Run database migrations up
	@echo "Running database migrations..."
	@flyway -configFiles=flyway.conf migrate

migrate-down: ## Run database migrations down
	@echo "Rolling back database migrations..."
	@flyway -configFiles=flyway.conf undo

migrate-info: ## Show migration status
	@echo "Database migration status..."
	@flyway -configFiles=flyway.conf info

profile-cpu: ## Run CPU profiling
	@echo "Running CPU profiling..."
	@go test -cpuprofile cpu.prof -bench=. ./internal/...
	@echo "CPU profile saved to cpu.prof"

profile-mem: ## Run memory profiling
	@echo "Running memory profiling..."
	@go test -memprofile mem.prof -bench=. ./internal/...
	@echo "Memory profile saved to mem.prof"

profile-trace: ## Run execution tracing
	@echo "Running execution tracing..."
	@go test -trace trace.out -bench=. ./internal/...
	@echo "Execution trace saved to trace.out"

view-cpu-profile: profile-cpu ## View CPU profile
	@go tool pprof cpu.prof

view-mem-profile: profile-mem ## View memory profile
	@go tool pprof mem.prof

generate: ## Run go generate
	@echo "Running go generate..."
	@go generate ./...

mod-graph: ## Show module dependency graph
	@echo "Module dependency graph:"
	@go mod graph

mod-why: ## Show why a module is needed
	@echo "Module dependency reasons:"
	@go mod why -m all

version: ## Show Go version and module info
	@echo "Go version:"
	@go version
	@echo ""
	@echo "Module info:"
	@go list -m

env: ## Show Go environment
	@go env

stats: ## Show project statistics
	@echo "Project Statistics:"
	@echo "=================="
	@echo "Go files:"
	@find . -name "*.go" -not -path "./vendor/*" | wc -l
	@echo "Test files:"
	@find . -name "*_test.go" -not -path "./vendor/*" | wc -l
	@echo "Total lines of code:"
	@find . -name "*.go" -not -path "./vendor/*" -exec wc -l {} + | tail -1
	@echo "Test coverage:"
	@go test -cover ./internal/... | grep -E "coverage:" | tail -1

todo: ## Show TODO and FIXME comments
	@echo "TODO and FIXME items:"
	@echo "==================="
	@grep -rn --include="*.go" -E "(TODO|FIXME|XXX|HACK)" . || echo "No TODO/FIXME items found"

watch: ## Watch for file changes and run tests
	@echo "Watching for changes..."
	@air -c .air.toml

# CI/CD targets
ci-test: deps test ## Run tests in CI environment
	@echo "CI tests complete"

ci-build: deps build ## Build in CI environment
	@echo "CI build complete"

ci-deploy: ci-test ci-build ## Deploy in CI environment
	@echo "CI deploy complete"

# Development workflow targets
dev-setup: install-tools deps setup-db ## Complete development setup
	@echo "Development environment setup complete! 🚀"
	@echo ""
	@echo "Next steps:"
	@echo "1. Copy .env.example to .env and configure"
	@echo "2. Run 'make run' to start the application"
	@echo "3. Run 'make test' to run tests"

quick-test: ## Run quick tests (unit tests only)
	@go test -short ./internal/util ./internal/config ./internal/service ./internal/handler

full-test: clean-db setup-db test ## Full test run with clean environment
	@echo "Full test run complete ✅"

release-check: clean deps test lint build ## Pre-release checks
	@echo "Release checks complete ✅"

# Help for specific test commands
test-help: ## Show detailed test help
	@echo "Test Commands Help:"
	@echo "=================="
	@echo "make test           - Run all tests"
	@echo "make test-unit      - Run unit tests only (fast)"
	@echo "make test-integration - Run integration tests (requires DB)"
	@echo "make test-e2e       - Run end-to-end tests (requires DB)"
	@echo "make test-race      - Run tests with race detection"
	@echo "make test-coverage  - Run tests with coverage analysis"
	@echo "make test-bench     - Run benchmark tests"
	@echo "make test-verbose   - Run tests with verbose output"
	@echo "make test-short     - Run tests in short mode"
	@echo ""
	@echo "Database Test Setup:"
	@echo "make setup-db       - Setup test database"
	@echo "make clean-db       - Clean test databases"
	@echo ""
	@echo "Environment Variables:"
	@echo "TEST_DB_HOST        - Database host (default: localhost)"
	@echo "TEST_DB_PORT        - Database port (default: 5432)"
	@echo "TEST_DB_USER        - Database user (default: postgres)"
	@echo "TEST_DB_PASSWORD    - Database password (default: postgres)"
