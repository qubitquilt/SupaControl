package apitypes

import "time"

// InstanceStatus represents the state of a Supabase instance
type InstanceStatus string

const (
	StatusProvisioning InstanceStatus = "PROVISIONING"
	StatusRunning      InstanceStatus = "RUNNING"
	StatusDeleting     InstanceStatus = "DELETING"
	StatusFailed       InstanceStatus = "FAILED"
)

// Instance represents a managed Supabase instance
type Instance struct {
	ID           int64          `json:"id" db:"id"`
	ProjectName  string         `json:"project_name" db:"project_name"`
	Namespace    string         `json:"namespace" db:"namespace"`
	Status       InstanceStatus `json:"status" db:"status"`
	StudioURL    string         `json:"studio_url" db:"studio_url"`
	APIURL       string         `json:"api_url" db:"api_url"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	ErrorMessage *string        `json:"error_message,omitempty" db:"error_message"`
}

// CreateInstanceRequest represents the request body for creating a new instance
type CreateInstanceRequest struct {
	Name string `json:"name" validate:"required,min=3,max=63"`
}

// CreateInstanceResponse represents the response for instance creation
type CreateInstanceResponse struct {
	Instance *Instance `json:"instance"`
	Message  string    `json:"message"`
}

// ListInstancesResponse represents the response for listing instances
type ListInstancesResponse struct {
	Instances []*Instance `json:"instances"`
	Count     int         `json:"count"`
}

// GetInstanceResponse represents the response for getting a single instance
type GetInstanceResponse struct {
	Instance *Instance `json:"instance"`
}

// DeleteInstanceResponse represents the response for deleting an instance
type DeleteInstanceResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

// UserInfo represents basic user information
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// APIKey represents an API key
type APIKey struct {
	ID        int64      `json:"id" db:"id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	Name      string     `json:"name" db:"name"`
	KeyHash   string     `json:"-" db:"key_hash"` // Never expose the hash
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsed  *time.Time `json:"last_used,omitempty" db:"last_used"`
}

// CreateAPIKeyRequest represents the request to create a new API key
type CreateAPIKeyRequest struct {
	Name      string     `json:"name" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents the response containing the new API key
type CreateAPIKeyResponse struct {
	Key     string  `json:"key"` // The actual key - only returned once!
	APIKey  *APIKey `json:"api_key"`
	Message string  `json:"message"`
}

// ListAPIKeysResponse represents the response for listing API keys
type ListAPIKeysResponse struct {
	APIKeys []*APIKey `json:"api_keys"`
	Count   int       `json:"count"`
}

// AuthMeResponse represents the response for /auth/me endpoint
type AuthMeResponse struct {
	User   *UserInfo `json:"user,omitempty"`
	APIKey *APIKey   `json:"api_key,omitempty"`
}
