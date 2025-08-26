package util

import (
	"regexp"
	"testing"
)

func TestGenerateCode(t *testing.T) {
	// Test that the generated code has the correct length
	code := GenerateCode()
	if len(code) != 6 {
		t.Errorf("Expected code length to be 6, got %d", len(code))
	}

	// Test that the code only contains valid characters
	validChars := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !validChars.MatchString(code) {
		t.Errorf("Generated code contains invalid characters: %s", code)
	}

	// Test that multiple calls generate different codes (with high probability)
	codes := make(map[string]bool)
	duplicates := 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		code := GenerateCode()
		if codes[code] {
			duplicates++
		}
		codes[code] = true
	}

	// With 62^6 possible combinations, duplicates should be very rare
	// Allow for a small number of duplicates due to randomness
	if duplicates > iterations/100 { // Allow up to 1% duplicates
		t.Errorf("Too many duplicate codes generated: %d out of %d", duplicates, iterations)
	}
}

func TestGenerateCodeCharacterSet(t *testing.T) {
	// Test that all expected character types appear in generated codes
	foundLowercase := false
	foundUppercase := false
	foundDigits := false

	// Generate many codes to ensure we see all character types
	for i := 0; i < 1000; i++ {
		code := GenerateCode()

		for _, char := range code {
			if char >= 'a' && char <= 'z' {
				foundLowercase = true
			}
			if char >= 'A' && char <= 'Z' {
				foundUppercase = true
			}
			if char >= '0' && char <= '9' {
				foundDigits = true
			}
		}

		// Early exit if we found all types
		if foundLowercase && foundUppercase && foundDigits {
			break
		}
	}

	if !foundLowercase {
		t.Error("Generated codes should contain lowercase letters")
	}
	if !foundUppercase {
		t.Error("Generated codes should contain uppercase letters")
	}
	if !foundDigits {
		t.Error("Generated codes should contain digits")
	}
}

func TestGenerateCodeConsistency(t *testing.T) {
	// Test that the function consistently generates 6-character codes
	for i := 0; i < 100; i++ {
		code := GenerateCode()
		if len(code) != 6 {
			t.Errorf("Iteration %d: Expected code length to be 6, got %d (code: %s)", i, len(code), code)
		}
	}
}

func BenchmarkGenerateCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateCode()
	}
}
