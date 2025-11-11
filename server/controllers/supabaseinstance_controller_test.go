package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
)

// TestReconcilePending_CreatesProvisioningJob tests that reconciling a Pending instance creates a provisioning Job
func TestReconcilePending_CreatesProvisioningJob(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create a test instance
	instance := createBasicInstance("test-pending-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	// Reconcile to initialize phase to Pending
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected first reconcile to requeue")
	}

	// Verify instance is in Pending phase
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found after first reconcile")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhasePending {
		t.Errorf("Expected phase Pending, got %s", current.Status.Phase)
	}

	// Reconcile again to create provisioning Job
	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected second reconcile to requeue")
	}

	// Verify instance transitioned to Provisioning
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found after second reconcile")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseProvisioning {
		t.Errorf("Expected phase Provisioning, got %s", current.Status.Phase)
	}

	// Verify provisioning Job was created
	if current.Status.ProvisioningJobName == "" {
		t.Error("ProvisioningJobName not set in status")
	}

	jobName := current.Status.ProvisioningJobName
	job := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: ControllerNamespace,
	}, job)
	if err != nil {
		t.Fatalf("Provisioning Job not found: %v", err)
	}

	// Verify Job has correct labels
	if job.Labels[JobInstanceLabel] != instance.Spec.ProjectName {
		t.Errorf("Job missing instance label")
	}
	if job.Labels[JobOperationLabel] != OperationProvision {
		t.Errorf("Job missing operation label")
	}

	// Verify Job uses correct ServiceAccount
	if job.Spec.Template.Spec.ServiceAccountName != ServiceAccountName {
		t.Errorf("Expected ServiceAccount %s, got %s", ServiceAccountName, job.Spec.Template.Spec.ServiceAccountName)
	}

	// Verify namespace and HelmReleaseName were set
	expectedNamespace := fmt.Sprintf("supa-%s", instance.Spec.ProjectName)
	if current.Status.Namespace != expectedNamespace {
		t.Errorf("Expected namespace %s, got %s", expectedNamespace, current.Status.Namespace)
	}
	if current.Status.HelmReleaseName != instance.Spec.ProjectName {
		t.Errorf("Expected HelmReleaseName %s, got %s", instance.Spec.ProjectName, current.Status.HelmReleaseName)
	}

	// Verify Ready condition is set to False
	readyCondition := meta.FindStatusCondition(current.Status.Conditions, supacontrolv1alpha1.ConditionTypeReady)
	if readyCondition == nil {
		t.Error("Ready condition not found")
	} else if readyCondition.Status != metav1.ConditionFalse {
		t.Errorf("Expected Ready condition to be False, got %s", readyCondition.Status)
	}
}

// TestReconcileProvisioning_TransitionsToInProgress tests transition from Provisioning to ProvisioningInProgress
func TestReconcileProvisioning_TransitionsToInProgress(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create a test instance in Provisioning phase
	instance := createBasicInstance("test-provisioning-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	// Initialize to Pending and then Provisioning
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Get current state
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseProvisioning {
		t.Fatalf("Instance not in Provisioning phase: %s", current.Status.Phase)
	}

	// Simulate Job becoming active by updating Job status
	jobName := current.Status.ProvisioningJobName
	job := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: ControllerNamespace,
	}, job)
	if err != nil {
		t.Fatalf("Job not found: %v", err)
	}

	job.Status.Active = 1
	err = k8sClient.Status().Update(ctx, job)
	if err != nil {
		t.Fatalf("Failed to update Job status: %v", err)
	}

	// Reconcile to detect active Job
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.RequeueAfter != 10*time.Second {
		t.Errorf("Expected requeue after 10s, got %v", result.RequeueAfter)
	}

	// Verify instance transitioned to ProvisioningInProgress
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found after reconcile")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseProvisioningInProgress {
		t.Errorf("Expected phase ProvisioningInProgress, got %s", current.Status.Phase)
	}

	// Verify Ready condition reflects in-progress state
	readyCondition := meta.FindStatusCondition(current.Status.Conditions, supacontrolv1alpha1.ConditionTypeReady)
	if readyCondition == nil {
		t.Error("Ready condition not found")
	} else {
		if readyCondition.Status != metav1.ConditionFalse {
			t.Errorf("Expected Ready condition False, got %s", readyCondition.Status)
		}
		if readyCondition.Reason != "ProvisioningInProgress" {
			t.Errorf("Expected reason ProvisioningInProgress, got %s", readyCondition.Reason)
		}
	}
}

// TestReconcileProvisioningInProgress_HandlesJobSuccess tests transition to Running when Job succeeds
func TestReconcileProvisioningInProgress_HandlesJobSuccess(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create and initialize instance
	instance := createBasicInstance("test-job-success-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Get Job and set it to active, then successful
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}

	jobName := current.Status.ProvisioningJobName

	// Simulate Job success (also creates instance namespace)
	setJobSucceeded(ctx, t, jobName)

	// Reconcile to detect Job success
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify instance transitioned to Running
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found after reconcile")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseRunning {
		t.Errorf("Expected phase Running, got %s", current.Status.Phase)
	}

	// Verify Ready condition is True
	readyCondition := meta.FindStatusCondition(current.Status.Conditions, supacontrolv1alpha1.ConditionTypeReady)
	if readyCondition == nil {
		t.Error("Ready condition not found")
	} else {
		if readyCondition.Status != metav1.ConditionTrue {
			t.Errorf("Expected Ready condition True, got %s", readyCondition.Status)
		}
		if readyCondition.Reason != "ProvisioningComplete" {
			t.Errorf("Expected reason ProvisioningComplete, got %s", readyCondition.Reason)
		}
	}

	// Verify URLs are set
	if current.Status.StudioURL == "" {
		t.Error("StudioURL not set")
	}
	if current.Status.APIURL == "" {
		t.Error("APIURL not set")
	}

	// Verify error message is cleared
	if current.Status.ErrorMessage != "" {
		t.Errorf("ErrorMessage should be empty, got: %s", current.Status.ErrorMessage)
	}

	// Verify observedGeneration is updated
	if current.Status.ObservedGeneration != current.Generation {
		t.Errorf("ObservedGeneration %d doesn't match Generation %d", current.Status.ObservedGeneration, current.Generation)
	}

	// Verify requeue is set for periodic health checks
	if result.RequeueAfter == 0 {
		t.Error("Expected periodic requeue for health checks")
	}
}

// TestReconcileProvisioningInProgress_HandlesJobFailure tests transition to Failed when Job fails
func TestReconcileProvisioningInProgress_HandlesJobFailure(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create and initialize instance
	instance := createBasicInstance("test-job-failure-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Get Job and simulate failure
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}

	jobName := current.Status.ProvisioningJobName
	job := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: ControllerNamespace,
	}, job)
	if err != nil {
		t.Fatalf("Job not found: %v", err)
	}

	// Simulate Job failure (exceeded backoff limit)
	backoffLimit := int32(3)
	job.Spec.BackoffLimit = &backoffLimit
	job.Status.Failed = 3
	job.Status.Active = 0
	job.Status.Conditions = []batchv1.JobCondition{
		{
			Type:    batchv1.JobFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackoffLimitExceeded",
			Message: "Job has reached the specified backoff limit",
		},
	}
	err = k8sClient.Status().Update(ctx, job)
	if err != nil {
		t.Fatalf("Failed to update Job status: %v", err)
	}

	// Reconcile to detect Job failure
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify instance transitioned to Failed
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found after reconcile")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseFailed {
		t.Errorf("Expected phase Failed, got %s", current.Status.Phase)
	}

	// Verify error message is set
	if current.Status.ErrorMessage == "" {
		t.Error("ErrorMessage not set")
	}

	// Verify Ready condition is False with failure reason
	readyCondition := meta.FindStatusCondition(current.Status.Conditions, supacontrolv1alpha1.ConditionTypeReady)
	if readyCondition == nil {
		t.Error("Ready condition not found")
	} else {
		if readyCondition.Status != metav1.ConditionFalse {
			t.Errorf("Expected Ready condition False, got %s", readyCondition.Status)
		}
		if readyCondition.Reason != "ProvisioningFailed" {
			t.Errorf("Expected reason ProvisioningFailed, got %s", readyCondition.Reason)
		}
	}

	// Verify reconcile returns without error (fail state is terminal)
	if result.RequeueAfter == 0 {
		t.Error("Expected delayed requeue for failed state")
	}
}

// TestReconcileDelete_CreatesCleanupJob tests that deleting an instance creates a cleanup Job
func TestReconcileDelete_CreatesCleanupJob(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create and initialize instance to Running
	instance := createBasicInstance("test-delete-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Get current state
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}

	// Simulate successful provisioning (also creates instance namespace)
	jobName := current.Status.ProvisioningJobName
	setJobSucceeded(ctx, t, jobName)

	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Failed to transition to Running: %v", err)
	}

	// Verify instance is Running
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseRunning {
		t.Fatalf("Instance not in Running phase: %s", current.Status.Phase)
	}

	// Delete the instance
	err = k8sClient.Delete(ctx, current)
	if err != nil {
		t.Fatalf("Failed to delete instance: %v", err)
	}

	// Reconcile to handle deletion
	result, err := reconciler.Reconcile(ctx, req)
	if err == nil {
		t.Error("Expected error indicating cleanup in progress")
	}
	if result.RequeueAfter == 0 {
		t.Error("Expected requeue for cleanup monitoring")
	}

	// Get updated state
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance should still exist during cleanup")
	}

	// Verify instance transitioned to Deleting or DeletingInProgress
	if current.Status.Phase != supacontrolv1alpha1.PhaseDeleting &&
		current.Status.Phase != supacontrolv1alpha1.PhaseDeletingInProgress {
		t.Errorf("Expected phase Deleting or DeletingInProgress, got %s", current.Status.Phase)
	}

	// Verify cleanup Job was created
	if current.Status.CleanupJobName == "" {
		t.Error("CleanupJobName not set in status")
	}

	cleanupJobName := current.Status.CleanupJobName
	cleanupJob := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      cleanupJobName,
		Namespace: ControllerNamespace,
	}, cleanupJob)
	if err != nil {
		t.Fatalf("Cleanup Job not found: %v", err)
	}

	// Verify Job has correct labels
	if cleanupJob.Labels[JobInstanceLabel] != instance.Spec.ProjectName {
		t.Error("Cleanup Job missing instance label")
	}
	if cleanupJob.Labels[JobOperationLabel] != OperationCleanup {
		t.Error("Cleanup Job missing cleanup operation label")
	}

	// Verify Job uses correct ServiceAccount
	if cleanupJob.Spec.Template.Spec.ServiceAccountName != ServiceAccountName {
		t.Errorf("Expected ServiceAccount %s, got %s", ServiceAccountName, cleanupJob.Spec.Template.Spec.ServiceAccountName)
	}
}

// TestCleanupViaJob_TransitionsToDeletingInProgress tests cleanup Job state transitions
func TestCleanupViaJob_TransitionsToDeletingInProgress(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create and prepare instance for deletion
	instance := createBasicInstance("test-cleanup-progress-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Transition to Running (simulate successful provision)
	current := getInstanceState(ctx, t, instance.Name)
	if current != nil && current.Status.ProvisioningJobName != "" {
		setJobSucceeded(ctx, t, current.Status.ProvisioningJobName)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Failed to reconcile Running state: %v", err)
	}

	// Delete and start cleanup
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}
	_ = k8sClient.Delete(ctx, current)
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Failed to create cleanup Job: %v", err)
	}

	// Get cleanup Job and make it active
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found during cleanup")
	}

	cleanupJobName := current.Status.CleanupJobName
	if cleanupJobName == "" {
		t.Fatal("Cleanup Job not created")
	}

	cleanupJob := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      cleanupJobName,
		Namespace: ControllerNamespace,
	}, cleanupJob)
	if err != nil {
		t.Fatalf("Cleanup Job not found: %v", err)
	}

	// Simulate Job becoming active
	cleanupJob.Status.Active = 1
	err = k8sClient.Status().Update(ctx, cleanupJob)
	if err != nil {
		t.Fatalf("Failed to update cleanup Job status: %v", err)
	}

	// Reconcile to detect active cleanup Job
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Logf("Reconcile error (may be expected): %v", err)
	}

	// Verify instance is in DeletingInProgress
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseDeletingInProgress {
		t.Errorf("Expected phase DeletingInProgress, got %s", current.Status.Phase)
	}

	// Simulate cleanup Job success
	cleanupJob.Status.Active = 0
	cleanupJob.Status.Succeeded = 1
	cleanupJob.Status.Conditions = []batchv1.JobCondition{
		{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
	}
	err = k8sClient.Status().Update(ctx, cleanupJob)
	if err != nil {
		t.Fatalf("Failed to update cleanup Job to success: %v", err)
	}

	// Final reconcile should complete deletion
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Logf("Reconcile error (may be expected): %v", err)
	}

	// Verify instance is eventually deleted
	success := waitForCondition(5*time.Second, func() bool {
		current = getInstanceState(ctx, t, instance.Name)
		return current == nil
	})
	if !success {
		t.Error("Instance not deleted after cleanup Job succeeded")
	}
}

// TestJobOwnerReferences_PreventOrphans tests that Jobs have proper owner references
func TestJobOwnerReferences_PreventOrphans(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create instance
	instance := createBasicInstance("test-owner-refs-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Get the created Job
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}

	jobName := current.Status.ProvisioningJobName
	job := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: ControllerNamespace,
	}, job)
	if err != nil {
		t.Fatalf("Job not found: %v", err)
	}

	// Verify owner references are set
	if len(job.OwnerReferences) == 0 {
		t.Fatal("Job has no owner references")
	}

	ownerRef := job.OwnerReferences[0]

	// Verify owner reference points to the instance
	if ownerRef.Name != instance.Name {
		t.Errorf("Owner reference name mismatch: expected %s, got %s", instance.Name, ownerRef.Name)
	}

	if ownerRef.UID != instance.UID {
		t.Errorf("Owner reference UID mismatch")
	}

	if ownerRef.Kind != "SupabaseInstance" {
		t.Errorf("Owner reference kind mismatch: expected SupabaseInstance, got %s", ownerRef.Kind)
	}

	// Verify controller reference is set
	if ownerRef.Controller == nil || !*ownerRef.Controller {
		t.Error("Owner reference Controller field not set to true")
	}

	// Verify BlockOwnerDeletion is set
	if ownerRef.BlockOwnerDeletion == nil || !*ownerRef.BlockOwnerDeletion {
		t.Error("Owner reference BlockOwnerDeletion not set to true")
	}

	// Test orphan prevention by deleting instance
	err = k8sClient.Delete(ctx, current)
	if err != nil {
		t.Fatalf("Failed to delete instance: %v", err)
	}

	// Reconcile deletion
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Logf("Reconcile error (may be expected): %v", err)
	}

	// Wait a bit for Kubernetes to process deletion
	time.Sleep(1 * time.Second)

	// Verify Job is eventually deleted due to owner reference
	// (Note: In envtest, garbage collection might not work exactly like a real cluster,
	// but we verify the owner references are properly set)
	t.Logf("Job has proper owner references that would trigger deletion in a real cluster")
}

// TestReconcilePaused_SkipsReconciliation tests that paused instances are not reconciled
func TestReconcilePaused_SkipsReconciliation(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create instance with paused=true
	instance := createBasicInstance("test-paused-001")
	instance.Spec.Paused = true
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	// Try to reconcile
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no requeue
	if result.Requeue || result.RequeueAfter > 0 {
		t.Error("Expected no requeue for paused instance")
	}

	// Verify instance has no phase set (not initialized)
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}
	if current.Status.Phase != "" {
		t.Errorf("Paused instance should not have phase set, got: %s", current.Status.Phase)
	}
}

// TestJobTimeout_HandlesActiveDeadlineSeconds tests that Job timeouts are respected
func TestJobTimeout_HandlesActiveDeadlineSeconds(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create instance
	instance := createBasicInstance("test-timeout-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Get Job
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}

	jobName := current.Status.ProvisioningJobName
	job := &batchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: ControllerNamespace,
	}, job)
	if err != nil {
		t.Fatalf("Job not found: %v", err)
	}

	// Verify ActiveDeadlineSeconds is set
	if job.Spec.ActiveDeadlineSeconds == nil {
		t.Fatal("Job does not have ActiveDeadlineSeconds set")
	}

	expectedDeadline := int64(900) // 15 minutes
	if *job.Spec.ActiveDeadlineSeconds != expectedDeadline {
		t.Errorf("Expected ActiveDeadlineSeconds %d, got %d", expectedDeadline, *job.Spec.ActiveDeadlineSeconds)
	}

	// Verify BackoffLimit is set for retries
	if job.Spec.BackoffLimit == nil {
		t.Fatal("Job does not have BackoffLimit set")
	}

	expectedBackoff := int32(3)
	if *job.Spec.BackoffLimit != expectedBackoff {
		t.Errorf("Expected BackoffLimit %d, got %d", expectedBackoff, *job.Spec.BackoffLimit)
	}

	// Verify TTLSecondsAfterFinished is set
	if job.Spec.TTLSecondsAfterFinished == nil {
		t.Fatal("Job does not have TTLSecondsAfterFinished set")
	}

	expectedTTL := int32(3600) // 1 hour
	if *job.Spec.TTLSecondsAfterFinished != expectedTTL {
		t.Errorf("Expected TTLSecondsAfterFinished %d, got %d", expectedTTL, *job.Spec.TTLSecondsAfterFinished)
	}
}

// TestFinalizer_AddedOnCreation tests that finalizer is added to new instances
func TestFinalizer_AddedOnCreation(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create instance without finalizer
	instance := createBasicInstance("test-finalizer-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	// Initial reconcile should add finalizer
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer was added
	current := getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}

	finalizerFound := false
	for _, f := range current.Finalizers {
		if f == FinalizerName {
			finalizerFound = true
			break
		}
	}

	if !finalizerFound {
		t.Errorf("Finalizer %s not found on instance. Finalizers: %v", FinalizerName, current.Finalizers)
	}
}

// TestReconcileRunning_PeriodicHealthChecks tests that Running instances are periodically requeued
func TestReconcileRunning_PeriodicHealthChecks(t *testing.T) {
	ctx := context.Background()
	reconciler := createTestReconciler()

	// Create and transition instance to Running
	instance := createBasicInstance("test-running-checks-001")
	err := k8sClient.Create(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}
	defer cleanupInstance(ctx, t, instance)

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
	reconcileToPending(ctx, t, reconciler, instance.Name)
	reconcileToProvisioning(ctx, t, reconciler, instance.Name)

	// Transition to Running
	current := getInstanceState(ctx, t, instance.Name)
	if current != nil && current.Status.ProvisioningJobName != "" {
		setJobSucceeded(ctx, t, current.Status.ProvisioningJobName)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("Failed to reconcile Running state: %v", err)
	}

	// Verify instance is Running
	current = getInstanceState(ctx, t, instance.Name)
	if current == nil {
		t.Fatal("Instance not found")
	}
	if current.Status.Phase != supacontrolv1alpha1.PhaseRunning {
		t.Fatalf("Instance not in Running phase: %s", current.Status.Phase)
	}

	// Reconcile Running instance
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile Running instance failed: %v", err)
	}

	// Verify periodic requeue is set
	if result.RequeueAfter == 0 {
		t.Error("Expected periodic requeue for Running instance health checks")
	}

	expectedRequeue := 5 * time.Minute
	if result.RequeueAfter != expectedRequeue {
		t.Errorf("Expected requeue after %v, got %v", expectedRequeue, result.RequeueAfter)
	}
}
