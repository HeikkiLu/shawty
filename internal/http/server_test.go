package http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/model"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sbowman/dotenv"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Setup test database
	var err error
	testDB, err = setupTestDB()
	if err != nil {
		fmt.Printf("Failed to setup test database: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if testDB != nil {
		cleanupTestDB()
		testDB.Close()
	}

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
	dsn := fmt.Sprintf("user=%s password=%s dbname=postgres host=%s port=%s sslmode=%s",
		user, pass, host, port, sslmode)

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
			id UUID PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			long_url TEXT NOT NULL UNIQUE,
			short_url TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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

func TestNewServer(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	server := NewServer(cfg, testDB)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	routes := server.Routes()
	if len(routes) == 0 {
		t.Fatal("expected server to have routes")
	}

	var (
		foundPostShorten bool
		foundGetCode     bool
	)

	// Check routes exist
	for _, r := range routes {
		if r.Method == http.MethodPost && r.Path == "/shorten" {
			foundPostShorten = true
		}
		if r.Method == http.MethodGet && r.Path == "/:code" {
			foundGetCode = true
		}
	}

	if !foundPostShorten {
		t.Error("expected route: POST /shorten")
	}
	if !foundGetCode {
		t.Error("expected route: GET /:code")
	}
}

func TestServer_ShortenEndpoint_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	server := NewServer(cfg, testDB)

	// Test creating a new short URL
	reqBody := model.CreateReq{
		URL: "https://example.com/integration-test",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response model.URLRecord
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.LongUrl != "https://example.com/integration-test" {
		t.Errorf("Expected long URL https://example.com/integration-test, got %s", response.LongUrl)
	}

	if len(response.Code) != 6 {
		t.Errorf("Expected code length 6, got %d", len(response.Code))
	}

	expectedShortURL := cfg.BaseURL + response.Code
	if response.ShortUrl != expectedShortURL {
		t.Errorf("Expected short URL %s, got %s", expectedShortURL, response.ShortUrl)
	}

	if response.ID == "" {
		t.Error("Expected ID to be set")
	}

	if response.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Verify the record was actually saved to the database
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE id = $1", response.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify database insert: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestServer_ShortenEndpoint_ExistingURL(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	server := NewServer(cfg, testDB)

	longURL := "https://example.com/existing-url-test"

	// First request - should create
	reqBody := model.CreateReq{URL: longURL}
	jsonBody, _ := json.Marshal(reqBody)

	req1 := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()

	server.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Errorf("First request: Expected status %d, got %d", http.StatusCreated, w1.Code)
	}

	var response1 model.URLRecord
	json.Unmarshal(w1.Body.Bytes(), &response1)

	// Second request - should return existing
	req2 := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	server.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request: Expected status %d, got %d", http.StatusOK, w2.Code)
	}

	var response2 model.URLRecord
	json.Unmarshal(w2.Body.Bytes(), &response2)

	// Should be the same record
	if response1.ID != response2.ID {
		t.Errorf("Expected same ID, got %s and %s", response1.ID, response2.ID)
	}

	if response1.Code != response2.Code {
		t.Errorf("Expected same code, got %s and %s", response1.Code, response2.Code)
	}

	// Verify only one record exists in database
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE long_url = $1", longURL).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestServer_ShortenEndpoint_InvalidInput(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	server := NewServer(cfg, testDB)

	testCases := []struct {
		name           string
		requestBody    string
		contentType    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid JSON",
			requestBody:    `{"url": invalid json`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing URL field",
			requestBody:    `{}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing field: url",
		},
		{
			name:           "Invalid URL",
			requestBody:    `{"url": "not-a-valid-url"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Malformed or unsupported URL",
		},
		{
			name:           "Unsupported protocol",
			requestBody:    `{"url": "ftp://example.com"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Malformed or unsupported URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer([]byte(tc.requestBody)))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedError != "" {
				var response map[string]string
				json.Unmarshal(w.Body.Bytes(), &response)

				if response["error"] != tc.expectedError {
					t.Errorf("Expected error %s, got %s", tc.expectedError, response["error"])
				}
			}
		})
	}
}

func TestServer_ShortenEndpoint_ConcurrentRequests(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	server := NewServer(cfg, testDB)

	// Test concurrent requests with the same URL
	longURL := "https://example.com/concurrent-test"
	numRequests := 10
	results := make(chan model.URLRecord, numRequests)
	errors := make(chan error, numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			reqBody := model.CreateReq{URL: longURL}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != http.StatusCreated && w.Code != http.StatusOK {
				errors <- fmt.Errorf("unexpected status code: %d", w.Code)
				return
			}

			var response model.URLRecord
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				errors <- err
				return
			}

			results <- response
		}()
	}

	// Collect results
	var responses []model.URLRecord
	for i := 0; i < numRequests; i++ {
		select {
		case result := <-results:
			responses = append(responses, result)
		case err := <-errors:
			t.Fatalf("Request failed: %v", err)
		}
	}

	// All responses should have the same code (same URL should get same short code)
	if len(responses) == 0 {
		t.Fatal("No responses received")
	}

	firstCode := responses[0].Code
	for i, response := range responses {
		if response.Code != firstCode {
			t.Errorf("Response %d: expected code %s, got %s", i, firstCode, response.Code)
		}

		if response.LongUrl != longURL {
			t.Errorf("Response %d: expected long URL %s, got %s", i, longURL, response.LongUrl)
		}
	}

	// Verify only one record exists in database
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE long_url = $1", longURL).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestServer_ShortenEndpoint_DifferentURLs(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	// Clean up before test
	testDB.Exec("DELETE FROM url_records")

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	server := NewServer(cfg, testDB)

	urls := []string{
		"https://example.com/test1",
		"https://example.com/test2",
		"https://example.com/test3",
		"https://different.com/page",
		"http://another.org/resource",
	}

	codes := make(map[string]bool)

	for i, url := range urls {
		reqBody := model.CreateReq{URL: url}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d or %d, got %d", i, http.StatusCreated, http.StatusOK, w.Code)
			continue
		}

		var response model.URLRecord
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Request %d: Failed to unmarshal response: %v", i, err)
			continue
		}

		// Each URL should get a unique code
		if codes[response.Code] {
			t.Errorf("Request %d: Duplicate code generated: %s", i, response.Code)
		}
		codes[response.Code] = true

		// Verify the response matches the input
		if response.LongUrl != url {
			t.Errorf("Request %d: Expected long URL %s, got %s", i, url, response.LongUrl)
		}
	}

	// Verify all records were saved
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM url_records").Scan(&count)
	if count != len(urls) {
		t.Errorf("Expected %d records in database, got %d", len(urls), count)
	}
}

func BenchmarkServer_ShortenEndpoint(b *testing.B) {
	if testDB == nil {
		b.Skip("Test database not available")
	}

	// Clean up before benchmark
	testDB.Exec("DELETE FROM url_records")

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	server := NewServer(cfg, testDB)

	reqBody := model.CreateReq{
		URL: "https://example.com/benchmark",
	}
	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			b.Fatalf("Expected status 200 or 201, got %d", w.Code)
		}
	}
}

func insertURL(t *testing.T, db *sql.DB, id, code, long, base string) {
	t.Helper()
	short := base + code
	_, err := db.Exec(`
		INSERT INTO url_records (id, code, long_url, short_url, created_at)
		VALUES ($1, $2, $3, $4, now())
	`, id, code, long, short)
	if err != nil {
		t.Fatalf("seed insert failed: %v", err)
	}

	// Verify the insert worked
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM url_records WHERE id = $1", id).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify insert: %v", err)
	}
	if count != 1 {
		t.Fatalf("insert verification failed: expected 1 record, got %d", count)
	}
}

func TestServer_Redirect_Success(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	// Clean database and verify it's empty
	if _, err := testDB.Exec("DELETE FROM url_records"); err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	srv := NewServer(cfg, testDB)

	id := "123e4567-e89b-12d3-a456-426614174000"
	code := "AbC123"
	long := "https://example.com/landing"
	insertURL(t, testDB, id, code, long, cfg.BaseURL)

	// Verify the record was inserted
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE code = $1", code).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify record insertion: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected 1 record with code %s, got %d", code, count)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		// Debug: check what's actually in the database
		rows, _ := testDB.Query("SELECT id, code, long_url FROM url_records")
		t.Log("Database contents:")
		for rows.Next() {
			var dbID, dbCode, dbLongURL string
			rows.Scan(&dbID, &dbCode, &dbLongURL)
			t.Logf("  ID: %s, Code: %s, Long URL: %s", dbID, dbCode, dbLongURL)
		}
		rows.Close()
		t.Fatalf("expected %d, got %d", http.StatusFound, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != long {
		t.Fatalf("expected Location=%q, got %q", long, loc)
	}
}

func TestServer_Redirect_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}
	testDB.Exec("DELETE FROM url_records")

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	srv := NewServer(cfg, testDB)

	req := httptest.NewRequest(http.MethodGet, "/NOPE42", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Fatalf("did not expect Location header, got %q", loc)
	}
}

func TestServer_RoutePrecedence(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}
	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	srv := NewServer(cfg, testDB)

	body, _ := json.Marshal(model.CreateReq{URL: "https://x"})
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("expected 201 or 200, got %d", w.Code)
	}
}
