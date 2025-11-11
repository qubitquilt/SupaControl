package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
)

// Handler holds dependencies for API handlers
type Handler struct {
	authService *auth.Service
	dbClient    DBClient
	crClient    CRClient
}

// NewHandler creates a new API handler
func NewHandler(authService *auth.Service, dbClient DBClient, crClient CRClient) *Handler {
	return &Handler{
		authService: authService,
		dbClient:    dbClient,
		crClient:    crClient,
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

	ctx := c.Request().Context()

	// Check if instance already exists in K8s
	_, err := h.crClient.GetSupabaseInstance(ctx, req.Name)
	if err == nil {
		return echo.NewHTTPError(http.StatusConflict, "instance with this name already exists")
	}
	if !apierrors.IsNotFound(err) {
		slog.Error("Failed to check instance existence", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check instance existence")
	}

	// Create SupabaseInstance CR
	instance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "supacontrol-api",
			},
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: req.Name,
		},
	}

	if err := h.crClient.CreateSupabaseInstance(ctx, instance); err != nil {
		slog.Error("Failed to create SupabaseInstance CR", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create instance")
	}

	// Convert CR to API response
	apiInstance := h.convertCRToAPIType(instance)

	return c.JSON(http.StatusAccepted, apitypes.CreateInstanceResponse{
		Instance: apiInstance,
		Message:  "Instance provisioning started",
	})
}

// ListInstances lists all Supabase instances
func (h *Handler) ListInstances(c echo.Context) error {
	ctx := c.Request().Context()

	crList, err := h.crClient.ListSupabaseInstances(ctx)
	if err != nil {
		slog.Error("Failed to list instances", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list instances")
	}

	// Convert CRs to API types
	instances := make([]*apitypes.Instance, 0, len(crList.Items))
	for i := range crList.Items {
		instances = append(instances, h.convertCRToAPIType(&crList.Items[i]))
	}

	return c.JSON(http.StatusOK, apitypes.ListInstancesResponse{
		Instances: instances,
		Count:     len(instances),
	})
}

// GetInstance gets a single Supabase instance
func (h *Handler) GetInstance(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	instance, err := h.crClient.GetSupabaseInstance(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found")
		}
		slog.Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	return c.JSON(http.StatusOK, apitypes.GetInstanceResponse{
		Instance: h.convertCRToAPIType(instance),
	})
}

// DeleteInstance deletes a Supabase instance
func (h *Handler) DeleteInstance(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	// Check if instance exists
	_, err := h.crClient.GetSupabaseInstance(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found")
		}
		slog.Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	// Delete SupabaseInstance CR (controller will handle cleanup via finalizer)
	if err := h.crClient.DeleteSupabaseInstance(ctx, name); err != nil {
		slog.Error("Failed to delete SupabaseInstance CR", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete instance")
	}

	return c.JSON(http.StatusAccepted, apitypes.DeleteInstanceResponse{
		Message: "Instance deletion started",
	})
}

// convertCRToAPIType converts a SupabaseInstance CR to API type
func (h *Handler) convertCRToAPIType(cr *supacontrolv1alpha1.SupabaseInstance) *apitypes.Instance {
	// Map CR phase to API status
	var status apitypes.InstanceStatus
	switch cr.Status.Phase {
	case supacontrolv1alpha1.PhasePending:
		status = apitypes.StatusProvisioning
	case supacontrolv1alpha1.PhaseProvisioning:
		status = apitypes.StatusProvisioning
	case supacontrolv1alpha1.PhaseRunning:
		status = apitypes.StatusRunning
	case supacontrolv1alpha1.PhaseDeleting:
		status = apitypes.StatusDeleting
	case supacontrolv1alpha1.PhaseFailed:
		status = apitypes.StatusFailed
	default:
		// Unknown phase - log warning and default to Provisioning
		slog.Warn("Unknown SupabaseInstance phase encountered",
			"projectName", cr.Spec.ProjectName,
			"phase", cr.Status.Phase,
			"defaulting_to", apitypes.StatusProvisioning)
		status = apitypes.StatusProvisioning
	}

	instance := &apitypes.Instance{
		ProjectName: cr.Spec.ProjectName,
		Namespace:   cr.Status.Namespace,
		Status:      status,
		StudioURL:   cr.Status.StudioURL,
		APIURL:      cr.Status.APIURL,
	}

	// Set error message if present
	if cr.Status.ErrorMessage != "" {
		instance.ErrorMessage = &cr.Status.ErrorMessage
	}

	// Set timestamps from CR metadata
	if !cr.CreationTimestamp.IsZero() {
		instance.CreatedAt = cr.CreationTimestamp.Time
	}
	if cr.Status.LastTransitionTime != nil {
		instance.UpdatedAt = cr.Status.LastTransitionTime.Time
	}

	return instance
}
