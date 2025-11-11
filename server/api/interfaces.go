package api

import (
	"context"
	"time"

	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
)

// DBClient defines the database operations needed by API handlers
// This interface allows for easy mocking in tests
type DBClient interface {
	// User operations
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id int64) (*User, error)

	// API key operations
	CreateAPIKey(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error)
	ListAPIKeysByUser(userID int64) ([]*apitypes.APIKey, error)
	ListAllAPIKeys() ([]*apitypes.APIKey, error)
	GetAPIKeyByID(id int64) (*apitypes.APIKey, error)
	DeleteAPIKey(id int64) error
	GetAPIKeyByHash(keyHash string) (*apitypes.APIKey, error)
	UpdateAPIKeyLastUsed(id int64) error
}

// CRClient defines the Kubernetes Custom Resource operations needed by API handlers
// This interface allows for easy mocking in tests
type CRClient interface {
	CreateSupabaseInstance(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error
	GetSupabaseInstance(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error)
	ListSupabaseInstances(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error)
	DeleteSupabaseInstance(ctx context.Context, name string) error
}

// User represents a user from the database
// This matches db.User but is defined here to avoid import cycles in tests
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    string
	UpdatedAt    string
}
