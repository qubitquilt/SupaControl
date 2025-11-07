package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	"github.com/qubitquilt/supacontrol/server/internal/db"
	"github.com/qubitquilt/supacontrol/server/internal/k8s"
)

// Handler holds dependencies for API handlers
type Handler struct {
	authService  *auth.Service
	dbClient     *db.Client
	orchestrator *k8s.Orchestrator
}

// NewHandler creates a new API handler
func NewHandler(authService *auth.Service, dbClient *db.Client, orchestrator *k8s.Orchestrator) *Handler {
	return &Handler{
		authService:  authService,
		dbClient:     dbClient,
		orchestrator: orchestrator,
	}
}

// Health check endpoint
func (h *Handler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Login handles user login
func (h *Handler) Login(c echo.Context) error {
	var req apitypes.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Get user
	user, err := h.dbClient.GetUserByUsername(req.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to authenticate")
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Verify password
	valid, err := h.authService.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify password")
	}

	if !valid {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Generate JWT
	token, err := h.authService.GenerateJWT(user.ID, user.Username, user.Role, 24*time.Hour)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, apitypes.LoginResponse{
		Token: token,
		User: &apitypes.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// GetAuthMe returns information about the authenticated user
func (h *Handler) GetAuthMe(c echo.Context) error {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	user, err := h.dbClient.GetUserByID(authCtx.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	return c.JSON(http.StatusOK, apitypes.AuthMeResponse{
		User: &apitypes.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// CreateAPIKey generates a new API key
func (h *Handler) CreateAPIKey(c echo.Context) error {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req apitypes.CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Generate new API key
	apiKey, err := h.authService.GenerateAPIKey()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate API key")
	}

	// Hash the key
	keyHash, err := h.authService.HashAPIKey(apiKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash API key")
	}

	// Store in database
	apiKeyRecord, err := h.dbClient.CreateAPIKey(authCtx.UserID, req.Name, keyHash, req.ExpiresAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create API key")
	}

	return c.JSON(http.StatusCreated, apitypes.CreateAPIKeyResponse{
		Key:     apiKey,
		APIKey:  apiKeyRecord,
		Message: "API key created successfully. Save this key securely - it won't be shown again!",
	})
}

// ListAPIKeys lists all API keys for the authenticated user
func (h *Handler) ListAPIKeys(c echo.Context) error {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var apiKeys []*apitypes.APIKey
	var err error

	// Admins can see all keys
	if authCtx.Role == "admin" {
		apiKeys, err = h.dbClient.ListAllAPIKeys()
	} else {
		apiKeys, err = h.dbClient.ListAPIKeysByUser(authCtx.UserID)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list API keys")
	}

	return c.JSON(http.StatusOK, apitypes.ListAPIKeysResponse{
		APIKeys: apiKeys,
		Count:   len(apiKeys),
	})
}

// DeleteAPIKey deletes an API key
func (h *Handler) DeleteAPIKey(c echo.Context) error {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id := c.Param("id")
	var apiKeyID int64
	if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid API key ID")
	}

	// Get the API key to verify ownership
	apiKey, err := h.dbClient.GetAPIKeyByID(apiKeyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get API key")
	}

	if apiKey == nil {
		return echo.NewHTTPError(http.StatusNotFound, "API key not found")
	}

	// Users can only delete their own keys, admins can delete any
	if authCtx.Role != "admin" && apiKey.UserID != authCtx.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "cannot delete other users' API keys")
	}

	if err := h.dbClient.DeleteAPIKey(apiKeyID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete API key")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key deleted successfully",
	})
}

// CreateInstance creates a new Supabase instance
func (h *Handler) CreateInstance(c echo.Context) error {
	var req apitypes.CreateInstanceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate project name
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "project name is required")
	}

	// Check if instance already exists
	exists, err := h.dbClient.InstanceExists(req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check instance existence")
	}

	if exists {
		return echo.NewHTTPError(http.StatusConflict, "instance with this name already exists")
	}

	// Create instance record with PROVISIONING status
	instance := &apitypes.Instance{
		ProjectName: req.Name,
		Namespace:   fmt.Sprintf("supa-%s", req.Name),
		Status:      apitypes.StatusProvisioning,
	}

	if err := h.dbClient.StoreInstance(instance); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to store instance")
	}

	// Provision instance asynchronously
	go h.provisionInstance(req.Name)

	return c.JSON(http.StatusAccepted, apitypes.CreateInstanceResponse{
		Instance: instance,
		Message:  "Instance provisioning started",
	})
}

// provisionInstance handles the async provisioning of an instance
func (h *Handler) provisionInstance(projectName string) {
	ctx := context.Background()

	instance, err := h.orchestrator.CreateInstance(ctx, projectName)
	if err != nil {
		// Update instance status to FAILED
		errMsg := err.Error()
		if updateErr := h.dbClient.UpdateInstanceStatus(projectName, apitypes.StatusFailed, &errMsg); updateErr != nil {
			fmt.Printf("Failed to update instance status: %v\n", updateErr)
		}
		return
	}

	// Update instance with full details
	if err := h.dbClient.UpdateInstance(instance); err != nil {
		errMsg := err.Error()
		if updateErr := h.dbClient.UpdateInstanceStatus(projectName, apitypes.StatusFailed, &errMsg); updateErr != nil {
			fmt.Printf("Failed to update instance status: %v\n", updateErr)
		}
		return
	}
}

// ListInstances lists all Supabase instances
func (h *Handler) ListInstances(c echo.Context) error {
	instances, err := h.dbClient.ListInstances()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list instances")
	}

	return c.JSON(http.StatusOK, apitypes.ListInstancesResponse{
		Instances: instances,
		Count:     len(instances),
	})
}

// GetInstance gets a single Supabase instance
func (h *Handler) GetInstance(c echo.Context) error {
	name := c.Param("name")

	instance, err := h.dbClient.GetInstance(name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	if instance == nil {
		return echo.NewHTTPError(http.StatusNotFound, "instance not found")
	}

	return c.JSON(http.StatusOK, apitypes.GetInstanceResponse{
		Instance: instance,
	})
}

// DeleteInstance deletes a Supabase instance
func (h *Handler) DeleteInstance(c echo.Context) error {
	name := c.Param("name")

	// Get instance
	instance, err := h.dbClient.GetInstance(name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	if instance == nil {
		return echo.NewHTTPError(http.StatusNotFound, "instance not found")
	}

	// Update status to DELETING
	if err := h.dbClient.UpdateInstanceStatus(name, apitypes.StatusDeleting, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update instance status")
	}

	// Delete instance asynchronously
	go h.deleteInstance(name, instance.Namespace)

	return c.JSON(http.StatusAccepted, apitypes.DeleteInstanceResponse{
		Message: "Instance deletion started",
	})
}

// deleteInstance handles the async deletion of an instance
func (h *Handler) deleteInstance(projectName, namespace string) {
	ctx := context.Background()

	if err := h.orchestrator.DeleteInstance(ctx, projectName, namespace); err != nil {
		// Update instance status to FAILED
		errMsg := fmt.Sprintf("failed to delete: %v", err)
		if updateErr := h.dbClient.UpdateInstanceStatus(projectName, apitypes.StatusFailed, &errMsg); updateErr != nil {
			fmt.Printf("Failed to update instance status: %v\n", updateErr)
		}
		return
	}

	// Remove from database
	if err := h.dbClient.DeleteInstance(projectName); err != nil {
		errMsg := fmt.Sprintf("failed to remove from database: %v", err)
		if updateErr := h.dbClient.UpdateInstanceStatus(projectName, apitypes.StatusFailed, &errMsg); updateErr != nil {
			fmt.Printf("Failed to update instance status: %v\n", updateErr)
		}
		return
	}
}
