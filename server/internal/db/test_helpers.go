// Package db provides database operations for SupaControl.
// This file contains test helper utilities for database testing.
package db

import (
	"os"
	"testing"

	_ "github.com/lib/pq" // PostgreSQL driver for database tests
)

// setupTestDB creates a test database client and runs migrations
// Returns the client and a cleanup function
func setupTestDB(t *testing.T) (*Client, func()) {
	t.Helper()

	// Get test database URL from environment
	testDSN := os.Getenv("TEST_DATABASE_URL")
	if testDSN == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database tests")
	}

	// Create new client
	client, err := NewClient(testDSN)
	if err != nil {
		t.Fatalf("Failed to create test database client: %v", err)
	}

	// Ping to verify connection
	if err := client.Ping(); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Failed to close client after ping failure: %v", closeErr)
		}
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Get migrations path (relative to this file)
	migrationsPath := "migrations"

	// Run migrations
	if err := client.RunMigrations(migrationsPath); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Failed to close client after migration failure: %v", closeErr)
		}
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Cleanup function to close connection and clean tables
	cleanup := func() {
		cleanTestData(t, client)
		if err := client.Close(); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}

	return client, cleanup
}

// cleanTestData removes all test data from tables
func cleanTestData(t *testing.T, client *Client) {
	t.Helper()

	// TRUNCATE is faster than DELETE and resets auto-incrementing counters.
	// CASCADE handles foreign key relationships automatically.
	query := "TRUNCATE TABLE users, api_keys RESTART IDENTITY CASCADE"
	_, err := client.db.Exec(query)
	if err != nil {
		t.Fatalf("Failed to clean test data: %v", err)
	}
}

// createTestUser creates a test user and returns it
func createTestUser(t *testing.T, client *Client, username, passwordHash, role string) *User {
	t.Helper()

	user, err := client.CreateUser(username, passwordHash, role)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// createTestUserWithDefaults creates a test user with default values
func createTestUserWithDefaults(t *testing.T, client *Client) *User {
	t.Helper()
	return createTestUser(t, client, "testuser", "testhash", "admin")
}
