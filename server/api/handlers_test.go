package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)


// User represents a database user (matching db.User)
// Removed - now defined in interfaces.go
// type User struct {
// 	ID           int64
// 	Username     string
// 	PasswordHash string
// 	Role         string
// 	CreatedAt    string
// 	UpdatedAt    string
// }
// mockDBClient is a mock implementation of db.Client for testing
type mockDBClient struct {
	getUserByUsernameFunc       func(username string) (*User, error)
	getUserByIDFunc             func(id int64) (*User, error)
	createAPIKeyFunc            func(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error)
	listAPIKeysByUserFunc       func(userID int64) ([]*apitypes.APIKey, error)
	listAllAPIKeysFunc          func() ([]*apitypes.APIKey, error)
	getAPIKeyByIDFunc           func(id int64) (*apitypes.APIKey, error)
	deleteAPIKeyFunc            func(id int64) error
	getAPIKeyByHashFunc         func(keyHash string) (*apitypes.APIKey, error)
	updateAPIKeyLastUsedFunc    func(id int64) error
// }

func (m *mockDBClient) GetUserByUsername(username string) (*User, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(username)
	}
	return nil, fmt.Errorf("GetUserByUsername not implemented")
// }

func (m *mockDBClient) GetUserByID(id int64) (*User, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(id)
	}
	return nil, fmt.Errorf("GetUserByID not implemented")
// }

func (m *mockDBClient) CreateAPIKey(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error) {
	if m.createAPIKeyFunc != nil {
		return m.createAPIKeyFunc(userID, name, keyHash, expiresAt)
	}
	return nil, fmt.Errorf("CreateAPIKey not implemented")
// }

func (m *mockDBClient) ListAPIKeysByUser(userID int64) ([]*apitypes.APIKey, error) {
	if m.listAPIKeysByUserFunc != nil {
		return m.listAPIKeysByUserFunc(userID)
	}
	return nil, fmt.Errorf("ListAPIKeysByUser not implemented")
// }

func (m *mockDBClient) ListAllAPIKeys() ([]*apitypes.APIKey, error) {
	if m.listAllAPIKeysFunc != nil {
		return m.listAllAPIKeysFunc()
	}
	return nil, fmt.Errorf("ListAllAPIKeys not implemented")
// }

func (m *mockDBClient) GetAPIKeyByID(id int64) (*apitypes.APIKey, error) {
	if m.getAPIKeyByIDFunc != nil {
		return m.getAPIKeyByIDFunc(id)
	}
	return nil, fmt.Errorf("GetAPIKeyByID not implemented")
// }

func (m *mockDBClient) DeleteAPIKey(id int64) error {
	if m.deleteAPIKeyFunc != nil {
		return m.deleteAPIKeyFunc(id)
	}
	return fmt.Errorf("DeleteAPIKey not implemented")
// }

func (m *mockDBClient) GetAPIKeyByHash(keyHash string) (*apitypes.APIKey, error) {
	if m.getAPIKeyByHashFunc != nil {
		return m.getAPIKeyByHashFunc(keyHash)
	}
	return nil, fmt.Errorf("GetAPIKeyByHash not implemented")
// }

func (m *mockDBClient) UpdateAPIKeyLastUsed(id int64) error {
	if m.updateAPIKeyLastUsedFunc != nil {
		return m.updateAPIKeyLastUsedFunc(id)
	}
	return nil
// }

// mockCRClient is a mock implementation of k8s.CRClient for testing
type mockCRClient struct {
	createSupabaseInstanceFunc func(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error
	getSupabaseInstanceFunc    func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error)
	listSupabaseInstancesFunc  func(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error)
	deleteSupabaseInstanceFunc func(ctx context.Context, name string) error
// }

func (m *mockCRClient) CreateSupabaseInstance(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	if m.createSupabaseInstanceFunc != nil {
		return m.createSupabaseInstanceFunc(ctx, instance)
	}
	return fmt.Errorf("CreateSupabaseInstance not implemented")
// }

func (m *mockCRClient) GetSupabaseInstance(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
	if m.getSupabaseInstanceFunc != nil {
		return m.getSupabaseInstanceFunc(ctx, name)
	}
	return nil, fmt.Errorf("GetSupabaseInstance not implemented")
// }

func (m *mockCRClient) ListSupabaseInstances(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error) {
	if m.listSupabaseInstancesFunc != nil {
		return m.listSupabaseInstancesFunc(ctx)
	}
	return nil, fmt.Errorf("ListSupabaseInstances not implemented")
// }

func (m *mockCRClient) DeleteSupabaseInstance(ctx context.Context, name string) error {
	if m.deleteSupabaseInstanceFunc != nil {
		return m.deleteSupabaseInstanceFunc(ctx, name)
	}
	return fmt.Errorf("DeleteSupabaseInstance not implemented")
// }

// Helper to create test echo context
func newTestContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
// }

// Helper to set auth context
func setAuthContext(c echo.Context, userID int64, username, role string) {
	c.Set("auth", &AuthContext{
		UserID:   userID,
		Username: username,
		Role:     role,
		IsAPIKey: false,
	})
// }

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	handler := &Handler{}
	c, rec := newTestContext(http.MethodGet, "/healthz", "")

	err := handler.HealthCheck(c)
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp["status"])
	}

	if resp["time"] == "" {
		t.Error("expected non-empty time field")
	}
// }

// TestLogin tests the login endpoint
func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(*mockDBClient, *auth.Service)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:        "successful login",
			requestBody: `{"username":"admin","password":"admin"}`,
			setupMock: func(db *mockDBClient, authSvc *auth.Service) {
				db.getUserByUsernameFunc = func(username string) (*User, error) {
					if username != "admin" {
						return nil, nil
					}
					// Hash of "admin"
					hash, _ := authSvc.HashPassword("admin")
					return &User{
						ID:           1,
						Username:     "admin",
						Role:         "admin",
						PasswordHash: hash,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:        "invalid request body",
			requestBody: `{invalid json}`,
			setupMock: func(db *mockDBClient, authSvc *auth.Service) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:        "user not found",
			requestBody: `{"username":"nonexistent","password":"admin"}`,
			setupMock: func(db *mockDBClient, authSvc *auth.Service) {
				db.getUserByUsernameFunc = func(username string) (*User, error) {
					return nil, nil // User not found
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:        "wrong password",
			requestBody: `{"username":"admin","password":"wrongpassword"}`,
			setupMock: func(db *mockDBClient, authSvc *auth.Service) {
				db.getUserByUsernameFunc = func(username string) (*User, error) {
					hash, _ := authSvc.HashPassword("admin")
					return &User{
						ID:           1,
						Username:     "admin",
						Role:         "admin",
						PasswordHash: hash,
					}, nil
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			authSvc := auth.NewService("test-secret-key")
			tt.setupMock(mockDB, authSvc)

			handler := NewHandler(authSvc, mockDB, nil)
			c, rec := newTestContext(http.MethodPost, "/api/v1/auth/login", tt.requestBody)

			err := handler.Login(c)

			// Echo returns errors as *echo.HTTPError
			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				// Verify response structure
				var resp apitypes.LoginResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Token == "" {
					t.Error("expected non-empty token")
				}
				if resp.User == nil {
					t.Error("expected user info")
				} else {
					if resp.User.Username != "admin" {
						t.Errorf("expected username 'admin', got '%s'", resp.User.Username)
					}
				}
			}
		})
	}
// }

// TestGetAuthMe tests the /auth/me endpoint
func TestGetAuthMe(t *testing.T) {
	tests := []struct {
		name           string
		setAuth        bool
		setupMock      func(*mockDBClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:    "successful get user info",
			setAuth: true,
			setupMock: func(db *mockDBClient) {
				db.getUserByIDFunc = func(id int64) (*User, error) {
					return &User{
						ID:       1,
						Username: "testuser",
						Role:     "admin",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:    "not authenticated",
			setAuth: false,
			setupMock: func(db *mockDBClient) {
				// No mock setup needed
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:    "user not found in database",
			setAuth: true,
			setupMock: func(db *mockDBClient) {
				db.getUserByIDFunc = func(id int64) (*User, error) {
					return nil, nil
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			tt.setupMock(mockDB)

			handler := NewHandler(nil, mockDB, nil)
			c, rec := newTestContext(http.MethodGet, "/api/v1/auth/me", "")

			if tt.setAuth {
				setAuthContext(c, 1, "testuser", "admin")
			}

			err := handler.GetAuthMe(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}
			}
		})
	}
// }

// TestCreateAPIKey tests the API key creation endpoint
func TestCreateAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		setAuth        bool
		setupMock      func(*mockDBClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:        "successful API key creation",
			requestBody: `{"name":"test-key"}`,
			setAuth:     true,
			setupMock: func(db *mockDBClient) {
				db.createAPIKeyFunc = func(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error) {
					return &apitypes.APIKey{
						ID:        1,
						UserID:    userID,
						Name:      name,
						KeyHash:   keyHash,
						CreatedAt: time.Now(),
						ExpiresAt: expiresAt,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name:        "not authenticated",
			requestBody: `{"name":"test-key"}`,
			setAuth:     false,
			setupMock:   func(db *mockDBClient) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:        "invalid request body",
			requestBody: `{invalid json}`,
			setAuth:     true,
			setupMock:   func(db *mockDBClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			tt.setupMock(mockDB)
			authSvc := auth.NewService("test-secret-key")

			handler := NewHandler(authSvc, mockDB, nil)
			c, rec := newTestContext(http.MethodPost, "/api/v1/auth/api-keys", tt.requestBody)

			if tt.setAuth {
				setAuthContext(c, 1, "testuser", "admin")
			}

			err := handler.CreateAPIKey(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				// Verify response contains the key
				var resp apitypes.CreateAPIKeyResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Key == "" {
					t.Error("expected non-empty key")
				}
				if !strings.HasPrefix(resp.Key, "sk_") {
					t.Errorf("expected key to start with 'sk_', got '%s'", resp.Key)
				}
			}
		})
	}
// }

// TestListAPIKeys tests listing API keys
func TestListAPIKeys(t *testing.T) {
	tests := []struct {
		name           string
		setAuth        bool
		userRole       string
		setupMock      func(*mockDBClient)
		expectedStatus int
		expectedError  bool
		expectedCount  int
	}{
		{
			name:     "list keys as regular user",
			setAuth:  true,
			userRole: "user",
			setupMock: func(db *mockDBClient) {
				db.listAPIKeysByUserFunc = func(userID int64) ([]*apitypes.APIKey, error) {
					return []*apitypes.APIKey{
						{ID: 1, UserID: userID, Name: "key1"},
						{ID: 2, UserID: userID, Name: "key2"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedCount:  2,
		},
		{
			name:     "list all keys as admin",
			setAuth:  true,
			userRole: "admin",
			setupMock: func(db *mockDBClient) {
				db.listAllAPIKeysFunc = func() ([]*apitypes.APIKey, error) {
					return []*apitypes.APIKey{
						{ID: 1, UserID: 1, Name: "key1"},
						{ID: 2, UserID: 1, Name: "key2"},
						{ID: 3, UserID: 2, Name: "key3"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedCount:  3,
		},
		{
			name:           "not authenticated",
			setAuth:        false,
			setupMock:      func(db *mockDBClient) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			tt.setupMock(mockDB)

			handler := NewHandler(nil, mockDB, nil)
			c, rec := newTestContext(http.MethodGet, "/api/v1/auth/api-keys", "")

			if tt.setAuth {
				setAuthContext(c, 1, "testuser", tt.userRole)
			}

			err := handler.ListAPIKeys(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				var resp apitypes.ListAPIKeysResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Count != tt.expectedCount {
					t.Errorf("expected count %d, got %d", tt.expectedCount, resp.Count)
				}
			}
		})
	}
// }

// TestDeleteAPIKey tests deleting an API key
func TestDeleteAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		apiKeyID       string
		setAuth        bool
		userRole       string
		setupMock      func(*mockDBClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:     "delete own key as user",
			apiKeyID: "1",
			setAuth:  true,
			userRole: "user",
			setupMock: func(db *mockDBClient) {
				db.getAPIKeyByIDFunc = func(id int64) (*apitypes.APIKey, error) {
					return &apitypes.APIKey{
						ID:     1,
						UserID: 1, // Same as authenticated user
						Name:   "test-key",
					}, nil
				}
				db.deleteAPIKeyFunc = func(id int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:     "delete other user's key as admin",
			apiKeyID: "2",
			setAuth:  true,
			userRole: "admin",
			setupMock: func(db *mockDBClient) {
				db.getAPIKeyByIDFunc = func(id int64) (*apitypes.APIKey, error) {
					return &apitypes.APIKey{
						ID:     2,
						UserID: 999, // Different user
						Name:   "other-key",
					}, nil
				}
				db.deleteAPIKeyFunc = func(id int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:     "forbidden: delete other user's key as regular user",
			apiKeyID: "2",
			setAuth:  true,
			userRole: "user",
			setupMock: func(db *mockDBClient) {
				db.getAPIKeyByIDFunc = func(id int64) (*apitypes.APIKey, error) {
					return &apitypes.APIKey{
						ID:     2,
						UserID: 999, // Different user
						Name:   "other-key",
					}, nil
				}
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  true,
		},
		{
			name:           "invalid API key ID",
			apiKeyID:       "invalid",
			setAuth:        true,
			userRole:       "admin",
			setupMock:      func(db *mockDBClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:     "API key not found",
			apiKeyID: "999",
			setAuth:  true,
			userRole: "admin",
			setupMock: func(db *mockDBClient) {
				db.getAPIKeyByIDFunc = func(id int64) (*apitypes.APIKey, error) {
					return nil, nil
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			tt.setupMock(mockDB)

			handler := NewHandler(nil, mockDB, nil)
			c, rec := newTestContext(http.MethodDelete, "/api/v1/auth/api-keys/"+tt.apiKeyID, "")
			c.SetParamNames("id")
			c.SetParamValues(tt.apiKeyID)

			if tt.setAuth {
				setAuthContext(c, 1, "testuser", tt.userRole)
			}

			err := handler.DeleteAPIKey(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}
			}
		})
	}
// }

// TestCreateInstance tests creating a Supabase instance
func TestCreateInstance(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(*mockCRClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:        "successful instance creation",
			requestBody: `{"name":"test-app"}`,
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					// Return NotFound to indicate instance doesn't exist
					return nil, apierrors.NewNotFound(schema.GroupResource{}, name)
				}
				cr.createSupabaseInstanceFunc = func(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
					return nil
				}
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name:        "instance already exists",
			requestBody: `{"name":"existing-app"}`,
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					// Return existing instance
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusConflict,
			expectedError:  true,
		},
		{
			name:           "empty instance name",
			requestBody:    `{"name":""}`,
			setupMock:      func(cr *mockCRClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "invalid request body",
			requestBody:    `{invalid json}`,
			setupMock:      func(cr *mockCRClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			tt.setupMock(mockCR)

			handler := NewHandler(nil, nil, mockCR)
			c, rec := newTestContext(http.MethodPost, "/api/v1/instances", tt.requestBody)

			err := handler.CreateInstance(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				var resp apitypes.CreateInstanceResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Instance == nil {
					t.Error("expected instance in response")
				}
				if resp.Message == "" {
					t.Error("expected message in response")
				}
			}
		})
	}
// }

// TestListInstances tests listing instances
func TestListInstances(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mockCRClient)
		expectedStatus int
		expectedError  bool
		expectedCount  int
	}{
		{
			name: "successful list with instances",
			setupMock: func(cr *mockCRClient) {
				cr.listSupabaseInstancesFunc = func(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error) {
					return &supacontrolv1alpha1.SupabaseInstanceList{
						Items: []supacontrolv1alpha1.SupabaseInstance{
							{
								ObjectMeta: metav1.ObjectMeta{Name: "app1"},
								Spec:       supacontrolv1alpha1.SupabaseInstanceSpec{ProjectName: "app1"},
								Status: supacontrolv1alpha1.SupabaseInstanceStatus{
									Phase:     supacontrolv1alpha1.PhaseRunning,
									Namespace: "supa-app1",
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Name: "app2"},
								Spec:       supacontrolv1alpha1.SupabaseInstanceSpec{ProjectName: "app2"},
								Status: supacontrolv1alpha1.SupabaseInstanceStatus{
									Phase:     supacontrolv1alpha1.PhaseProvisioning,
									Namespace: "supa-app2",
								},
							},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedCount:  2,
		},
		{
			name: "empty list",
			setupMock: func(cr *mockCRClient) {
				cr.listSupabaseInstancesFunc = func(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error) {
					return &supacontrolv1alpha1.SupabaseInstanceList{
						Items: []supacontrolv1alpha1.SupabaseInstance{},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedCount:  0,
		},
		{
			name: "kubernetes error",
			setupMock: func(cr *mockCRClient) {
				cr.listSupabaseInstancesFunc = func(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error) {
					return nil, fmt.Errorf("kubernetes api error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			tt.setupMock(mockCR)

			handler := NewHandler(nil, nil, mockCR)
			c, rec := newTestContext(http.MethodGet, "/api/v1/instances", "")

			err := handler.ListInstances(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				var resp apitypes.ListInstancesResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Count != tt.expectedCount {
					t.Errorf("expected count %d, got %d", tt.expectedCount, resp.Count)
				}
			}
		})
	}
// }

// TestGetInstance tests getting a single instance
func TestGetInstance(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		setupMock      func(*mockCRClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:         "successful get",
			instanceName: "test-app",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{Name: name},
						Spec:       supacontrolv1alpha1.SupabaseInstanceSpec{ProjectName: name},
						Status: supacontrolv1alpha1.SupabaseInstanceStatus{
							Phase:     supacontrolv1alpha1.PhaseRunning,
							Namespace: "supa-" + name,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:         "instance not found",
			instanceName: "nonexistent",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return nil, apierrors.NewNotFound(schema.GroupResource{}, name)
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			tt.setupMock(mockCR)

			handler := NewHandler(nil, nil, mockCR)
			c, rec := newTestContext(http.MethodGet, "/api/v1/instances/"+tt.instanceName, "")
			c.SetParamNames("name")
			c.SetParamValues(tt.instanceName)

			err := handler.GetInstance(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}

				var resp apitypes.GetInstanceResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Instance == nil {
					t.Error("expected instance in response")
				}
			}
		})
	}
// }

// TestDeleteInstance tests deleting an instance
func TestDeleteInstance(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		setupMock      func(*mockCRClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:         "successful delete",
			instanceName: "test-app",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{Name: name},
						Spec:       supacontrolv1alpha1.SupabaseInstanceSpec{ProjectName: name},
					}, nil
				}
				cr.deleteSupabaseInstanceFunc = func(ctx context.Context, name string) error {
					return nil
				}
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name:         "instance not found",
			instanceName: "nonexistent",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return nil, apierrors.NewNotFound(schema.GroupResource{}, name)
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			tt.setupMock(mockCR)

			handler := NewHandler(nil, nil, mockCR)
			c, rec := newTestContext(http.MethodDelete, "/api/v1/instances/"+tt.instanceName, "")
			c.SetParamNames("name")
			c.SetParamValues(tt.instanceName)

			err := handler.DeleteInstance(c)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected *echo.HTTPError, got %T", err)
				}
				if httpErr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, httpErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
				}
			}
		})
	}
// }
