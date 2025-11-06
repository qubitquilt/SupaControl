package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	service := NewService("test-secret-key")

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "mySecurePassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // Argon2 can hash empty strings
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 1000),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := service.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				if !strings.HasPrefix(hash, "$argon2id$") {
					t.Error("HashPassword() hash doesn't have correct format")
				}
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	service := NewService("test-secret-key")
	password := "mySecurePassword123"

	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			password: "wrongPassword",
			hash:     hash,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.VerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	service := NewService("test-secret-key")

	key1, err := service.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if !strings.HasPrefix(key1, "sk_") {
		t.Error("GenerateAPIKey() key doesn't start with 'sk_'")
	}

	if len(key1) < 40 {
		t.Error("GenerateAPIKey() key is too short")
	}

	// Generate another key to ensure uniqueness
	key2, err := service.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if key1 == key2 {
		t.Error("GenerateAPIKey() generated identical keys")
	}
}

func TestGenerateJWT(t *testing.T) {
	service := NewService("test-secret-key")

	token, err := service.GenerateJWT(1, "testuser", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateJWT() returned empty token")
	}

	// JWT tokens have 3 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("GenerateJWT() token has %d parts, want 3", len(parts))
	}
}

func TestValidateJWT(t *testing.T) {
	service := NewService("test-secret-key")

	userID := int64(1)
	username := "testuser"
	role := "admin"

	token, err := service.GenerateJWT(userID, username, role, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateJWT(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if claims.UserID != userID {
					t.Errorf("ValidateJWT() UserID = %v, want %v", claims.UserID, userID)
				}
				if claims.Username != username {
					t.Errorf("ValidateJWT() Username = %v, want %v", claims.Username, username)
				}
				if claims.Role != role {
					t.Errorf("ValidateJWT() Role = %v, want %v", claims.Role, role)
				}
			}
		})
	}
}

func TestValidateJWTExpired(t *testing.T) {
	service := NewService("test-secret-key")

	// Generate a token that expires immediately
	token, err := service.GenerateJWT(1, "testuser", "admin", -1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	_, err = service.ValidateJWT(token)
	if err == nil {
		t.Error("ValidateJWT() should fail for expired token")
	}
}

func TestValidateJWTWrongSecret(t *testing.T) {
	service1 := NewService("secret1")
	service2 := NewService("secret2")

	token, err := service1.GenerateJWT(1, "testuser", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	_, err = service2.ValidateJWT(token)
	if err == nil {
		t.Error("ValidateJWT() should fail for token signed with different secret")
	}
}
