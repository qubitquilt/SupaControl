// Package config provides configuration management for SupaControl.
// It handles loading configuration from environment variables and .env files.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	ServerPort string
	ServerHost string

	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// JWT configuration
	JWTSecret string

	// Kubernetes configuration
	KubeConfig            string // Path to kubeconfig (empty means in-cluster)
	DefaultIngressClass   string
	DefaultIngressDomain  string
	CertManagerIssuer     string // cert-manager ClusterIssuer name for TLS
	LeaderElectionEnabled bool   // Enable leader election for HA deployments

	// Supabase Helm chart configuration
	SupabaseChartRepo    string
	SupabaseChartName    string
	SupabaseChartVersion string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := loadDotEnv(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT", "8091"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "supacontrol"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "supacontrol"),

		JWTSecret: getEnv("JWT_SECRET", ""),

		KubeConfig:            getEnv("KUBECONFIG", ""),
		DefaultIngressClass:   getEnv("DEFAULT_INGRESS_CLASS", "nginx"),
		DefaultIngressDomain:  getEnv("DEFAULT_INGRESS_DOMAIN", "supabase.example.com"),
		CertManagerIssuer:     getEnv("CERT_MANAGER_ISSUER", "letsencrypt-prod"),
		LeaderElectionEnabled: getEnvBool("LEADER_ELECTION_ENABLED", false),

		SupabaseChartRepo:    getEnv("SUPABASE_CHART_REPO", "https://supabase-community.github.io/supabase-kubernetes"),
		SupabaseChartName:    getEnv("SUPABASE_CHART_NAME", "supabase"),
		SupabaseChartVersion: getEnv("SUPABASE_CHART_VERSION", ""),
	}

	// Validate required fields
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// GetDSN returns the PostgreSQL connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

// GetServerAddr returns the server address
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool gets a boolean environment variable with a fallback default value
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Accept "true", "1", "yes" as true, everything else as false
	return value == "true" || value == "1" || value == "yes"
}

// loadDotEnv loads environment variables from .env file
func loadDotEnv() error {
	// Try to load from current directory first
	if err := loadEnvFile(".env"); err != nil {
		return err
	}
	return nil
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		// If file doesn't exist, return without error
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't return it since we're in a deferred function
			// In a real implementation, you might want to use proper logging
			_ = closeErr
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			if setErr := os.Setenv(key, value); setErr != nil {
				// Log the error but continue processing other variables
				// In a real implementation, you might want to use proper logging
				_ = setErr
			}
		}
	}

	return scanner.Err()
}
