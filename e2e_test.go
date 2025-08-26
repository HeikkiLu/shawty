package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/db"
	"urlshortener/urlshortener/internal/model"

	httpserver "urlshortener/urlshortener/internal/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sbowman/dotenv"
)

var (
	testServer *httptest.Server
	testDB     *sql.DB
	testConfig config.Config
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Setup test environment
	if err := setupE2ETest(); err != nil {
		fmt.Printf("Failed to setup E2E tests: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	teardownE2ETest()

	os.Exit(code)
}

func setupE2ETest() error {
	dotenv.Load()

	// Configure test database
	testConfig = config.Config{
		DBUser:  dotenv.GetString("TEST_DB_USER"),
		DBPass:  dotenv.GetString("TEST_DB_PASSWORD"),
		DBName:  dotenv.GetString("TEST_DB_NAME"),
		DBHost:  dotenv.GetString("TEST_DB_HOST"),
		DBPort:  dotenv.GetString("TEST_DB_PORT"),
		SSLMode: dotenv.GetString("TEST_DB_SSLMODE"),
		BaseURL: "https://e2e.test/",
		Domain:  "localhost",
		Port:    "0", // Let test server choose port
	}

	// Create test database if needed
	if err := createTestDatabase(); err != nil {
		return fmt.Errorf("failed to create test database: %w", err)
	}

	// Connect to test database
	var err error
	testDB, err = db.Open(testConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Create table schema
	if err := createTableSchema(); err != nil {
		return fmt.Errorf("failed to create table schema: %w", err)
	}

	// Start test server
	engine := httpserver.NewServer(testConfig, testDB)
	testServer = httptest.NewServer(engine)

	return nil
}

func teardownE2ETest() {
	if testServer != nil {
		testServer.Close()
	}
	if testDB != nil {
		testDB.Close()
	}
}

func createTestDatabase() error {
	// Connect to postgres to create test database
	adminConfig := testConfig
	adminConfig.DBName = "postgres"

	adminDB, err := sql.Open("postgres", adminConfig.DSN())
	if err != nil {
		return err
	}
	defer adminDB.Close()

	// Drop test database if exists and create new one
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testConfig.DBName))
	if err != nil {
		return err
	}

	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testConfig.DBName))
	return err
}

func createTableSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS url_records (
			id VARCHAR(36) PRIMARY KEY,
			code VARCHAR(10) UNIQUE NOT NULL,
			long_url TEXT UNIQUE NOT NULL,
			short_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := testDB.Exec(schema)
	return err
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func clearDatabase() error {
	_, err := testDB.Exec("DELETE FROM url_records")
	return err
}

func TestE2E_ShortenURL_NewURL(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("Failed to clear database: %v", err)
	}

	// Prepare request
	reqBody := model.CreateReq{
		URL: "https://example.com/e2e-new-url-test",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	// Parse response
	var urlRecord model.URLRecord
	if err := json.NewDecoder(resp.Body).Decode(&urlRecord); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response
	if urlRecord.LongUrl != reqBody.URL {
		t.Errorf("Expected long URL %s, got %s", reqBody.URL, urlRecord.LongUrl)
	}

	if len(urlRecord.Code) != 6 {
		t.Errorf("Expected code length 6, got %d", len(urlRecord.Code))
	}

	expectedShortURL := testConfig.BaseURL + urlRecord.Code
	if urlRecord.ShortUrl != expectedShortURL {
		t.Errorf("Expected short URL %s, got %s", expectedShortURL, urlRecord.ShortUrl)
	}

	if urlRecord.ID == "" {
		t.Error("Expected ID to be non-empty")
	}

	if urlRecord.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Verify data was persisted in database
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE id = $1", urlRecord.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestE2E_ShortenURL_ExistingURL(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("Failed to clear database: %v", err)
	}

	testURL := "https://example.com/e2e-existing-url-test"

	// First request - create new short URL
	reqBody := model.CreateReq{URL: testURL}
	jsonBody, _ := json.Marshal(reqBody)

	resp1, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to make first request: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusCreated {
		t.Errorf("First request: expected status %d, got %d", http.StatusCreated, resp1.StatusCode)
	}

	var firstRecord model.URLRecord
	if err := json.NewDecoder(resp1.Body).Decode(&firstRecord); err != nil {
		t.Fatalf("Failed to decode first response: %v", err)
	}

	// Second request - should return existing short URL
	resp2, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("Failed to make second request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Second request: expected status %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	var secondRecord model.URLRecord
	if err := json.NewDecoder(resp2.Body).Decode(&secondRecord); err != nil {
		t.Fatalf("Failed to decode second response: %v", err)
	}

	// Verify both responses are identical
	if firstRecord.ID != secondRecord.ID {
		t.Errorf("Expected same ID, got %s and %s", firstRecord.ID, secondRecord.ID)
	}

	if firstRecord.Code != secondRecord.Code {
		t.Errorf("Expected same code, got %s and %s", firstRecord.Code, secondRecord.Code)
	}

	if firstRecord.LongUrl != secondRecord.LongUrl {
		t.Errorf("Expected same long URL, got %s and %s", firstRecord.LongUrl, secondRecord.LongUrl)
	}

	// Verify only one record exists in database
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE long_url = $1", testURL).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestE2E_ShortenURL_InvalidRequests(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    interface{}
		contentType    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Empty request body",
			requestBody:    "",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"url": invalid}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing URL field",
			requestBody:    map[string]string{},
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing field: url",
		},
		{
			name:           "Empty URL",
			requestBody:    map[string]string{"url": ""},
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing field: url",
		},
		{
			name:           "Invalid URL",
			requestBody:    map[string]string{"url": "not-a-url"},
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Malformed or unsupported URL",
		},
		{
			name:           "Unsupported protocol",
			requestBody:    map[string]string{"url": "ftp://example.com"},
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Malformed or unsupported URL",
		},
		{
			name:           "Wrong content type",
			requestBody:    map[string]string{"url": "https://example.com"},
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer

			if str, ok := tc.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				jsonData, _ := json.Marshal(tc.requestBody)
				body.Write(jsonData)
			}

			resp, err := http.Post(testServer.URL+"/shorten", tc.contentType, &body)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if tc.expectedError != "" {
				var errorResp map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if errorResp["error"] != tc.expectedError {
					t.Errorf("Expected error %s, got %s", tc.expectedError, errorResp["error"])
				}
			}
		})
	}
}

func TestE2E_ShortenURL_ConcurrentRequests(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("Failed to clear database: %v", err)
	}

	testURL := "https://example.com/e2e-concurrent-test"
	numRequests := 20

	// Channel to collect results
	results := make(chan model.URLRecord, numRequests)
	errors := make(chan error, numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			reqBody := model.CreateReq{URL: testURL}
			jsonBody, _ := json.Marshal(reqBody)

			resp, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				return
			}

			var urlRecord model.URLRecord
			if err := json.NewDecoder(resp.Body).Decode(&urlRecord); err != nil {
				errors <- err
				return
			}

			results <- urlRecord
		}()
	}

	// Collect results
	var responses []model.URLRecord
	timeout := time.After(10 * time.Second)

	for i := 0; i < numRequests; i++ {
		select {
		case result := <-results:
			responses = append(responses, result)
		case err := <-errors:
			t.Fatalf("Request failed: %v", err)
		case <-timeout:
			t.Fatalf("Timeout waiting for responses")
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

		if response.LongUrl != testURL {
			t.Errorf("Response %d: expected long URL %s, got %s", i, testURL, response.LongUrl)
		}
	}

	// Verify only one record exists in database
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM url_records WHERE long_url = $1", testURL).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record in database, got %d", count)
	}
}

func TestE2E_ShortenURL_MultipleUniqueURLs(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("Failed to clear database: %v", err)
	}

	urls := []string{
		"https://example.com/e2e-test-1",
		"https://example.com/e2e-test-2",
		"https://test.org/page/1",
		"https://test.org/page/2",
		"https://different.net/resource",
	}

	var records []model.URLRecord
	codes := make(map[string]bool)

	// Create short URLs for each unique URL
	for i, url := range urls {
		reqBody := model.CreateReq{URL: url}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Request %d: expected status %d, got %d", i, http.StatusCreated, resp.StatusCode)
		}

		var record model.URLRecord
		if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
			t.Fatalf("Request %d: failed to decode response: %v", i, err)
		}

		// Verify unique codes
		if codes[record.Code] {
			t.Errorf("Request %d: duplicate code generated: %s", i, record.Code)
		}
		codes[record.Code] = true

		// Verify correct URL mapping
		if record.LongUrl != url {
			t.Errorf("Request %d: expected long URL %s, got %s", i, url, record.LongUrl)
		}

		records = append(records, record)
	}

	// Verify all records were persisted
	var totalCount int
	err := testDB.QueryRow("SELECT COUNT(*) FROM url_records").Scan(&totalCount)
	if err != nil {
		t.Fatalf("Failed to query total count: %v", err)
	}

	if totalCount != len(urls) {
		t.Errorf("Expected %d records in database, got %d", len(urls), totalCount)
	}

	// Verify each record exists in database
	for i, record := range records {
		var dbRecord model.URLRecord
		err := testDB.QueryRow(
			"SELECT id, code, long_url, short_url FROM url_records WHERE code = $1",
			record.Code).Scan(&dbRecord.ID, &dbRecord.Code, &dbRecord.LongUrl, &dbRecord.ShortUrl)
		if err != nil {
			t.Errorf("Record %d not found in database: %v", i, err)
			continue
		}

		if dbRecord.ID != record.ID {
			t.Errorf("Record %d: ID mismatch, expected %s, got %s", i, record.ID, dbRecord.ID)
		}

		if dbRecord.LongUrl != record.LongUrl {
			t.Errorf("Record %d: long URL mismatch, expected %s, got %s", i, record.LongUrl, dbRecord.LongUrl)
		}
	}
}

func TestE2E_ShortenURL_LongURLs(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("Failed to clear database: %v", err)
	}

	// Test with very long URLs
	longURLs := []string{
		"https://example.com/" + string(make([]byte, 1000)),           // Very long path
		"https://example.com/path?param=" + string(make([]byte, 500)), // Long query parameter
		"https://subdomain.very-long-domain-name-for-testing.example.com/path/to/resource/with/many/segments/in/the/url",
	}

	for i, longURL := range longURLs {
		// Fill the long URLs with actual characters
		if i == 0 {
			longURL = "https://example.com/" + generateString('a', 1000)
		} else if i == 1 {
			longURL = "https://example.com/path?param=" + generateString('x', 500)
		}

		reqBody := model.CreateReq{URL: longURL}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Request %d: expected status %d, got %d", i, http.StatusCreated, resp.StatusCode)
		}

		var record model.URLRecord
		if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
			t.Fatalf("Request %d: failed to decode response: %v", i, err)
		}

		// Verify the long URL was stored correctly
		if record.LongUrl != longURL {
			t.Errorf("Request %d: long URL mismatch", i)
		}

		// Verify short code is still standard length
		if len(record.Code) != 6 {
			t.Errorf("Request %d: expected code length 6, got %d", i, len(record.Code))
		}
	}
}

func generateString(char rune, length int) string {
	result := make([]rune, length)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

func BenchmarkE2E_ShortenURL(b *testing.B) {
	if err := clearDatabase(); err != nil {
		b.Fatalf("Failed to clear database: %v", err)
	}

	reqBody := model.CreateReq{
		URL: "https://example.com/benchmark-test",
	}
	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(testServer.URL+"/shorten", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			b.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}
	}
}
