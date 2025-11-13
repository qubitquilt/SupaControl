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
			if err := os.Setenv("DB_PASSWORD", origDBPassword); err != nil {
				t.Errorf("Failed to restore DB_PASSWORD: %v", err)
			}
		} else {
			if err := os.Unsetenv("DB_PASSWORD"); err != nil {
				t.Errorf("Failed to unset DB_PASSWORD: %v", err)
			}
		}
		if origJWTSecret != "" {
			if err := os.Setenv("JWT_SECRET", origJWTSecret); err != nil {
				t.Errorf("Failed to restore JWT_SECRET: %v", err)
			}
		} else {
			if err := os.Unsetenv("JWT_SECRET"); err != nil {
				t.Errorf("Failed to unset JWT_SECRET: %v", err)
			}
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
				if err := os.Setenv("DB_PASSWORD", tt.dbPassword); err != nil {
					t.Fatalf("Failed to set DB_PASSWORD: %v", err)
				}
			} else {
				if err := os.Unsetenv("DB_PASSWORD"); err != nil {
					t.Fatalf("Failed to unset DB_PASSWORD: %v", err)
				}
			}

			if tt.jwtSecret != "" {
				if err := os.Setenv("JWT_SECRET", tt.jwtSecret); err != nil {
					t.Fatalf("Failed to set JWT_SECRET: %v", err)
				}
			} else {
				if err := os.Unsetenv("JWT_SECRET"); err != nil {
					t.Fatalf("Failed to unset JWT_SECRET: %v", err)
				}
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
	if err := os.Setenv("DB_PASSWORD", "testpass"); err != nil {
		t.Fatalf("Failed to set DB_PASSWORD: %v", err)
	}
	if err := os.Setenv("JWT_SECRET", "testsecret"); err != nil {
		t.Fatalf("Failed to set JWT_SECRET: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("DB_PASSWORD"); err != nil {
			t.Errorf("Failed to unset DB_PASSWORD: %v", err)
		}
		if err := os.Unsetenv("JWT_SECRET"); err != nil {
			t.Errorf("Failed to unset JWT_SECRET: %v", err)
		}
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
