package k8s

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes client operations
type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
			}
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfig, err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// GetConfig returns the Kubernetes REST config
func (c *Client) GetConfig() *rest.Config {
	return c.config
}

// CreateNamespace creates a new Kubernetes namespace
func (c *Client) CreateNamespace(ctx context.Context, name string, labels map[string]string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}

	return nil
}

// DeleteNamespace deletes a Kubernetes namespace
func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", name, err)
	}

	return nil
}

// NamespaceExists checks if a namespace exists
func (c *Client) NamespaceExists(ctx context.Context, name string) (bool, error) {
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if err.Error() == "not found" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check namespace: %w", err)
	}
	return true, nil
}

// CreateSecret creates a Kubernetes secret
func (c *Client) CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	_, err := c.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

// DeleteSecret deletes a Kubernetes secret
func (c *Client) DeleteSecret(ctx context.Context, namespace, name string) error {
	err := c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

// CreateIngress creates a Kubernetes Ingress resource
func (c *Client) CreateIngress(ctx context.Context, namespace, name, host, serviceName string, servicePort int32, ingressClass string) error {
	pathTypePrefix := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer": "letsencrypt-prod",
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "supacontrol",
				"supacontrol.io/instance":      name,
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClass,
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{host},
					SecretName: fmt.Sprintf("%s-tls", name),
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: servicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ingress %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

// DeleteIngress deletes a Kubernetes Ingress resource
func (c *Client) DeleteIngress(ctx context.Context, namespace, name string) error {
	err := c.clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete ingress %s in namespace %s: %w", name, namespace, err)
	}

	return nil
}

// GenerateRandomString generates a random base64 encoded string
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateSecurePassword generates a secure random password
func GenerateSecurePassword() (string, error) {
	return GenerateRandomString(32)
}

// GenerateJWTSecret generates a JWT secret
func GenerateJWTSecret() (string, error) {
	return GenerateRandomString(64)
}
