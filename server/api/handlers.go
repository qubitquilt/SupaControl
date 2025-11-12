package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	corev1 "k8s.io/api/core/v1"
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
	k8sClient   K8sClient
}

// NewHandler creates a new API handler
func NewHandler(authService *auth.Service, dbClient DBClient, crClient CRClient, k8sClient K8sClient) *Handler {
	return &Handler{
		authService: authService,
		dbClient:    dbClient,
		crClient:    crClient,
		k8sClient:   k8sClient,
	}
}

// getInstanceNamespace returns the namespace for an instance
// It uses the namespace from the instance status if available, otherwise generates it from the name
func getInstanceNamespace(instance *supacontrolv1alpha1.SupabaseInstance) string {
	if instance.Status.Namespace != "" {
		return instance.Status.Namespace
	}
	return fmt.Sprintf("supa-%s", instance.Spec.ProjectName)
}

// containerLogResult holds the result of fetching logs from a container
type containerLogResult struct {
	podName       string
	containerName string
	logs          string
	err           error
	index         int // For maintaining order
}

// fetchContainerLogs fetches logs from a single container
func fetchContainerLogs(ctx context.Context, clientset K8sClient, namespace, podName, containerName string, lines int64, index int) containerLogResult {
	result := containerLogResult{
		podName:       podName,
		containerName: containerName,
		index:         index,
	}

	logOptions := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &lines,
	}

	req := clientset.GetClientset().CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		result.err = err
		return result
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		result.err = err
		return result
	}

	result.logs = buf.String()
	return result
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
		GetLogger(c).Error("Failed to check instance existence", "error", err)
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
		GetLogger(c).Error("Failed to create SupabaseInstance CR", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create instance")
	}

	// Convert CR to API response
	apiInstance := h.convertCRToAPIType(c, instance)

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
		GetLogger(c).Error("Failed to list instances", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list instances")
	}

	// Convert CRs to API types
	instances := make([]*apitypes.Instance, 0, len(crList.Items))
	for i := range crList.Items {
		instances = append(instances, h.convertCRToAPIType(c, &crList.Items[i]))
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
		GetLogger(c).Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	return c.JSON(http.StatusOK, apitypes.GetInstanceResponse{
		Instance: h.convertCRToAPIType(c, instance),
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
		GetLogger(c).Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	// Delete SupabaseInstance CR (controller will handle cleanup via finalizer)
	if err := h.crClient.DeleteSupabaseInstance(ctx, name); err != nil {
		GetLogger(c).Error("Failed to delete SupabaseInstance CR", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete instance")
	}

	return c.JSON(http.StatusAccepted, apitypes.DeleteInstanceResponse{
		Message: "Instance deletion started",
	})
}

// StartInstance starts a stopped instance by setting Paused=false
func (h *Handler) StartInstance(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	// Get the instance
	instance, err := h.crClient.GetSupabaseInstance(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found")
		}
		GetLogger(c).Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	// Check if already running
	if !instance.Spec.Paused {
		return echo.NewHTTPError(http.StatusConflict, "instance is already running")
	}

	// Update the instance to set Paused=false
	instance.Spec.Paused = false
	if err := h.crClient.UpdateSupabaseInstance(ctx, instance); err != nil {
		GetLogger(c).Error("Failed to start instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start instance")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Instance start initiated",
		"status":  "Starting",
	})
}

// StopInstance stops a running instance by setting Paused=true
func (h *Handler) StopInstance(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	// Get the instance
	instance, err := h.crClient.GetSupabaseInstance(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found")
		}
		GetLogger(c).Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	// Check if already stopped
	if instance.Spec.Paused {
		return echo.NewHTTPError(http.StatusConflict, "instance is already stopped")
	}

	// Update the instance to set Paused=true
	instance.Spec.Paused = true
	if err := h.crClient.UpdateSupabaseInstance(ctx, instance); err != nil {
		GetLogger(c).Error("Failed to stop instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to stop instance")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Instance stopped successfully",
		"status":  "Stopped",
	})
}

// RestartInstance restarts an instance by deleting its pods
func (h *Handler) RestartInstance(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	// Get the instance to verify it exists
	instance, err := h.crClient.GetSupabaseInstance(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found")
		}
		GetLogger(c).Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	// Get the namespace
	namespace := getInstanceNamespace(instance)

	// Get all deployments in the namespace and restart them by adding an annotation
	clientset := h.k8sClient.GetClientset()
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		GetLogger(c).Error("Failed to list deployments", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to restart instance")
	}

	// Restart each deployment by updating the restart annotation
	restartedCount := 0
	for i := range deployments.Items {
		deployment := &deployments.Items[i]
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

		_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			GetLogger(c).Error("Failed to restart deployment", "deployment", deployment.Name, "error", err)
			continue
		}
		restartedCount++
	}

	if restartedCount == 0 {
		return echo.NewHTTPError(http.StatusInternalServerError, "no deployments found or failed to restart")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Instance restart initiated",
		"status":    "Restarting",
		"restarted": restartedCount,
	})
}

// GetLogs retrieves logs from instance pods using concurrent fetching for better performance
func (h *Handler) GetLogs(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	// Parse query params
	linesParam := c.QueryParam("lines")
	lines := int64(100)
	if linesParam != "" {
		parsed, err := strconv.ParseInt(linesParam, 10, 64)
		if err == nil && parsed > 0 {
			lines = parsed
		}
	}

	// Get the instance to verify it exists
	instance, err := h.crClient.GetSupabaseInstance(ctx, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found")
		}
		GetLogger(c).Error("Failed to get instance", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get instance")
	}

	// Get the namespace
	namespace := getInstanceNamespace(instance)

	// Get all pods in the namespace
	clientset := h.k8sClient.GetClientset()
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		GetLogger(c).Error("Failed to list pods", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get logs")
	}

	if len(pods.Items) == 0 {
		return c.String(http.StatusOK, "No pods found for this instance\n")
	}

	// Count total containers to fetch logs from
	totalContainers := 0
	for _, pod := range pods.Items {
		totalContainers += len(pod.Spec.Containers)
	}

	// Fetch logs concurrently from all containers
	var wg sync.WaitGroup
	resultsChan := make(chan containerLogResult, totalContainers)

	index := 0
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			wg.Add(1)
			go func(p corev1.Pod, c corev1.Container, idx int) {
				defer wg.Done()
				result := fetchContainerLogs(ctx, h.k8sClient, namespace, p.Name, c.Name, lines, idx)
				resultsChan <- result
			}(pod, container, index)
			index++
		}
	}

	// Close the results channel once all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results and sort by index to maintain order
	results := make([]containerLogResult, 0, totalContainers)
	for result := range resultsChan {
		results = append(results, result)
	}

	// Sort results by index to maintain the original pod/container order
	// Using a simple bubble sort for small arrays (most instances won't have many containers)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].index > results[j].index {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Build the aggregated logs output
	var aggregatedLogs strings.Builder
	currentPod := ""
	for _, result := range results {
		// Add pod header if this is a new pod
		if result.podName != currentPod {
			aggregatedLogs.WriteString(fmt.Sprintf("=== Logs from pod: %s ===\n", result.podName))
			currentPod = result.podName
		}

		aggregatedLogs.WriteString(fmt.Sprintf("--- Container: %s ---\n", result.containerName))

		if result.err != nil {
			aggregatedLogs.WriteString(fmt.Sprintf("Error getting logs: %v\n", result.err))
		} else {
			aggregatedLogs.WriteString(result.logs)
			aggregatedLogs.WriteString("\n")
		}
	}

	return c.String(http.StatusOK, aggregatedLogs.String())
}

// convertCRToAPIType converts a SupabaseInstance CR to API type
func (h *Handler) convertCRToAPIType(c echo.Context, cr *supacontrolv1alpha1.SupabaseInstance) *apitypes.Instance {
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
		GetLogger(c).Warn("Unknown SupabaseInstance phase encountered",
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
