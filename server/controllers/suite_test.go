package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
)

func TestMain(m *testing.M) {
	// Set up test logger
	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	// Bootstrap test environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		panic(err)
	}
	if cfg == nil {
		panic("test environment config is nil")
	}

	// Add SupabaseInstance CRD to the scheme
	err = supacontrolv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}

	// Create the Kubernetes client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(err)
	}
	if k8sClient == nil {
		panic("k8s client is nil")
	}

	// Run tests
	code := m.Run()

	// Tear down test environment
	err = testEnv.Stop()
	if err != nil {
		panic(err)
	}

	os.Exit(code)
}

// Helper function to wait for a condition with timeout
func waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// Helper function to create a test reconciler
func createTestReconciler() *SupabaseInstanceReconciler {
	return &SupabaseInstanceReconciler{
		Client:               k8sClient,
		Scheme:               scheme.Scheme,
		ChartRepo:            "https://supabase-community.github.io/supabase-kubernetes",
		ChartName:            "supabase",
		ChartVersion:         "0.1.0",
		DefaultIngressClass:  "nginx",
		DefaultIngressDomain: "test.example.com",
		CertManagerIssuer:    "letsencrypt-test",
	}
}

// Helper function to create a basic SupabaseInstance
func createBasicInstance(name string) *supacontrolv1alpha1.SupabaseInstance {
	return &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: ctrl.ObjectMeta{
			Name: name,
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: name,
		},
	}
}

// Helper function to get the latest instance state
func getInstanceState(ctx context.Context, t *testing.T, name string) *supacontrolv1alpha1.SupabaseInstance {
	t.Helper()
	instance := &supacontrolv1alpha1.SupabaseInstance{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, instance)
	if err != nil {
		return nil
	}
	return instance
}

// Helper function to cleanup test resources
func cleanupInstance(ctx context.Context, t *testing.T, instance *supacontrolv1alpha1.SupabaseInstance) {
	t.Helper()
	if instance == nil {
		return
	}

	// Delete the instance (this will trigger deletion with finalizer)
	err := k8sClient.Delete(ctx, instance)
	if client.IgnoreNotFound(err) != nil {
		t.Logf("Warning: failed to delete instance %s: %v", instance.Name, err)
	}

	// Wait for deletion to complete (with timeout)
	waitForCondition(30*time.Second, func() bool {
		current := getInstanceState(ctx, t, instance.Name)
		return current == nil
	})
}
