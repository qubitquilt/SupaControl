# SupaControl Operator Architecture

## Overview

This document describes the architectural transformation of SupaControl from an **imperative orchestration** model to a **declarative Kubernetes Operator** pattern.

## Table of Contents

- [Why the Operator Pattern](#why-the-operator-pattern)
- [Architecture Before vs After](#architecture-before-vs-after)
- [Core Components](#core-components)
- [State Management](#state-management)
- [Reconciliation Loop](#reconciliation-loop)
- [Deployment Guide](#deployment-guide)
- [Development Guide](#development-guide)

## Why the Operator Pattern

### Problems with the Imperative Approach

The original architecture had several critical limitations:

1. **Coupling**: API handlers directly invoked Helm SDK operations via goroutines
2. **No Crash Recovery**: If the server crashed mid-provision, the operation was lost
3. **State Drift**: No reconciliation between Postgres DB and Kubernetes state
4. **Dual Sources of Truth**: Instance state stored in both Postgres and K8s
5. **No Idempotency**: Retrying failed operations could create duplicates
6. **Complex Error Handling**: Each failure path required manual state updates

Reference: `server/api/handlers.go:244` (old imperative approach)

### Benefits of the Operator Pattern

The Kubernetes Operator pattern solves these problems:

1. **Declarative API**: Create a `SupabaseInstance` CR → controller handles the rest
2. **Automatic Reconciliation**: Controller continuously ensures actual state matches desired state
3. **Crash Resilience**: Controller recovers and continues provisioning after restart
4. **Single Source of Truth**: Kubernetes API (etcd) is the authoritative state store
5. **Idempotency**: Creating the same CR multiple times is safe
6. **Kubernetes-Native**: Leverages K8s patterns (watches, finalizers, conditions)

## Architecture Before vs After

### Before: Imperative Orchestration

```
User Request → API Handler → Goroutine → Orchestrator → Helm SDK
                    ↓
                Postgres DB (state storage)
```

**Flow:**
1. API receives POST `/instances`
2. Handler creates DB record with `PROVISIONING` status
3. Handler spawns goroutine calling `orchestrator.CreateInstance()`
4. Orchestrator directly calls Helm SDK
5. On completion, updates DB record to `RUNNING` or `FAILED`

**Issues:**
- No recovery if server crashes
- State drift between DB and K8s
- Fire-and-forget goroutines

### After: Declarative Operator

```
User Request → API Handler → Creates SupabaseInstance CR
                                        ↓
                            Controller watches CR
                                        ↓
                            Reconciliation Loop
                                        ↓
                            Updates CR Status
```

**Flow:**
1. API receives POST `/instances`
2. Handler creates `SupabaseInstance` CR
3. Controller detects new CR via watch
4. Controller reconciles state (namespace → secrets → helm → ingress)
5. Controller updates CR status subresource
6. API reads status from CR on GET requests

**Benefits:**
- Controller automatically reconciles after crash
- K8s API is single source of truth
- Declarative desired state

## Core Components

### 1. Custom Resource Definition (CRD)

**File:** `server/api/v1alpha1/supabaseinstance_types.go`

Defines the `SupabaseInstance` resource:

```go
type SupabaseInstanceSpec struct {
    ProjectName   string // Required: unique identifier
    IngressClass  string // Optional: override default
    IngressDomain string // Optional: override default
    ChartVersion  string // Optional: specific Helm chart version
    Paused        bool   // Optional: pause reconciliation
}

type SupabaseInstanceStatus struct {
    Phase              SupabaseInstancePhase // Pending|Provisioning|Running|Deleting|Failed
    Conditions         []metav1.Condition    // Detailed status tracking
    Namespace          string                // K8s namespace
    StudioURL          string                // Generated URL
    APIURL             string                // Generated URL
    ErrorMessage       string                // Failure details
    ObservedGeneration int64                 // Spec version tracking
    LastTransitionTime *metav1.Time          // Last phase change
    HelmReleaseName    string                // Helm release identifier
}
```

**Key Design Decisions:**

- **Cluster-scoped**: `SupabaseInstance` resources are cluster-scoped (not namespaced)
- **Status Subresource**: Enables optimistic concurrency for status updates
- **Conditions**: Standard K8s pattern for detailed status reporting
- **Phases**: Simple high-level state machine

### 2. Controller

**File:** `server/controllers/supabaseinstance_controller.go`

The controller implements the reconciliation loop:

```go
func (r *SupabaseInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```

**Reconciliation Flow:**

1. **Fetch CR**: Get the `SupabaseInstance` resource
2. **Check Deletion**: Handle finalizer cleanup if being deleted
3. **State Machine**: Route to phase-specific handler:
   - `Pending` → Start provisioning
   - `Provisioning` → Create resources
   - `Running` → Health checks
   - `Failed` → Retry or wait
4. **Update Status**: Write observed state back to CR

**Finalizer Pattern:**

The controller adds a finalizer (`supacontrol.qubitquilt.com/finalizer`) to ensure cleanup:

```go
if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
    // Clean up resources
    r.cleanup(ctx, instance)
    // Remove finalizer
    controllerutil.RemoveFinalizer(instance, FinalizerName)
}
```

### 3. API Handlers (Refactored)

**File:** `server/api/handlers.go`

The API layer has been simplified dramatically:

**Before:**
```go
func (h *Handler) CreateInstance(c echo.Context) error {
    // Create DB record
    h.dbClient.StoreInstance(instance)
    // Spawn goroutine
    go h.provisionInstance(projectName)
    // Return 202
}
```

**After:**
```go
func (h *Handler) CreateInstance(c echo.Context) error {
    // Create SupabaseInstance CR
    instance := &supacontrolv1alpha1.SupabaseInstance{
        ObjectMeta: metav1.ObjectMeta{Name: req.Name},
        Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
            ProjectName: req.Name,
        },
    }
    h.crClient.CreateSupabaseInstance(ctx, instance)
    // Return 202
}
```

**Key Changes:**

- **No goroutines**: Controller handles async operations
- **No DB writes**: K8s API is the store
- **Immediate return**: 202 Accepted, controller does the work
- **Status from CR**: `ListInstances` queries K8s API, not Postgres

### 4. CR Client

**File:** `server/internal/k8s/crclient.go`

Wrapper around controller-runtime client for API handlers:

```go
type CRClient struct {
    client.Client
    scheme *runtime.Scheme
}

func (c *CRClient) CreateSupabaseInstance(ctx context.Context, instance *SupabaseInstance) error
func (c *CRClient) GetSupabaseInstance(ctx context.Context, name string) (*SupabaseInstance, error)
func (c *CRClient) ListSupabaseInstances(ctx context.Context) (*SupabaseInstanceList, error)
func (c *CRClient) DeleteSupabaseInstance(ctx context.Context, name string) error
```

## State Management

### Single Source of Truth

**Before:** Dual state stores (Postgres + K8s)

```
Postgres DB:
- instances table
- status: PROVISIONING|RUNNING|FAILED

Kubernetes:
- Namespaces
- Secrets
- Helm releases
- Ingresses
```

**Risk:** State could diverge (e.g., DB says `RUNNING` but K8s resources deleted)

**After:** Kubernetes API (etcd) is the sole source of truth

```
SupabaseInstance CR:
- metadata.name: unique identifier
- spec: desired state
- status: observed state

All other resources:
- Owned by SupabaseInstance via controller
- Deleted when CR is deleted
```

**Benefits:**
- **Consistency**: Status always reflects K8s reality
- **Recovery**: Controller re-syncs state on restart
- **Drift Detection**: Controller can detect manual changes

### Condition Types

The controller sets detailed conditions for status tracking:

```go
ConditionTypeReady             // Overall ready status
ConditionTypeNamespaceReady    // Namespace created
ConditionTypeSecretsReady      // Secrets generated
ConditionTypeHelmReleaseReady  // Helm chart installed
ConditionTypeIngressReady      // Ingress configured
```

Example:
```yaml
status:
  phase: Running
  conditions:
    - type: Ready
      status: "True"
      reason: ProvisioningComplete
      message: Instance is running and ready
    - type: NamespaceReady
      status: "True"
      reason: NamespaceCreated
    - type: HelmReleaseReady
      status: "True"
      reason: HelmReleaseInstalled
```

## Reconciliation Loop

### Phase State Machine

```
┌─────────┐
│ Pending │ Initial state when CR is created
└────┬────┘
     │
     v
┌──────────────┐
│ Provisioning │ Creating namespace, secrets, helm, ingress
└──────┬───────┘
       │
       v  (success)
   ┌─────────┐
   │ Running │ Healthy, periodic checks
   └────┬────┘
        │
        │ (delete request)
        v
   ┌──────────┐
   │ Deleting │ Cleanup via finalizer
   └──────────┘

   (failure)
        │
        v
   ┌────────┐
   │ Failed │ Error state, retry or manual intervention
   └────────┘
```

### Provisioning Steps

When transitioning from `Pending` → `Provisioning` → `Running`, the controller:

1. **Ensures Namespace**
   - Creates namespace `supa-{projectName}`
   - Labels: `supacontrol.io/instance={projectName}`
   - Sets condition: `NamespaceReady`

2. **Ensures Secrets**
   - Generates secure passwords/keys (postgres, JWT, anon, service-role)
   - Creates K8s secret: `{projectName}-secrets`
   - Sets condition: `SecretsReady`
   - **Note**: Secrets are auto-generated, not stored elsewhere

3. **Ensures Helm Release**
   - Installs Supabase Helm chart
   - Uses generated secrets in values
   - Release name: `{projectName}`
   - Sets condition: `HelmReleaseReady`

4. **Ensures Ingresses**
   - Creates Studio ingress: `{projectName}-studio.{domain}`
   - Creates API ingress: `{projectName}-api.{domain}`
   - Sets condition: `IngressReady`

5. **Updates Status**
   - Phase: `Running`
   - URLs populated
   - Overall condition: `Ready=True`

### Deletion Flow

When a `SupabaseInstance` CR is deleted:

1. **Finalizer Blocks Deletion**
   - K8s marks `deletionTimestamp` but doesn't remove CR
   - Controller detects deletion

2. **Cleanup**
   - Phase: `Deleting`
   - Uninstalls Helm release
   - Deletes namespace (cascade deletes all resources)

3. **Remove Finalizer**
   - Controller removes finalizer
   - K8s deletes the CR

**Key Benefit:** No orphaned resources

### Idempotency

Every reconciliation step is idempotent:

```go
func (r *SupabaseInstanceReconciler) ensureNamespace(ctx, instance) error {
    ns := &corev1.Namespace{Name: instance.Status.Namespace}
    if err := r.Create(ctx, ns); err != nil {
        if apierrors.IsAlreadyExists(err) {
            return nil // Idempotent: already exists
        }
        return err
    }
    return nil
}
```

**Result:** Controller can safely re-run reconciliation multiple times

## Deployment Guide

### Prerequisites

1. Kubernetes cluster with kubectl access
2. Helm 3.x installed
3. Cert-manager for TLS (optional but recommended)
4. Ingress controller (nginx, traefik, etc.)

### Installation Steps

#### 1. Install CRD

```bash
kubectl apply -f deploy/crds/supacontrol.qubitquilt.com_supabaseinstances.yaml
```

Verify:
```bash
kubectl get crd supabaseinstances.supacontrol.qubitquilt.com
```

#### 2. Create Namespace

```bash
kubectl create namespace supacontrol-system
```

#### 3. Apply RBAC

```bash
kubectl apply -f deploy/rbac/rbac.yaml
```

This creates:
- ServiceAccount: `supacontrol-controller`
- ClusterRole: `supacontrol-controller-role`
- ClusterRoleBinding: `supacontrol-controller-rolebinding`

#### 4. Deploy the Server

The server now includes both the API and the controller manager.

Update your Helm chart or Kubernetes manifests to include:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: supacontrol-server
  namespace: supacontrol-system
spec:
  replicas: 1
  template:
    spec:
      serviceAccountName: supacontrol-controller
      containers:
        - name: server
          image: your-registry/supacontrol-server:latest
          env:
            - name: DATABASE_URL
              value: "postgres://..."
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: supacontrol-secrets
                  key: jwt-secret
            - name: SUPABASE_CHART_REPO
              value: "supabase-community"
            - name: SUPABASE_CHART_NAME
              value: "supabase"
            - name: DEFAULT_INGRESS_CLASS
              value: "nginx"
            - name: DEFAULT_INGRESS_DOMAIN
              value: "example.com"
            - name: CERT_MANAGER_ISSUER
              value: "letsencrypt-prod"  # or letsencrypt-staging for testing
            - name: LEADER_ELECTION_ENABLED
              value: "false"  # Set to "true" for multi-replica HA deployments
```

**Important:** The server binary now runs both:
- HTTP API server (port 8080)
- Controller manager (background)

**High Availability (HA) Deployment:**

For production deployments, run multiple replicas with leader election enabled:

```yaml
spec:
  replicas: 3  # Multiple replicas for HA
  template:
    spec:
      serviceAccountName: supacontrol-controller
      containers:
        - name: server
          image: your-registry/supacontrol-server:latest
          env:
            # ... other env vars ...
            - name: LEADER_ELECTION_ENABLED
              value: "true"  # REQUIRED for multi-replica deployments
```

**Why Leader Election is Critical for HA:**
- Prevents multiple controllers from reconciling the same `SupabaseInstance` simultaneously
- Avoids race conditions and resource conflicts
- Only the elected leader actively reconciles; other replicas remain on standby
- If the leader pod fails, a new leader is automatically elected

**Environment Variables:**
- `LEADER_ELECTION_ENABLED`: Set to `"true"` for multi-replica deployments (default: `false`)

#### 5. Verify Controller

Check logs:
```bash
kubectl logs -n supacontrol-system deployment/supacontrol-server
```

Expected output:
```
Starting SupaControl server...
Connected to database
Initialized CR client
Initialized controller manager
Starting controller manager...
Waiting for controller cache to sync...
Controller cache synced
Server listening on :8080
```

### Testing the Operator

#### Create an Instance

```bash
kubectl apply -f - <<EOF
apiVersion: supacontrol.qubitquilt.com/v1alpha1
kind: SupabaseInstance
metadata:
  name: my-test-instance
spec:
  projectName: my-test-instance
EOF
```

#### Watch Reconciliation

```bash
kubectl get supabaseinstance my-test-instance -w
```

Output:
```
NAME               PROJECT            PHASE          NAMESPACE               AGE
my-test-instance   my-test-instance   Pending                                0s
my-test-instance   my-test-instance   Provisioning   supa-my-test-instance   2s
my-test-instance   my-test-instance   Running        supa-my-test-instance   45s
```

#### Check Status

```bash
kubectl get supabaseinstance my-test-instance -o yaml
```

#### Delete Instance

```bash
kubectl delete supabaseinstance my-test-instance
```

The controller will:
1. Update phase to `Deleting`
2. Uninstall Helm release
3. Delete namespace
4. Remove finalizer
5. CR is deleted

## Development Guide

### Project Structure

```
server/
├── api/
│   ├── v1alpha1/                    # CRD API types
│   │   ├── groupversion_info.go     # API group metadata
│   │   ├── supabaseinstance_types.go # CR definition
│   │   └── zz_generated.deepcopy.go  # Generated deepcopy methods
│   ├── handlers.go                   # HTTP API handlers (refactored)
│   └── router.go
├── controllers/
│   └── supabaseinstance_controller.go # Reconciliation logic
├── internal/
│   ├── k8s/
│   │   ├── k8s.go                   # Basic K8s client
│   │   ├── crclient.go              # CR client wrapper
│   │   ├── orchestrator.go          # (Legacy, kept for secrets generation)
│   │   └── utils.go
│   ├── db/                          # (Optional: can be removed)
│   └── ...
├── main.go                          # Entry point (API + controller)
└── ...

deploy/
├── crds/
│   └── supacontrol.qubitquilt.com_supabaseinstances.yaml
└── rbac/
    └── rbac.yaml
```

### Adding New Reconciliation Steps

To add a new provisioning step:

1. **Add to Status**
   ```go
   // In supabaseinstance_types.go
   const ConditionTypeMyNewStep = "MyNewStepReady"
   ```

2. **Implement Ensure Function**
   ```go
   // In supabaseinstance_controller.go
   func (r *SupabaseInstanceReconciler) ensureMyNewStep(ctx, instance) error {
       // Idempotent resource creation
       // Set condition on success
       meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
           Type:   ConditionTypeMyNewStep,
           Status: metav1.ConditionTrue,
           Reason: "MyNewStepCreated",
       })
       return nil
   }
   ```

3. **Call from Provisioning Phase**
   ```go
   func (r *SupabaseInstanceReconciler) reconcileProvisioning(ctx, instance) (ctrl.Result, error) {
       // ... existing steps ...
       if err := r.ensureMyNewStep(ctx, instance); err != nil {
           return r.transitionToFailed(ctx, instance, err.Error())
       }
       // ...
   }
   ```

### Local Development

#### Running the Operator Locally

```bash
export KUBECONFIG=~/.kube/config
export DATABASE_URL="postgres://localhost/supacontrol"
export JWT_SECRET="your-secret"
export SUPABASE_CHART_REPO="supabase-community"
export SUPABASE_CHART_NAME="supabase"
export DEFAULT_INGRESS_CLASS="nginx"
export DEFAULT_INGRESS_DOMAIN="local.dev"

cd server
go run main.go
```

The operator will:
- Connect to your local Kubernetes cluster
- Start the controller manager
- Start the HTTP API server

#### Testing Reconciliation

Create a test CR:
```bash
kubectl apply -f - <<EOF
apiVersion: supacontrol.qubitquilt.com/v1alpha1
kind: SupabaseInstance
metadata:
  name: local-test
spec:
  projectName: local-test
EOF
```

Watch logs for reconciliation:
```bash
# In your terminal running the operator
# You'll see:
# Reconciling SupabaseInstance projectName=local-test phase=Pending
# Created namespace supa-local-test
# Created secrets
# Installed Helm release
# ...
```

### Troubleshooting

#### Controller Not Reconciling

Check RBAC:
```bash
kubectl auth can-i create supabaseinstances.supacontrol.qubitquilt.com --as=system:serviceaccount:supacontrol-system:supacontrol-controller
```

Check controller logs:
```bash
kubectl logs -n supacontrol-system deployment/supacontrol-server --tail=100
```

#### Instance Stuck in Provisioning

Check conditions:
```bash
kubectl get supabaseinstance <name> -o jsonpath='{.status.conditions}' | jq
```

Describe the instance:
```bash
kubectl describe supabaseinstance <name>
```

Check namespace resources:
```bash
kubectl get all -n supa-<projectName>
```

#### Orphaned Resources

If you delete the CRD without deleting instances, resources may be orphaned.

Find orphaned namespaces:
```bash
kubectl get ns -l supacontrol.io/instance
```

Clean up manually:
```bash
kubectl delete ns <namespace>
```

## Migration from Old Architecture

### Step 1: Deploy New CRD and RBAC

```bash
kubectl apply -f deploy/crds/
kubectl apply -f deploy/rbac/
```

### Step 2: Deploy New Server Version

Update your deployment to use the new server image with operator support.

**Important:** The new server is backward compatible:
- API still accepts REST requests
- Creates CRs instead of calling orchestrator
- Can coexist with old instances in Postgres

### Step 3: Migrate Existing Instances (Optional)

For each existing instance in Postgres, create a corresponding CR:

```bash
# Example script
kubectl apply -f - <<EOF
apiVersion: supacontrol.qubitquilt.com/v1alpha1
kind: SupabaseInstance
metadata:
  name: existing-instance-1
spec:
  projectName: existing-instance-1
EOF
```

The controller will detect existing resources and adopt them.

### Step 4: Phase Out Postgres State (Optional)

Once all instances are migrated to CRs:
1. Update API to remove Postgres queries
2. Keep DB for users/auth only
3. Remove instance-related DB tables

## Future Enhancements

### 1. Drift Detection

Currently, the controller creates resources once. Future enhancement:

```go
func (r *SupabaseInstanceReconciler) reconcileRunning(ctx, instance) (ctrl.Result, error) {
    // Check if namespace still exists
    // Check if Helm release is healthy
    // Check if ingresses are configured correctly
    // Remediate drift
}
```

### 2. Health Checks

Add probes to check Supabase instance health:

```go
// Check if PostgreSQL is accessible
// Check if Kong gateway is running
// Update Ready condition accordingly
```

### 3. Backup/Restore

Add CRDs for backup operations:

```yaml
apiVersion: supacontrol.qubitquilt.com/v1alpha1
kind: SupabaseBackup
metadata:
  name: my-instance-backup-1
spec:
  instanceRef: my-instance
  retentionDays: 30
```

### 4. Upgrades

Support in-place upgrades:

```yaml
spec:
  chartVersion: "1.0.0"  # Update this
```

Controller detects change and performs Helm upgrade.

### 5. Secret Management Strategy

Current: Secrets are auto-generated and stored only in K8s.

Future options:
- **External Secrets Operator** integration
- **HashiCorp Vault** integration
- **Sealed Secrets** for GitOps
- **Secret rotation** automation

## Conclusion

The operator pattern transforms SupaControl from an imperative orchestration tool into a **Kubernetes-native declarative platform**. This aligns with the strategic vision of building a sustainable, scalable control plane for multi-tenant Supabase infrastructure.

**Key Wins:**
- ✅ Declarative API via CRDs
- ✅ Automatic reconciliation
- ✅ Crash resilience
- ✅ Single source of truth (K8s API)
- ✅ Kubernetes-native patterns
- ✅ Simplified API handlers
- ✅ State drift detection (foundation)

This architecture scales from a "project" to an "organizational platform" as described in the Principal Engineer feedback.
