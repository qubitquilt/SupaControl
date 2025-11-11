# ADR 002: Job-Based Provisioning and Cleanup Pattern

**Status**: Accepted

**Date**: 2025-11-11

**Decision Makers**: Engineering Team

**Technical Story**: Improving controller reliability, observability, and failure handling by delegating long-running operations to Kubernetes Jobs.

---

## Context and Problem Statement

The current SupabaseInstance controller directly executes long-running operations (Helm chart installation/uninstallation) within the reconciliation loop. This creates several operational challenges:

### Current Implementation Problems

1. **Blocking Reconcile Loop**: Helm operations can take 5-10 minutes, blocking the controller thread
   - During this time, the controller cannot process other instances
   - Controller manager worker pool gets exhausted under load
   - No parallelism for provisioning multiple instances

2. **Poor Observability**: Helm operations are opaque
   - No visibility into what step is executing
   - Logs are buried in controller output
   - Hard to debug failures ("Helm install failed" - but why?)

3. **Insufficient Error Handling**:
   - Transient failures (network issues, temporary API unavailability) cause immediate failure
   - No built-in retry/backoff mechanism (only controller requeue)
   - Difficult to distinguish between retryable and permanent failures

4. **Resource Constraints**:
   - Helm operations run in the controller pod
   - No CPU/memory limits for provisioning operations
   - Large Helm charts can OOM the controller

5. **Cleanup Reliability**:
   - Finalizer cleanup blocks deletion
   - If Helm uninstall hangs, deletion is stuck indefinitely
   - No timeout mechanism

### Example Failure Scenario

```
User creates SupabaseInstance → Controller reconciles → ensureHelmRelease() called
  → Helm chart download starts (2 min) → Network timeout
  → Error returned → Status set to Failed
  → Manual intervention required (no automatic retry)
```

## Decision Drivers

### Technical Drivers

1. **Kubernetes-Native Patterns**: Use Jobs for batch/one-time operations (recommended K8s pattern)
2. **Separation of Concerns**: Controller orchestrates, Jobs execute
3. **Observability**: Job status/events provide clear visibility
4. **Resource Management**: Jobs have resource limits, timeouts, and backoff policies
5. **Concurrency**: Multiple Jobs can run in parallel without blocking controller

### Operational Drivers

1. **Reliability**: Built-in retry with exponential backoff
2. **Debuggability**: Job logs persist after completion
3. **Monitoring**: Job metrics (duration, success/failure rate) via Prometheus
4. **Scalability**: Provision N instances without blocking controller
5. **Failure Recovery**: Jobs can be manually re-triggered or deleted

### Industry Precedent

Similar patterns used by:
- **Argo CD**: Uses Jobs for sync operations
- **Crossplane**: Uses provider pods (Job-like) for resource provisioning
- **Tekton**: Pipeline runs as Pods/Jobs
- **Velero**: Backup/restore operations as Jobs

## Decision Outcome

**Chosen Option**: **Use Kubernetes Jobs for Provisioning and Cleanup Operations**

### Architecture Overview

```
┌────────────────────────────────────────────────────────────┐
│                   SupabaseInstance CRD                     │
│         (Declarative State: Pending → Provisioning)       │
└─────────────────────┬──────────────────────────────────────┘
                      │
                      ▼
┌────────────────────────────────────────────────────────────┐
│              SupabaseInstance Controller                   │
│  • Watches CRDs and Jobs                                   │
│  • Creates Jobs for provisioning/cleanup                   │
│  • Monitors Job status → Updates CRD status                │
│  • Does NOT execute Helm operations directly               │
└─────────────────────┬──────────────────────────────────────┘
                      │
                      ▼
┌────────────────────────────────────────────────────────────┐
│                 Kubernetes Job                             │
│  Name: supacontrol-provision-{instance-name}-{hash}        │
│  Container: Uses Helm CLI image                            │
│  Command: /scripts/provision.sh                            │
│  Mounts: Secret with config, ServiceAccount with RBAC     │
│  Limits: CPU=500m, Memory=512Mi, Timeout=15min             │
└────────────────────┬───────────────────────────────────────┘
                     │
                     ▼
    ┌────────────────────────────────────┐
    │  Job Execution Steps:              │
    │  1. Create namespace               │
    │  2. Generate secrets               │
    │  3. helm repo add                  │
    │  4. helm install (with values)     │
    │  5. Create ingresses               │
    │  6. Report status (success/failure)│
    └────────────────────────────────────┘
```

### State Machine with Jobs

```
Pending → Provisioning (Job Created) → ProvisioningInProgress (Job Running)
   ↓              ↓                              ↓
   ↓          Job Failed ← Retry? → Job Succeeded
   ↓              ↓                              ↓
   └─────────→ Failed                      Running
```

### New Phase Definitions

- **Pending**: Initial state, no action taken yet
- **Provisioning**: Provisioning Job created, not yet started
- **ProvisioningInProgress**: Job Pod is running
- **Running**: Job completed successfully, instance is ready
- **Failed**: Job failed after retries exhausted
- **Deleting**: Cleanup Job created
- **DeletingInProgress**: Cleanup Job is running

## Implementation Details

### 1. Job Creation (Provisioning)

```go
func (r *SupabaseInstanceReconciler) createProvisioningJob(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("supacontrol-provision-%s", instance.Spec.ProjectName),
            Namespace: "supacontrol-system", // Jobs run in controller namespace
            Labels: map[string]string{
                "supacontrol.io/instance":   instance.Spec.ProjectName,
                "supacontrol.io/operation": "provision",
            },
            OwnerReferences: []metav1.OwnerReference{
                {
                    APIVersion: instance.APIVersion,
                    Kind:       instance.Kind,
                    Name:       instance.Name,
                    UID:        instance.UID,
                    Controller: pointer.Bool(true),
                },
            },
        },
        Spec: batchv1.JobSpec{
            BackoffLimit: pointer.Int32(3), // Retry up to 3 times
            ActiveDeadlineSeconds: pointer.Int64(900), // 15 minute timeout
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    ServiceAccountName: "supacontrol-provisioner",
                    RestartPolicy:      corev1.RestartPolicyNever,
                    Containers: []corev1.Container{
                        {
                            Name:  "provisioner",
                            Image: "alpine/helm:3.13.0", // Helm CLI image
                            Command: []string{"/scripts/provision.sh"},
                            Env: []corev1.EnvVar{
                                {Name: "INSTANCE_NAME", Value: instance.Spec.ProjectName},
                                {Name: "NAMESPACE", Value: fmt.Sprintf("supa-%s", instance.Spec.ProjectName)},
                                {Name: "CHART_REPO", Value: r.ChartRepo},
                                {Name: "CHART_NAME", Value: r.ChartName},
                                {Name: "CHART_VERSION", Value: instance.Spec.ChartVersion},
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

    return r.Create(ctx, job)
}
```

### 2. Job Monitoring

```go
func (r *SupabaseInstanceReconciler) checkProvisioningJobStatus(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
    jobName := fmt.Sprintf("supacontrol-provision-%s", instance.Spec.ProjectName)
    job := &batchv1.Job{}

    err := r.Get(ctx, client.ObjectKey{Namespace: "supacontrol-system", Name: jobName}, job)
    if err != nil {
        return ctrl.Result{}, err
    }

    // Check Job status
    if job.Status.Succeeded > 0 {
        // Success! Transition to Running
        return r.transitionToRunning(ctx, instance)
    }

    if job.Status.Failed > 0 {
        // Check if backoff limit reached
        if job.Status.Failed >= *job.Spec.BackoffLimit {
            // Permanent failure
            return r.transitionToFailed(ctx, instance, "Provisioning job failed after retries")
        }
        // Still retrying, requeue
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
    }

    // Job still running, requeue
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
```

### 3. Provisioning Script (provision.sh)

Mounted as ConfigMap in Job Pod:

```bash
#!/bin/sh
set -euo pipefail

echo "Starting provisioning for instance: $INSTANCE_NAME"

# Step 1: Create namespace
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Step 2: Generate and create secrets
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: $INSTANCE_NAME-secrets
  namespace: $NAMESPACE
stringData:
  postgres-password: "$(openssl rand -base64 32)"
  jwt-secret: "$(openssl rand -base64 64)"
  anon-key: "$(openssl rand -base64 32)"
  service-role-key: "$(openssl rand -base64 32)"
EOF

# Step 3: Add Helm repo
helm repo add supabase "$CHART_REPO"
helm repo update

# Step 4: Install Helm chart
helm install "$INSTANCE_NAME" supabase/"$CHART_NAME" \
  --namespace "$NAMESPACE" \
  --version "$CHART_VERSION" \
  --wait \
  --timeout 10m

echo "Provisioning complete for instance: $INSTANCE_NAME"
```

### 4. Cleanup Job (Deletion)

Similar pattern for deletion:

```go
func (r *SupabaseInstanceReconciler) createCleanupJob(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("supacontrol-cleanup-%s", instance.Spec.ProjectName),
            Namespace: "supacontrol-system",
            Labels: map[string]string{
                "supacontrol.io/instance":   instance.Spec.ProjectName,
                "supacontrol.io/operation": "cleanup",
            },
        },
        Spec: batchv1.JobSpec{
            BackoffLimit: pointer.Int32(2),
            ActiveDeadlineSeconds: pointer.Int64(600), // 10 minute timeout
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    ServiceAccountName: "supacontrol-provisioner",
                    RestartPolicy:      corev1.RestartPolicyNever,
                    Containers: []corev1.Container{
                        {
                            Name:  "cleanup",
                            Image: "alpine/helm:3.13.0",
                            Command: []string{"/scripts/cleanup.sh"},
                            Env: []corev1.EnvVar{
                                {Name: "INSTANCE_NAME", Value: instance.Spec.ProjectName},
                                {Name: "NAMESPACE", Value: instance.Status.Namespace},
                            },
                        },
                    },
                },
            },
        },
    }

    return r.Create(ctx, job)
}
```

### 5. RBAC for Provisioning Jobs

ServiceAccount with namespace-scoped permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: supacontrol-provisioner
  namespace: supacontrol-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: supacontrol-provisioner
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "delete", "get", "list"]
  - apiGroups: [""]
    resources: ["secrets", "configmaps"]
    verbs: ["create", "delete", "get", "list", "update", "patch"]
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["create", "delete", "get", "list", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: supacontrol-provisioner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: supacontrol-provisioner
subjects:
  - kind: ServiceAccount
    name: supacontrol-provisioner
    namespace: supacontrol-system
```

## Consequences

### Positive Consequences

1. **Non-Blocking Reconciliation**:
   - Controller can process other instances while Jobs run
   - Improved throughput for multi-instance provisioning

2. **Better Observability**:
   - `kubectl get jobs -n supacontrol-system` shows all operations
   - `kubectl logs job/supacontrol-provision-myapp` shows detailed logs
   - Job events show retry attempts and failures

3. **Built-in Retry Logic**:
   - Kubernetes handles backoff automatically
   - Configurable retry limits (`BackoffLimit`)
   - Exponential backoff between attempts

4. **Resource Isolation**:
   - Provisioning operations have CPU/memory limits
   - Can't OOM the controller pod
   - Failed Jobs don't crash the controller

5. **Timeout Guarantees**:
   - `ActiveDeadlineSeconds` prevents indefinite hangs
   - Deletion finalizers won't block forever

6. **Audit Trail**:
   - Job history shows what happened and when
   - Failed Jobs can be inspected post-mortem
   - TTL controller can auto-clean old Jobs

7. **Testability**:
   - Provisioning scripts can be tested independently
   - Mock Job status in controller tests
   - Integration tests can verify Job creation

### Negative Consequences

1. **Increased Complexity**:
   - More Kubernetes resources to manage (Jobs, ConfigMaps, ServiceAccounts)
   - State machine has more phases
   - *Mitigation*: Clear documentation and helper functions

2. **Delayed Feedback**:
   - Job creation is async, user doesn't see immediate result
   - *Mitigation*: Update status conditions with "Provisioning in progress"

3. **RBAC Overhead**:
   - Need ServiceAccount with provisioning permissions
   - *Mitigation*: Document required permissions, include in Helm chart

4. **Image Dependency**:
   - Requires Helm CLI image in cluster (alpine/helm)
   - *Mitigation*: Common image, can use custom image if needed

5. **Job Cleanup**:
   - Need to clean up completed Jobs
   - *Mitigation*: Set `ttlSecondsAfterFinished: 3600` (auto-delete after 1 hour)

### Neutral Consequences

- **Migration**: Existing instances won't automatically use Jobs (only new ones)
- **Monitoring**: Need to monitor Job success/failure rates
- **Cost**: Minimal (Jobs are short-lived, resource limits are low)

## Alternatives Considered

### Alternative 1: Keep Direct Helm Operations (Status Quo)

**Why Rejected**: All the problems listed above remain unsolved

### Alternative 2: Use Argo Workflows or Tekton Pipelines

**Pros**:
- Mature workflow orchestration
- Rich UI for monitoring
- DAG-based task execution

**Cons**:
- Requires additional operators (Argo/Tekton) as dependencies
- Over-engineered for simple install/uninstall operations
- Adds operational complexity

**Why Rejected**: Too heavy for this use case

### Alternative 3: Use StatefulSet with Init Containers

**Pros**:
- Could run provisioning in init container

**Cons**:
- StatefulSet is for long-running workloads, not one-time operations
- Misuse of Kubernetes primitives
- No built-in retry/backoff

**Why Rejected**: Jobs are the correct primitive for batch operations

### Alternative 4: External Queue (RabbitMQ, SQS) + Worker Pods

**Pros**:
- Fully async with queue guarantees
- Can scale workers independently

**Cons**:
- Requires external dependency (queue system)
- Much more complex than needed
- Reinventing what Jobs already provide

**Why Rejected**: Over-engineered, adds external dependency

## Validation and Testing

### Unit Tests

```go
func TestReconciler_CreateProvisioningJob(t *testing.T) {
    // Test that controller creates Job with correct spec
}

func TestReconciler_MonitorJobStatus_Success(t *testing.T) {
    // Mock Job with Succeeded=1, verify transition to Running
}

func TestReconciler_MonitorJobStatus_Failure(t *testing.T) {
    // Mock Job with Failed >= BackoffLimit, verify transition to Failed
}
```

### Integration Tests (envtest)

```go
func TestProvisioningFlow_EndToEnd(t *testing.T) {
    // 1. Create SupabaseInstance CR
    // 2. Verify Job is created
    // 3. Manually set Job.Status.Succeeded = 1
    // 4. Verify instance transitions to Running
}
```

### Manual Testing

```bash
# Create instance
kubectl apply -f - <<EOF
apiVersion: supacontrol.qubitquilt.com/v1alpha1
kind: SupabaseInstance
metadata:
  name: test-instance
spec:
  projectName: test-app
EOF

# Watch Job creation and completion
kubectl get jobs -n supacontrol-system -w

# Check Job logs
kubectl logs -n supacontrol-system job/supacontrol-provision-test-app

# Verify instance status
kubectl get supabaseinstance test-instance -o yaml | grep -A 10 status
```

## Rollout Plan

### Phase 1: Implementation (Workstream III)

1. Create ADR-002 (this document)
2. Implement Job creation functions
3. Update reconciliation state machine
4. Add Job monitoring logic
5. Create provisioning/cleanup scripts
6. Add RBAC resources

### Phase 2: Testing

1. Unit tests for Job creation/monitoring
2. Integration tests with envtest
3. Manual testing in dev cluster

### Phase 3: Documentation

1. Update ARCHITECTURE.md with Job-based flow
2. Update CLAUDE.md with implementation details
3. Add troubleshooting guide for Job failures

### Phase 4: Deployment

1. Deploy updated controller
2. Test with new instance creation
3. Monitor Job metrics in production
4. Gather operational feedback

## Monitoring and Observability

### Metrics to Track

```
supacontrol_provisioning_jobs_total{status="success|failure"}
supacontrol_provisioning_job_duration_seconds{quantile="0.5|0.9|0.99"}
supacontrol_cleanup_jobs_total{status="success|failure"}
supacontrol_active_provisioning_jobs
```

### Alerts

```
# Provisioning job failing repeatedly
ALERT ProvisioningJobHighFailureRate
  IF rate(supacontrol_provisioning_jobs_total{status="failure"}[5m]) > 0.5
  FOR 10m

# Provisioning job taking too long
ALERT ProvisioningJobSlow
  IF supacontrol_provisioning_job_duration_seconds{quantile="0.9"} > 600
  FOR 5m
```

## References

- [Kubernetes Jobs Documentation](https://kubernetes.io/docs/concepts/workloads/controllers/job/)
- [Operator Pattern Best Practices](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Helm Go SDK](https://helm.sh/docs/topics/advanced/)
- ADR-001: CRD as Single Source of Truth
- Technical Brief: "Evolving SupaControl to a Production-Grade Control Plane" (2025-11-11)

## Review and Approval

- **Proposed By**: Engineering Team
- **Reviewed By**: Technical Lead
- **Approved By**: Engineering Team
- **Date**: 2025-11-11

---

## Changelog

- **2025-11-11**: Initial ADR created, decision accepted
