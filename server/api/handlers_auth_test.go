package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	"github.com/qubitquilt/supacontrol/server/internal/db"
)

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
			setupMock: func(mockDB *mockDBClient, authSvc *auth.Service) {
				mockDB.getUserByUsernameFunc = func(username string) (*db.User, error) {
					if username != "admin" {
						return nil, nil
					}
					// Hash of "admin"
					hash, err := authSvc.HashPassword("admin")
					if err != nil {
						panic(fmt.Sprintf("failed to hash password in test setup: %v", err))
					}
					return &db.User{
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
			setupMock: func(_ *mockDBClient, _ *auth.Service) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:        "user not found",
			requestBody: `{"username":"nonexistent","password":"admin"}`,
			setupMock: func(mockDB *mockDBClient, _ *auth.Service) {
				mockDB.getUserByUsernameFunc = func(_ string) (*db.User, error) {
					return nil, nil // User not found
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:        "wrong password",
			requestBody: `{"username":"admin","password":"wrongpassword"}`,
			setupMock: func(mockDB *mockDBClient, authSvc *auth.Service) {
				mockDB.getUserByUsernameFunc = func(_ string) (*db.User, error) {
					hash, err := authSvc.HashPassword("admin")
					if err != nil {
						panic(fmt.Sprintf("failed to hash password in test setup: %v", err))
					}
					return &db.User{
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

			handler := NewHandler(authSvc, mockDB, nil, nil)
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
				} else if resp.User.Username != "admin" {
					t.Errorf("expected username 'admin', got '%s'", resp.User.Username)
				}
			}
		})
	}
}

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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.getUserByIDFunc = func(_ int64) (*db.User, error) {
					return &db.User{
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
			setupMock: func(_ *mockDBClient) {
				// No mock setup needed
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:    "user not found in database",
			setAuth: true,
			setupMock: func(mockDB *mockDBClient) {
				mockDB.getUserByIDFunc = func(_ int64) (*db.User, error) {
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

			handler := NewHandler(nil, mockDB, nil, nil)
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
}

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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.createAPIKeyFunc = func(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error) {
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
			name:           "not authenticated",
			requestBody:    `{"name":"test-key"}`,
			setAuth:        false,
			setupMock:      func(_ *mockDBClient) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:           "invalid request body",
			requestBody:    `{invalid json}`,
			setAuth:        true,
			setupMock:      func(_ *mockDBClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			tt.setupMock(mockDB)
			authSvc := auth.NewService("test-secret-key")

			handler := NewHandler(authSvc, mockDB, nil, nil)
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
}

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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.listAPIKeysByUserFunc = func(userID int64) ([]*apitypes.APIKey, error) {
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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.listAllAPIKeysFunc = func() ([]*apitypes.APIKey, error) {
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
			setupMock:      func(_ *mockDBClient) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDBClient{}
			tt.setupMock(mockDB)

			handler := NewHandler(nil, mockDB, nil, nil)
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
}

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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.getAPIKeyByIDFunc = func(_ int64) (*apitypes.APIKey, error) {
					return &apitypes.APIKey{
						ID:     1,
						UserID: 1, // Same as authenticated user
						Name:   "test-key",
					}, nil
				}
				mockDB.deleteAPIKeyFunc = func(_ int64) error {
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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.getAPIKeyByIDFunc = func(_ int64) (*apitypes.APIKey, error) {
					return &apitypes.APIKey{
						ID:     2,
						UserID: 999, // Different user
						Name:   "other-key",
					}, nil
				}
				mockDB.deleteAPIKeyFunc = func(_ int64) error {
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
			setupMock: func(mockDB *mockDBClient) {
				mockDB.getAPIKeyByIDFunc = func(_ int64) (*apitypes.APIKey, error) {
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
			setupMock:      func(_ *mockDBClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:     "API key not found",
			apiKeyID: "999",
			setAuth:  true,
			userRole: "admin",
			setupMock: func(mockDB *mockDBClient) {
				mockDB.getAPIKeyByIDFunc = func(_ int64) (*apitypes.APIKey, error) {
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

			handler := NewHandler(nil, mockDB, nil, nil)
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
}
