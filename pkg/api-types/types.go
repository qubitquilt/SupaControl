package apitypes

import "time"

// UserInfo represents user information
type UserInfo struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

// AuthMeResponse represents an auth/me response
type AuthMeResponse struct {
	User *UserInfo `json:"user"`
}

// CreateAPIKeyRequest represents an API key creation request
type CreateAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents an API key creation response
type CreateAPIKeyResponse struct {
	Key     string  `json:"key"`
	APIKey  *APIKey `json:"key"`
	Message string  `json:"message"`
}

// APIKey represents an API key
type APIKey struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	Hash       string     `json:"hash"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// ListAPIKeysResponse represents a list API keys response
type ListAPIKeysResponse struct {
	APIKeys []*APIKey `json:"api_keys"`
	Count   int       `json:"count"`
}

// InstanceStatus represents the status of an instance
type InstanceStatus string

const (
	StatusProvisioning InstanceStatus = "provisioning"
	StatusRunning      InstanceStatus = "running"
	StatusDeleting     InstanceStatus = "deleting"
	StatusFailed       InstanceStatus = "failed"
)

// Instance represents a Supabase instance
type Instance struct {
	ProjectName  string         `json:"project_name"`
	Namespace    string         `json:"namespace"`
	Status       InstanceStatus `json:"status"`
	StudioURL    string         `json:"studio_url,omitempty"`
	APIURL       string         `json:"api_url,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at,omitempty"`
	ErrorMessage *string        `json:"error_message,omitempty"`
}

// CreateInstanceRequest represents an instance creation request
type CreateInstanceRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateInstanceResponse represents an instance creation response
type CreateInstanceResponse struct {
	Instance *Instance `json:"instance"`
	Message  string    `json:"message"`
}

// ListInstancesResponse represents a list instances response
type ListInstancesResponse struct {
	Instances []*Instance `json:"instances"`
	Count     int         `json:"count"`
}

// GetInstanceResponse represents a get instance response
type GetInstanceResponse struct {
	Instance *Instance `json:"instance"`
}

// DeleteInstanceResponse represents a delete instance response
type DeleteInstanceResponse struct {
	Message string `json:"message"`
}
