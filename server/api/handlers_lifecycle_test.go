package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
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
				cr.getSupabaseInstanceFunc = func(_ context.Context, _ string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-instance",
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: "test-instance",
							Paused:      true, // Instance is stopped
						},
					}, nil
				}
				cr.updateSupabaseInstanceFunc = func(_ context.Context, _ *supacontrolv1alpha1.SupabaseInstance) error {
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
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
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
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
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
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
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
				cr.updateSupabaseInstanceFunc = func(_ context.Context, _ *supacontrolv1alpha1.SupabaseInstance) error {
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
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
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
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
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
			c, _ := newTestContext(http.MethodGet, "/", "")
			result := handler.convertCRToAPIType(c, tt.cr)
			if result.Status != tt.expected {
				t.Errorf("expected status %s, got %s", tt.expected, result.Status)
			}
			if result.ProjectName != tt.cr.Spec.ProjectName {
				t.Errorf("expected project name %s, got %s", tt.cr.Spec.ProjectName, result.ProjectName)
			}
		})
	}
}

// TestRestartInstance tests the RestartInstance handler
func TestRestartInstance(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		setupMock      func(*mockCRClient, *fake.Clientset)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:         "successful restart",
			instanceName: "test-instance",
			setupMock: func(cr *mockCRClient, k8s *fake.Clientset) {
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: name,
						},
						Status: supacontrolv1alpha1.SupabaseInstanceStatus{
							Namespace: "supa-test-instance",
						},
					}, nil
				}
				// Create a deployment in the namespace
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "supa-test-instance",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": "test"},
							},
						},
					},
				}
				if _, err := k8s.AppsV1().Deployments("supa-test-instance").Create(context.Background(), deployment, metav1.CreateOptions{}); err != nil {
					t.Fatalf("failed to create deployment: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:         "instance not found",
			instanceName: "nonexistent",
			setupMock: func(cr *mockCRClient, _ *fake.Clientset) {
				cr.getSupabaseInstanceFunc = func(_ context.Context, _ string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			fakeClientset := fake.NewSimpleClientset()
			mockK8s := &mockK8sClient{clientset: fakeClientset}
			tt.setupMock(mockCR, fakeClientset)

			handler := NewHandler(nil, nil, mockCR, mockK8s)
			c, rec := newTestContext(http.MethodPost, fmt.Sprintf("/api/v1/instances/%s/restart", tt.instanceName), "")
			c.SetParamNames("name")
			c.SetParamValues(tt.instanceName)

			err := handler.RestartInstance(c)

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

// TestGetLogs tests the GetLogs handler
func TestGetLogs(t *testing.T) {
	tests := []struct {
		name           string
		instanceName   string
		queryParams    map[string]string
		setupMock      func(*mockCRClient, *fake.Clientset)
		expectedStatus int
		expectedError  bool
		expectInOutput string
	}{
		{
			name:         "successful log retrieval",
			instanceName: "test-instance",
			setupMock: func(cr *mockCRClient, k8s *fake.Clientset) {
				cr.getSupabaseInstanceFunc = func(_ context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: name,
						},
						Status: supacontrolv1alpha1.SupabaseInstanceStatus{
							Namespace: "supa-test-instance",
						},
					}, nil
				}
				// Create a pod in the namespace
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "supa-test-instance",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "test-container"},
						},
					},
				}
				if _, err := k8s.CoreV1().Pods("supa-test-instance").Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
					t.Fatalf("failed to create pod: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectInOutput: "=== Logs from pod: test-pod ===",
		},
		{
			name:         "instance not found",
			instanceName: "nonexistent",
			setupMock: func(cr *mockCRClient, _ *fake.Clientset) {
				cr.getSupabaseInstanceFunc = func(_ context.Context, _ string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:         "no pods found",
			instanceName: "empty-instance",
			setupMock: func(cr *mockCRClient, _ *fake.Clientset) {
				cr.getSupabaseInstanceFunc = func(_ context.Context, _ string) (*supacontrolv1alpha1.SupabaseInstance, error) {
					return &supacontrolv1alpha1.SupabaseInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "empty-instance",
						},
						Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
							ProjectName: "empty-instance",
						},
						Status: supacontrolv1alpha1.SupabaseInstanceStatus{
							Namespace: "supa-empty-instance",
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectInOutput: "No pods found for this instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCR := &mockCRClient{}
			fakeClientset := fake.NewSimpleClientset()
			mockK8s := &mockK8sClient{clientset: fakeClientset}
			tt.setupMock(mockCR, fakeClientset)

			handler := NewHandler(nil, nil, mockCR, mockK8s)

			url := fmt.Sprintf("/api/v1/instances/%s/logs", tt.instanceName)
			if tt.queryParams != nil {
				params := []string{}
				for k, v := range tt.queryParams {
					params = append(params, fmt.Sprintf("%s=%s", k, v))
				}
				url = url + "?" + strings.Join(params, "&")
			}

			c, rec := newTestContext(http.MethodGet, url, "")
			c.SetParamNames("name")
			c.SetParamValues(tt.instanceName)

			err := handler.GetLogs(c)

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
				if tt.expectInOutput != "" {
					body := rec.Body.String()
					if !strings.Contains(body, tt.expectInOutput) {
						t.Errorf("expected output to contain %q, got %q", tt.expectInOutput, body)
					}
				}
			}
		})
	}
}
