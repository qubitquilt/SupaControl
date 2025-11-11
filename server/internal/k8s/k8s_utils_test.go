package k8s

import (
	"testing"
)

// Note: Full integration tests for Client operations are in k8s_test.go
// These tests focus on utility functions that don't require K8s connectivity

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "32 characters",
			length: 32,
		},
		{
			name:   "64 characters",
			length: 64,
		},
		{
			name:   "16 characters",
			length: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomString(tt.length)
			if err != nil {
				t.Errorf("GenerateRandomString() error = %v", err)
				return
			}

			if len(result) != tt.length {
				t.Errorf("GenerateRandomString() length = %v, want %v", len(result), tt.length)
			}

			// Generate another to ensure uniqueness
			result2, err := GenerateRandomString(tt.length)
			if err != nil {
				t.Errorf("GenerateRandomString() error = %v", err)
				return
			}

			if result == result2 {
				t.Error("GenerateRandomString() generated identical strings")
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	password1, err := GenerateSecurePassword()
	if err != nil {
		t.Fatalf("GenerateSecurePassword() error = %v", err)
	}

	if len(password1) != 32 {
		t.Errorf("GenerateSecurePassword() length = %v, want 32", len(password1))
	}

	// Generate another to ensure uniqueness
	password2, err := GenerateSecurePassword()
	if err != nil {
		t.Fatalf("GenerateSecurePassword() error = %v", err)
	}

	if password1 == password2 {
		t.Error("GenerateSecurePassword() generated identical passwords")
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	secret1, err := GenerateJWTSecret()
	if err != nil {
		t.Fatalf("GenerateJWTSecret() error = %v", err)
	}

	if len(secret1) != 64 {
		t.Errorf("GenerateJWTSecret() length = %v, want 64", len(secret1))
	}

	// Generate another to ensure uniqueness
	secret2, err := GenerateJWTSecret()
	if err != nil {
		t.Fatalf("GenerateJWTSecret() error = %v", err)
	}

	if secret1 == secret2 {
		t.Error("GenerateJWTSecret() generated identical secrets")
	}
}

// Benchmark tests
func BenchmarkGenerateRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateRandomString(32)
	}
}

func BenchmarkGenerateSecurePassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateSecurePassword()
	}
}

func BenchmarkGenerateJWTSecret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateJWTSecret()
	}
}
