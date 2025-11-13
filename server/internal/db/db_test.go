package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "invalid DSN format",
			dsn:     "invalid-dsn",
			wantErr: true,
		},
		{
			name:    "invalid host",
			dsn:     "postgres://user:pass@invalid-host:5432/db?sslmode=disable",
			wantErr: true,
		},
		{
			name:    "empty DSN",
			dsn:     "",
			wantErr: true,
		},
		{
			name:    "unreachable database",
			dsn:     "postgres://user:pass@10.255.255.1:5432/db?sslmode=disable",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
			if client != nil {
				if err := client.Close(); err != nil {
					t.Errorf("Failed to close client: %v", err)
				}
			}
		})
	}
}

func TestNewClient_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.db == nil {
		t.Fatal("Expected non-nil underlying database connection")
	}
}

func TestClient_Ping(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	err := client.Ping()
	if err != nil {
		t.Errorf("Ping() failed: %v", err)
	}
}

func TestClient_Close(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// First close should succeed
	err := client.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Operations after close should fail
	err = client.Ping()
	if err == nil {
		t.Error("Expected Ping() to fail after Close()")
	}
}

func TestClient_GetDB(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	db := client.GetDB()
	if db == nil {
		t.Error("GetDB() returned nil")
		return
	}

	// Verify it's the same underlying connection
	if db != client.db {
		t.Error("GetDB() returned different connection than client.db")
	}

	// Verify we can use the returned DB
	err := db.Ping()
	if err != nil {
		t.Errorf("Ping on returned DB failed: %v", err)
	}
}

func TestClient_ConnectionPoolConfiguration(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	db := client.GetDB()
	stats := db.Stats()

	// Verify connection pool settings
	if stats.MaxOpenConnections != 25 {
		t.Errorf("MaxOpenConnections = %d, want 25", stats.MaxOpenConnections)
	}

	// Note: MaxIdleConnections cannot be directly verified from DBStats
	// as it's not exposed, but we verify MaxOpenConnections is set correctly
}

func TestClient_CreateUser(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name         string
		username     string
		passwordHash string
		role         string
		wantErr      bool
	}{
		{
			name:         "valid admin user",
			username:     "admin1",
			passwordHash: "hash123",
			role:         "admin",
			wantErr:      false,
		},
		{
			name:         "valid operator user",
			username:     "operator1",
			passwordHash: "hash456",
			role:         "operator",
			wantErr:      false,
		},
		{
			name:         "duplicate username",
			username:     "duplicate",
			passwordHash: "hash789",
			role:         "admin",
			wantErr:      false, // First creation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := client.CreateUser(tt.username, tt.passwordHash, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if user == nil {
					t.Error("Expected non-nil user")
					return
				}

				if user.Username != tt.username {
					t.Errorf("Username = %v, want %v", user.Username, tt.username)
				}

				if user.PasswordHash != tt.passwordHash {
					t.Errorf("PasswordHash = %v, want %v", user.PasswordHash, tt.passwordHash)
				}

				if user.Role != tt.role {
					t.Errorf("Role = %v, want %v", user.Role, tt.role)
				}

				if user.ID == 0 {
					t.Error("Expected non-zero user ID")
				}

				if user.CreatedAt == "" {
					t.Error("Expected non-empty CreatedAt")
				}

				if user.UpdatedAt == "" {
					t.Error("Expected non-empty UpdatedAt")
				}
			}
		})
	}

	// Test duplicate username error
	t.Run("duplicate username error", func(t *testing.T) {
		_, err := client.CreateUser("duplicate", "hash999", "admin")
		if err == nil {
			t.Error("Expected error for duplicate username")
		}
	})

	// Test database error during CreateUser
	t.Run("database error during CreateUser", func(t *testing.T) {
		// Create a separate client for error simulation
		testDSN := os.Getenv("TEST_DATABASE_URL")
		if testDSN == "" {
			t.Skip("TEST_DATABASE_URL not set")
		}
		errorClient, err := NewClient(testDSN)
		if err != nil {
			t.Fatalf("Failed to create error client: %v", err)
		}
		// Close it immediately to simulate failure
		_ = errorClient.Close()

		_, err = errorClient.CreateUser("failuser", "failhash", "admin")
		if err == nil {
			t.Error("Expected error when database connection is closed")
		}
	})
}

func TestClient_GetUserByUsername(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test user
	created := createTestUser(t, client, "testuser", "testhash", "admin")

	tests := []struct {
		name     string
		username string
		wantNil  bool
		wantErr  bool
	}{
		{
			name:     "existing user",
			username: "testuser",
			wantNil:  false,
			wantErr:  false,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			wantNil:  true,
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			wantNil:  true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := client.GetUserByUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (user == nil) != tt.wantNil {
				t.Errorf("GetUserByUsername() user = %v, wantNil %v", user, tt.wantNil)
				return
			}

			if user != nil {
				if user.ID != created.ID {
					t.Errorf("ID = %v, want %v", user.ID, created.ID)
				}
				if user.Username != created.Username {
					t.Errorf("Username = %v, want %v", user.Username, created.Username)
				}
			}
		})
	}

	// Test database error during GetUserByUsername
	t.Run("database error during GetUserByUsername", func(t *testing.T) {
		// Create a separate client for error simulation
		testDSN := os.Getenv("TEST_DATABASE_URL")
		if testDSN == "" {
			t.Skip("TEST_DATABASE_URL not set")
		}
		errorClient, err := NewClient(testDSN)
		if err != nil {
			t.Fatalf("Failed to create error client: %v", err)
		}
		// Close it immediately to simulate failure
		_ = errorClient.Close()

		_, err = errorClient.GetUserByUsername("testuser")
		if err == nil {
			t.Error("Expected error when database connection is closed")
		}
	})
}

func TestClient_GetUserByID(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test user
	created := createTestUser(t, client, "testuser", "testhash", "admin")

	tests := []struct {
		name    string
		id      int64
		wantNil bool
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      created.ID,
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      99999,
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "zero ID",
			id:      0,
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "negative ID",
			id:      -1,
			wantNil: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := client.GetUserByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (user == nil) != tt.wantNil {
				t.Errorf("GetUserByID() user = %v, wantNil %v", user, tt.wantNil)
				return
			}

			if user != nil {
				if user.ID != created.ID {
					t.Errorf("ID = %v, want %v", user.ID, created.ID)
				}
				if user.Username != created.Username {
					t.Errorf("Username = %v, want %v", user.Username, created.Username)
				}
			}
		})
	}

	// Test database error during GetUserByID
	t.Run("database error during GetUserByID", func(t *testing.T) {
		// Create a separate client for error simulation
		testDSN := os.Getenv("TEST_DATABASE_URL")
		if testDSN == "" {
			t.Skip("TEST_DATABASE_URL not set")
		}
		errorClient, err := NewClient(testDSN)
		if err != nil {
			t.Fatalf("Failed to create error client: %v", err)
		}
		// Close it immediately to simulate failure
		_ = errorClient.Close()

		_, err = errorClient.GetUserByID(created.ID)
		if err == nil {
			t.Error("Expected error when database connection is closed")
		}
	})
}

func TestClient_WithinTransaction_Success(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	var userID int64

	err := client.WithinTransaction(func(tx *sqlx.Tx) error {
		// Create a user within transaction
		var user User
		err := tx.QueryRowx(
			`INSERT INTO users (username, password_hash, role)
			 VALUES ($1, $2, $3)
			 RETURNING *`,
			"txuser", "txhash", "admin",
		).StructScan(&user)
		if err != nil {
			return err
		}

		userID = user.ID
		return nil
	})

	if err != nil {
		t.Fatalf("WithinTransaction() failed: %v", err)
	}

	// Verify user was committed
	user, err := client.GetUserByID(userID)
	if err != nil {
		t.Fatalf("Failed to get user after transaction: %v", err)
	}

	if user == nil {
		t.Fatal("Expected user to be committed")
	}

	if user.Username != "txuser" {
		t.Errorf("Username = %v, want %v", user.Username, "txuser")
	}
}

func TestClient_WithinTransaction_Rollback(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	testErr := errors.New("test error")
	var userID int64

	err := client.WithinTransaction(func(tx *sqlx.Tx) error {
		// Create a user within transaction
		var user User
		err := tx.QueryRowx(
			`INSERT INTO users (username, password_hash, role)
			 VALUES ($1, $2, $3)
			 RETURNING *`,
			"rollbackuser", "rollbackhash", "admin",
		).StructScan(&user)
		if err != nil {
			return err
		}

		userID = user.ID

		// Return error to trigger rollback
		return testErr
	})

	if err != testErr {
		t.Errorf("WithinTransaction() error = %v, want %v", err, testErr)
	}

	// Verify user was NOT committed
	user, err := client.GetUserByID(userID)
	if err != nil {
		t.Fatalf("Failed to query user after rollback: %v", err)
	}

	if user != nil {
		t.Error("Expected user to be rolled back, but found in database")
	}
}

func TestClient_WithinTransaction_Panic(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	var userID int64

	// Test panic recovery
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to be re-raised")
			}
		}()

		_ = client.WithinTransaction(func(tx *sqlx.Tx) error {
			// Create a user within transaction
			var user User
			err := tx.QueryRowx(
				`INSERT INTO users (username, password_hash, role)
				 VALUES ($1, $2, $3)
				 RETURNING *`,
				"panicuser", "panichash", "admin",
			).StructScan(&user)
			if err != nil {
				return err
			}

			userID = user.ID

			// Trigger panic
			panic("test panic")
		})
	}()

	// Verify user was rolled back after panic
	user, err := client.GetUserByID(userID)
	if err != nil {
		t.Fatalf("Failed to query user after panic: %v", err)
	}

	if user != nil {
		t.Error("Expected user to be rolled back after panic, but found in database")
	}
}

func TestClient_WithinTransaction_CommitError(t *testing.T) {
	// Create a separate client for this test to avoid interfering with shared connections
	testDSN := os.Getenv("TEST_DATABASE_URL")
	if testDSN == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	client, err := NewClient(testDSN)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Failed to close test client: %v", closeErr)
		}
	}()

	// Test commit failure by closing the database connection during transaction
	// This simulates a connection loss that would cause commit to fail
	var commitError error
	err = client.WithinTransaction(func(tx *sqlx.Tx) error {
		// Perform some operation
		_, err := tx.Exec("SELECT 1")
		if err != nil {
			return err
		}

		// Close the underlying database connection to force commit failure
		_ = client.db.Close()

		return nil
	})

	// The transaction should fail during commit due to closed connection
	if err == nil {
		t.Error("Expected WithinTransaction to fail due to commit error after connection close")
	}

	// Store the error for verification
	commitError = err

	if commitError != nil && !contains(commitError.Error(), "commit") && !contains(commitError.Error(), "closed") {
		t.Errorf("Expected commit-related error, got: %v", commitError)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || strings.Contains(s, substr)))
}

func TestClient_BeginTx(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	tx, err := client.BeginTx()
	if err != nil {
		t.Fatalf("BeginTx() failed: %v", err)
	}

	if tx == nil {
		t.Fatal("Expected non-nil transaction")
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil {
			t.Errorf("Failed to rollback transaction: %v", rbErr)
		}
	}()

	// Test that we can use the transaction
	_, err = tx.Exec("SELECT 1")
	if err != nil {
		t.Errorf("Failed to execute query on transaction: %v", err)
	}
}

func TestClient_BeginTx_AfterClose(t *testing.T) {
	// Create a separate client for this test
	testDSN := os.Getenv("TEST_DATABASE_URL")
	if testDSN == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	client, err := NewClient(testDSN)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Failed to close test client: %v", closeErr)
		}
	}()

	// Close the client
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}

	// BeginTx should fail after Close
	_, err = client.BeginTx()
	if err == nil {
		t.Error("Expected BeginTx to fail after Close")
	}
}

func TestClient_RunMigrations(t *testing.T) {
	// Get test database URL from environment
	testDSN := os.Getenv("TEST_DATABASE_URL")
	if testDSN == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping migration tests")
	}

	// Create fresh client (don't use setupTestDB as it runs migrations)
	client, err := NewClient(testDSN)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Failed to close client: %v", closeErr)
		}
	}()

	// Clean all tables first
	tables := []string{"api_keys", "users"}
	for _, table := range tables {
		_, _ = client.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}

	// Test migration
	migrationsPath := "migrations"
	err = client.RunMigrations(migrationsPath)
	if err != nil {
		t.Fatalf("RunMigrations() failed: %v", err)
	}

	// Verify tables exist
	expectedTables := []string{"users", "api_keys"}
	for _, table := range expectedTables {
		var exists bool
		err := client.db.Get(&exists,
			`SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)`, table)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Expected table %s to exist after migrations", table)
		}
	}

	// Test running migrations again (should be idempotent)
	err = client.RunMigrations(migrationsPath)
	if err != nil {
		t.Errorf("RunMigrations() second run failed: %v", err)
	}
}

func TestClient_RunMigrations_InvalidPath(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	err := client.RunMigrations("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for invalid migrations path")
	}
}

func TestClient_RunMigrations_InvalidSQL(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a temporary directory for test migrations
	tempDir := t.TempDir()

	// Create a migration file with invalid SQL
	invalidSQL := "INVALID SQL STATEMENT THAT WILL FAIL"
	migrationFile := filepath.Join(tempDir, "001_invalid.sql")
	err := os.WriteFile(migrationFile, []byte(invalidSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create test migration file: %v", err)
	}

	// Test that RunMigrations fails with invalid SQL
	err = client.RunMigrations(tempDir)
	if err == nil {
		t.Error("Expected error for invalid SQL in migration file")
	}
}

func TestClient_RunMigrations_FileReadError(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a temporary directory for test migrations
	tempDir := t.TempDir()

	// Create a migration file
	validSQL := "SELECT 1;"
	migrationFile := filepath.Join(tempDir, "001_test.sql")
	err := os.WriteFile(migrationFile, []byte(validSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create test migration file: %v", err)
	}

	// Remove read permissions from the file (if possible)
	// Note: This may not work on all systems, but we can still test the error handling
	err = os.Chmod(migrationFile, 0000)
	if err != nil {
		// If we can't change permissions, skip this test
		t.Skip("Cannot change file permissions for test")
		return
	}
	defer func() {
		// Restore permissions for cleanup
		_ = os.Chmod(migrationFile, 0644)
	}()

	// Test that RunMigrations fails when file cannot be read
	err = client.RunMigrations(tempDir)
	if err == nil {
		t.Error("Expected error when migration file cannot be read")
	}
}
