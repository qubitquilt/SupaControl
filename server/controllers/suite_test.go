package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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

var envtestEnabled bool

func TestMain(m *testing.M) {
	// Set up test logger
	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	// Bootstrap test environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deploy", "crds")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		// Controller tests require Kubernetes test binaries (etcd, kube-apiserver).
		// These tests are skipped when envtest is not available.
		//
		// In CI, envtest is configured automatically via the setup-envtest tool.
		// For local development, see server/controllers/README_TEST.md for setup instructions.
		//
		// Skipping controller tests due to missing envtest binaries.
		logf.Log.Info("Skipping controller tests: envtest binaries not available", "error", err.Error())
		envtestEnabled = false
		os.Exit(0)
	}

	envtestEnabled = true

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

	// Create the controller namespace for tests
	// This is needed because the controller creates Jobs in this namespace
	ctx := context.Background()
	controllerNs := &corev1.Namespace{}
	controllerNs.Name = "supacontrol-system"
	err = k8sClient.Create(ctx, controllerNs)
	if err != nil {
		panic(fmt.Sprintf("failed to create controller namespace: %v", err))
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

// Helper function to reconcile and advance instance to Pending phase
func reconcileToPending(ctx context.Context, t *testing.T, reconciler *SupabaseInstanceReconciler, instanceName string) {
	t.Helper()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instanceName}}
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Failed to reconcile to Pending: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected requeue for Pending phase initialization")
	}
}

// Helper function to reconcile and advance instance to Provisioning phase
func reconcileToProvisioning(ctx context.Context, t *testing.T, reconciler *SupabaseInstanceReconciler, instanceName string) {
	t.Helper()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instanceName}}
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Failed to reconcile to Provisioning: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected requeue for Provisioning phase")
	}
}

// Helper function to simulate Job success
func setJobSucceeded(ctx context.Context, t *testing.T, jobName string) {
	t.Helper()
	job := &batchv1.Job{}
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: ControllerNamespace,
	}, job)
	if err != nil {
		t.Fatalf("Failed to get Job: %v", err)
	}

	// Extract instance name from job name and create instance namespace
	// Job names follow pattern: supacontrol-provision-{instance-name} or supacontrol-cleanup-{instance-name}
	// We need to create the supa-{instance-name} namespace that Helm would normally create
	var instanceName string
	if len(jobName) > len("supacontrol-provision-") && jobName[:len("supacontrol-provision-")] == "supacontrol-provision-" {
		instanceName = jobName[len("supacontrol-provision-"):]
	} else if len(jobName) > len("supacontrol-cleanup-") && jobName[:len("supacontrol-cleanup-")] == "supacontrol-cleanup-" {
		instanceName = jobName[len("supacontrol-cleanup-"):]
	}

	if instanceName != "" {
		instanceNs := &corev1.Namespace{}
		instanceNs.Name = "supa-" + instanceName
		// Try to create namespace - ignore if it already exists
		err = k8sClient.Create(ctx, instanceNs)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Logf("Warning: failed to create instance namespace %s: %v", instanceNs.Name, err)
		}
	}

	job.Status.Succeeded = 1
	job.Status.Active = 0
	job.Status.Conditions = []batchv1.JobCondition{
		{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionTrue,
		},
	}
	err = k8sClient.Status().Update(ctx, job)
	if err != nil {
		t.Fatalf("Failed to update Job status to succeeded: %v", err)
	}
}
