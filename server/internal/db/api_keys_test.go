package db

import (
	"testing"
	"time"
)

func TestClient_CreateAPIKey(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test user
	user := createTestUserWithDefaults(t, client)

	tests := []struct {
		name      string
		userID    int64
		keyName   string
		keyHash   string
		expiresAt *time.Time
		wantErr   bool
	}{
		{
			name:      "valid API key without expiration",
			userID:    user.ID,
			keyName:   "test-key-1",
			keyHash:   "hash123",
			expiresAt: nil,
			wantErr:   false,
		},
		{
			name:      "valid API key with expiration",
			userID:    user.ID,
			keyName:   "test-key-2",
			keyHash:   "hash456",
			expiresAt: timePtr(time.Now().Add(24 * time.Hour)),
			wantErr:   false,
		},
		{
			name:      "valid API key with past expiration",
			userID:    user.ID,
			keyName:   "test-key-3",
			keyHash:   "hash789",
			expiresAt: timePtr(time.Now().Add(-24 * time.Hour)),
			wantErr:   false, // Creation should succeed, but GetByHash will return nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := client.CreateAPIKey(tt.userID, tt.keyName, tt.keyHash, tt.expiresAt)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if apiKey == nil {
					t.Fatal("Expected non-nil API key")
				}

				if apiKey.ID == 0 {
					t.Error("Expected non-zero API key ID")
				}

				if apiKey.UserID != tt.userID {
					t.Errorf("UserID = %v, want %v", apiKey.UserID, tt.userID)
				}

				if apiKey.Name != tt.keyName {
					t.Errorf("Name = %v, want %v", apiKey.Name, tt.keyName)
				}

				if apiKey.KeyHash != tt.keyHash {
					t.Errorf("KeyHash = %v, want %v", apiKey.KeyHash, tt.keyHash)
				}

				if apiKey.CreatedAt.IsZero() {
					t.Error("Expected non-zero CreatedAt")
				}

				if tt.expiresAt != nil {
					if apiKey.ExpiresAt == nil {
						t.Error("Expected non-nil ExpiresAt")
					} else if !apiKey.ExpiresAt.Equal(*tt.expiresAt) {
						// Allow for small time differences (database rounding)
						diff := apiKey.ExpiresAt.Sub(*tt.expiresAt)
						if diff > time.Second || diff < -time.Second {
							t.Errorf("ExpiresAt = %v, want %v", apiKey.ExpiresAt, tt.expiresAt)
						}
					}
				} else {
					if apiKey.ExpiresAt != nil {
						t.Error("Expected nil ExpiresAt")
					}
				}

				if apiKey.LastUsed != nil {
					t.Error("Expected nil LastUsed for new key")
				}
			}
		})
	}
}

func TestClient_CreateAPIKey_InvalidUserID(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Try to create API key for non-existent user
	_, err := client.CreateAPIKey(99999, "test-key", "hash123", nil)
	if err == nil {
		t.Error("Expected error for invalid user ID")
	}
}

func TestClient_CreateAPIKey_DuplicateHash(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create first API key
	_, err := client.CreateAPIKey(user.ID, "key1", "duplicatehash", nil)
	if err != nil {
		t.Fatalf("Failed to create first API key: %v", err)
	}

	// Try to create second API key with same hash
	_, err = client.CreateAPIKey(user.ID, "key2", "duplicatehash", nil)
	if err == nil {
		t.Error("Expected error for duplicate key hash")
	}
}

func TestClient_GetAPIKeyByHash(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create test API keys
	validKey, _ := client.CreateAPIKey(user.ID, "valid-key", "validhash", nil)
	_, _ = client.CreateAPIKey(user.ID, "expired-key", "expiredhash",
		timePtr(time.Now().Add(-24*time.Hour)))

	tests := []struct {
		name    string
		keyHash string
		wantNil bool
		wantErr bool
		wantID  int64
	}{
		{
			name:    "existing valid key",
			keyHash: "validhash",
			wantNil: false,
			wantErr: false,
			wantID:  validKey.ID,
		},
		{
			name:    "expired key returns nil",
			keyHash: "expiredhash",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "non-existent key",
			keyHash: "nonexistent",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "empty hash",
			keyHash: "",
			wantNil: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := client.GetAPIKeyByHash(tt.keyHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAPIKeyByHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (apiKey == nil) != tt.wantNil {
				t.Errorf("GetAPIKeyByHash() key = %v, wantNil %v", apiKey, tt.wantNil)
				return
			}

			if apiKey != nil && apiKey.ID != tt.wantID {
				t.Errorf("ID = %v, want %v", apiKey.ID, tt.wantID)
			}
		})
	}
}

func TestClient_GetAPIKeyByID(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create test API key
	created, err := client.CreateAPIKey(user.ID, "test-key", "testhash", nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	tests := []struct {
		name    string
		id      int64
		wantNil bool
		wantErr bool
	}{
		{
			name:    "existing key",
			id:      created.ID,
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "non-existent key",
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
			apiKey, err := client.GetAPIKeyByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAPIKeyByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (apiKey == nil) != tt.wantNil {
				t.Errorf("GetAPIKeyByID() key = %v, wantNil %v", apiKey, tt.wantNil)
				return
			}

			if apiKey != nil {
				if apiKey.ID != created.ID {
					t.Errorf("ID = %v, want %v", apiKey.ID, created.ID)
				}
				if apiKey.Name != created.Name {
					t.Errorf("Name = %v, want %v", apiKey.Name, created.Name)
				}
			}
		})
	}
}

func TestClient_ListAPIKeysByUser(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user1 := createTestUser(t, client, "user1", "hash1", "admin")
	user2 := createTestUser(t, client, "user2", "hash2", "admin")

	// Create API keys for user1
	_, err := client.CreateAPIKey(user1.ID, "key1", "hash1", nil)
	if err != nil {
		t.Fatalf("Failed to create key1: %v", err)
	}
	_, err = client.CreateAPIKey(user1.ID, "key2", "hash2", nil)
	if err != nil {
		t.Fatalf("Failed to create key2: %v", err)
	}

	// Create API key for user2
	_, err = client.CreateAPIKey(user2.ID, "key3", "hash3", nil)
	if err != nil {
		t.Fatalf("Failed to create key3: %v", err)
	}

	tests := []struct {
		name      string
		userID    int64
		wantCount int
		wantErr   bool
	}{
		{
			name:      "user with multiple keys",
			userID:    user1.ID,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "user with single key",
			userID:    user2.ID,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "user with no keys",
			userID:    99999,
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := client.ListAPIKeysByUser(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListAPIKeysByUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(keys) != tt.wantCount {
				t.Errorf("ListAPIKeysByUser() count = %v, want %v", len(keys), tt.wantCount)
			}

			// Verify all keys belong to the user
			for _, key := range keys {
				if key.UserID != tt.userID {
					t.Errorf("Key UserID = %v, want %v", key.UserID, tt.userID)
				}
			}
		})
	}
}

func TestClient_ListAPIKeysByUser_OrderedByCreatedAt(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create keys with slight delays to ensure different timestamps
	key1, _ := client.CreateAPIKey(user.ID, "key1", "hash1", nil)
	time.Sleep(10 * time.Millisecond)
	key2, _ := client.CreateAPIKey(user.ID, "key2", "hash2", nil)
	time.Sleep(10 * time.Millisecond)
	key3, _ := client.CreateAPIKey(user.ID, "key3", "hash3", nil)

	keys, err := client.ListAPIKeysByUser(user.ID)
	if err != nil {
		t.Fatalf("ListAPIKeysByUser() failed: %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Expected 3 keys, got %d", len(keys))
	}

	// Verify descending order (newest first)
	if keys[0].ID != key3.ID {
		t.Errorf("First key ID = %v, want %v (newest)", keys[0].ID, key3.ID)
	}
	if keys[1].ID != key2.ID {
		t.Errorf("Second key ID = %v, want %v", keys[1].ID, key2.ID)
	}
	if keys[2].ID != key1.ID {
		t.Errorf("Third key ID = %v, want %v (oldest)", keys[2].ID, key1.ID)
	}
}

func TestClient_ListAllAPIKeys(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user1 := createTestUser(t, client, "user1", "hash1", "admin")
	user2 := createTestUser(t, client, "user2", "hash2", "admin")

	// Create API keys for both users
	_, _ = client.CreateAPIKey(user1.ID, "key1", "hash1key", nil)
	_, _ = client.CreateAPIKey(user1.ID, "key2", "hash2key", nil)
	_, _ = client.CreateAPIKey(user2.ID, "key3", "hash3key", nil)

	keys, err := client.ListAllAPIKeys()
	if err != nil {
		t.Fatalf("ListAllAPIKeys() failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("ListAllAPIKeys() count = %v, want 3", len(keys))
	}

	// Verify we got keys from both users
	userIDs := make(map[int64]bool)
	for _, key := range keys {
		userIDs[key.UserID] = true
	}

	if !userIDs[user1.ID] {
		t.Error("Expected keys from user1")
	}
	if !userIDs[user2.ID] {
		t.Error("Expected keys from user2")
	}
}

func TestClient_ListAllAPIKeys_Empty(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	keys, err := client.ListAllAPIKeys()
	if err != nil {
		t.Fatalf("ListAllAPIKeys() failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("ListAllAPIKeys() count = %v, want 0", len(keys))
	}
}

func TestClient_UpdateAPIKeyLastUsed(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create API key
	key, err := client.CreateAPIKey(user.ID, "test-key", "testhash", nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	if key.LastUsed != nil {
		t.Fatal("Expected nil LastUsed for new key")
	}

	// Update last used
	err = client.UpdateAPIKeyLastUsed(key.ID)
	if err != nil {
		t.Fatalf("UpdateAPIKeyLastUsed() failed: %v", err)
	}

	// Verify last used was updated
	updated, err := client.GetAPIKeyByID(key.ID)
	if err != nil {
		t.Fatalf("Failed to get updated key: %v", err)
	}

	if updated.LastUsed == nil {
		t.Fatal("Expected non-nil LastUsed after update")
	}

	if updated.LastUsed.IsZero() {
		t.Error("Expected non-zero LastUsed timestamp")
	}

	// Verify timestamp is recent (within last minute)
	if time.Since(*updated.LastUsed) > time.Minute {
		t.Errorf("LastUsed timestamp too old: %v", updated.LastUsed)
	}
}

func TestClient_UpdateAPIKeyLastUsed_NonExistent(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Update non-existent key (should succeed but affect 0 rows)
	err := client.UpdateAPIKeyLastUsed(99999)
	if err != nil {
		t.Errorf("UpdateAPIKeyLastUsed() unexpected error: %v", err)
	}
}

func TestClient_DeleteAPIKey(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create API key
	key, err := client.CreateAPIKey(user.ID, "test-key", "testhash", nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Delete the key
	err = client.DeleteAPIKey(key.ID)
	if err != nil {
		t.Fatalf("DeleteAPIKey() failed: %v", err)
	}

	// Verify key is deleted
	deleted, err := client.GetAPIKeyByID(key.ID)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if deleted != nil {
		t.Error("Expected key to be deleted")
	}
}

func TestClient_DeleteAPIKey_NotFound(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	// Try to delete non-existent key
	err := client.DeleteAPIKey(99999)
	if err == nil {
		t.Error("Expected error when deleting non-existent key")
	}
}

func TestClient_DeleteExpiredAPIKeys(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create various API keys
	_, _ = client.CreateAPIKey(user.ID, "valid-key", "validhash", nil)
	_, _ = client.CreateAPIKey(user.ID, "future-key", "futurehash",
		timePtr(time.Now().Add(24*time.Hour)))
	expiredKey1, _ := client.CreateAPIKey(user.ID, "expired-key-1", "expiredhash1",
		timePtr(time.Now().Add(-24*time.Hour)))
	expiredKey2, _ := client.CreateAPIKey(user.ID, "expired-key-2", "expiredhash2",
		timePtr(time.Now().Add(-48*time.Hour)))

	// Delete expired keys
	count, err := client.DeleteExpiredAPIKeys()
	if err != nil {
		t.Fatalf("DeleteExpiredAPIKeys() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("DeleteExpiredAPIKeys() count = %v, want 2", count)
	}

	// Verify expired keys are deleted
	deleted1, _ := client.GetAPIKeyByID(expiredKey1.ID)
	if deleted1 != nil {
		t.Error("Expected expired key 1 to be deleted")
	}

	deleted2, _ := client.GetAPIKeyByID(expiredKey2.ID)
	if deleted2 != nil {
		t.Error("Expected expired key 2 to be deleted")
	}

	// Verify valid keys still exist
	remaining, err := client.ListAPIKeysByUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to list remaining keys: %v", err)
	}

	if len(remaining) != 2 {
		t.Errorf("Remaining keys count = %v, want 2", len(remaining))
	}
}

func TestClient_DeleteExpiredAPIKeys_NoneExpired(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	user := createTestUserWithDefaults(t, client)

	// Create only valid keys
	_, _ = client.CreateAPIKey(user.ID, "key1", "hash1validkey", nil)
	_, _ = client.CreateAPIKey(user.ID, "key2", "hash2validkey", timePtr(time.Now().Add(24*time.Hour)))

	count, err := client.DeleteExpiredAPIKeys()
	if err != nil {
		t.Fatalf("DeleteExpiredAPIKeys() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("DeleteExpiredAPIKeys() count = %v, want 0", count)
	}
}

func TestClient_DeleteExpiredAPIKeys_Empty(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := client.DeleteExpiredAPIKeys()
	if err != nil {
		t.Fatalf("DeleteExpiredAPIKeys() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("DeleteExpiredAPIKeys() count = %v, want 0", count)
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
