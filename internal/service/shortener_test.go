package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/lib/pq"
	"urlshortener/urlshortener/internal/model"
)

// Mock repository for testing
type mockURLRepo struct {
	urls           map[string]model.URLRecord // key: long_url
	codes          map[string]model.URLRecord // key: code
	insertError    error
	getByLongError error
	getByCodeError error
	insertFunc     func(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error)
}

func newMockURLRepo() *mockURLRepo {
	return &mockURLRepo{
		urls:  make(map[string]model.URLRecord),
		codes: make(map[string]model.URLRecord),
	}
}

func (m *mockURLRepo) GetByLong(ctx context.Context, long string) (model.URLRecord, error) {
	if m.getByLongError != nil {
		return model.URLRecord{}, m.getByLongError
	}

	if rec, exists := m.urls[long]; exists {
		return rec, nil
	}
	return model.URLRecord{}, sql.ErrNoRows
}

func (m *mockURLRepo) GetByCode(ctx context.Context, code string) (model.URLRecord, error) {
	if m.getByCodeError != nil {
		return model.URLRecord{}, m.getByCodeError
	}

	if rec, exists := m.codes[code]; exists {
		return rec, nil
	}
	return model.URLRecord{}, sql.ErrNoRows
}

func (m *mockURLRepo) Insert(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error) {
	// If custom insert function is provided, use it
	if m.insertFunc != nil {
		return m.insertFunc(ctx, id, code, long, short)
	}

	if m.insertError != nil {
		return model.URLRecord{}, m.insertError
	}

	// Check for code collision
	if _, exists := m.codes[code]; exists {
		pqErr := &pq.Error{
			Code:   PgUniqueViolation,
			Detail: "Key (code)=(" + code + ") already exists.",
		}
		return model.URLRecord{}, pqErr
	}

	// Check for long URL collision
	if _, exists := m.urls[long]; exists {
		pqErr := &pq.Error{
			Code:   PgUniqueViolation,
			Detail: "Key (long_url)=(" + long + ") already exists.",
		}
		return model.URLRecord{}, pqErr
	}

	rec := model.URLRecord{
		ID:       id,
		Code:     code,
		LongUrl:  long,
		ShortUrl: short,
	}

	m.urls[long] = rec
	m.codes[code] = rec

	return rec, nil
}

func TestShortener_Shorten_NewURL(t *testing.T) {
	repo := newMockURLRepo()
	s := NewShortener(repo)

	ctx := context.Background()
	baseURL := "https://shawt.ly/"
	longURL := "https://example.com/very/long/url"

	rec, created, err := s.Shorten(ctx, baseURL, longURL)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !created {
		t.Error("Expected created to be true for new URL")
	}

	if rec.LongUrl != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, rec.LongUrl)
	}

	if len(rec.Code) != 6 {
		t.Errorf("Expected code length to be 6, got %d", len(rec.Code))
	}

	if rec.ShortUrl != baseURL+rec.Code {
		t.Errorf("Expected short URL %s, got %s", baseURL+rec.Code, rec.ShortUrl)
	}
}

func TestShortener_Shorten_ExistingURL(t *testing.T) {
	repo := newMockURLRepo()
	s := NewShortener(repo)

	ctx := context.Background()
	baseURL := "https://shawt.ly/"
	longURL := "https://example.com/existing"

	// First call - should create
	rec1, created1, err1 := s.Shorten(ctx, baseURL, longURL)
	if err1 != nil {
		t.Fatalf("First call failed: %v", err1)
	}
	if !created1 {
		t.Error("Expected first call to create new record")
	}

	// Second call - should return existing
	rec2, created2, err2 := s.Shorten(ctx, baseURL, longURL)
	if err2 != nil {
		t.Errorf("Second call failed: %v", err2)
	}
	if created2 {
		t.Error("Expected second call to not create new record")
	}

	if rec1.Code != rec2.Code {
		t.Errorf("Expected same code for same URL, got %s and %s", rec1.Code, rec2.Code)
	}
}

func TestShortener_Shorten_CodeCollision(t *testing.T) {
	repo := newMockURLRepo()

	// Pre-populate with a code to force collision
	existingRec := model.URLRecord{
		ID:       "existing-id",
		Code:     "ABC123",
		LongUrl:  "https://example.com/existing",
		ShortUrl: "https://shawt.ly/ABC123",
	}
	repo.codes[existingRec.Code] = existingRec
	repo.urls[existingRec.LongUrl] = existingRec

	s := NewShortener(repo)

	ctx := context.Background()
	baseURL := "https://shawt.ly/"
	longURL := "https://example.com/new"

	// Override insert to simulate code collision on first attempt
	callCount := 0
	repo.insertFunc = func(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error) {
		callCount++
		if callCount == 1 && code == "ABC123" {
			pqErr := &pq.Error{
				Code:   PgUniqueViolation,
				Detail: "Key (code)=(" + code + ") already exists.",
			}
			return model.URLRecord{}, pqErr
		}
		// For subsequent calls, use the normal logic
		return repo.normalInsert(ctx, id, code, long, short)
	}

	rec, created, err := s.Shorten(ctx, baseURL, longURL)
	if err != nil {
		t.Errorf("Expected no error after retry, got %v", err)
	}

	if !created {
		t.Error("Expected created to be true")
	}

	if rec.Code == existingRec.Code {
		t.Error("Expected different code after collision")
	}
}

// normalInsert is the default insert behavior
func (m *mockURLRepo) normalInsert(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error) {
	// Check for code collision
	if _, exists := m.codes[code]; exists {
		pqErr := &pq.Error{
			Code:   PgUniqueViolation,
			Detail: "Key (code)=(" + code + ") already exists.",
		}
		return model.URLRecord{}, pqErr
	}

	// Check for long URL collision
	if _, exists := m.urls[long]; exists {
		pqErr := &pq.Error{
			Code:   PgUniqueViolation,
			Detail: "Key (long_url)=(" + long + ") already exists.",
		}
		return model.URLRecord{}, pqErr
	}

	rec := model.URLRecord{
		ID:       id,
		Code:     code,
		LongUrl:  long,
		ShortUrl: short,
	}

	m.urls[long] = rec
	m.codes[code] = rec

	return rec, nil
}

func TestShortener_Shorten_MaxRetries(t *testing.T) {
	repo := newMockURLRepo()

	// Set up repo to always return code collision
	repo.insertError = &pq.Error{
		Code:   PgUniqueViolation,
		Detail: "Key (code)=(test) already exists.",
	}

	s := NewShortener(repo)

	ctx := context.Background()
	baseURL := "https://shawt.ly/"
	longURL := "https://example.com/test"

	_, created, err := s.Shorten(ctx, baseURL, longURL)

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if created {
		t.Error("Expected created to be false on error")
	}

	expectedErr := "Could not allocate unique code"
	if err.Error() != expectedErr {
		t.Errorf("Expected error message %s, got %s", expectedErr, err.Error())
	}
}

func TestShortener_Shorten_LongURLCollisionRace(t *testing.T) {
	repo := newMockURLRepo()
	s := NewShortener(repo)

	ctx := context.Background()
	baseURL := "https://shawt.ly/"
	longURL := "https://example.com/race"

	// Override insert to simulate long URL collision
	repo.insertFunc = func(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error) {
		// Simulate race condition - another request inserted the same long URL
		pqErr := &pq.Error{
			Code:   PgUniqueViolation,
			Detail: "Key (long_url)=(" + long + ") already exists.",
		}

		// Add the record to simulate it was inserted by another request
		existingRec := model.URLRecord{
			ID:       "race-id",
			Code:     "RACE01",
			LongUrl:  long,
			ShortUrl: baseURL + "RACE01",
		}
		repo.urls[long] = existingRec
		repo.codes["RACE01"] = existingRec

		return model.URLRecord{}, pqErr
	}

	rec, created, err := s.Shorten(ctx, baseURL, longURL)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if created {
		t.Error("Expected created to be false when returning existing record")
	}

	if rec.Code != "RACE01" {
		t.Errorf("Expected code RACE01, got %s", rec.Code)
	}
}

func TestShortener_Resolve_Success(t *testing.T) {
	repo := newMockURLRepo()

	// Pre-populate with a record
	rec := model.URLRecord{
		ID:       "test-id",
		Code:     "TEST01",
		LongUrl:  "https://example.com/test",
		ShortUrl: "https://shawt.ly/TEST01",
	}
	repo.codes[rec.Code] = rec

	s := NewShortener(repo)

	ctx := context.Background()
	longURL, err := s.Resolve(ctx, "TEST01")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if longURL != "https://example.com/test" {
		t.Errorf("Expected long URL https://example.com/test, got %s", longURL)
	}
}

func TestShortener_Resolve_NotFound(t *testing.T) {
	repo := newMockURLRepo()
	s := NewShortener(repo)

	ctx := context.Background()
	_, err := s.Resolve(ctx, "NOTFOUND")

	if err == nil {
		t.Error("Expected error for non-existent code")
	}

	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestShortener_Resolve_RepoError(t *testing.T) {
	repo := newMockURLRepo()
	repo.getByCodeError = errors.New("database connection error")

	s := NewShortener(repo)

	ctx := context.Background()
	_, err := s.Resolve(ctx, "TEST01")

	if err == nil {
		t.Error("Expected error from repository")
	}

	expectedErr := "database connection error"
	if err.Error() != expectedErr {
		t.Errorf("Expected error message %s, got %s", expectedErr, err.Error())
	}
}

func BenchmarkShortener_Shorten(b *testing.B) {
	repo := newMockURLRepo()
	s := NewShortener(repo)
	ctx := context.Background()
	baseURL := "https://shawt.ly/"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		longURL := "https://example.com/benchmark/" + string(rune(i))
		s.Shorten(ctx, baseURL, longURL)
	}
}

func BenchmarkShortener_Resolve(b *testing.B) {
	repo := newMockURLRepo()

	// Pre-populate with test data
	for i := 0; i < 1000; i++ {
		code := "CODE" + string(rune(i))
		rec := model.URLRecord{
			Code:    code,
			LongUrl: "https://example.com/" + string(rune(i)),
		}
		repo.codes[code] = rec
	}

	s := NewShortener(repo)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := "CODE" + string(rune(i%1000))
		s.Resolve(ctx, code)
	}
}
