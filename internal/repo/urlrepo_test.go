package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/sbowman/dotenv"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	// Setup test database
	var err error
	testDB, err = setupTestDB()
	if err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}
	defer testDB.Close()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestDB()

	os.Exit(code)
}

func setupTestDB() (*sql.DB, error) {
	dotenv.Load()

	// Use environment variables or defaults for test database
	dbUser := dotenv.GetString("TEST_DB_USER")
	dbPass := dotenv.GetString("TEST_DB_PASSWORD")
	dbName := dotenv.GetString("TEST_DB_NAME")
	dbHost := dotenv.GetString("TEST_DB_HOST")
	dbPort := dotenv.GetString("TEST_DB_PORT")
	dbSSLMode := dotenv.GetString("TEST_DB_SSLMODE")
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		dbUser, dbPass, dbName, dbHost, dbPort, dbSSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		// If test database doesn't exist, try to create it
		if createErr := createTestDatabase(dbUser, dbPass, dbName, dbHost, dbPort, dbSSLMode); createErr != nil {
			return nil, fmt.Errorf("failed to ping database and create test db: %w", err)
		}

		// Try connecting again
		if err = db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping database after creation: %w", err)
		}
	}

	// Create the table if it doesn't exist
	if err = createTestTable(db); err != nil {
		return nil, fmt.Errorf("failed to create test table: %w", err)
	}

	return db, nil
}

func createTestDatabase(user, pass, dbname, host, port, sslmode string) error {
	// Connect to postgres database to create test database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		user, pass, dbname, host, port, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
	if err != nil {
		// Ignore error if database already exists
		return nil
	}

	return nil
}

func createTestTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS url_records (
			id VARCHAR(36) PRIMARY KEY,
			code VARCHAR(10) UNIQUE NOT NULL,
			long_url TEXT UNIQUE NOT NULL,
			short_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := db.Exec(query)
	return err
}

func cleanupTestDB() {
	if testDB != nil {
		// Clean up test data
		testDB.Exec("DELETE FROM url_records")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestPostgresRepo_Insert(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	id := "test-id-1"
	code := "ABC123"
	longURL := "https://example.com/test"
	shortURL := "https://shawt.ly/ABC123"

	rec, err := repo.Insert(ctx, id, code, longURL, shortURL)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if rec.ID != id {
		t.Errorf("Expected ID %s, got %s", id, rec.ID)
	}

	if rec.Code != code {
		t.Errorf("Expected code %s, got %s", code, rec.Code)
	}

	if rec.LongUrl != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, rec.LongUrl)
	}

	if rec.ShortUrl != shortURL {
		t.Errorf("Expected short URL %s, got %s", shortURL, rec.ShortUrl)
	}

	if rec.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Verify it was actually inserted
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE id = $1", id).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify insert: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestPostgresRepo_Insert_DuplicateCode(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	// Insert first record
	_, err := repo.Insert(ctx, "id1", "DUP123", "https://example.com/1", "https://shawt.ly/DUP123")
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}

	// Try to insert with same code
	_, err = repo.Insert(ctx, "id2", "DUP123", "https://example.com/2", "https://shawt.ly/DUP123")
	if err == nil {
		t.Error("Expected error for duplicate code")
	}

	// Verify only one record exists
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE code = $1", "DUP123").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 record, got %d", count)
	}
}

func TestPostgresRepo_Insert_DuplicateLongURL(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	longURL := "https://example.com/duplicate"

	// Insert first record
	_, err := repo.Insert(ctx, "id1", "CODE1", longURL, "https://shawt.ly/CODE1")
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}

	// Try to insert with same long URL
	_, err = repo.Insert(ctx, "id2", "CODE2", longURL, "https://shawt.ly/CODE2")
	if err == nil {
		t.Error("Expected error for duplicate long URL")
	}

	// Verify only one record exists
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE long_url = $1", longURL).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 record, got %d", count)
	}
}

func TestPostgresRepo_GetByLong(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up and insert test data
	testDB.Exec("DELETE FROM url_records")

	id := "test-id-get-long"
	code := "GETLONG"
	longURL := "https://example.com/get-by-long"
	shortURL := "https://shawt.ly/GETLONG"

	// Insert test record
	insertedRec, err := repo.Insert(ctx, id, code, longURL, shortURL)
	if err != nil {
		t.Fatalf("Failed to insert test record: %v", err)
	}

	// Test GetByLong
	rec, err := repo.GetByLong(ctx, longURL)
	if err != nil {
		t.Fatalf("GetByLong failed: %v", err)
	}

	if rec.ID != insertedRec.ID {
		t.Errorf("Expected ID %s, got %s", insertedRec.ID, rec.ID)
	}

	if rec.Code != code {
		t.Errorf("Expected code %s, got %s", code, rec.Code)
	}

	if rec.LongUrl != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, rec.LongUrl)
	}

	if rec.ShortUrl != shortURL {
		t.Errorf("Expected short URL %s, got %s", shortURL, rec.ShortUrl)
	}
}

func TestPostgresRepo_GetByLong_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up
	testDB.Exec("DELETE FROM url_records")

	_, err := repo.GetByLong(ctx, "https://nonexistent.com")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestPostgresRepo_GetByCode(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up and insert test data
	testDB.Exec("DELETE FROM url_records")

	id := "test-id-get-code"
	code := "GETCODE"
	longURL := "https://example.com/get-by-code"
	shortURL := "https://shawt.ly/GETCODE"

	// Insert test record
	insertedRec, err := repo.Insert(ctx, id, code, longURL, shortURL)
	if err != nil {
		t.Fatalf("Failed to insert test record: %v", err)
	}

	// Test GetByCode
	rec, err := repo.GetByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetByCode failed: %v", err)
	}

	if rec.ID != insertedRec.ID {
		t.Errorf("Expected ID %s, got %s", insertedRec.ID, rec.ID)
	}

	if rec.Code != code {
		t.Errorf("Expected code %s, got %s", code, rec.Code)
	}

	if rec.LongUrl != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, rec.LongUrl)
	}

	if rec.ShortUrl != shortURL {
		t.Errorf("Expected short URL %s, got %s", shortURL, rec.ShortUrl)
	}
}

func TestPostgresRepo_GetByCode_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up
	testDB.Exec("DELETE FROM url_records")

	_, err := repo.GetByCode(ctx, "NOTFOUND")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestPostgresRepo_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up
	testDB.Exec("DELETE FROM url_records")

	// Test data
	testCases := []struct {
		id       string
		code     string
		longURL  string
		shortURL string
	}{
		{"id1", "CODE1", "https://example.com/1", "https://shawt.ly/CODE1"},
		{"id2", "CODE2", "https://example.com/2", "https://shawt.ly/CODE2"},
		{"id3", "CODE3", "https://example.com/3", "https://shawt.ly/CODE3"},
	}

	// Insert all records
	for _, tc := range testCases {
		_, err := repo.Insert(ctx, tc.id, tc.code, tc.longURL, tc.shortURL)
		if err != nil {
			t.Fatalf("Failed to insert record %s: %v", tc.id, err)
		}
	}

	// Test retrieval by long URL
	for _, tc := range testCases {
		rec, err := repo.GetByLong(ctx, tc.longURL)
		if err != nil {
			t.Errorf("Failed to get record by long URL %s: %v", tc.longURL, err)
			continue
		}

		if rec.Code != tc.code {
			t.Errorf("Expected code %s, got %s", tc.code, rec.Code)
		}
	}

	// Test retrieval by code
	for _, tc := range testCases {
		rec, err := repo.GetByCode(ctx, tc.code)
		if err != nil {
			t.Errorf("Failed to get record by code %s: %v", tc.code, err)
			continue
		}

		if rec.LongUrl != tc.longURL {
			t.Errorf("Expected long URL %s, got %s", tc.longURL, rec.LongUrl)
		}
	}

	// Verify total count
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM url_records").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}

	if count != len(testCases) {
		t.Errorf("Expected %d records, got %d", len(testCases), count)
	}
}

func BenchmarkPostgresRepo_Insert(b *testing.B) {
	if testDB == nil {
		b.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up
	testDB.Exec("DELETE FROM url_records")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("bench-id-%d", i)
		code := fmt.Sprintf("BENCH%d", i)
		longURL := fmt.Sprintf("https://example.com/bench/%d", i)
		shortURL := fmt.Sprintf("https://shawt.ly/BENCH%d", i)

		_, err := repo.Insert(ctx, id, code, longURL, shortURL)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

func BenchmarkPostgresRepo_GetByCode(b *testing.B) {
	if testDB == nil {
		b.Skip("Test database not available")
	}

	repo := NewPostgres(testDB)
	ctx := context.Background()

	// Clean up and prepare test data
	testDB.Exec("DELETE FROM url_records")

	// Insert test data
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("bench-id-%d", i)
		code := fmt.Sprintf("BENCH%d", i)
		longURL := fmt.Sprintf("https://example.com/bench/%d", i)
		shortURL := fmt.Sprintf("https://shawt.ly/BENCH%d", i)

		repo.Insert(ctx, id, code, longURL, shortURL)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := fmt.Sprintf("BENCH%d", i%1000)
		_, err := repo.GetByCode(ctx, code)
		if err != nil {
			b.Fatalf("GetByCode failed: %v", err)
		}
	}
}
