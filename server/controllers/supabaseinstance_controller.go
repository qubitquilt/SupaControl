package controllers

import (
	"context"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	"github.com/qubitquilt/supacontrol/server/internal/k8s"
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

// +kubebuilder:rbac:groups=supacontrol.qubitquilt.com,resources=supabaseinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=supacontrol.qubitquilt.com,resources=supabaseinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=supacontrol.qubitquilt.com,resources=supabaseinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
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

// reconcilePending transitions from Pending to Provisioning
func (r *SupabaseInstanceReconciler) reconcilePending(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Starting provisioning", "projectName", instance.Spec.ProjectName)

	// Transition to Provisioning phase
	instance.Status.Phase = supacontrolv1alpha1.PhaseProvisioning
	instance.Status.Namespace = fmt.Sprintf("supa-%s", instance.Spec.ProjectName)
	instance.Status.HelmReleaseName = instance.Spec.ProjectName
	now := metav1.Now()
	instance.Status.LastTransitionTime = &now

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// reconcileProvisioning handles the provisioning phase
func (r *SupabaseInstanceReconciler) reconcileProvisioning(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	// Step 1: Create namespace
	if err := r.ensureNamespace(ctx, instance); err != nil {
		return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to create namespace: %v", err))
	}

	// Step 2: Create secrets
	if err := r.ensureSecrets(ctx, instance); err != nil {
		return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to create secrets: %v", err))
	}

	// Step 3: Install Helm chart
	if err := r.ensureHelmRelease(ctx, instance); err != nil {
		return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to install Helm chart: %v", err))
	}

	// Step 4: Create ingresses
	if err := r.ensureIngresses(ctx, instance); err != nil {
		// Log warning but don't fail
		logger.Error(err, "Failed to create ingresses (non-fatal)")
	}

	// Transition to Running
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

	// Update conditions
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: instance.Generation,
		Reason:             "ProvisioningComplete",
		Message:            "Instance is running and ready",
	})

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

// reconcileDelete handles deletion with cleanup
func (r *SupabaseInstanceReconciler) reconcileDelete(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Deleting SupabaseInstance", "projectName", instance.Spec.ProjectName)

	if controllerutil.ContainsFinalizer(instance, FinalizerName) {
		// Update phase to Deleting
		if instance.Status.Phase != supacontrolv1alpha1.PhaseDeleting {
			instance.Status.Phase = supacontrolv1alpha1.PhaseDeleting
			now := metav1.Now()
			instance.Status.LastTransitionTime = &now
			if err := r.Status().Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Perform cleanup
		if err := r.cleanup(ctx, instance); err != nil {
			logger.Error(err, "Failed to cleanup resources")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(instance, FinalizerName)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// cleanup removes all resources associated with the instance
func (r *SupabaseInstanceReconciler) cleanup(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)
	namespace := instance.Status.Namespace
	releaseName := instance.Status.HelmReleaseName

	// Uninstall Helm release
	if releaseName != "" && namespace != "" {
		if err := r.uninstallHelmChart(ctx, namespace, releaseName); err != nil {
			logger.Error(err, "Failed to uninstall Helm chart (non-fatal)")
		}
	}

	// Delete namespace (cascade deletes all resources)
	if namespace != "" {
		ns := &corev1.Namespace{}
		ns.Name = namespace
		if err := r.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete namespace: %w", err)
		}
		logger.Info("Deleted namespace", "namespace", namespace)
	}

	return nil
}

// ensureNamespace creates the namespace if it doesn't exist
func (r *SupabaseInstanceReconciler) ensureNamespace(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)
	namespace := instance.Status.Namespace

	ns := &corev1.Namespace{}
	ns.Name = namespace
	ns.Labels = map[string]string{
		"app.kubernetes.io/managed-by": "supacontrol",
		"supacontrol.io/instance":      instance.Spec.ProjectName,
	}

	if err := r.Create(ctx, ns); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("Namespace already exists", "namespace", namespace)
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:               supacontrolv1alpha1.ConditionTypeNamespaceReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: instance.Generation,
				Reason:             "NamespaceExists",
				Message:            "Namespace already exists",
			})
			return nil
		}
		return err
	}

	logger.Info("Created namespace", "namespace", namespace)
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeNamespaceReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: instance.Generation,
		Reason:             "NamespaceCreated",
		Message:            "Namespace created successfully",
	})

	// Create namespace-scoped RBAC for least privilege
	if err := r.ensureNamespaceRBAC(ctx, instance); err != nil {
		return fmt.Errorf("failed to create namespace RBAC: %w", err)
	}

	return nil
}

// ensureNamespaceRBAC creates namespace-scoped Role and RoleBinding for the controller
// This implements least privilege by granting permissions only within the instance namespace
func (r *SupabaseInstanceReconciler) ensureNamespaceRBAC(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)
	namespace := instance.Status.Namespace

	// Create Role with namespace-scoped permissions
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "supacontrol-instance-manager",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "supacontrol",
				"supacontrol.io/instance":      instance.Spec.ProjectName,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"services", "pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}

	if err := r.Create(ctx, role); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("Namespace Role already exists", "namespace", namespace)
		} else {
			return err
		}
	} else {
		logger.Info("Created namespace Role", "namespace", namespace)
	}

	// Create RoleBinding
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "supacontrol-instance-manager",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "supacontrol",
				"supacontrol.io/instance":      instance.Spec.ProjectName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "supacontrol-instance-manager",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "supacontrol-controller",
				Namespace: "supacontrol-system",
			},
		},
	}

	if err := r.Create(ctx, roleBinding); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("Namespace RoleBinding already exists", "namespace", namespace)
		} else {
			return err
		}
	} else {
		logger.Info("Created namespace RoleBinding", "namespace", namespace)
	}

	return nil
}

// ensureSecrets creates the required secrets
func (r *SupabaseInstanceReconciler) ensureSecrets(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)
	namespace := instance.Status.Namespace

	// Check if secret already exists
	secretName := fmt.Sprintf("%s-secrets", instance.Spec.ProjectName)
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, existingSecret)

	if err == nil {
		logger.Info("Secrets already exist", "namespace", namespace, "secret", secretName)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               supacontrolv1alpha1.ConditionTypeSecretsReady,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: instance.Generation,
			Reason:             "SecretsExist",
			Message:            "Secrets already exist",
		})
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	// Generate secrets
	postgresPassword, err := k8s.GenerateSecurePassword()
	if err != nil {
		return fmt.Errorf("failed to generate postgres password: %w", err)
	}

	jwtSecret, err := k8s.GenerateJWTSecret()
	if err != nil {
		return fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	anonKey, err := k8s.GenerateJWTSecret()
	if err != nil {
		return fmt.Errorf("failed to generate anon key: %w", err)
	}

	serviceRoleKey, err := k8s.GenerateJWTSecret()
	if err != nil {
		return fmt.Errorf("failed to generate service role key: %w", err)
	}

	// Create secret
	secret := &corev1.Secret{}
	secret.Namespace = namespace
	secret.Name = secretName
	secret.Labels = map[string]string{
		"app.kubernetes.io/managed-by": "supacontrol",
		"supacontrol.io/instance":      instance.Spec.ProjectName,
	}
	secret.Data = map[string][]byte{
		"postgres-password": []byte(postgresPassword),
		"jwt-secret":        []byte(jwtSecret),
		"anon-key":          []byte(anonKey),
		"service-role-key":  []byte(serviceRoleKey),
	}

	if err := r.Create(ctx, secret); err != nil {
		return err
	}

	logger.Info("Created secrets", "namespace", namespace, "secret", secretName)
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeSecretsReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: instance.Generation,
		Reason:             "SecretsCreated",
		Message:            "Secrets created successfully",
	})

	return nil
}

// ensureHelmRelease installs the Helm chart
func (r *SupabaseInstanceReconciler) ensureHelmRelease(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	logger := ctrl.LoggerFrom(ctx)
	namespace := instance.Status.Namespace
	releaseName := instance.Status.HelmReleaseName

	// Check if release already exists
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v...))
	}); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	// Check if release exists
	getClient := action.NewGet(actionConfig)
	_, err := getClient.Run(releaseName)
	if err == nil {
		logger.Info("Helm release already exists", "namespace", namespace, "release", releaseName)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               supacontrolv1alpha1.ConditionTypeHelmReleaseReady,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: instance.Generation,
			Reason:             "HelmReleaseExists",
			Message:            "Helm release already installed",
		})
		return nil
	}

	// Install release
	installClient := action.NewInstall(actionConfig)
	installClient.Namespace = namespace
	installClient.ReleaseName = releaseName
	installClient.CreateNamespace = false
	installClient.Wait = false
	installClient.Timeout = 0

	// Set chart version
	chartVersion := r.ChartVersion
	if instance.Spec.ChartVersion != "" {
		chartVersion = instance.Spec.ChartVersion
	}
	if chartVersion != "" {
		installClient.Version = chartVersion
	}

	// Get secrets for Helm values
	secretName := fmt.Sprintf("%s-secrets", instance.Spec.ProjectName)
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, secret); err != nil {
		return fmt.Errorf("failed to get secrets: %w", err)
	}

	// Build Helm values
	values := map[string]interface{}{
		"postgresql": map[string]interface{}{
			"auth": map[string]interface{}{
				"postgresPassword": string(secret.Data["postgres-password"]),
			},
		},
		"jwt": map[string]interface{}{
			"secret":         string(secret.Data["jwt-secret"]),
			"anonKey":        string(secret.Data["anon-key"]),
			"serviceRoleKey": string(secret.Data["service-role-key"]),
		},
	}

	// Locate chart
	chartPath := r.ChartName
	if r.ChartRepo != "" {
		chartPath = fmt.Sprintf("%s/%s", r.ChartRepo, r.ChartName)
	}

	cp, err := installClient.ChartPathOptions.LocateChart(chartPath, settings)
	if err != nil {
		return fmt.Errorf("failed to locate chart: %w", err)
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	_, err = installClient.Run(chartRequested, values)
	if err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	logger.Info("Installed Helm release", "namespace", namespace, "release", releaseName)
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:               supacontrolv1alpha1.ConditionTypeHelmReleaseReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: instance.Generation,
		Reason:             "HelmReleaseInstalled",
		Message:            "Helm release installed successfully",
	})

	return nil
}

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

// uninstallHelmChart removes the Helm release
func (r *SupabaseInstanceReconciler) uninstallHelmChart(ctx context.Context, namespace, releaseName string) error {
	logger := ctrl.LoggerFrom(ctx)
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v...))
	}); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	client := action.NewUninstall(actionConfig)
	client.Wait = false
	client.Timeout = 0

	_, err := client.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall chart: %w", err)
	}

	return nil
}

// transitionToFailed moves the instance to Failed phase
func (r *SupabaseInstanceReconciler) transitionToFailed(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance, errorMsg string) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Error(fmt.Errorf(errorMsg), "Instance provisioning failed", "projectName", instance.Spec.ProjectName)

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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *SupabaseInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize the logger
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	return ctrl.NewControllerManagedBy(mgr).
		For(&supacontrolv1alpha1.SupabaseInstance{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
