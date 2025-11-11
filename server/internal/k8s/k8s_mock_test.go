package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// createTestClient creates a client with a fake clientset for testing
func createTestClient() *Client {
	return &Client{
		clientset: fake.NewSimpleClientset(),
		config:    &rest.Config{},
	}
}

func TestClient_CreateNamespace(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		labels        map[string]string
		wantErr       bool
	}{
		{
			name:          "create namespace with labels",
			namespaceName: "test-namespace",
			labels: map[string]string{
				"app": "supabase",
				"env": "test",
			},
			wantErr: false,
		},
		{
			name:          "create namespace without labels",
			namespaceName: "simple-namespace",
			labels:        nil,
			wantErr:       false,
		},
		{
			name:          "create namespace with empty labels",
			namespaceName: "empty-labels",
			labels:        map[string]string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createTestClient()
			ctx := context.Background()

			err := client.CreateNamespace(ctx, tt.namespaceName, tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify namespace was created
				ns, err := client.clientset.CoreV1().Namespaces().Get(ctx, tt.namespaceName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Failed to get created namespace: %v", err)
					return
				}

				if ns.Name != tt.namespaceName {
					t.Errorf("Namespace name = %v, want %v", ns.Name, tt.namespaceName)
				}

				// Verify labels
				if tt.labels != nil {
					for key, value := range tt.labels {
						if ns.Labels[key] != value {
							t.Errorf("Label %s = %v, want %v", key, ns.Labels[key], value)
						}
					}
				}
			}
		})
	}
}

func TestClient_CreateNamespace_Duplicate(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create namespace first time
	err := client.CreateNamespace(ctx, "duplicate-ns", nil)
	if err != nil {
		t.Fatalf("First CreateNamespace() failed: %v", err)
	}

	// Try to create again - should fail
	err = client.CreateNamespace(ctx, "duplicate-ns", nil)
	if err == nil {
		t.Error("Expected error when creating duplicate namespace")
	}
}

func TestClient_DeleteNamespace(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create a namespace first
	namespaceName := "delete-test"
	err := client.CreateNamespace(ctx, namespaceName, nil)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	// Delete the namespace
	err = client.DeleteNamespace(ctx, namespaceName)
	if err != nil {
		t.Errorf("DeleteNamespace() error = %v", err)
	}

	// Verify it's deleted
	_, err = client.clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err == nil {
		t.Error("Expected error when getting deleted namespace")
	}
}

func TestClient_DeleteNamespace_NotFound(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Try to delete non-existent namespace
	err := client.DeleteNamespace(ctx, "nonexistent-namespace")
	if err == nil {
		t.Error("Expected error when deleting non-existent namespace")
	}
}

func TestClient_NamespaceExists(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create a test namespace
	testNS := "exists-test"
	err := client.CreateNamespace(ctx, testNS, nil)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	tests := []struct {
		name      string
		namespace string
		want      bool
		wantErr   bool
	}{
		{
			name:      "existing namespace",
			namespace: testNS,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "non-existent namespace",
			namespace: "nonexistent",
			want:      false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := client.NamespaceExists(ctx, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("NamespaceExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if exists != tt.want {
				t.Errorf("NamespaceExists() = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestClient_CreateSecret(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create test namespace
	namespace := "secret-test"
	err := client.CreateNamespace(ctx, namespace, nil)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	tests := []struct {
		name       string
		secretName string
		data       map[string][]byte
		labels     map[string]string
		wantErr    bool
	}{
		{
			name:       "create secret with data",
			secretName: "test-secret",
			data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
			labels: map[string]string{
				"app": "test",
			},
			wantErr: false,
		},
		{
			name:       "create secret with empty data",
			secretName: "empty-secret",
			data:       map[string][]byte{},
			labels:     nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CreateSecret(ctx, namespace, tt.secretName, tt.data, tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify secret was created
				secret, err := client.clientset.CoreV1().Secrets(namespace).Get(ctx, tt.secretName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Failed to get created secret: %v", err)
					return
				}

				if secret.Name != tt.secretName {
					t.Errorf("Secret name = %v, want %v", secret.Name, tt.secretName)
				}

				// Verify data
				for key, value := range tt.data {
					if string(secret.Data[key]) != string(value) {
						t.Errorf("Secret data[%s] = %v, want %v", key, secret.Data[key], value)
					}
				}

				// Verify labels
				if tt.labels != nil {
					for key, value := range tt.labels {
						if secret.Labels[key] != value {
							t.Errorf("Secret label %s = %v, want %v", key, secret.Labels[key], value)
						}
					}
				}
			}
		})
	}
}

func TestClient_DeleteSecret(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create test namespace
	namespace := "secret-delete-test"
	err := client.CreateNamespace(ctx, namespace, nil)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	// Create a secret
	secretName := "delete-me"
	err = client.CreateSecret(ctx, namespace, secretName, map[string][]byte{"key": []byte("value")}, nil)
	if err != nil {
		t.Fatalf("Failed to create test secret: %v", err)
	}

	// Delete the secret
	err = client.DeleteSecret(ctx, namespace, secretName)
	if err != nil {
		t.Errorf("DeleteSecret() error = %v", err)
	}

	// Verify it's deleted
	_, err = client.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		t.Error("Expected error when getting deleted secret")
	}
}

func TestClient_CreateIngress(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create test namespace
	namespace := "ingress-test"
	err := client.CreateNamespace(ctx, namespace, nil)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	tests := []struct {
		name         string
		ingressName  string
		host         string
		serviceName  string
		servicePort  int32
		ingressClass string
		wantErr      bool
	}{
		{
			name:         "create ingress",
			ingressName:  "test-ingress",
			host:         "test.example.com",
			serviceName:  "test-service",
			servicePort:  8000,
			ingressClass: "nginx",
			wantErr:      false,
		},
		{
			name:         "create ingress with different port",
			ingressName:  "another-ingress",
			host:         "another.example.com",
			serviceName:  "another-service",
			servicePort:  3000,
			ingressClass: "traefik",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CreateIngress(ctx, namespace, tt.ingressName, tt.host, tt.serviceName, tt.servicePort, tt.ingressClass)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIngress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify ingress was created
				ingress, err := client.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, tt.ingressName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Failed to get created ingress: %v", err)
					return
				}

				if ingress.Name != tt.ingressName {
					t.Errorf("Ingress name = %v, want %v", ingress.Name, tt.ingressName)
				}

				if *ingress.Spec.IngressClassName != tt.ingressClass {
					t.Errorf("Ingress class = %v, want %v", *ingress.Spec.IngressClassName, tt.ingressClass)
				}

				// Verify host
				if len(ingress.Spec.Rules) == 0 {
					t.Error("Expected at least one ingress rule")
					return
				}

				if ingress.Spec.Rules[0].Host != tt.host {
					t.Errorf("Ingress host = %v, want %v", ingress.Spec.Rules[0].Host, tt.host)
				}

				// Verify service backend
				if ingress.Spec.Rules[0].HTTP == nil || len(ingress.Spec.Rules[0].HTTP.Paths) == 0 {
					t.Error("Expected at least one HTTP path")
					return
				}

				backend := ingress.Spec.Rules[0].HTTP.Paths[0].Backend
				if backend.Service.Name != tt.serviceName {
					t.Errorf("Service name = %v, want %v", backend.Service.Name, tt.serviceName)
				}

				if backend.Service.Port.Number != tt.servicePort {
					t.Errorf("Service port = %v, want %v", backend.Service.Port.Number, tt.servicePort)
				}
			}
		})
	}
}

func TestClient_DeleteIngress(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	// Create test namespace
	namespace := "ingress-delete-test"
	err := client.CreateNamespace(ctx, namespace, nil)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	// Create an ingress
	ingressName := "delete-ingress"
	err = client.CreateIngress(ctx, namespace, ingressName, "test.example.com", "test-service", 8000, "nginx")
	if err != nil {
		t.Fatalf("Failed to create test ingress: %v", err)
	}

	// Delete the ingress
	err = client.DeleteIngress(ctx, namespace, ingressName)
	if err != nil {
		t.Errorf("DeleteIngress() error = %v", err)
	}

	// Verify it's deleted
	_, err = client.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err == nil {
		t.Error("Expected error when getting deleted ingress")
	}
}

func TestClient_GetConfig(t *testing.T) {
	client := createTestClient()

	config := client.GetConfig()
	if config == nil {
		t.Error("GetConfig() returned nil")
	}

	if config != client.config {
		t.Error("GetConfig() returned different config than client.config")
	}
}

// TestClient_CreateMultipleResources tests creating multiple resources in sequence
func TestClient_CreateMultipleResources(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	namespace := "multi-test"

	// Create namespace
	err := client.CreateNamespace(ctx, namespace, map[string]string{"test": "multi"})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// Create secret
	err = client.CreateSecret(ctx, namespace, "test-secret", map[string][]byte{
		"key": []byte("value"),
	}, nil)
	if err != nil {
		t.Errorf("Failed to create secret: %v", err)
	}

	// Create ingress
	err = client.CreateIngress(ctx, namespace, "test-ingress", "test.example.com", "svc", 8000, "nginx")
	if err != nil {
		t.Errorf("Failed to create ingress: %v", err)
	}

	// Verify all resources exist
	_, err = client.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Namespace not found: %v", err)
	}

	_, err = client.clientset.CoreV1().Secrets(namespace).Get(ctx, "test-secret", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Secret not found: %v", err)
	}

	_, err = client.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, "test-ingress", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Ingress not found: %v", err)
	}
}

// TestClient_CreateSecret_WithPreexistingObjects tests that creating a secret works with existing resources
func TestClient_CreateSecret_WithPreexistingObjects(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	namespace := "preset-test"

	// Pre-create the namespace manually
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := client.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to pre-create namespace: %v", err)
	}

	// Now create secret using client method
	err = client.CreateSecret(ctx, namespace, "test-secret", map[string][]byte{
		"key": []byte("value"),
	}, nil)
	if err != nil {
		t.Errorf("CreateSecret() failed with pre-existing namespace: %v", err)
	}
}

// TestClient_CreateIngress_WithAnnotations tests that ingress is created with correct annotations
func TestClient_CreateIngress_WithAnnotations(t *testing.T) {
	client := createTestClient()
	ctx := context.Background()

	namespace := "annotation-test"
	err := client.CreateNamespace(ctx, namespace, nil)
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	ingressName := "annotated-ingress"
	err = client.CreateIngress(ctx, namespace, ingressName, "test.example.com", "svc", 8000, "nginx")
	if err != nil {
		t.Fatalf("CreateIngress() failed: %v", err)
	}

	// Verify annotations
	ingress, err := client.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get ingress: %v", err)
	}

	expectedAnnotations := map[string]string{
		"cert-manager.io/cluster-issuer": "letsencrypt-prod",
	}

	for key, expected := range expectedAnnotations {
		if actual, ok := ingress.Annotations[key]; !ok {
			t.Errorf("Missing annotation %s", key)
		} else if actual != expected {
			t.Errorf("Annotation %s = %v, want %v", key, actual, expected)
		}
	}

	// Verify labels
	expectedLabels := map[string]string{
		"app.kubernetes.io/managed-by": "supacontrol",
		"supacontrol.io/instance":      ingressName,
	}

	for key, expected := range expectedLabels {
		if actual, ok := ingress.Labels[key]; !ok {
			t.Errorf("Missing label %s", key)
		} else if actual != expected {
			t.Errorf("Label %s = %v, want %v", key, actual, expected)
		}
	}

	// Verify TLS configuration
	if len(ingress.Spec.TLS) == 0 {
		t.Error("Expected TLS configuration")
	} else {
		tls := ingress.Spec.TLS[0]
		if len(tls.Hosts) == 0 || tls.Hosts[0] != "test.example.com" {
			t.Errorf("TLS hosts = %v, want [test.example.com]", tls.Hosts)
		}
		expectedSecretName := ingressName + "-tls"
		if tls.SecretName != expectedSecretName {
			t.Errorf("TLS secret name = %v, want %v", tls.SecretName, expectedSecretName)
		}
	}
}

// Benchmarks
func BenchmarkGenerateRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateRandomString(32)
	}
}

func BenchmarkGenerateSecurePassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateSecurePassword()
	}
}

func BenchmarkGenerateJWTSecret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateJWTSecret()
	}
}

// Mock setup for client creation tests
func TestNewClient_WithInvalidKubeconfig(t *testing.T) {
	_, err := NewClient("/nonexistent/kubeconfig")
	if err == nil {
		t.Error("Expected error with invalid kubeconfig path")
	}
}

func TestNewClient_EmptyString(t *testing.T) {
	// This will try in-cluster config first, then fall back to default kubeconfig
	// In a test environment, this should fail
	_, err := NewClient("")
	if err == nil {
		t.Skip("Skipping test - appears to have valid kubeconfig or in-cluster config")
	}
	// If we get here, the error is expected
}
