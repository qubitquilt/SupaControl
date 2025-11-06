package db

import (
	"database/sql"
	"fmt"

	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
)

// StoreInstance creates a new instance record in the database
func (c *Client) StoreInstance(instance *apitypes.Instance) error {
	query := `
		INSERT INTO instances (project_name, namespace, status, studio_url, api_url, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := c.db.QueryRowx(
		query,
		instance.ProjectName,
		instance.Namespace,
		instance.Status,
		instance.StudioURL,
		instance.APIURL,
		instance.ErrorMessage,
	).Scan(&instance.ID, &instance.CreatedAt, &instance.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store instance: %w", err)
	}

	return nil
}

// GetInstance retrieves an instance by project name
func (c *Client) GetInstance(projectName string) (*apitypes.Instance, error) {
	var instance apitypes.Instance

	query := `SELECT * FROM instances WHERE project_name = $1`

	err := c.db.Get(&instance, query, projectName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// GetInstanceByID retrieves an instance by ID
func (c *Client) GetInstanceByID(id int64) (*apitypes.Instance, error) {
	var instance apitypes.Instance

	query := `SELECT * FROM instances WHERE id = $1`

	err := c.db.Get(&instance, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// ListInstances retrieves all instances
func (c *Client) ListInstances() ([]*apitypes.Instance, error) {
	var instances []*apitypes.Instance

	query := `SELECT * FROM instances ORDER BY created_at DESC`

	err := c.db.Select(&instances, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, nil
}

// UpdateInstanceStatus updates the status of an instance
func (c *Client) UpdateInstanceStatus(projectName string, status apitypes.InstanceStatus, errorMessage *string) error {
	query := `
		UPDATE instances
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE project_name = $3
	`

	result, err := c.db.Exec(query, status, errorMessage, projectName)
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", projectName)
	}

	return nil
}

// UpdateInstance updates an instance's details
func (c *Client) UpdateInstance(instance *apitypes.Instance) error {
	query := `
		UPDATE instances
		SET namespace = $1, status = $2, studio_url = $3, api_url = $4, error_message = $5, updated_at = NOW()
		WHERE project_name = $6
	`

	result, err := c.db.Exec(
		query,
		instance.Namespace,
		instance.Status,
		instance.StudioURL,
		instance.APIURL,
		instance.ErrorMessage,
		instance.ProjectName,
	)
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", instance.ProjectName)
	}

	return nil
}

// DeleteInstance removes an instance from the database
func (c *Client) DeleteInstance(projectName string) error {
	query := `DELETE FROM instances WHERE project_name = $1`

	result, err := c.db.Exec(query, projectName)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", projectName)
	}

	return nil
}

// InstanceExists checks if an instance exists by project name
func (c *Client) InstanceExists(projectName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM instances WHERE project_name = $1)`

	err := c.db.Get(&exists, query, projectName)
	if err != nil {
		return false, fmt.Errorf("failed to check instance existence: %w", err)
	}

	return exists, nil
}
