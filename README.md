# Shawty - URL Shortener

A lightweigh URL shortener written in Go with PostgreSQL as the backend.

## Features

- **URL Shortening**: Generate short URLs with 6-character codes
- **URL Deduplication**: Same URLs get the same short code
- **PostgreSQL Storage**: Reliable database persistence
- **REST API**: Simple JSON API for integration
- **Testing**: 100+ tests with 80%+ coverage
- **Race-Safe**: Handles concurrent requests safely

## Quick Start

### Prerequisites

- Go 1.23 or later
- PostgreSQL 12 or later
- Git

### Installation

```bash
git clone https://github.com/yourusername/urlshortener
cd urlshortener
cp .env.example .env   # update with your DB config
make migrate-up
make run
```

The service will be available at `http://localhost:3001`

## API Usage

### Shorten a URL

**POST** `/shorten`

```bash
curl -X POST http://localhost:3001/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/url"}'
```

**Response:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "code": "abc123",
  "long_url": "https://example.com/very/long/url",
  "short_url": "https://shawt.ly/abc123",
  "created_at": "2023-01-01T12:00:00Z"
}
```

### (TODO) Use the Short URL

Simply visit the `short_url` in your browser or make a GET request:

```bash
curl -L https://shawt.ly/abc123
```

This will redirect you to the original URL.

## Development

### Development Setup

```bash
make dev-setup
```

This will:
- Install required development tools
- Download dependencies
- Set up test databases
- Verify the environment

### Running Tests

- Unit tests with mocks
- Integration tests (PostgreSQL)
- End-to-end API tests
- Race detection & benchmarks

```bash
# Run all tests
make test

# Run specific test types
make test-unit          # Unit tests only (fast)
make test-integration   # Integration tests (requires DB)
make test-e2e          # End-to-end tests
make test-race         # Race condition tests
make test-coverage     # Generate coverage report
make test-bench        # Performance benchmarks
```

### Test Architecture

Test suite includes:

- **Unit Tests**: Test individual components in isolation using mocks
- **Integration Tests**: Test database interactions with real PostgreSQL
- **End-to-End Tests**: Full HTTP request/response cycle tests
- **Benchmark Tests**: Performance and memory allocation testing
- **Race Detection**: Concurrent access safety verification

#### Test Coverage

- **Utility Functions**: Code generation, validation
- **Configuration**: Environment loading, URL parsing
- **Service Layer**: Business logic, error handling, retries
- **HTTP Handlers**: Request validation, response formatting
- **Database Layer**: CRUD operations, constraints, transactions
- **Integration**: Full application flow testing

### Code Quality

```bash
# Format code
make fmt

# Run all quality checks
make check
```
## Testing

### Test Structure

```
internal/
├── config/config_test.go          # Configuration tests
├── handler/shorten_test.go        # HTTP handler tests
├── http/server_test.go           # Server integration tests
├── repo/urlrepo_test.go          # Database repository tests
├── service/shortener_test.go     # Business logic tests
├── util/codegen_test.go          # Utility function tests
└── testutil/fixtures.go          # Test helpers and fixtures
e2e_test.go                       # End-to-end API tests
```

### Running Tests in CI

The project includes GitHub Actions workflows for:

- Automated testing on push/PR
- Multiple Go versions support
- Database integration testing
- Code coverage reporting
- Security scanning
- Performance benchmarking

### Test Database Setup

For integration and E2E tests, you need PostgreSQL:

```bash
# Using the test script
./test.sh --setup-db

# Or manually
export TEST_DB_NAME=urlshortener_test
export TEST_DB_USER=test_user
export TEST_DB_PASSWORD=test_password
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
```

## Architecture

```
┌─────────────────┐
│   HTTP Handler  │  ← REST API endpoints
├─────────────────┤
│   Service Layer │  ← Business logic
├─────────────────┤
│  Repository     │  ← Data access
├─────────────────┤
│   Database      │  ← PostgreSQL
└─────────────────┘
```

### Components

- **HTTP Server** (`internal/http`): Gin-based REST API
- **Handlers** (`internal/handler`): Request/response handling
- **Service** (`internal/service`): Business logic and URL shortening
- **Repository** (`internal/repo`): Database operations
- **Models** (`internal/model`): Data structures
- **Config** (`internal/config`): Configuration management
- **Utils** (`internal/util`): Utility functions

## Deployment

### Docker

```bash
# Build image
make docker-build

# Run with Docker
make docker-run

# Run tests in Docker
make docker-test
```

### Environment Variables

| Variable                  | Description                   | Example                                                               |
|---------------------------|-------------------------------|-----------------------------------------------------------------------------------|
| `DB_USER`                 | Main database username        | `user`                                                                        |
| `DB_USER_PASSWORD`        | Main database password        | `password`                                                                |
| `DB_NAME`                 | Main database name            | `urlshortener`                                                                    |
| `DB_HOST`                 | Database host                 | `localhost`                                                                       |
| `DB_PORT`                 | Database port                 | `5432`                                                                            |
| `DB_DRIVER`               | Database driver               | `postgres`                                                                        |
| `DB_SSLMODE`              | SSL mode                      | `disable`                                                                         |
| `DB_URI`                  | Full database URI             | `postgres://user:password@localhost:5432/urlshortener?sslmode=disable` |
| `DB_USER_FLYWAY`          | Flyway migration username     | `flyway_user`                                                                     |
| `DB_USER_PASSWORD_FLYWAY` | Flyway migration password     | `password`                                                                |
| `DB_URI_FLYWAY`           | Flyway connection URI         | `postgres://flyway_user:password@localhost:5432/urlshortener?sslmode=disable` |
| `TEST_DB_USER`            | Test database username        | `test_user`                                                                       |
| `TEST_DB_PASSWORD`        | Test database password        | `test_password`                                                                   |
| `TEST_DB_NAME`            | Test database name            | `urlshortener_test`                                                               |
| `TEST_DB_HOST`            | Test database host            | `localhost`                                                                       |
| `TEST_DB_PORT`            | Test database port            | `5432`                                                                            |
| `TEST_DB_SSLMODE`         | Test database SSL mode        | `disable`                                                                         |
| `BASE_URL`                | Base URL for short links      | `http://localhost:3001/`                                                          |
| `DOMAIN`                  | Server bind domain            | `localhost`                                                                       |
| `PORT`                    | Server port                   | `3001`                                                                            |

## Performance

Based on our benchmarks:

- **URL Generation**: ~1,700 ns/op (590k ops/sec)
- **URL Shortening**: ~3,800 ns/op (265k ops/sec)
- **URL Resolution**: ~100 ns/op (10M ops/sec)
- **HTTP Requests**: ~9,700 ns/op (103k req/sec)

## Tooling note

The **Makefile**, **test suite layout**, and **GitHub Actions workflow** were initially generated by Claude and adapted for this project.

## License

This project is licensed under the MIT License – see the [LICENSE](./LICENSE) file for details.
