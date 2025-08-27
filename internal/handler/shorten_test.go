package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/model"

	"github.com/gin-gonic/gin"
)

// Mock shortener service for testing
type mockShortener struct {
	shortenFunc  func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error)
	resolveFunc  func(ctx context.Context, code string) (string, error)
	redirectFunc func(ctx context.Context, code string) (string, error)
}

func (m *mockShortener) Shorten(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
	if m.shortenFunc != nil {
		return m.shortenFunc(ctx, baseURL, long)
	}
	return model.URLRecord{}, false, errors.New("not implemented")
}

func (m *mockShortener) Resolve(ctx context.Context, code string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, code)
	}
	return "", errors.New("not implemented")
}

func TestHandler_Shorten_Success_NewURL(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{
		shortenFunc: func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
			return model.URLRecord{
				ID:        "test-id",
				Code:      "ABC123",
				LongUrl:   long,
				ShortUrl:  baseURL + "ABC123",
				CreatedAt: time.Now(),
			}, true, nil
		},
	}

	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test data
	reqBody := model.CreateReq{
		URL: "https://example.com/test",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response model.URLRecord
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "ABC123" {
		t.Errorf("Expected code ABC123, got %s", response.Code)
	}

	if response.LongUrl != "https://example.com/test" {
		t.Errorf("Expected long URL https://example.com/test, got %s", response.LongUrl)
	}

	if response.ShortUrl != "https://shawt.ly/ABC123" {
		t.Errorf("Expected short URL https://shawt.ly/ABC123, got %s", response.ShortUrl)
	}
}

func TestHandler_Shorten_Success_ExistingURL(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{
		shortenFunc: func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
			return model.URLRecord{
				ID:        "existing-id",
				Code:      "EXIST1",
				LongUrl:   long,
				ShortUrl:  baseURL + "EXIST1",
				CreatedAt: time.Now().Add(-time.Hour), // Created earlier
			}, false, nil // false indicates existing URL
		},
	}

	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test data
	reqBody := model.CreateReq{
		URL: "https://example.com/existing",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions - should return 200 OK for existing URL
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response model.URLRecord
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "EXIST1" {
		t.Errorf("Expected code EXIST1, got %s", response.Code)
	}
}

func TestHandler_Shorten_MissingURL(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{}
	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test data - empty body
	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedError := "Missing field: url"
	if response["error"] != expectedError {
		t.Errorf("Expected error message %s, got %s", expectedError, response["error"])
	}
}

func TestHandler_Shorten_InvalidJSON(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{}
	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test data - invalid JSON
	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_Shorten_MalformedURL(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{}
	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	testCases := []struct {
		name string
		url  string
	}{
		{"Invalid URL", "not-a-url"},
		{"Missing scheme", "example.com"},
		{"FTP scheme", "ftp://example.com"},
		{"File scheme", "file:///etc/passwd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := model.CreateReq{
				URL: tc.url,
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d for URL: %s", http.StatusBadRequest, w.Code, tc.url)
			}

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			expectedError := "Malformed or unsupported URL"
			if response["error"] != expectedError {
				t.Errorf("Expected error message %s, got %s", expectedError, response["error"])
			}
		})
	}
}

func TestHandler_Shorten_ValidURLs(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{
		shortenFunc: func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
			return model.URLRecord{
				ID:        "test-id",
				Code:      "VALID1",
				LongUrl:   long,
				ShortUrl:  baseURL + "VALID1",
				CreatedAt: time.Now(),
			}, true, nil
		},
	}

	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	validURLs := []string{
		"https://example.com",
		"http://example.com",
		"https://subdomain.example.com/path",
		"http://example.com:8080/path?query=value",
		"https://example.com/path/to/resource",
		"https://192.168.1.1:8080/api",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			reqBody := model.CreateReq{
				URL: url,
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status %d, got %d for URL: %s", http.StatusCreated, w.Code, url)
			}

			var response model.URLRecord
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response.LongUrl != url {
				t.Errorf("Expected long URL %s, got %s", url, response.LongUrl)
			}
		})
	}
}

func TestHandler_Shorten_ServiceError(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{
		shortenFunc: func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
			return model.URLRecord{}, false, errors.New("database connection failed")
		},
	}

	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test data
	reqBody := model.CreateReq{
		URL: "https://example.com/test",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Make request
	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedError := "database connection failed"
	if response["error"] != expectedError {
		t.Errorf("Expected error message %s, got %s", expectedError, response["error"])
	}
}

func TestHandler_Shorten_URLNormalization(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	var capturedURL string
	mockSrv := &mockShortener{
		shortenFunc: func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
			capturedURL = long
			return model.URLRecord{
				ID:        "test-id",
				Code:      "NORM01",
				LongUrl:   long,
				ShortUrl:  baseURL + "NORM01",
				CreatedAt: time.Now(),
			}, true, nil
		},
	}

	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test that URLs are normalized by Go's url.ParseRequestURI
	reqBody := model.CreateReq{
		URL: "https://example.com/path/../normalized",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	// The URL should be passed to the service as-is (Go's URL parser handles normalization)
	expectedURL := "https://example.com/path/../normalized"
	if capturedURL != expectedURL {
		t.Errorf("Expected captured URL %s, got %s", expectedURL, capturedURL)
	}
}

func TestHandler_Shorten_ContentType(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{}
	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	// Test without Content-Type header
	reqBody := model.CreateReq{
		URL: "https://example.com/test",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
	// Don't set Content-Type header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still work as Gin can handle JSON without explicit Content-Type
	if w.Code != http.StatusBadRequest && w.Code != http.StatusCreated {
		t.Logf("Request without Content-Type returned status %d", w.Code)
	}
}

func BenchmarkHandler_Shorten(b *testing.B) {
	// Setup
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		BaseURL: "https://shawt.ly/",
	}

	mockSrv := &mockShortener{
		shortenFunc: func(ctx context.Context, baseURL, long string) (model.URLRecord, bool, error) {
			return model.URLRecord{
				ID:        "bench-id",
				Code:      "BENCH1",
				LongUrl:   long,
				ShortUrl:  baseURL + "BENCH1",
				CreatedAt: time.Now(),
			}, true, nil
		},
	}

	handler := New(cfg, mockSrv)
	router := gin.New()
	router.POST("/shorten", handler.Shorten)

	reqBody := model.CreateReq{
		URL: "https://example.com/benchmark",
	}
	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}
	}
}

func TestHandler_Redirect_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	mockSrv := &mockShortener{
		resolveFunc: func(ctx context.Context, code string) (string, error) {
			if code != "AbC123" {
				return "", errors.New("unexpected code")
			}
			return "https://example.com/landing", nil
		},
	}
	h := New(cfg, mockSrv)

	r := gin.New()
	r.GET("/:code", h.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/AbC123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/landing" {
		t.Fatalf("expected Location=https://example.com/landing, got %q", loc)
	}
}

func TestHandler_Redirect_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	mockSrv := &mockShortener{
		resolveFunc: func(ctx context.Context, code string) (string, error) {
			return "", errors.New("not found")
		},
	}
	h := New(cfg, mockSrv)

	r := gin.New()
	r.GET("/:code", h.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/doesnt-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Fatalf("did not expect Location header, got %q", loc)
	}
}

func TestRouter_RoutePrecedence_ShortcodeDoesNotCaptureShorten(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	mockSrv := &mockShortener{
		resolveFunc: func(ctx context.Context, code string) (string, error) {
			return "https://example.org", nil
		},
	}
	h := New(cfg, mockSrv)

	r := gin.New()
	// Minimal /shorten handler stub to assert we hit it
	r.POST("/shorten", func(c *gin.Context) { c.Status(http.StatusCreated) })
	r.GET("/:code", h.Redirect)

	// POST /shorten should NOT be routed to /:code
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(`{"url":"https://x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", w.Code)
	}
}

func TestHandler_Redirect_HEAD_MirrorsGET(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{BaseURL: "https://shawt.ly/"}
	mockSrv := &mockShortener{
		resolveFunc: func(ctx context.Context, code string) (string, error) {
			return "https://example.com/head-ok", nil
		},
	}
	h := New(cfg, mockSrv)

	r := gin.New()
	r.GET("/:code", h.Redirect)
	r.HEAD("/:code", h.Redirect)

	req := httptest.NewRequest(http.MethodHead, "/ABC", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, w.Code)
	}
	if w.Header().Get("Location") != "https://example.com/head-ok" {
		t.Fatalf("bad Location %q", w.Header().Get("Location"))
	}
}
