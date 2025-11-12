package api

import (
	"context"
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

// TestStartInstance tests the StartInstance handler
func TestStartInstance(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		setupMock      func(*mockCRClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:         "successful start",
			instanceName: "test-instance",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: name,
							Paused:      true, // Instance is stopped
						},
					}, nil
				}
				cr.updateSupabaseInstanceFunc = func(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
					return nil
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
		{
			name:         "instance already running",
			instanceName: "running-instance",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: name,
							Paused:      false, // Already running
						},
					}, nil
				}
			},
			expectedStatus: http.StatusConflict,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			tt.setupMock(mockCR)

			handler := NewHandler(nil, nil, mockCR, nil)
			c, rec := newTestContext(http.MethodPost, fmt.Sprintf("/api/v1/instances/%s/start", tt.instanceName), "")
			c.SetParamNames("name")
			c.SetParamValues(tt.instanceName)

			err := handler.StartInstance(c)

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

// TestStopInstance tests the StopInstance handler
func TestStopInstance(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		setupMock      func(*mockCRClient)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:         "successful stop",
			instanceName: "test-instance",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: name,
							Paused:      false, // Instance is running
						},
					}, nil
				}
				cr.updateSupabaseInstanceFunc = func(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
					return nil
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
		{
			name:         "instance already stopped",
			instanceName: "stopped-instance",
			setupMock: func(cr *mockCRClient) {
				cr.getSupabaseInstanceFunc = func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: name,
							Paused:      true, // Already stopped
						},
					}, nil
				}
			},
			expectedStatus: http.StatusConflict,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			tt.setupMock(mockCR)

			handler := NewHandler(nil, nil, mockCR, nil)
			c, rec := newTestContext(http.MethodPost, fmt.Sprintf("/api/v1/instances/%s/stop", tt.instanceName), "")
			c.SetParamNames("name")
			c.SetParamValues(tt.instanceName)

			err := handler.StopInstance(c)

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

// TestGetInstanceNamespace tests the namespace helper function
func TestGetInstanceNamespace(t *testing.T) {
	tests := []struct {
		name     string
		instance *supacontrolv1alpha1.SupabaseInstance
		expected string
	}{
		{
			name: "namespace from status",
			instance: &supacontrolv1alpha1.SupabaseInstance{
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "test-project",
				},
				Status: supacontrolv1alpha1.SupabaseInstanceStatus{
					Namespace: "custom-namespace",
				},
			},
			expected: "custom-namespace",
		},
		{
			name: "namespace generated from project name",
			instance: &supacontrolv1alpha1.SupabaseInstance{
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "test-project",
				},
				Status: supacontrolv1alpha1.SupabaseInstanceStatus{
					Namespace: "",
				},
			},
			expected: "supa-test-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstanceNamespace(tt.instance)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestConvertCRToAPIType tests the CR to API type conversion
func TestConvertCRToAPIType(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil)

	tests := []struct {
		name     string
		cr       *supacontrolv1alpha1.SupabaseInstance
		expected apitypes.InstanceStatus
	}{
		{
			name: "pending phase",
			cr: &supacontrolv1alpha1.SupabaseInstance{
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "test",
				},
				Status: supacontrolv1alpha1.SupabaseInstanceStatus{
					Phase: supacontrolv1alpha1.PhasePending,
				},
			},
			expected: apitypes.StatusProvisioning,
		},
		{
			name: "running phase",
			cr: &supacontrolv1alpha1.SupabaseInstance{
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "test",
				},
				Status: supacontrolv1alpha1.SupabaseInstanceStatus{
					Phase: supacontrolv1alpha1.PhaseRunning,
				},
			},
			expected: apitypes.StatusRunning,
		},
		{
			name: "failed phase",
			cr: &supacontrolv1alpha1.SupabaseInstance{
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "test",
				},
				Status: supacontrolv1alpha1.SupabaseInstanceStatus{
					Phase: supacontrolv1alpha1.PhaseFailed,
				},
			},
			expected: apitypes.StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.convertCRToAPIType(tt.cr)
			if result.Status != tt.expected {
				t.Errorf("expected status %s, got %s", tt.expected, result.Status)
			}
			if result.ProjectName != tt.cr.Spec.ProjectName {
				t.Errorf("expected project name %s, got %s", tt.cr.Spec.ProjectName, result.ProjectName)
			}
		})
	}
}
