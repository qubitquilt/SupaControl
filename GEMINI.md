# GEMINI.md - AI Assistant Guide for SupaControl

This document provides guidance for AI assistants (like Gemini) working with the SupaControl codebase. It includes architecture overview, development patterns, and common workflows.

## Project Overview

**SupaControl** is a self-hosted management platform for orchestrating multi-tenant Supabase instances on Kubernetes. It provides:
- A REST API for programmatic control.
- A React-based web dashboard for visual management.
- A Kubernetes Operator (Controller) for robust, native orchestration.
- An interactive CLI for easy installation.

### Key Technologies
- **Backend**: Go (Echo framework), PostgreSQL (for operational data).
- **Frontend**: React + Vite.
- **Orchestration**: Kubernetes, client-go, controller-runtime, Helm.
- **Infrastructure**: Docker, Helm charts.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      SupaControl                         │
├─────────────────────────────────────────────────────────┤
│  Web UI (React) ──► REST API (Echo/Go)                  │
│   (Creates/deletes SupabaseInstance CRDs)                │
│                          │                               │
│                    Controller (K8s Operator)            │
│                   (Reconciles CRDs via Jobs)            │
│                          │                               │
│                    PostgreSQL DB                         │
│                (Users, API Keys ONLY)                   │
└─────────────────────────────────────────────────────────┘
                          │
                  Kubernetes API
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   Instance1         Instance2         Instance3
 (Namespace)       (Namespace)       (Namespace)

Note: Instance state is stored in SupabaseInstance CRDs, not PostgreSQL (per ADR-001).
The controller uses a Job-based pattern for provisioning (per ADR-002).
```

## Code Organization

### Directory Structure

```
/SupaControl
├── /server/              # Go backend
│   ├── /api             # REST API routes and handlers
│   ├── /controllers     # Kubernetes controller/operator logic
│   ├── /internal        # Core business logic
│   │   ├── /auth        # JWT & password hashing
│   │   ├── /config      # Environment configuration
│   │   ├── /db          # Database layer (for users/API keys)
│   │   └── /k8s         # K8s client wrappers
│   ├── main.go          # Application entry point
│   └── go.mod           # Go dependencies
│
├── /ui/                 # React frontend
│   ├── /src
│   │   ├── /pages       # Route components (Dashboard, Login)
│   │   ├── api.js       # API client
│   │   └── main.jsx     # Entry point
│
├── /cli/                # Interactive installer
│
├── /charts/             # Helm chart for SupaControl deployment
│
├── /docs/
│   ├── /adr             # Architecture Decision Records (IMPORTANT)
│   │   ├── 001-crd-as-single-source-of-truth.md
│   │   └── 002-job-based-provisioning-pattern.md
│   └── ARCHITECTURE.md  # Detailed architecture
│
├── README.md            # User documentation
├── CONTRIBUTING.md      # Contribution guide
└── GEMINI.md            # This file
```

## Key Components

### 1. API Handlers (`server/api/handlers.go`)

**Purpose**: Handles HTTP requests. For instance management, it acts as a thin wrapper around the Kubernetes API to manage `SupabaseInstance` CRDs.

**Key Functions**:
- `CreateInstance()`: Creates a new `SupabaseInstance` CRD in Kubernetes.
- `ListInstances()`: Lists all `SupabaseInstance` CRDs from the cluster.
- `GetInstance()`: Gets a specific `SupabaseInstance` CRD.
- `DeleteInstance()`: Deletes a `SupabaseInstance` CRD.

**Important**: The API handlers **do not** interact with the database for instance management. They only manage CRDs.

### 2. Kubernetes Controller (`server/controllers/supabaseinstance_controller.go`)

**Purpose**: The core of the operator. It watches for changes to `SupabaseInstance` CRDs and takes action to make the cluster state match the desired state.

**Architecture (per ADR-002)**:
- The controller does not perform long-running tasks directly.
- For provisioning, it creates a Kubernetes `Job` (`supacontrol-provision-{name}`).
- For cleanup, it creates a Kubernetes `Job` (`supacontrol-cleanup-{name}`).
- It monitors the status of these Jobs and updates the `SupabaseInstance` CRD's `.status` field accordingly.

**Reconciliation Phases**:
- `Pending` → `Provisioning` (Job created) → `Running` (Job succeeded)
- Any phase → `Failed` (Job failed)
- `Deleting` (Cleanup Job created)

### 3. Database Layer (`server/internal/db/`)

**Purpose**: Manages persistence for SupaControl's **operational data only**.

**IMPORTANT**: Per **ADR-001**, this layer is **NOT USED** for instance state. The `instances` table and its repository have been removed.

**Files**:
- `db.go`: Connection, migrations, health checks.
- `api_keys.go`: CRUD for API keys.

**Schema**:
```sql
-- users table
id, username, password_hash, role, created_at, updated_at

-- api_keys table
id, name, key_hash, user_id, created_at, expires_at, last_used
```

### 4. React Dashboard (`ui/src/pages/Dashboard.jsx`)

**Purpose**: Web UI for instance management.

**Functionality**:
- Lists instances by querying the SupaControl API (which in turn lists CRDs).
- Creates new instances by sending a request to the API (which creates a CRD).
- Deletes instances (which deletes the CRD).

## Development Workflows

### Adding a New API Endpoint

1. **Define types** in `pkg/api-types/` if needed.
2. **Add handler function** in `server/api/handlers.go`.
3. **Register route** in `server/api/router.go`.
4. **Write tests** for the handler.
5. **Update frontend** API client in `ui/src/api.js`.

**Example**: For an action on an instance, the handler should typically modify the `SupabaseInstance` CRD, and the controller will handle the rest.

```go
// server/api/handlers.go
func (h *Handler) PauseInstance(c echo.Context) error {
    name := c.Param("name")
    
    // Get the CRD
    instance, err := h.crClient.GetSupabaseInstance(c.Request().Context(), name)
    if err != nil {
        // handle error
    }

    // Modify the spec
    instance.Spec.Paused = true

    // Update the CRD
    if err := h.crClient.UpdateSupabaseInstance(c.Request().Context(), instance); err != nil {
        // handle error
    }

    return c.JSON(http.StatusOK, "Instance pause initiated")
}
```

### Modifying the Controller

1. **Locate the reconciler**: `server/controllers/supabaseinstance_controller.go`.
2. **Identify the state**: Find the `if instance.Status.Phase == ...` block you need to change.
3. **Modify the logic**: For example, to add a new resource, add a `createMyResource()` function and call it from the provisioning Job script.
4. **Update the Job script**: The provisioning and cleanup logic is in shell scripts defined in `server/controllers/job_helpers.go`.
5. **Write tests**: Add or update tests in `server/controllers/supabaseinstance_controller_test.go`.

### Testing Changes

**Backend**:
```bash
# Run all backend tests, including controller tests
# Make sure KUBEBUILDER_ASSETS is set for controller tests
make test-backend

# Run only controller tests
make test-controller
```

**Frontend**:
```bash
cd ui && npm test
```

## Important Patterns and Conventions

### 1. CRDs are the Single Source of Truth (ADR-001)

- **DO NOT** store instance information (like name, status, namespace) in the PostgreSQL database.
- All instance state must be read from and written to the `SupabaseInstance` CRD.
- The API serves as a proxy to the Kubernetes API for CRD management.

### 2. Job-Based Provisioning (ADR-002)

- The controller **delegates** long-running tasks to Kubernetes `Jobs`.
- **DO NOT** put blocking calls (like `helm install`) inside the main reconciliation loop.
- The controller's role is to create a Job and monitor its status.
- The actual logic is in the shell scripts executed by the Job.

### 3. Instance Lifecycle

1. **Create**: `POST /api/v1/instances` → Creates `SupabaseInstance` CRD.
2. **Reconcile**: Controller sees CRD, creates a provisioning `Job`.
3. **Provision**: `Job` runs, installs Helm chart, creates resources.
4. **Monitor**: Controller watches `Job` status and updates CRD `.status` field.
5. **Delete**: `DELETE /api/v1/instances/:name` → Deletes `SupabaseInstance` CRD.
6. **Cleanup**: Controller's finalizer creates a cleanup `Job` before the CRD is removed.

## Common Tasks

### Task: Add a New Field to an Instance

**Example**: Add a `size` field (`small`, `medium`, `large`).

1. **Update CRD Spec**:
   ```go
   // server/api/v1alpha1/supabaseinstance_types.go
   type SupabaseInstanceSpec struct {
       ProjectName string `json:"projectName"`
       Size        string `json:"size,omitempty"` // NEW
   }
   ```

2. **Update Provisioning Job**: Modify the script in `server/controllers/job_helpers.go` to use the new field.
   ```go
   // server/controllers/job_helpers.go
   job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
       Name:  "INSTANCE_SIZE",
       Value: instance.Spec.Size, // Pass size to the job
   })
   ```
   And in the script itself:
   ```bash
   # In provision.sh script
   echo "Instance size: $INSTANCE_SIZE"
   # Use this value to select different Helm values, e.g., --set resources.requests.cpu=...
   ```

3. **Update API Handler**: The handler in `server/api/handlers.go` needs to accept the new field and put it in the CRD spec.
   ```go
   // In CreateInstance handler
   instance.Spec.Size = req.Size // Assuming 'req' has the new field
   ```

4. **Update Frontend**: Add a dropdown or input in `ui/src/pages/Dashboard.jsx` to select the size.

### Task: Fix Test Coverage

**Priority Areas**:
1. **Controller Logic**: `server/controllers/supabaseinstance_controller_test.go` is the highest priority. Use `envtest` to write integration tests for the reconciliation loop.
2. **API Handlers**: `server/api/handlers_*.go` need tests to verify they correctly interact with the `crClient`.
3. **Database Operations**: `server/internal/db/` needs tests for user and API key repositories.
4. **React Components**: `ui/src/pages/` components need tests for user interactions.

**Example: Add a controller test**:
```go
// server/controllers/supabaseinstance_controller_test.go
var _ = Describe("SupabaseInstance Controller", func() {
    It("Should create a provisioning Job when a new instance is created", func() {
        // 1. Create a SupabaseInstance resource
        // 2. Trigger reconciliation
        // 3. Expect a Job with the correct name and spec to be created
    })
})
```

## Gotchas and Important Notes

- **Controller vs. API**: Remember the separation of duties. The **API** modifies the *desired state* (the CRD). The **Controller** acts on the desired state to change the *actual state* (the cluster).
- **Idempotency**: The controller's reconciliation loop must be idempotent. It will run many times. Logic should be "if X doesn't exist, create it" not just "create X".
- **Finalizers**: The controller uses a finalizer on the `SupabaseInstance` CRD to ensure cleanup (the cleanup Job) runs before Kubernetes deletes the CRD. If cleanup fails, the CRD will be stuck in `Terminating` state until the issue is resolved.

## Resources

- **ADR-001**: `docs/adr/001-crd-as-single-source-of-truth.md`
- **ADR-02**: `docs/adr/002-job-based-provisioning-pattern.md`
- **Architecture**: `ARCHITECTURE.md`
- **Controller Tests**: `server/controllers/README_TEST.md`

---

**Last Updated**: November 25, 2025
**Maintained By**: SupaControl Contributors
**Questions?**: Open an issue on GitHub