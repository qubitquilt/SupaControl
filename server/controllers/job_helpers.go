package controllers

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
)

const (
	// JobOperationLabel is the label key for job operation type
	JobOperationLabel = "supacontrol.io/operation"

	// JobInstanceLabel is the label key for instance name
	JobInstanceLabel = "supacontrol.io/instance"

	// OperationProvision is the provision operation value
	OperationProvision = "provision"

	// OperationCleanup is the cleanup operation value
	OperationCleanup = "cleanup"

	// ProvisionerImage is the Docker image used for provisioning Jobs
	ProvisionerImage = "alpine/helm:3.13.0"

	// ServiceAccountName is the name of the ServiceAccount used by Jobs
	ServiceAccountName = "supacontrol-provisioner"

	// ControllerNamespace is the namespace where the controller runs
	ControllerNamespace = "supacontrol-system"
)

// createProvisioningJob creates a Kubernetes Job for provisioning a Supabase instance
func (r *SupabaseInstanceReconciler) createProvisioningJob(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (*batchv1.Job, error) {
	logger := ctrl.LoggerFrom(ctx)

	jobName := fmt.Sprintf("supacontrol-provision-%s", instance.Spec.ProjectName)
	namespace := fmt.Sprintf("supa-%s", instance.Spec.ProjectName)

	// Check if job already exists
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: ControllerNamespace, Name: jobName}, existingJob)
	if err == nil {
		logger.Info("Provisioning Job already exists", "jobName", jobName)
		return existingJob, nil
	}

	// Determine chart version
	chartVersion := r.ChartVersion
	if instance.Spec.ChartVersion != "" {
		chartVersion = instance.Spec.ChartVersion
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: ControllerNamespace,
			Labels: map[string]string{
				JobInstanceLabel:              instance.Spec.ProjectName,
				JobOperationLabel:             OperationProvision,
				"app.kubernetes.io/name":      "supacontrol",
				"app.kubernetes.io/component": "provisioner",
			},
			Annotations: map[string]string{
				"supacontrol.io/instance-uid": string(instance.UID),
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(instance, supacontrolv1alpha1.GroupVersion.WithKind("SupabaseInstance"))},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            pointer.Int32(3),    // Retry up to 3 times
			ActiveDeadlineSeconds:   pointer.Int64(900),  // 15 minute timeout
			TTLSecondsAfterFinished: pointer.Int32(3600), // Clean up after 1 hour
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						JobInstanceLabel:  instance.Spec.ProjectName,
						JobOperationLabel: OperationProvision,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "provisioner",
							Image:   ProvisionerImage,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{`
set -euo pipefail

echo "========================================"
echo "SupaControl Provisioning Job"
echo "Instance: $INSTANCE_NAME"
echo "Namespace: $NAMESPACE"
echo "========================================"

# Step 1: Create namespace
echo "[1/5] Creating namespace: $NAMESPACE"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace "$NAMESPACE" \
  app.kubernetes.io/managed-by=supacontrol \
  supacontrol.io/instance="$INSTANCE_NAME" \
  --overwrite

# Step 2: Generate and create secrets
echo "[2/5] Generating secrets"
POSTGRES_PASSWORD=$(openssl rand -base64 32 | tr -d '\n')
JWT_SECRET=$(openssl rand -base64 64 | tr -d '\n')
ANON_KEY=$(openssl rand -base64 32 | tr -d '\n')
SERVICE_ROLE_KEY=$(openssl rand -base64 32 | tr -d '\n')

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: $INSTANCE_NAME-secrets
  namespace: $NAMESPACE
  labels:
    app.kubernetes.io/managed-by: supacontrol
    supacontrol.io/instance: $INSTANCE_NAME
stringData:
  postgres-password: "$POSTGRES_PASSWORD"
  jwt-secret: "$JWT_SECRET"
  anon-key: "$ANON_KEY"
  service-role-key: "$SERVICE_ROLE_KEY"
EOF

echo "[2/5] Secrets created successfully"

# Step 3: Add Helm repository
echo "[3/5] Adding Helm repository: $CHART_REPO"
helm repo add supabase-community "$CHART_REPO" || true
helm repo update

# Step 4: Install Helm chart
echo "[4/5] Installing Helm chart: $CHART_NAME (version: $CHART_VERSION)"
helm install "$INSTANCE_NAME" supabase-community/"$CHART_NAME" \
  --namespace "$NAMESPACE" \
  --version "$CHART_VERSION" \
  --set postgresql.auth.postgresPassword="$POSTGRES_PASSWORD" \
  --set jwt.secret="$JWT_SECRET" \
  --set jwt.anonKey="$ANON_KEY" \
  --set jwt.serviceRoleKey="$SERVICE_ROLE_KEY" \
  --wait \
  --timeout 10m

echo "[4/5] Helm chart installed successfully"

# Step 5: Report completion
echo "[5/5] Provisioning complete!"
echo "========================================"
echo "Instance '$INSTANCE_NAME' is now running"
echo "Namespace: $NAMESPACE"
echo "========================================"
`},
							Env: []corev1.EnvVar{
								{
									Name:  "INSTANCE_NAME",
									Value: instance.Spec.ProjectName,
								},
								{
									Name:  "NAMESPACE",
									Value: namespace,
								},
								{
									Name:  "CHART_REPO",
									Value: r.ChartRepo,
								},
								{
									Name:  "CHART_NAME",
									Value: r.ChartName,
								},
								{
									Name:  "CHART_VERSION",
									Value: chartVersion,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, job, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create provisioning Job: %w", err)
	}

	logger.Info("Created provisioning Job", "jobName", jobName, "namespace", ControllerNamespace)
	return job, nil
}

// createCleanupJob creates a Kubernetes Job for cleaning up a Supabase instance
func (r *SupabaseInstanceReconciler) createCleanupJob(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (*batchv1.Job, error) {
	logger := ctrl.LoggerFrom(ctx)

	jobName := fmt.Sprintf("supacontrol-cleanup-%s", instance.Spec.ProjectName)
	namespace := instance.Status.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("supa-%s", instance.Spec.ProjectName)
	}

	// Check if job already exists
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: ControllerNamespace, Name: jobName}, existingJob)
	if err == nil {
		logger.Info("Cleanup Job already exists", "jobName", jobName)
		return existingJob, nil
	}

	releaseName := instance.Status.HelmReleaseName
	if releaseName == "" {
		releaseName = instance.Spec.ProjectName
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: ControllerNamespace,
			Labels: map[string]string{
				JobInstanceLabel:              instance.Spec.ProjectName,
				JobOperationLabel:             OperationCleanup,
				"app.kubernetes.io/name":      "supacontrol",
				"app.kubernetes.io/component": "provisioner",
			},
			Annotations: map[string]string{
				"supacontrol.io/instance-uid": string(instance.UID),
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(instance, supacontrolv1alpha1.GroupVersion.WithKind("SupabaseInstance"))},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            pointer.Int32(2),    // Retry up to 2 times
			ActiveDeadlineSeconds:   pointer.Int64(600),  // 10 minute timeout
			TTLSecondsAfterFinished: pointer.Int32(3600), // Clean up after 1 hour
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						JobInstanceLabel:  instance.Spec.ProjectName,
						JobOperationLabel: OperationCleanup,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "cleanup",
							Image:   ProvisionerImage,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{`
set -euo pipefail

echo "========================================"
echo "SupaControl Cleanup Job"
echo "Instance: $INSTANCE_NAME"
echo "Namespace: $NAMESPACE"
echo "========================================"

# Step 1: Uninstall Helm release (if it exists)
echo "[1/3] Uninstalling Helm release: $RELEASE_NAME"
if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
  helm uninstall "$RELEASE_NAME" --namespace "$NAMESPACE" --wait --timeout 5m || true
  echo "[1/3] Helm release uninstalled"
else
  echo "[1/3] Helm release not found, skipping"
fi

# Step 2: Delete namespace
echo "[2/3] Deleting namespace: $NAMESPACE"
kubectl delete namespace "$NAMESPACE" --wait=false --ignore-not-found=true

# Step 3: Report completion
echo "[3/3] Cleanup complete!"
echo "========================================"
echo "Instance '$INSTANCE_NAME' has been deleted"
echo "========================================"
`},
							Env: []corev1.EnvVar{
								{
									Name:  "INSTANCE_NAME",
									Value: instance.Spec.ProjectName,
								},
								{
									Name:  "NAMESPACE",
									Value: namespace,
								},
								{
									Name:  "RELEASE_NAME",
									Value: releaseName,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create cleanup Job: %w", err)
	}

	logger.Info("Created cleanup Job", "jobName", jobName, "namespace", ControllerNamespace)
	return job, nil
}

// getJobStatus retrieves the status of a Job
func (r *SupabaseInstanceReconciler) getJobStatus(ctx context.Context, jobName string) (*batchv1.Job, error) {
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: ControllerNamespace, Name: jobName}, job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// isJobSucceeded checks if a Job has completed successfully
func isJobSucceeded(job *batchv1.Job) bool {
	return job.Status.Succeeded > 0
}

// isJobFailed checks if a Job has failed permanently (exhausted retries)
func isJobFailed(job *batchv1.Job) bool {
	if job.Spec.BackoffLimit == nil {
		return false
	}
	return job.Status.Failed >= *job.Spec.BackoffLimit
}

// isJobActive checks if a Job is currently running
func isJobActive(job *batchv1.Job) bool {
	return job.Status.Active > 0
}

// getJobConditionMessage extracts a human-readable message from Job conditions
func getJobConditionMessage(job *batchv1.Job) string {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return fmt.Sprintf("Job failed: %s - %s", condition.Reason, condition.Message)
		}
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return "Job completed successfully"
		}
	}
	return ""
}
