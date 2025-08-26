package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"urlshortener/urlshortener/internal/model"
)

// TestURLRecord creates a test URL record with optional overrides
type URLRecordBuilder struct {
	id       string
	code     string
	longURL  string
	shortURL string
	baseURL  string
}

// NewURLRecordBuilder creates a new builder with default values
func NewURLRecordBuilder() *URLRecordBuilder {
	code := RandomCode()
	return &URLRecordBuilder{
		id:       uuid.New().String(),
		code:     code,
		longURL:  "https://example.com/test",
		shortURL: "https://shawt.ly/" + code,
		baseURL:  "https://shawt.ly/",
	}
}

// WithID sets the ID
func (b *URLRecordBuilder) WithID(id string) *URLRecordBuilder {
	b.id = id
	return b
}

// WithCode sets the code and updates the short URL
func (b *URLRecordBuilder) WithCode(code string) *URLRecordBuilder {
	b.code = code
	b.shortURL = b.baseURL + code
	return b
}

// WithLongURL sets the long URL
func (b *URLRecordBuilder) WithLongURL(longURL string) *URLRecordBuilder {
	b.longURL = longURL
	return b
}

// WithBaseURL sets the base URL and updates the short URL
func (b *URLRecordBuilder) WithBaseURL(baseURL string) *URLRecordBuilder {
	b.baseURL = baseURL
	b.shortURL = baseURL + b.code
	return b
}

// Build creates the URLRecord
func (b *URLRecordBuilder) Build() model.URLRecord {
	return model.URLRecord{
		ID:        b.id,
		Code:      b.code,
		LongUrl:   b.longURL,
		ShortUrl:  b.shortURL,
		CreatedAt: time.Now(),
	}
}

// RandomCode generates a random 6-character code for testing
func RandomCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// RandomURL generates a random URL for testing
func RandomURL() string {
	domains := []string{"example.com", "test.org", "sample.net", "demo.io"}
	paths := []string{"path", "resource", "page", "item", "content"}

	domain := domains[rand.Intn(len(domains))]
	path := paths[rand.Intn(len(paths))]
	id := rand.Intn(10000)

	return fmt.Sprintf("https://%s/%s/%d", domain, path, id)
}

// ValidURLs returns a slice of valid test URLs
func ValidURLs() []string {
	return []string{
		"https://example.com",
		"http://example.com",
		"https://subdomain.example.com/path",
		"http://example.com:8080/path?query=value",
		"https://example.com/path/to/resource#fragment",
		"https://192.168.1.1:8080/api",
		"https://user:pass@example.com/secure",
		"http://localhost:3000/development",
		"https://api.github.com/repos/user/repo",
		"https://cdn.jsdelivr.net/npm/package@1.0.0/dist/file.min.js",
	}
}

// InvalidURLs returns a slice of invalid test URLs
func InvalidURLs() []string {
	return []string{
		"not-a-url",
		"example.com",
		"ftp://example.com",
		"file:///etc/passwd",
		"javascript:alert('xss')",
		"data:text/html,<script>alert('xss')</script>",
		"mailto:user@example.com",
		"tel:+1234567890",
		"",
		"   ",
	}
}

// DatabaseCleaner helps clean up test data
type DatabaseCleaner struct {
	db *sql.DB
}

// NewDatabaseCleaner creates a new database cleaner
func NewDatabaseCleaner(db *sql.DB) *DatabaseCleaner {
	return &DatabaseCleaner{db: db}
}

// Clean removes all test data from the database
func (c *DatabaseCleaner) Clean() error {
	_, err := c.db.Exec("DELETE FROM url_records")
	return err
}

// CleanAndSeed removes all data and inserts test records
func (c *DatabaseCleaner) CleanAndSeed(records []model.URLRecord) error {
	ctx := context.Background()

	// Start transaction
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean existing data
	_, err = tx.Exec("DELETE FROM url_records")
	if err != nil {
		return err
	}

	// Insert test records
	for _, record := range records {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO url_records (id, code, long_url, short_url, created_at) VALUES ($1, $2, $3, $4, $5)",
			record.ID, record.Code, record.LongUrl, record.ShortUrl, record.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CreateTestRecords creates a slice of test URL records
func CreateTestRecords(count int, baseURL string) []model.URLRecord {
	records := make([]model.URLRecord, count)

	for i := 0; i < count; i++ {
		code := fmt.Sprintf("TEST%02d", i+1)
		records[i] = model.URLRecord{
			ID:        uuid.New().String(),
			Code:      code,
			LongUrl:   fmt.Sprintf("https://example.com/test/%d", i+1),
			ShortUrl:  baseURL + code,
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		}
	}

	return records
}

// AssertURLRecordEqual compares two URL records and returns detailed error messages
func AssertURLRecordEqual(expected, actual model.URLRecord) error {
	if expected.ID != actual.ID {
		return fmt.Errorf("ID mismatch: expected %s, got %s", expected.ID, actual.ID)
	}

	if expected.Code != actual.Code {
		return fmt.Errorf("Code mismatch: expected %s, got %s", expected.Code, actual.Code)
	}

	if expected.LongUrl != actual.LongUrl {
		return fmt.Errorf("LongUrl mismatch: expected %s, got %s", expected.LongUrl, actual.LongUrl)
	}

	if expected.ShortUrl != actual.ShortUrl {
		return fmt.Errorf("ShortUrl mismatch: expected %s, got %s", expected.ShortUrl, actual.ShortUrl)
	}

	// Allow some tolerance for CreatedAt timestamps
	if !expected.CreatedAt.IsZero() && !actual.CreatedAt.IsZero() {
		diff := expected.CreatedAt.Sub(actual.CreatedAt)
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Second {
			return fmt.Errorf("CreatedAt mismatch: expected %s, got %s (diff: %s)",
				expected.CreatedAt.Format(time.RFC3339),
				actual.CreatedAt.Format(time.RFC3339),
				diff)
		}
	}

	return nil
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}

	return false
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(operation func() error, maxRetries int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		if i < maxRetries-1 {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

// TestConfig provides common test configuration values
type TestConfig struct {
	BaseURL        string
	TestDBName     string
	TestDBUser     string
	TestDBPassword string
	TestDBHost     string
	TestDBPort     string
	TestDBSSLMode  string
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		BaseURL:        "https://test.short/",
		TestDBName:     "urlshortener_test",
		TestDBUser:     "postgres",
		TestDBPassword: "postgres",
		TestDBHost:     "localhost",
		TestDBPort:     "5432",
		TestDBSSLMode:  "disable",
	}
}

// DSN returns the database connection string for the test config
func (c TestConfig) DSN() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		c.TestDBUser, c.TestDBPassword, c.TestDBName, c.TestDBHost, c.TestDBPort, c.TestDBSSLMode)
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t interface{ Skip(...interface{}) }) {
	// This would be used with testing.T.Skip() in actual tests
	// For now, it's a placeholder for the pattern
}

// MockTime allows for deterministic time testing
type MockTime struct {
	current time.Time
}

// NewMockTime creates a new mock time starting at the given time
func NewMockTime(start time.Time) *MockTime {
	return &MockTime{current: start}
}

// Now returns the current mock time
func (m *MockTime) Now() time.Time {
	return m.current
}

// Advance moves the mock time forward by the given duration
func (m *MockTime) Advance(d time.Duration) {
	m.current = m.current.Add(d)
}

// Set sets the mock time to a specific time
func (m *MockTime) Set(t time.Time) {
	m.current = t
}

// RandomSeed initializes the random number generator with a predictable seed for testing
func RandomSeed(seed int64) {
	rand.Seed(seed)
}

// ConcurrentRunner helps run multiple goroutines and collect results
type ConcurrentRunner struct {
	goroutineCount int
	results        chan interface{}
	errors         chan error
}

// NewConcurrentRunner creates a new concurrent runner
func NewConcurrentRunner(goroutineCount int) *ConcurrentRunner {
	return &ConcurrentRunner{
		goroutineCount: goroutineCount,
		results:        make(chan interface{}, goroutineCount),
		errors:         make(chan error, goroutineCount),
	}
}

// Run executes the function in multiple goroutines
func (r *ConcurrentRunner) Run(fn func() (interface{}, error)) ([]interface{}, []error) {
	// Start goroutines
	for i := 0; i < r.goroutineCount; i++ {
		go func() {
			result, err := fn()
			if err != nil {
				r.errors <- err
			} else {
				r.results <- result
			}
		}()
	}

	// Collect results
	var results []interface{}
	var errors []error

	for i := 0; i < r.goroutineCount; i++ {
		select {
		case result := <-r.results:
			results = append(results, result)
		case err := <-r.errors:
			errors = append(errors, err)
		}
	}

	return results, errors
}
