package config

import (
	"fmt"
	"os"
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
	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
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
