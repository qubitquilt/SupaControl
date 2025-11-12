package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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

			handler := NewHandler(nil, nil, mockCR, nil)
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
}

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

			handler := NewHandler(nil, nil, mockCR, nil)
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
}

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

			handler := NewHandler(nil, nil, mockCR, nil)
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
}

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

			handler := NewHandler(nil, nil, mockCR, nil)
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
}
