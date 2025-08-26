package config

import (
	"os"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"DB_USER", "DB_USER_PASSWORD", "DB_NAME", "DB_HOST", "DB_PORT", "DB_SSLMODE", "BASE_URL", "DOMAIN", "PORT"}

	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Clean up function
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment variables
	testEnv := map[string]string{
		"DB_USER":          "testuser",
		"DB_USER_PASSWORD": "testpass",
		"DB_NAME":          "testdb",
		"DB_HOST":          "localhost",
		"DB_PORT":          "5432",
		"DB_SSLMODE":       "disable",
		"BASE_URL":         "https://short.ly",
		"DOMAIN":           "0.0.0.0",
		"PORT":             "8080",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test all fields
	if cfg.DBUser != "testuser" {
		t.Errorf("Expected DBUser 'testuser', got '%s'", cfg.DBUser)
	}

	if cfg.DBPass != "testpass" {
		t.Errorf("Expected DBPass 'testpass', got '%s'", cfg.DBPass)
	}

	if cfg.DBName != "testdb" {
		t.Errorf("Expected DBName 'testdb', got '%s'", cfg.DBName)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("Expected DBHost 'localhost', got '%s'", cfg.DBHost)
	}

	if cfg.DBPort != "5432" {
		t.Errorf("Expected DBPort '5432', got '%s'", cfg.DBPort)
	}

	if cfg.SSLMode != "disable" {
		t.Errorf("Expected SSLMode 'disable', got '%s'", cfg.SSLMode)
	}

	if cfg.BaseURL != "https://short.ly/" {
		t.Errorf("Expected BaseURL 'https://short.ly/', got '%s'", cfg.BaseURL)
	}

	if cfg.Domain != "0.0.0.0" {
		t.Errorf("Expected Domain '0.0.0.0', got '%s'", cfg.Domain)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected Port '8080', got '%s'", cfg.Port)
	}
}

func TestConfig_Load_EmptyEnvironment(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"DB_USER", "DB_USER_PASSWORD", "DB_NAME", "DB_HOST", "DB_PORT", "DB_SSLMODE", "BASE_URL", "DOMAIN", "PORT"}

	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Clean up function
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear all environment variables
	for _, key := range envVars {
		os.Unsetenv(key)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// All fields should be empty strings
	if cfg.DBUser != "" {
		t.Errorf("Expected empty DBUser, got '%s'", cfg.DBUser)
	}

	if cfg.DBPass != "" {
		t.Errorf("Expected empty DBPass, got '%s'", cfg.DBPass)
	}

	if cfg.DBName != "" {
		t.Errorf("Expected empty DBName, got '%s'", cfg.DBName)
	}

	if cfg.DBHost != "" {
		t.Errorf("Expected empty DBHost, got '%s'", cfg.DBHost)
	}

	if cfg.DBPort != "" {
		t.Errorf("Expected empty DBPort, got '%s'", cfg.DBPort)
	}

	if cfg.SSLMode != "" {
		t.Errorf("Expected empty SSLMode, got '%s'", cfg.SSLMode)
	}

	if cfg.BaseURL != "/" {
		t.Errorf("Expected BaseURL '/', got '%s'", cfg.BaseURL)
	}

	if cfg.Domain != "" {
		t.Errorf("Expected empty Domain, got '%s'", cfg.Domain)
	}

	if cfg.Port != "" {
		t.Errorf("Expected empty Port, got '%s'", cfg.Port)
	}
}

func TestConfig_BaseURL_TrailingSlash(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No trailing slash",
			input:    "https://short.ly",
			expected: "https://short.ly/",
		},
		{
			name:     "With trailing slash",
			input:    "https://short.ly/",
			expected: "https://short.ly/",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "/",
		},
		{
			name:     "Just slash",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Multiple trailing slashes",
			input:    "https://short.ly//",
			expected: "https://short.ly//",
		},
	}

	// Save original environment
	originalBaseURL := os.Getenv("BASE_URL")
	defer func() {
		if originalBaseURL == "" {
			os.Unsetenv("BASE_URL")
		} else {
			os.Setenv("BASE_URL", originalBaseURL)
		}
	}()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("BASE_URL", tc.input)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			if cfg.BaseURL != tc.expected {
				t.Errorf("Expected BaseURL '%s', got '%s'", tc.expected, cfg.BaseURL)
			}
		})
	}
}

func TestConfig_BindAddr(t *testing.T) {
	testCases := []struct {
		name     string
		domain   string
		port     string
		expected string
	}{
		{
			name:     "Standard configuration",
			domain:   "localhost",
			port:     "8080",
			expected: "localhost:8080",
		},
		{
			name:     "All interfaces",
			domain:   "0.0.0.0",
			port:     "3000",
			expected: "0.0.0.0:3000",
		},
		{
			name:     "Empty domain",
			domain:   "",
			port:     "8080",
			expected: ":8080",
		},
		{
			name:     "Empty port",
			domain:   "localhost",
			port:     "",
			expected: "localhost:",
		},
		{
			name:     "Both empty",
			domain:   "",
			port:     "",
			expected: ":",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Domain: tc.domain,
				Port:   tc.port,
			}

			bindAddr := cfg.BindAddr()
			if bindAddr != tc.expected {
				t.Errorf("Expected BindAddr '%s', got '%s'", tc.expected, bindAddr)
			}
		})
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := Config{
		DBUser:  "testuser",
		DBPass:  "testpass",
		DBName:  "testdb",
		DBHost:  "localhost",
		DBPort:  "5432",
		SSLMode: "disable",
	}

	expectedDSN := "user=testuser password=testpass dbname=testdb host=localhost port=5432 sslmode=disable"
	dsn := cfg.DSN()

	if dsn != expectedDSN {
		t.Errorf("Expected DSN '%s', got '%s'", expectedDSN, dsn)
	}
}

func TestConfig_DSN_EmptyValues(t *testing.T) {
	cfg := Config{}

	expectedDSN := "user= password= dbname= host= port= sslmode="
	dsn := cfg.DSN()

	if dsn != expectedDSN {
		t.Errorf("Expected DSN '%s', got '%s'", expectedDSN, dsn)
	}
}

func TestConfig_DSN_SpecialCharacters(t *testing.T) {
	cfg := Config{
		DBUser:  "user@domain",
		DBPass:  "pass word!@#$",
		DBName:  "test-db",
		DBHost:  "db.example.com",
		DBPort:  "5432",
		SSLMode: "require",
	}

	expectedDSN := "user=user@domain password=pass word!@#$ dbname=test-db host=db.example.com port=5432 sslmode=require"
	dsn := cfg.DSN()

	if dsn != expectedDSN {
		t.Errorf("Expected DSN '%s', got '%s'", expectedDSN, dsn)
	}
}

func BenchmarkConfig_Load(b *testing.B) {
	// Set up test environment
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_USER_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("BASE_URL", "https://short.ly")
	os.Setenv("DOMAIN", "0.0.0.0")
	os.Setenv("PORT", "8080")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Load()
	}
}

func BenchmarkConfig_DSN(b *testing.B) {
	cfg := Config{
		DBUser:  "testuser",
		DBPass:  "testpass",
		DBName:  "testdb",
		DBHost:  "localhost",
		DBPort:  "5432",
		SSLMode: "disable",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.DSN()
	}
}

func BenchmarkConfig_BindAddr(b *testing.B) {
	cfg := Config{
		Domain: "localhost",
		Port:   "8080",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.BindAddr()
	}
}
