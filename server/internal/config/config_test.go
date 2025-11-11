package config

import (
	"os"
	"testing"
)

func TestGetDSN(t *testing.T) {
	cfg := &Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBName:     "testdb",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	got := cfg.GetDSN()

	if got != expected {
		t.Errorf("GetDSN() = %v, want %v", got, expected)
	}
}

func TestGetServerAddr(t *testing.T) {
	tests := []struct {
		name       string
		serverHost string
		serverPort string
		want       string
	}{
		{
			name:       "default",
			serverHost: "0.0.0.0",
			serverPort: "8091",
			want:       "0.0.0.0:8091",
		},
		{
			name:       "custom port",
			serverHost: "127.0.0.1",
			serverPort: "3000",
			want:       "127.0.0.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ServerHost: tt.serverHost,
				ServerPort: tt.serverPort,
			}

			got := cfg.GetServerAddr()
			if got != tt.want {
				t.Errorf("GetServerAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	origDBPassword := os.Getenv("DB_PASSWORD")
	origJWTSecret := os.Getenv("JWT_SECRET")

	// Restore env vars after test
	defer func() {
		if origDBPassword != "" {
			os.Setenv("DB_PASSWORD", origDBPassword)
		} else {
			os.Unsetenv("DB_PASSWORD")
		}
		if origJWTSecret != "" {
			os.Setenv("JWT_SECRET", origJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	tests := []struct {
		name        string
		dbPassword  string
		jwtSecret   string
		expectError bool
	}{
		{
			name:        "valid config",
			dbPassword:  "testpassword",
			jwtSecret:   "testsecret",
			expectError: false,
		},
		{
			name:        "missing db password",
			dbPassword:  "",
			jwtSecret:   "testsecret",
			expectError: true,
		},
		{
			name:        "missing jwt secret",
			dbPassword:  "testpassword",
			jwtSecret:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbPassword != "" {
				os.Setenv("DB_PASSWORD", tt.dbPassword)
			} else {
				os.Unsetenv("DB_PASSWORD")
			}

			if tt.jwtSecret != "" {
				os.Setenv("JWT_SECRET", tt.jwtSecret)
			} else {
				os.Unsetenv("JWT_SECRET")
			}

			cfg, err := Load()

			if tt.expectError {
				if err == nil {
					t.Error("Load() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
				}
			}
		})
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Set required env vars
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("JWT_SECRET", "testsecret")
	defer func() {
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults
	if cfg.ServerPort != "8091" {
		t.Errorf("ServerPort = %v, want 8091", cfg.ServerPort)
	}

	if cfg.ServerHost != "0.0.0.0" {
		t.Errorf("ServerHost = %v, want 0.0.0.0", cfg.ServerHost)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost = %v, want localhost", cfg.DBHost)
	}

	if cfg.DBPort != "5432" {
		t.Errorf("DBPort = %v, want 5432", cfg.DBPort)
	}

	if cfg.DefaultIngressClass != "nginx" {
		t.Errorf("DefaultIngressClass = %v, want nginx", cfg.DefaultIngressClass)
	}
}
