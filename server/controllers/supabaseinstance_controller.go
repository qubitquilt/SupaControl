package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
)

const (
	// FinalizerName is the name of the finalizer added to SupabaseInstance resources
	FinalizerName = "supacontrol.qubitquilt.com/finalizer"
)

// SupabaseInstanceReconciler reconciles a SupabaseInstance object
type SupabaseInstanceReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	ChartRepo            string
	ChartName            string
	ChartVersion         string
	DefaultIngressClass  string
	DefaultIngressDomain string
	CertManagerIssuer    string
}

// +kubebuilder:rbac:groups=supacontrol.qubitquilt.com,resources=supabaseinstances,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=supacontrol.qubitquilt.com,resources=supabaseinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=supacontrol.qubitquilt.com,resources=supabaseinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop
func (r *SupabaseInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	// Fetch the SupabaseInstance resource
	instance := &supacontrolv1alpha1.SupabaseInstance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource deleted, nothing to do
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get SupabaseInstance")
		return ctrl.Result{}, err
	}

	// Check if reconciliation is paused
	if instance.Spec.Paused {
		logger.Info("Reconciliation paused for instance", "projectName", instance.Spec.ProjectName)
		return ctrl.Result{}, nil
	}

	// Handle deletion with finalizer
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(instance, FinalizerName) {
		controllerutil.AddFinalizer(instance, FinalizerName)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile based on current phase
	return r.reconcileNormal(ctx, instance)
}

// reconcileNormal handles the normal reconciliation flow
func (r *SupabaseInstanceReconciler) reconcileNormal(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Reconciling SupabaseInstance", "projectName", instance.Spec.ProjectName, "phase", instance.Status.Phase)

	// Initialize phase if empty
	if instance.Status.Phase == "" {
		instance.Status.Phase = supacontrolv1alpha1.PhasePending
		instance.Status.ObservedGeneration = instance.Generation
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// State machine based on phase
	switch instance.Status.Phase {
	case supacontrolv1alpha1.PhasePending:
		return r.reconcilePending(ctx, instance)
	case supacontrolv1alpha1.PhaseProvisioning:
		return r.reconcileProvisioning(ctx, instance)
	case supacontrolv1alpha1.PhaseProvisioningInProgress:
		return r.reconcileProvisioningInProgress(ctx, instance)
	case supacontrolv1alpha1.PhaseRunning:
		return r.reconcileRunning(ctx, instance)
	case supacontrolv1alpha1.PhaseFailed:
		return r.reconcileFailed(ctx, instance)
	default:
		logger.Info("Unknown phase, resetting to Pending", "phase", instance.Status.Phase)
		instance.Status.Phase = supacontrolv1alpha1.PhasePending
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
}

// reconcilePending transitions from Pending to Provisioning by creating a Job
func (r *SupabaseInstanceReconciler) reconcilePending(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Starting provisioning via Job", "projectName", instance.Spec.ProjectName)

	// Create provisioning Job
	job, err := r.createProvisioningJob(ctx, instance)
	if err != nil {
		return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to create provisioning Job: %v", err))
	}

	// Transition to Provisioning phase
	instance.Status.Phase = supacontrolv1alpha1.PhaseProvisioning
	instance.Status.Namespace = fmt.Sprintf("supa-%s", instance.Spec.ProjectName)
	instance.Status.HelmReleaseName = instance.Spec.ProjectName
	instance.Status.ProvisioningJobName = job.Name
	now := metav1.Now()
	instance.Status.LastTransitionTime = &now

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: instance.Generation,
		Reason:             "ProvisioningJobCreated",
		Message:            fmt.Sprintf("Provisioning Job '%s' created", job.Name),
	})

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue immediately to check Job status
	return ctrl.Result{Requeue: true}, nil
}

// reconcileProvisioning checks the status of the provisioning Job
func (r *SupabaseInstanceReconciler) reconcileProvisioning(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	// Get the provisioning Job status
	jobName := instance.Status.ProvisioningJobName
	if jobName == "" {
		// Job name not set, this shouldn't happen - create job
		logger.Info("Provisioning Job name not set, creating new Job", "projectName", instance.Spec.ProjectName)
		job, err := r.createProvisioningJob(ctx, instance)
		if err != nil {
			return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to create provisioning Job: %v", err))
		}
		instance.Status.ProvisioningJobName = job.Name
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	job, err := r.getJobStatus(ctx, jobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "Provisioning Job not found", "jobName", jobName)
			return r.transitionToFailed(ctx, instance, fmt.Sprintf("Provisioning Job '%s' not found", jobName))
		}
		return ctrl.Result{}, err
	}

	// Check if Job is running - transition to ProvisioningInProgress
	if isJobActive(job) {
		logger.Info("Provisioning Job is running", "jobName", jobName)
		instance.Status.Phase = supacontrolv1alpha1.PhaseProvisioningInProgress
		now := metav1.Now()
		instance.Status.LastTransitionTime = &now

		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               supacontrolv1alpha1.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: instance.Generation,
			Reason:             "ProvisioningInProgress",
			Message:            fmt.Sprintf("Provisioning Job '%s' is running", jobName),
		})

		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}

		// Requeue to check status again
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Check if Job succeeded
	if isJobSucceeded(job) {
		return r.transitionToRunning(ctx, instance)
	}

	// Check if Job failed
	if isJobFailed(job) {
		errMsg := getJobConditionMessage(job)
		if errMsg == "" {
			errMsg = "Provisioning Job failed after retries"
		}
		return r.transitionToFailed(ctx, instance, errMsg)
	}

	// Job exists but hasn't started yet, requeue
	logger.Info("Provisioning Job exists but hasn't started", "jobName", jobName)
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// reconcileProvisioningInProgress monitors the running provisioning Job
func (r *SupabaseInstanceReconciler) reconcileProvisioningInProgress(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	jobName := instance.Status.ProvisioningJobName
	job, err := r.getJobStatus(ctx, jobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "Provisioning Job not found", "jobName", jobName)
			return r.transitionToFailed(ctx, instance, fmt.Sprintf("Provisioning Job '%s' not found", jobName))
		}
		return ctrl.Result{}, err
	}

	// Check if Job succeeded
	if isJobSucceeded(job) {
		logger.Info("Provisioning Job succeeded", "jobName", jobName)
		return r.transitionToRunning(ctx, instance)
	}

	// Check if Job failed
	if isJobFailed(job) {
		errMsg := getJobConditionMessage(job)
		if errMsg == "" {
			errMsg = "Provisioning Job failed after retries"
		}
		logger.Error(errors.New(errMsg), "Provisioning Job failed", "jobName", jobName)
		return r.transitionToFailed(ctx, instance, errMsg)
	}

	// Job still running, requeue
	logger.V(1).Info("Provisioning Job still running", "jobName", jobName, "active", job.Status.Active)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// transitionToRunning transitions the instance to Running phase
func (r *SupabaseInstanceReconciler) transitionToRunning(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Provisioning complete, transitioning to Running", "projectName", instance.Spec.ProjectName)

	instance.Status.Phase = supacontrolv1alpha1.PhaseRunning
	instance.Status.ErrorMessage = ""
	now := metav1.Now()
	instance.Status.LastTransitionTime = &now

	// Set URLs
	ingressDomain := r.DefaultIngressDomain
	if instance.Spec.IngressDomain != "" {
		ingressDomain = instance.Spec.IngressDomain
	}
	instance.Status.StudioURL = fmt.Sprintf("https://%s-studio.%s", instance.Spec.ProjectName, ingressDomain)
	instance.Status.APIURL = fmt.Sprintf("https://%s-api.%s", instance.Spec.ProjectName, ingressDomain)

	// Create ingresses
	if err := r.ensureIngresses(ctx, instance); err != nil {
		// Log warning but don't fail
		logger.Error(err, "Failed to create ingresses (non-fatal)")
	}

	// Update conditions
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: instance.Generation,
		Reason:             "ProvisioningComplete",
		Message:            "Instance is running and ready",
	})

	// Update observedGeneration to indicate this spec has been reconciled
	instance.Status.ObservedGeneration = instance.Generation

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue with delay for periodic health checks
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// reconcileRunning handles the running phase (health checks, drift detection)
func (r *SupabaseInstanceReconciler) reconcileRunning(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	// In a production operator, you would:
	// 1. Check if namespace still exists
	// 2. Check if Helm release is healthy
	// 3. Check if ingresses are configured correctly
	// 4. Detect and reconcile drift
	//
	// For now, we'll just requeue periodically for basic health checks
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// reconcileFailed handles the failed phase
func (r *SupabaseInstanceReconciler) reconcileFailed(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	// In a production operator, you might:
	// 1. Implement retry logic with backoff
	// 2. Alert/notify about the failure
	// 3. Attempt automatic remediation
	//
	// For now, we'll just log and wait
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Instance in failed state", "projectName", instance.Spec.ProjectName, "error", instance.Status.ErrorMessage)

	// Requeue after a delay to allow manual intervention
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// reconcileDelete handles deletion with cleanup using a Job
func (r *SupabaseInstanceReconciler) reconcileDelete(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Deleting SupabaseInstance", "projectName", instance.Spec.ProjectName)

	if controllerutil.ContainsFinalizer(instance, FinalizerName) {
		// Update phase to Deleting if not already
		if instance.Status.Phase != supacontrolv1alpha1.PhaseDeleting && instance.Status.Phase != supacontrolv1alpha1.PhaseDeletingInProgress {
			instance.Status.Phase = supacontrolv1alpha1.PhaseDeleting
			now := metav1.Now()
			instance.Status.LastTransitionTime = &now
			if err := r.Status().Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Perform cleanup via Job
		if err := r.cleanupViaJob(ctx, instance); err != nil {
			logger.Error(err, "Failed to cleanup resources")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		// Remove finalizer after cleanup complete
		controllerutil.RemoveFinalizer(instance, FinalizerName)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// cleanupViaJob performs cleanup using a Kubernetes Job
func (r *SupabaseInstanceReconciler) cleanupViaJob(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)

	// Check if cleanup Job already exists
	jobName := instance.Status.CleanupJobName
	if jobName == "" {
		// Create cleanup Job
		job, err := r.createCleanupJob(ctx, instance)
		if err != nil {
			return fmt.Errorf("failed to create cleanup Job: %w", err)
		}
		instance.Status.CleanupJobName = job.Name
		instance.Status.Phase = supacontrolv1alpha1.PhaseDeletingInProgress
		now := metav1.Now()
		instance.Status.LastTransitionTime = &now
		if err := r.Status().Update(ctx, instance); err != nil {
			return err
		}
		logger.Info("Created cleanup Job", "jobName", job.Name)
		// Return error to requeue and wait for Job completion
		return fmt.Errorf("cleanup Job created, waiting for completion")
	}

	// Get Job status
	job, err := r.getJobStatus(ctx, jobName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Cleanup Job not found, assuming cleanup complete", "jobName", jobName)
			return nil
		}
		return err
	}

	// Check if Job succeeded
	if isJobSucceeded(job) {
		logger.Info("Cleanup Job succeeded", "jobName", jobName)
		return nil
	}

	// Check if Job failed
	if isJobFailed(job) {
		errMsg := getJobConditionMessage(job)
		logger.Error(errors.New(errMsg), "Cleanup Job failed", "jobName", jobName)
		// Don't block deletion on cleanup failure, just log it
		return nil
	}

	// Job still running - ensure phase is DeletingInProgress
	if isJobActive(job) && instance.Status.Phase != supacontrolv1alpha1.PhaseDeletingInProgress {
		logger.Info("Cleanup Job is running, transitioning to DeletingInProgress", "jobName", jobName)
		instance.Status.Phase = supacontrolv1alpha1.PhaseDeletingInProgress
		now := metav1.Now()
		instance.Status.LastTransitionTime = &now
		if err := r.Status().Update(ctx, instance); err != nil {
			return err
		}
	}

	logger.V(1).Info("Cleanup Job still running", "jobName", jobName, "active", job.Status.Active)
	return fmt.Errorf("cleanup Job still running")
}

// transitionToFailed moves the instance to Failed phase
// ensureIngresses creates the ingress resources
func (r *SupabaseInstanceReconciler) ensureIngresses(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)
	namespace := instance.Status.Namespace
	releaseName := instance.Status.HelmReleaseName

	ingressClass := r.DefaultIngressClass
	if instance.Spec.IngressClass != "" {
		ingressClass = instance.Spec.IngressClass
	}

	ingressDomain := r.DefaultIngressDomain
	if instance.Spec.IngressDomain != "" {
		ingressDomain = instance.Spec.IngressDomain
	}

	// Create Studio ingress
	studioHost := fmt.Sprintf("%s-studio.%s", instance.Spec.ProjectName, ingressDomain)
	if err := r.createIngress(ctx, namespace, fmt.Sprintf("%s-studio-ingress", instance.Spec.ProjectName),
		studioHost, fmt.Sprintf("%s-studio", releaseName), 3000, ingressClass, instance); err != nil {
		logger.Error(err, "Failed to create Studio ingress")
	}

	// Create API ingress
	apiHost := fmt.Sprintf("%s-api.%s", instance.Spec.ProjectName, ingressDomain)
	if err := r.createIngress(ctx, namespace, fmt.Sprintf("%s-api-ingress", instance.Spec.ProjectName),
		apiHost, fmt.Sprintf("%s-kong", releaseName), 8000, ingressClass, instance); err != nil {
		logger.Error(err, "Failed to create API ingress")
	}

	logger.Info("Created ingresses", "namespace", namespace)
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeIngressReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: instance.Generation,
		Reason:             "IngressCreated",
		Message:            "Ingresses created successfully",
	})

	return nil
}

// createIngress creates an ingress resource
func (r *SupabaseInstanceReconciler) createIngress(ctx context.Context, namespace, name, host, serviceName string, port int32, ingressClass string, instance *supacontrolv1alpha1.SupabaseInstance) error {
	pathTypePrefix := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{}
	ingress.Namespace = namespace
	ingress.Name = name
	ingress.Labels = map[string]string{
		"app.kubernetes.io/managed-by": "supacontrol",
		"supacontrol.io/instance":      instance.Spec.ProjectName,
	}
	ingress.Annotations = map[string]string{
		"cert-manager.io/cluster-issuer": r.CertManagerIssuer,
	}
	ingress.Spec = networkingv1.IngressSpec{
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
											Number: port,
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

	if err := r.Create(ctx, ingress); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

func (r *SupabaseInstanceReconciler) transitionToFailed(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance, errorMsg string) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Error(errors.New(errorMsg), "Instance provisioning failed", "projectName", instance.Spec.ProjectName)

	instance.Status.Phase = supacontrolv1alpha1.PhaseFailed
	instance.Status.ErrorMessage = errorMsg
	now := metav1.Now()
	instance.Status.LastTransitionTime = &now

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: instance.Generation,
		Reason:             "ProvisioningFailed",
		Message:            errorMsg,
	})

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue with delay for periodic monitoring of failed state
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *SupabaseInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize the logger
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	return ctrl.NewControllerManagedBy(mgr).
		For(&supacontrolv1alpha1.SupabaseInstance{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
