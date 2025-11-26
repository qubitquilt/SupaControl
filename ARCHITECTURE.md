# Architecture Documentation

This document provides a comprehensive architectural overview of SupaControl, covering system design, component interactions, data flows, and technical decision rationale.

## Table of Contents

- [System Overview](#system-overview)
- [Architecture Principles](#architecture-principles)
- [Component Architecture](#component-architecture)
- [Data Architecture](#data-architecture)
- [API Design](#api-design)
- [Security Architecture](#security-architecture)
- [Deployment Architecture](#deployment-architecture)
- [Scalability & Performance](#scalability--performance)
- [Technology Stack Rationale](#technology-stack-rationale)
- [Design Decisions](#design-decisions)
- [Future Architecture](#future-architecture)

## System Overview

SupaControl is a **control plane** for managing multi-tenant Supabase instances on Kubernetes. It follows a layered architecture pattern with clear separation of concerns.

### System Context

```
┌──────────────────────────────────────────────────────────────┐
│                      External Systems                         │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐             │
│   │   Users  │    │   CLI    │    │  CI/CD   │             │
│   │ (Browser)│    │ (supactl)│    │ Pipelines│             │
│   └────┬─────┘    └────┬─────┘    └────┬─────┘             │
│        │               │               │                     │
│        └───────────────┼───────────────┘                     │
│                        │                                     │
└────────────────────────┼─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                      SupaControl                              │
│                    (This System)                              │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│   ┌──────────────────────────────────────────────┐          │
│   │         Application Layer                     │          │
│   │  - Web UI (React SPA)                        │          │
│   │  - REST API (Echo/Go)                        │          │
│   │  - Authentication & Authorization            │          │
│   └──────────────────────────────────────────────┘          │
│                         │                                     │
│   ┌──────────────────────────────────────────────┐          │
│   │         Business Logic Layer                  │          │
│   │  - Instance Lifecycle Management             │          │
│   │  - Validation & State Management             │          │
│   │  - API Key Management                        │          │
│   └──────────────────────────────────────────────┘          │
│                         │                                     │
│   ┌──────────────────────────────────────────────┐          │
│   │         Data/Integration Layer                │          │
│   │  - PostgreSQL Repository (Users, API Keys)   │          │
│   │  - Kubernetes CRD Client                     │          │
│   │  - Helm SDK Integration                      │          │
│   └──────────────────────────────────────────────┘          │
│                                                               │
└────────────────────────┬──────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                   External Dependencies                       │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│   ┌──────────────┐         ┌──────────────┐                 │
│   │  PostgreSQL  │         │  Kubernetes  │                 │
│   │   Database   │         │   Cluster    │                 │
│   │              │         │              │                 │
│   │ - Users      │         │ - CRDs       │                 │
│   │ - API Keys   │         │ - Namespaces │                 │
│   └──────────────┘         └──────────────┘                 │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

### High-Level Flow

1. **User Interaction**: Users interact via Web UI or CLI.
2. **API Gateway**: All requests go through the REST API.
3. **Authentication**: JWT/API Key validation.
4. **Business Logic**: Request processing and validation.
5. **Orchestration**: Creates/updates `SupabaseInstance` CRDs in Kubernetes.
6. **Persistence**:
   - **Instance State**: Stored in Kubernetes as CRDs (Single Source of Truth).
   - **Operational State**: Users and API keys stored in PostgreSQL.
7. **Response**: Asynchronous acceptance; status polled from CRD.

## Architecture Principles

SupaControl is built on these core architectural principles:

### 1. API-First Design

- All functionality exposed via REST API.
- Web UI and CLI are API consumers.
- Enables programmatic access and automation.

### 2. Separation of Concerns

- **API Layer**: HTTP handling, routing, middleware.
- **Business Logic**: Validation, state management, orchestration logic.
- **Data Layer**: PostgreSQL for operational data, Kubernetes client for instance state.

### 3. Stateless Application

- No session state stored in the application.
- Horizontally scalable for high availability.
- All persistent state managed externally (PostgreSQL and Kubernetes CRDs).

### 4. Declarative Infrastructure

- Kubernetes resources managed declaratively via CRDs.
- Helm charts for reproducible deployments.
- Aligns with Infrastructure as Code (IaC) and GitOps principles.

### 5. Security by Default

- Authentication required for all sensitive endpoints.
- Secrets encrypted at rest (in K8s and DB).
- Least privilege RBAC for all components.

### 6. Observability

- Structured logging for clear, machine-readable logs.
- Health check endpoints for liveness and readiness.
- Metrics exposure for Prometheus (future).

## Component Architecture

### Backend Components

```
server/
├── main.go                 # Application bootstrap
├── /api                    # API Layer
│   ├── handlers.go         # Request handlers
│   ├── router.go           # Route definitions
│   └── middleware.go       # Auth, CORS, logging
├── /internal               # Business Logic
│   ├── /auth
│   │   └── auth.go         # JWT & password service
│   ├── /config
│   │   └── config.go       # Configuration management
│   ├── /db
│   │   ├── db.go           # Database connection & migrations
│   │   └── api_keys.go     # API key repository
│   └── /k8s
│       ├── crclient.go     # Controller runtime CRD client
│       └── orchestrator.go # Legacy direct Helm operations
├── /controllers
│   └── supabaseinstance_controller.go # Operator logic
└── /pkg
    └── /api-types          # Shared API types
```

### Component Responsibilities

#### 1. API Layer (`server/api/`)

**Responsibility**: HTTP request/response handling.

**Functions**:
- Route registration and URL mapping.
- Request validation, binding, and response formatting (JSON).
- Middleware execution (auth, logging, CORS).

**Pattern**: Handler → Business Logic → Response.

```go
func (h *Handler) CreateInstance(c echo.Context) error {
    // 1. Parse and validate request
    // 2. Create a SupabaseInstance CRD via the crClient
    // 3. Return 202 Accepted
}
```

#### 2. Authentication Service (`server/internal/auth/`)

**Responsibility**: Security and identity management.

**Functions**:
- JWT token generation and validation.
- Password hashing (bcrypt) and verification.
- API key generation, hashing, and validation.

#### 3. Database Layer (`server/internal/db/`)

**Responsibility**: Persistence for SupaControl's **operational data only**.

**Technology**: PostgreSQL + sqlx.

**IMPORTANT**: Per **ADR-001**, instance state is **NOT** stored in PostgreSQL. The `instances` table and its corresponding repository have been removed. The database only manages users, API keys, and (in the future) audit logs.

**Schema Design**:
```sql
-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL
);

-- api_keys table
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    user_id INT REFERENCES users(id)
);
```

#### 4. Kubernetes Controller (`server/controllers/`)

**Responsibility**: Reconciliation of `SupabaseInstance` CRDs. This is the core of the operator pattern.

**Functions**:
- Watches for changes to `SupabaseInstance` resources.
- Creates, updates, or deletes Kubernetes resources (Namespaces, Jobs, Secrets) to match the desired state defined in the CRD.
- Updates the `.status` field of the CRD to reflect the actual state.

**Pattern**: Follows the Job-Based Provisioning pattern from **ADR-002**.

**Instance Creation Flow (per ADR-002)**:
1. API creates a `SupabaseInstance` CRD with `status.phase: Pending`.
2. Controller sees the new CRD.
3. Controller creates a Kubernetes `Job` to handle the provisioning.
4. The Job runs a script that:
   - Creates the namespace.
   - Generates secrets.
   - Installs the Supabase Helm chart.
5. Controller monitors the Job's status.
6. Upon Job success, the controller updates the CRD status to `Running`.
7. Upon Job failure, the controller updates the CRD status to `Failed`.

### Frontend Components

```
ui/
├── /src
│   ├── main.jsx            # Entry point
│   ├── App.jsx             # Main app component
│   ├── api.js              # API client functions
│   └── /pages
│       ├── Login.jsx       # Authentication page
│       ├── Dashboard.jsx   # Instance management
│       └── Settings.jsx    # API key management
```

**Technology**: React + Vite.
**Pattern**: Component → API Client → Backend.

## Data Architecture

### Data Flow Diagrams

#### Instance Creation Data Flow (per ADR-001 & ADR-002)

```
┌──────────┐
│  Client  │
└─────┬────┘
      │ POST /api/v1/instances {"name": "myapp"}
      ▼
┌──────────────────┐
│   API Handler    │
│  CreateInstance  │
└─────┬────────────┘
      │ 1. Validate request
      │ 2. Create SupabaseInstance CRD
      │    (status: "Pending")
      ▼
┌──────────────────┐
│   Controller     │
│  (watches CRDs)  │
└─────┬────────────┘
      │ 3. Sees "Pending" CRD
      │ 4. Creates Provisioning Job
      ▼
┌──────────────────┐
│ Provisioning Job │
│  (runs in K8s)   │
└─────┬────────────┘
      │ 5. Creates Namespace, Secrets, Helm Release
      │ 6. Exits with success/failure code
      ▼
┌──────────────────┐
│   Controller     │
│ (watches Jobs)   │
└─────┬────────────┘
      │ 7. Sees Job result
      │ 8. Updates CRD status to "Running" or "Failed"
      ▼
┌──────────┐
│  Client  │
│ (polling)│
└─────┬────┘
      │ GET /api/v1/instances/myapp
      │ (reads updated CRD status)
```

#### Authentication Flow

(No changes to this flow)

### Database Schema

#### Entity Relationship Diagram

```
┌─────────────────┐
│     users       │
├─────────────────┤
│ id (PK)         │
│ username        │
│ password_hash   │
│ role            │
│ created_at      │
│ updated_at      │
└────────┬────────┘
         │
         │ 1:N
         │
         ▼
┌─────────────────┐
│    api_keys     │
├─────────────────┤
│ id (PK)         │
│ name            │
│ key_hash        │
│ user_id (FK)    │
│ created_at      │
│ expires_at      │
│ last_used       │
└─────────────────┘

PostgreSQL Database Schema Notes:
- This database is for OPERATIONAL DATA ONLY (users, API keys).
- Instance state is NOT stored here. See ADR-001.
```

### State Management

**Application State**: Stateless (no in-memory session state).

**Persistent State**:
1. **Instance State**: Kubernetes `SupabaseInstance` CRDs are the **Single Source of Truth**.
   - Managed by the controller.
   - Queried via the Kubernetes API.
2. **Operational State**:
   - **User accounts**: PostgreSQL (`users` table).
   - **API keys**: PostgreSQL (`api_keys` table).
3. **Ephemeral State**:
   - **Provisioning/Cleanup**: Kubernetes `Jobs` manage the lifecycle of these operations.

This architecture eliminates state synchronization complexity by relying on the Kubernetes control plane for instance management.

## API Design

(No significant changes, but the backend implementation relies on CRDs)

### RESTful Principles

- Follows standard REST style: resource-based URLs, standard HTTP methods, and status codes.
- Asynchronous operations return `202 Accepted` and require polling for status.

## Security Architecture

### Defense in Depth

SupaControl implements multiple security layers:

```
┌─────────────────────────────────────────────────┐
│  Layer 1: Network Security                      │
│  - TLS/HTTPS encryption                         │
│  - Ingress firewall rules                       │
│  - Network policies (Kubernetes)                │
└─────────────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│  Layer 2: API Gateway                           │
│  - Rate limiting (future)                       │
│  - Request validation                           │
│  - CORS configuration                           │
└─────────────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│  Layer 3: Authentication & Authorization        │
│  - JWT validation (HS256)                       │
│  - API key validation                           │
│  - User identity verification                   │
└─────────────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│  Layer 4: Application Security                  │
│  - Input validation                             │
│  - SQL injection prevention (prepared statements)│
│  - XSS prevention (React escaping)              │
└─────────────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│  Layer 5: Data Security                         │
│  - Password hashing (bcrypt)                    │
│  - API key hashing (SHA-256)                    │
│  - Secrets encryption (Kubernetes Secrets)      │
└─────────────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│  Layer 6: Infrastructure Security               │
│  - Least-privilege RBAC (see below)             │
│  - Namespace isolation                          │
│  - Pod Security Standards                       │
└─────────────────────────────────────────────────┘
```

### Secrets Management

**Types of Secrets**:
1. **JWT Signing Secret**: Used to sign/verify JWTs for the API.
2. **Database Password**: For connecting to the operational PostgreSQL database.
3. **User Passwords**: Stored as bcrypt hashes in the database.
4. **API Keys**: Stored as SHA-256 hashes in the database.

**Storage**:
- **In Production**: All secrets are stored in Kubernetes Secrets, which should be encrypted at rest.
- **In Code**: Secrets are NEVER stored or committed in the codebase.
- **In Logs**: Secrets are NEVER logged.

### RBAC (Kubernetes)

SupaControl follows the principle of least privilege by using a two-tiered RBAC model. This approach avoids granting broad, cluster-wide permissions to provisioning jobs, significantly reducing the security risk. See **Security Advisory ADVISORY-001** for details.

#### 1. Controller RBAC (`ClusterRole`)

The main SupaControl controller runs with a `ClusterRole` that grants it only the permissions needed to manage `SupabaseInstance` CRDs, create namespaces, and manage RBAC resources within those namespaces.

```yaml
# Simplified main controller ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: supacontrol-controller-manager
rules:
  - apiGroups: ["supacontrol.qubitquilt.com"]
    resources: ["supabaseinstances", "supabaseinstances/finalizers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "get", "list", "watch"] # Note: delete is handled by finalizer logic
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["roles", "rolebindings"]
    verbs: ["create", "get", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["create", "get", "list", "watch", "delete"]
```

#### 2. Provisioning Job RBAC (`Role` - Per-Instance)

For each Supabase instance, the controller creates a dedicated, namespace-scoped `Role` and `RoleBinding`. The provisioning `Job` for that instance is then bound to this role, limiting its permissions to **only its own namespace**.

**Instance Creation Flow (RBAC)**:
1. Controller sees a new `SupabaseInstance` CRD.
2. Controller creates a new namespace (e.g., `supa-my-app`).
3. Controller creates a `Role` and `RoleBinding` inside the `supa-my-app` namespace.
4. Controller creates a provisioning `Job` that uses a `ServiceAccount` bound to this new, scoped `Role`.

This ensures that a compromised provisioning Job for `supa-my-app` **cannot** access resources in any other namespace (e.g., `supa-another-app` or `kube-system`).

```yaml
# Example Role created by the controller for a single instance
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: supacontrol-provisioner
  namespace: supa-my-app # Scoped to the instance's namespace
rules:
  # Permissions to manage resources ONLY within this namespace
  - apiGroups: ["", "apps", "networking.k8s.io", "batch"]
    resources: ["secrets", "services", "deployments", "ingresses", "jobs", ...]
    verbs: ["create", "delete", "get", "list", "update", "watch"]
```

**Principle of Least Privilege**:
- The main controller has limited cluster-wide permissions.
- Provisioning Jobs have **zero** cluster-wide permissions.
- The blast radius of a compromised provisioning Job is contained to a single instance's namespace.
- This is a critical security feature for a multi-tenant platform.

### Audit Logging

**What is Logged**:
- User authentication attempts (success/failure).
- API key creation and revocation.
- Instance creation and deletion requests.
- All API requests (endpoint, user, timestamp).

**What is NOT Logged**:
- Passwords, API keys, or JWT tokens.
- Sensitive user data from instance databases.


## Deployment Architecture

(No changes to this section)

## Scalability & Performance

(No changes to this section)

## Technology Stack Rationale

(No changes to this section)

## Design Decisions

### Decision 1: Stateless Application

**Context**: Need for horizontal scalability and cloud-native architecture.

**Decision**: Store all persistent state externally. No in-memory sessions.

**Rationale**:
- Enables horizontal scaling (add more pods).
- Simplifies deployment and recovery.
- Aligns with cloud-native best practices.
- State is managed by dedicated systems: Kubernetes for instance state and PostgreSQL for operational data.

### Decision 2: One Namespace Per Instance

(No changes to this decision)

### Decision 3: API-First Design

(No changes to this decision)

### Decision 4: JWT + API Keys

(No changes to this decision)

### Decision 5: CRD as Single Source of Truth (ADR-001)

**Context**: Need to store and manage instance state reliably and in a Kubernetes-native way.

**Decision**: Use Kubernetes `SupabaseInstance` CRDs as the Single Source of Truth for all instance state, not PostgreSQL.

**Rationale**:
- **Kubernetes-Native**: Aligns with the operator pattern.
- **Declarative**: Enables declarative state management and GitOps workflows.
- **No Sync Complexity**: Eliminates the risk of divergence between a database and the cluster's actual state.
- **Leverages K8s Features**: Utilizes metadata, status conditions, finalizers, and controller-runtime reconciliation.
- **Source of Truth Proximity**: State lives where the instances run.

**Tradeoffs**:
- Cannot query instance state without K8s API access (acceptable for a K8s control plane).
- No complex SQL queries on instance data (client-side filtering is sufficient).

**Reference**: See `docs/adr/001-crd-as-single-source-of-truth.md` for full details.

### Decision 6: Job-Based Provisioning (ADR-002)

**Context**: The need for reliable, observable, and non-blocking long-running operations like Helm installations.

**Decision**: Delegate provisioning and cleanup tasks to Kubernetes `Jobs`.

**Rationale**:
- **Non-Blocking**: The main controller remains responsive while Jobs run.
- **Observability**: Job status, logs, and events are easily monitored.
- **Reliability**: Jobs have built-in retry and timeout mechanisms.
- **Resource Isolation**: Provisioning tasks run in dedicated pods with their own resource limits.

**Reference**: See `docs/adr/002-job-based-provisioning-pattern.md` for full details.

## Future Architecture

(No changes to this section)

---

## Conclusion

SupaControl's architecture is designed for:
- **Simplicity**: Easy to understand and maintain.
- **Scalability**: Horizontal scaling and multi-tenancy.
- **Security**: Defense in depth, least privilege.
- **Extensibility**: Modular design for future enhancements.

The architecture balances pragmatism with best practices, focusing on delivering value while maintaining code quality and operational excellence.

For questions or suggestions about the architecture, please open an issue on GitHub.

---

**Document Version**: 1.1
**Last Updated**: November 2025
**Maintained By**: SupaControl Contributors
