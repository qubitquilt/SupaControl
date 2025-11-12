package controllers

import (
	"context"
	"crypto/sha256"
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

// Global counter for unique suffix generation
var nameSuffixCounter int64 = 0

// Helper function to create a basic SupabaseInstance
func createBasicInstance(name string) *supacontrolv1alpha1.SupabaseInstance {
	// Create a short, unique identifier based on the test name
	// Use hash of test name for uniqueness but keep it short
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(name)))[:6]

	// Use nanosecond timestamp and an incrementing counter for absolute uniqueness
	now := time.Now()

	// Create a unique suffix using time and counter
	// Format time as hex and append counter
	timeHex := fmt.Sprintf("%x", now.UnixNano())[:10] // Use first 10 chars of time hex
	counter := nameSuffixCounter
	nameSuffixCounter++ // Increment for next call

	// Combine hash, time, and counter for absolute uniqueness
	// Format: t-{hash}-{timeHex}-{counter} where:
	// - hash: 6 chars (from test name)
	// - separator: 1 char ("-")
	// - timeHex: 10 chars (from nanosecond timestamp)
	// - separator: 1 char ("-")
	// - counter: variable length (but we keep it short)
	// Total: 2 + 6 + 1 + 10 + 1 + ~3 = ~23 chars (well under 39 char limit)
	projectName := fmt.Sprintf("t-%s-%s-%d", hash, timeHex, counter)

	return &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: ctrl.ObjectMeta{
			Name: projectName,
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: projectName,
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
	if result.RequeueAfter == 0 {
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
	if result.RequeueAfter == 0 {
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
	if strings.HasPrefix(jobName, "supacontrol-provision-") {
		instanceName = strings.TrimPrefix(jobName, "supacontrol-provision-")
	} else if strings.HasPrefix(jobName, "supacontrol-cleanup-") {
		instanceName = strings.TrimPrefix(jobName, "supacontrol-cleanup-")
	}

	if instanceName != "" {
		instanceNs := &corev1.Namespace{}
		instanceNs.Name = "supa-" + instanceName
		t.Logf("Creating instance namespace: %s for job: %s", instanceNs.Name, jobName)
		err = k8sClient.Create(ctx, instanceNs)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				t.Logf("Instance namespace %s already exists (OK)", instanceNs.Name)
			} else {
				t.Fatalf("Failed to create instance namespace %s: %v", instanceNs.Name, err)
			}
		} else {
			t.Logf("Successfully created instance namespace: %s", instanceNs.Name)
		}
	} else {
		t.Logf("Warning: could not extract instance name from job name: %s", jobName)
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

// TestCreateBasicInstance_NameLengthCompliance tests that generated names comply with Kubernetes limits
func TestCreateBasicInstance_NameLengthCompliance(t *testing.T) {
	// Test the problematic test names mentioned in the CI output
	testNames := []string{
		"TestReconcilePending_CreatesProvisioningJob",
		"TestReconcileProvisioning_TransitionsToInProgress",
		"TestJobTimeout_HandlesActiveDeadlineSeconds",
		"TestJobOwnerReferences_PreventOrphans",
		"TestCleanupViaJob_TransitionsToDeletingInProgress",
		"TestReconcileDelete_CreatesCleanupJob",
		"TestReconcileProvisioningInProgress_HandlesJobSuccess",
		"TestReconcileProvisioningInProgress_HandlesJobFailure",
		"TestReconcilePaused_SkipsReconciliation",
		"TestFinalizer_AddedOnCreation",
		"TestReconcileRunning_PeriodicHealthChecks",
	}

	maxNameLength := 0
	maxJobNameLength := 0

	for _, testName := range testNames {
		// Use the actual createBasicInstance function
		instance := createBasicInstance(testName)

		provisionJobName := fmt.Sprintf("supacontrol-provision-%s", instance.Spec.ProjectName)
		cleanupJobName := fmt.Sprintf("supacontrol-cleanup-%s", instance.Spec.ProjectName)

		maxNameLength = max(maxNameLength, len(instance.Spec.ProjectName))
		maxJobNameLength = max(maxJobNameLength, len(provisionJobName))
		maxJobNameLength = max(maxJobNameLength, len(cleanupJobName))

		t.Logf("Test: %s", testName)
		t.Logf("  Project Name: %s (%d chars)", instance.Spec.ProjectName, len(instance.Spec.ProjectName))
		t.Logf("  Provision Job: %s (%d chars)", provisionJobName, len(provisionJobName))
		t.Logf("  Cleanup Job: %s (%d chars)", cleanupJobName, len(cleanupJobName))

		if len(provisionJobName) > 63 {
			t.Errorf("Provision Job name exceeds 63 chars: %s", provisionJobName)
		}
		if len(cleanupJobName) > 63 {
			t.Errorf("Cleanup Job name exceeds 63 chars: %s", cleanupJobName)
		}
	}

	t.Logf("Maximum project name length: %d (limit: 63)", maxNameLength)
	t.Logf("Maximum job name length: %d (limit: 63)", maxJobNameLength)

	if maxJobNameLength > 63 {
		t.Errorf("FAILURE: Some job names exceed 63 characters! Max: %d", maxJobNameLength)
	} else {
		t.Logf("SUCCESS: All job names comply with Kubernetes 63-character label limit!")
	}
}

// TestCreateBasicInstance_Uniqueness tests that generated names are unique
func TestCreateBasicInstance_Uniqueness(t *testing.T) {
	instance1 := createBasicInstance("TestSameName")
	instance2 := createBasicInstance("TestSameName")

	if instance1.Spec.ProjectName == instance2.Spec.ProjectName {
		t.Errorf("Generated names should be unique: %s == %s",
			instance1.Spec.ProjectName, instance2.Spec.ProjectName)
	}

	t.Logf("Instance 1: %s", instance1.Spec.ProjectName)
	t.Logf("Instance 2: %s", instance2.Spec.ProjectName)
}

// TestCreateBasicInstance_NameSanitization tests that names are properly sanitized
func TestCreateBasicInstance_NameSanitization(t *testing.T) {
	// Test with problematic characters
	instance := createBasicInstance("Test/With\\Special!Chars@And_Long_Name_That_Would_Exceed_Limits")

	// Should contain only lowercase letters, numbers, hyphens
	if !isValidK8sName(instance.Spec.ProjectName) {
		t.Errorf("Generated name is not a valid Kubernetes resource name: %s", instance.Spec.ProjectName)
	}

	// Should start with letter (our prefix "t-" ensures this)
	if !strings.HasPrefix(instance.Spec.ProjectName, "t-") {
		t.Errorf("Generated name should start with 't-': %s", instance.Spec.ProjectName)
	}

	t.Logf("Sanitized name: %s", instance.Spec.ProjectName)
}

// Helper function to check if a name is valid for Kubernetes
func isValidK8sName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	// Kubernetes resource names must start and end with alphanumeric
	// and can contain hyphens in between
	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '-' && i > 0 && i < len(name)-1)) {
			return false
		}
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
