package db

import (
	"database/sql"
	"fmt"
	"time"

	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
)

// CreateAPIKey creates a new API key in the database
func (c *Client) CreateAPIKey(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error) {
	var apiKey apitypes.APIKey

	query := `
		INSERT INTO api_keys (user_id, name, key_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, key_hash, created_at, expires_at, last_used
	`

	err := c.db.QueryRowx(query, userID, name, keyHash, expiresAt).StructScan(&apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &apiKey, nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (c *Client) GetAPIKeyByHash(keyHash string) (*apitypes.APIKey, error) {
	var apiKey apitypes.APIKey

	query := `SELECT * FROM api_keys WHERE key_hash = $1`

	err := c.db.Get(&apiKey, query, keyHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, nil // Treat expired keys as non-existent
	}

	return &apiKey, nil
}

// GetAPIKeyByID retrieves an API key by its ID
func (c *Client) GetAPIKeyByID(id int64) (*apitypes.APIKey, error) {
	var apiKey apitypes.APIKey

	query := `SELECT * FROM api_keys WHERE id = $1`

	err := c.db.Get(&apiKey, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// ListAPIKeysByUser retrieves all API keys for a user
func (c *Client) ListAPIKeysByUser(userID int64) ([]*apitypes.APIKey, error) {
	var apiKeys []*apitypes.APIKey

	query := `SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`

	err := c.db.Select(&apiKeys, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return apiKeys, nil
}

// ListAllAPIKeys retrieves all API keys (admin function)
func (c *Client) ListAllAPIKeys() ([]*apitypes.APIKey, error) {
	var apiKeys []*apitypes.APIKey

	query := `SELECT * FROM api_keys ORDER BY created_at DESC`

	err := c.db.Select(&apiKeys, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return apiKeys, nil
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key
func (c *Client) UpdateAPIKeyLastUsed(id int64) error {
	query := `UPDATE api_keys SET last_used = NOW() WHERE id = $1`

	_, err := c.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key
func (c *Client) DeleteAPIKey(id int64) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := c.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// DeleteExpiredAPIKeys deletes all expired API keys
func (c *Client) DeleteExpiredAPIKeys() (int64, error) {
	query := `DELETE FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < NOW()`

	result, err := c.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired API keys: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
