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
│   │  - PostgreSQL Repository                     │          │
│   │  - Kubernetes Client                         │          │
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
│   │ - State      │         │ - Instances  │                 │
│   │ - Audit Logs │         │ - Namespaces │                 │
│   └──────────────┘         └──────────────┘                 │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

### High-Level Flow

1. **User Interaction**: Users interact via Web UI or CLI
2. **API Gateway**: All requests go through the REST API
3. **Authentication**: JWT/API Key validation
4. **Business Logic**: Request processing and validation
5. **Orchestration**: Kubernetes/Helm operations
6. **Persistence**: State stored in PostgreSQL
7. **Response**: Results returned to user

## Architecture Principles

SupaControl is built on these core architectural principles:

### 1. API-First Design

- All functionality exposed via REST API
- Web UI and CLI are API consumers
- Enables programmatic access and automation
- Facilitates testing and integration

### 2. Separation of Concerns

- **API Layer**: HTTP handling, routing, middleware
- **Business Logic**: Validation, state management, orchestration
- **Data Layer**: Database operations, Kubernetes client

### 3. Stateless Application

- No session state stored in application
- Horizontally scalable
- Cloud-native design
- All state in PostgreSQL

### 4. Declarative Infrastructure

- Kubernetes resources managed declaratively
- Helm charts for reproducible deployments
- Infrastructure as Code principles

### 5. Security by Default

- Authentication required (except public endpoints)
- Secrets encrypted at rest
- Least privilege RBAC
- Audit logging for compliance

### 6. Observability

- Structured logging
- Health check endpoints
- Metrics exposure (future: Prometheus)
- Error tracking and reporting

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
│   │   ├── api_keys.go     # API key repository
│   │   └── instances.go    # Instance repository
│   └── /k8s
│       ├── k8s.go          # Kubernetes client wrapper
│       ├── orchestrator.go # Helm operations
│       └── crclient.go     # Controller runtime client
└── /pkg
    └── /api-types          # Shared API types
```

### Component Responsibilities

#### 1. API Layer (`server/api/`)

**Responsibility**: HTTP request/response handling

**Functions**:
- Route registration and URL mapping
- Request validation and binding
- Response formatting (JSON)
- Error handling and status codes
- Middleware execution (auth, logging, CORS)

**Key Files**:
- `handlers.go`: Endpoint handlers (Login, CreateInstance, etc.)
- `router.go`: Echo router setup and route registration
- `middleware.go`: Authentication, logging, error recovery

**Pattern**: Handler → Business Logic → Response

```go
func (h *Handler) CreateInstance(c echo.Context) error {
    // 1. Parse request
    var req CreateInstanceRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    // 2. Call business logic
    instance, err := h.orchestrator.CreateInstance(c.Request().Context(), req.Name)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    // 3. Return response
    return c.JSON(http.StatusCreated, instance)
}
```

#### 2. Authentication Service (`server/internal/auth/`)

**Responsibility**: Security and identity management

**Functions**:
- JWT token generation and validation
- Password hashing (bcrypt)
- API key generation and validation
- Claims extraction and verification

**Security Design**:
- Passwords hashed with bcrypt (cost 10)
- JWTs signed with HS256 (HMAC-SHA256)
- API keys stored as hashed values
- 24-hour JWT expiration
- Revokable API keys

**Example Flow**:
```
1. User submits username/password
2. Hash compared with stored hash
3. Generate JWT with user claims
4. Return token to user
5. User includes token in subsequent requests
6. Middleware validates token on each request
```

#### 3. Database Layer (`server/internal/db/`)

**Responsibility**: Data persistence and retrieval

**Technology**: PostgreSQL + sqlx (prepared statements)

**Schema Design**:

```sql
-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- api_keys table
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    user_id INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP NULL
);

-- instances table
CREATE TABLE instances (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);
```

**Repository Pattern**:
- Each entity has a repository file
- CRUD operations encapsulated
- Database transactions where needed
- Soft deletes for instances

**Migrations**:
- Located in `server/internal/db/migrations/`
- Applied automatically on startup
- Sequentially numbered (001, 002, 003...)
- Idempotent (CREATE IF NOT EXISTS)

#### 4. Kubernetes Orchestrator (`server/internal/k8s/`)

**Responsibility**: Kubernetes and Helm operations

**Functions**:
- Create/delete Kubernetes namespaces
- Install/uninstall Helm releases
- Query instance status
- Manage Kubernetes resources

**Design**:
- Wrapper around client-go and Helm SDK
- In-cluster configuration support
- Out-of-cluster (KUBECONFIG) support
- Error handling and retries

**Instance Creation Flow**:
```
1. Validate instance name (DNS compliant)
2. Check if instance already exists in DB
3. Create Kubernetes namespace: supa-{name}
4. Prepare Helm values (ingress, domain, etc.)
5. Install Helm chart in namespace
6. Wait for installation to complete
7. Save instance record to database
8. Return instance details
```

**Namespace Isolation**:
- Each instance in separate namespace
- Naming convention: `supa-{instance-name}`
- Network policies for isolation (optional)
- Resource quotas per namespace (future)

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
└── /public
    └── assets/             # Static assets
```

**Technology**: React + Vite

**State Management**: React hooks (useState, useEffect)

**API Client**: Fetch API with Bearer token auth

**Routing**: React Router (client-side)

**Pattern**: Component → API Client → Backend

## Data Architecture

### Data Flow Diagrams

#### Instance Creation Data Flow

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
      │ 2. Extract user from JWT
      ▼
┌──────────────────┐
│  Orchestrator    │
│  CreateInstance  │
└─────┬────────────┘
      │
      ├─────► 3. Check DB: instance exists?
      │       └─[Database Repository]
      │
      ├─────► 4. Create namespace
      │       └─[Kubernetes API]
      │
      ├─────► 5. Install Helm chart
      │       └─[Helm SDK]
      │
      └─────► 6. Save instance record
              └─[Database Repository]

Success path:
  Database ──► Return instance ──► API ──► Client (201 Created)

Error path:
  Any step ──► Rollback (delete namespace) ──► API ──► Client (500 Error)
```

#### Authentication Flow

```
┌──────────┐
│  Client  │
└─────┬────┘
      │ POST /api/v1/auth/login
      │ {"username": "admin", "password": "admin"}
      ▼
┌──────────────────┐
│   API Handler    │
│      Login       │
└─────┬────────────┘
      │ 1. Bind request
      ▼
┌──────────────────┐
│ Database Repo    │
│ GetUserByUsername│
└─────┬────────────┘
      │ 2. Fetch user record
      ▼
┌──────────────────┐
│  Auth Service    │
│ CheckPassword    │
└─────┬────────────┘
      │ 3. Compare password hash
      │ (bcrypt.CompareHashAndPassword)
      ▼
    Match?
      │
      ├─ Yes ──► 4. Generate JWT
      │          └─[Auth Service: GenerateToken]
      │
      └─ No ───► Return 401 Unauthorized

Success:
  JWT Token ──► API ──► Client (200 OK)
  Client stores token in localStorage

Subsequent Requests:
  Client ──► Authorization: Bearer <token> ──► API
          └─[Middleware validates token]
```

### Database Schema

#### Entity Relationship Diagram

```
┌─────────────────┐
│     users       │
├─────────────────┤
│ id (PK)         │
│ username        │
│ password_hash   │
│ created_at      │
└────────┬────────┘
         │
         │ 1:N
         │
         ▼
┌─────────────────┐         ┌─────────────────┐
│    api_keys     │         │    instances    │
├─────────────────┤         ├─────────────────┤
│ id (PK)         │         │ id (PK)         │
│ name            │         │ name (unique)   │
│ key_hash        │         │ namespace       │
│ user_id (FK)    │         │ status          │
│ created_at      │         │ created_at      │
│ revoked_at      │         │ updated_at      │
└─────────────────┘         │ deleted_at      │
                            └─────────────────┘

Notes:
- users.password_hash: bcrypt hashed passwords
- api_keys.key_hash: SHA-256 hashed API keys
- api_keys.revoked_at: NULL = active, timestamp = revoked
- instances.deleted_at: NULL = active (soft delete pattern)
- instances.status: Pending, Running, Failed, Deleting
```

### State Management

**Application State**: Stateless (no in-memory state)

**Persistent State**:
1. **User accounts**: PostgreSQL (users table)
2. **API keys**: PostgreSQL (api_keys table)
3. **Instance metadata**: PostgreSQL (instances table)
4. **Kubernetes resources**: Kubernetes API (declarative)
5. **Helm releases**: Helm storage (in-cluster secrets)

**State Synchronization**:
- Database is source of truth for business state
- Kubernetes is source of truth for deployment state
- Reconciliation happens on GET requests (status checks)

## API Design

### RESTful Principles

SupaControl follows REST architectural style:

1. **Resource-Based URLs**: `/api/v1/instances`, `/api/v1/auth/api-keys`
2. **HTTP Methods**: GET (read), POST (create), DELETE (delete)
3. **Stateless**: No session state on server
4. **Standard Status Codes**: 200, 201, 400, 401, 404, 500
5. **JSON**: Request and response format

### API Versioning

- Version in URL path: `/api/v1/`
- Allows backward compatibility
- Future versions: `/api/v2/`

### Endpoint Design Patterns

#### Collection Endpoints

```
GET    /api/v1/instances       # List all instances
POST   /api/v1/instances       # Create new instance
```

#### Resource Endpoints

```
GET    /api/v1/instances/:name # Get specific instance
DELETE /api/v1/instances/:name # Delete instance
```

#### Action Endpoints

```
POST /api/v1/auth/login          # Login action
POST /api/v1/auth/api-keys       # Create API key
```

### Request/Response Patterns

**Standard Request**:
```http
POST /api/v1/instances
Authorization: Bearer eyJhbGc...
Content-Type: application/json

{
  "name": "my-app"
}
```

**Success Response**:
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": 1,
  "name": "my-app",
  "namespace": "supa-my-app",
  "status": "Pending",
  "created_at": "2024-01-15T10:00:00Z"
}
```

**Error Response**:
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "message": "instance name is required"
}
```

### Authentication Middleware

All protected endpoints use authentication middleware:

```go
func AuthMiddleware(authService *auth.Service) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // 1. Extract Authorization header
            authHeader := c.Request().Header.Get("Authorization")

            // 2. Parse Bearer token
            token := strings.TrimPrefix(authHeader, "Bearer ")

            // 3. Validate token (JWT or API key)
            claims, err := authService.ValidateToken(token)
            if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
            }

            // 4. Set user context
            c.Set("user", claims.Username)

            // 5. Continue to handler
            return next(c)
        }
    }
}
```

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
│  - RBAC (Kubernetes)                            │
│  - Least privilege ServiceAccount               │
│  - Namespace isolation                          │
└─────────────────────────────────────────────────┘
```

### Secrets Management

**Types of Secrets**:
1. **JWT Signing Secret**: Used to sign/verify JWTs
2. **Database Password**: PostgreSQL connection
3. **User Passwords**: Bcrypt hashed
4. **API Keys**: SHA-256 hashed

**Storage**:
- **In Production**: Kubernetes Secrets (base64 encoded, encrypted at rest if enabled)
- **In Code**: NEVER stored or committed
- **In Logs**: NEVER logged

**Best Practices**:
- Secrets generated with cryptographically secure random generator
- Minimum secret length enforced (64 bytes for JWT)
- Regular rotation recommended
- Separate secrets per environment

### RBAC (Kubernetes)

SupaControl ServiceAccount requires cluster-wide permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: supacontrol
rules:
  # Namespace management
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "delete", "get", "list"]

  # Resource management
  - apiGroups: [""]
    resources: ["secrets", "configmaps", "services", "persistentvolumeclaims"]
    verbs: ["create", "delete", "get", "list", "update"]

  # Workload management
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets"]
    verbs: ["create", "delete", "get", "list", "update"]

  # Ingress management
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["create", "delete", "get", "list", "update"]

  # Read-only access to pods (for status checks)
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
```

**Principle of Least Privilege**:
- Only permissions needed for operation
- No write access to pods (uses Helm/deployments)
- No cluster-admin privileges required

### Audit Logging

**What is Logged**:
- User authentication attempts (success/failure)
- API key creation and revocation
- Instance creation and deletion
- All API requests (endpoint, user, timestamp)

**What is NOT Logged**:
- Passwords or API keys
- JWT tokens
- Sensitive user data

**Future Enhancements**:
- Centralized logging (ELK stack, Loki)
- Log retention policies
- Compliance reporting

## Deployment Architecture

### Kubernetes Deployment

```
┌─────────────────────────────────────────────────────────┐
│              Kubernetes Cluster                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │  Namespace: supacontrol                        │    │
│  ├────────────────────────────────────────────────┤    │
│  │                                                 │    │
│  │  ┌──────────────────┐  ┌──────────────────┐   │    │
│  │  │   Deployment     │  │   Deployment     │   │    │
│  │  │   supacontrol    │  │   postgresql     │   │    │
│  │  │                  │  │                  │   │    │
│  │  │  ┌────────────┐  │  │  ┌────────────┐ │   │    │
│  │  │  │    Pod     │  │  │  │    Pod     │ │   │    │
│  │  │  │  Server    │  │  │  │  Database  │ │   │    │
│  │  │  └────────────┘  │  │  └────────────┘ │   │    │
│  │  │                  │  │                  │   │    │
│  │  │ (Replicas: 1-N) │  │ (Replicas: 1)   │   │    │
│  │  └──────────────────┘  └──────────────────┘   │    │
│  │           │                     │              │    │
│  │  ┌────────▼─────────────────────▼──────┐      │    │
│  │  │          Services                    │      │    │
│  │  │  - supacontrol (ClusterIP)          │      │    │
│  │  │  - postgresql (ClusterIP)           │      │    │
│  │  └──────────────┬──────────────────────┘      │    │
│  │                 │                              │    │
│  │  ┌──────────────▼──────────────────────┐      │    │
│  │  │        Ingress                      │      │    │
│  │  │  host: supacontrol.example.com     │      │    │
│  │  │  TLS: enabled                      │      │    │
│  │  └─────────────────────────────────────┘      │    │
│  │                                                 │    │
│  │  ┌─────────────────────────────────────┐      │    │
│  │  │        Secrets                       │      │    │
│  │  │  - JWT_SECRET                       │      │    │
│  │  │  - DB_PASSWORD                      │      │    │
│  │  └─────────────────────────────────────┘      │    │
│  │                                                 │    │
│  │  ┌─────────────────────────────────────┐      │    │
│  │  │      ConfigMap                       │      │    │
│  │  │  - Configuration values             │      │    │
│  │  └─────────────────────────────────────┘      │    │
│  │                                                 │    │
│  │  ┌─────────────────────────────────────┐      │    │
│  │  │  ServiceAccount + RBAC              │      │    │
│  │  │  - ClusterRole                      │      │    │
│  │  │  - ClusterRoleBinding               │      │    │
│  │  └─────────────────────────────────────┘      │    │
│  │                                                 │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Managed Instance Namespaces                    │   │
│  ├─────────────────────────────────────────────────┤   │
│  │  - supa-app1 (Supabase instance 1)             │   │
│  │  - supa-app2 (Supabase instance 2)             │   │
│  │  - supa-app3 (Supabase instance 3)             │   │
│  │  ...                                            │   │
│  └─────────────────────────────────────────────────┘   │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### High Availability Configuration

For production deployments:

```yaml
# Multiple replicas
replicaCount: 3

# Pod anti-affinity (spread across nodes)
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - supacontrol
          topologyKey: kubernetes.io/hostname

# Resource limits (prevent resource starvation)
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

# Health checks
livenessProbe:
  httpGet:
    path: /healthz
    port: 8091
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthz
    port: 8091
  initialDelaySeconds: 5
  periodSeconds: 5

# PostgreSQL HA
postgresql:
  architecture: replication
  replication:
    enabled: true
    numSynchronousReplicas: 1
```

## Scalability & Performance

### Horizontal Scalability

**Application Tier**:
- Stateless design allows horizontal scaling
- Multiple replicas behind load balancer (Kubernetes Service)
- Session-free architecture (JWT tokens)

**Database Tier**:
- PostgreSQL supports read replicas
- Connection pooling (future: PgBouncer)
- Indexed queries for performance

**Limitations**:
- Single database write leader (PostgreSQL constraint)
- Namespace creation is sequential (Kubernetes API)

### Performance Considerations

**Current Performance**:
- API response time: < 100ms (simple operations)
- Instance creation: 2-5 minutes (depends on Supabase Helm chart)
- Instance deletion: 1-2 minutes
- JWT validation: < 1ms

**Optimization Strategies**:
1. **Caching** (future):
   - Cache instance list in memory (with TTL)
   - Cache Kubernetes status queries

2. **Database Optimization**:
   - Indexes on frequently queried columns
   - Connection pooling
   - Prepared statements (already implemented with sqlx)

3. **Asynchronous Operations** (future):
   - Background job queue for long-running tasks
   - Webhooks for completion notifications

4. **Rate Limiting** (future):
   - Protect against abuse
   - Per-user quotas

### Resource Requirements

**Minimum (Development)**:
- CPU: 250m (0.25 cores)
- Memory: 256Mi
- Storage: 1Gi (PostgreSQL)

**Recommended (Production)**:
- CPU: 500m-1000m (0.5-1 cores)
- Memory: 512Mi-1Gi
- Storage: 20Gi (PostgreSQL with retention)

## Technology Stack Rationale

### Backend: Go + Echo

**Why Go?**
- Excellent Kubernetes client library (client-go)
- Fast compilation and execution
- Built-in concurrency (goroutines)
- Static typing and strong tooling
- Single binary deployment

**Why Echo?**
- Lightweight, high-performance framework
- Excellent middleware support
- Built-in JWT support
- Active community and maintenance

**Alternatives Considered**:
- **Fiber**: Similar performance, less mature
- **Gin**: Popular, but Echo has better middleware ecosystem

### Frontend: React + Vite

**Why React?**
- Large ecosystem and community
- Component-based architecture
- Excellent developer tools
- Wide adoption and talent availability

**Why Vite?**
- Fast dev server with HMR
- Modern build tool (ESBuild)
- Better DX than Webpack/CRA
- Optimized production builds

**Alternatives Considered**:
- **Vue**: Good, but smaller community for this use case
- **Svelte**: Interesting, but less mature ecosystem

### Database: PostgreSQL

**Why PostgreSQL?**
- Robust, battle-tested RDBMS
- Excellent support for JSON (for future extensions)
- ACID compliance
- Open source and widely deployed
- Strong Kubernetes integration (operators available)

**Alternatives Considered**:
- **MySQL**: Less feature-rich, weaker JSON support
- **SQLite**: Not suitable for multi-pod deployment
- **NoSQL**: Overkill for relational data model

### Orchestration: Kubernetes + Helm

**Why Kubernetes?**
- Industry standard for container orchestration
- SupaControl's primary use case is K8s environments
- Native namespace isolation
- Declarative resource management

**Why Helm?**
- Package manager for Kubernetes
- Supabase community provides Helm chart
- Enables repeatable deployments
- Template-based configuration

## Design Decisions

### Decision 1: Stateless Application

**Context**: Need for horizontal scalability and cloud-native architecture

**Decision**: Store all state in PostgreSQL, no in-memory sessions

**Rationale**:
- Enables horizontal scaling (add more pods)
- Simplifies deployment and recovery
- Cloud-native best practice

**Tradeoffs**:
- Database becomes single point of failure (mitigated with HA)
- Slight latency increase vs. in-memory (acceptable for use case)

### Decision 2: One Namespace Per Instance

**Context**: Instance isolation and resource management

**Decision**: Each Supabase instance in separate Kubernetes namespace

**Rationale**:
- Strong isolation between tenants
- Easy resource quota management per instance
- Simplified cleanup (delete namespace = delete all resources)
- Aligns with Supabase Helm chart expectations

**Tradeoffs**:
- Namespace proliferation (can reach K8s limits with 1000s of instances)
- Slightly more complex RBAC (cluster-level permissions needed)

### Decision 3: API-First Design

**Context**: Need to support UI, CLI, and programmatic access

**Decision**: All functionality exposed via REST API first

**Rationale**:
- Enables multiple client types (web, CLI, CI/CD)
- Facilitates testing (test API directly)
- Supports automation and integration

**Tradeoffs**:
- More initial development effort
- Need to maintain API compatibility

### Decision 4: JWT + API Keys

**Context**: Need for both interactive and programmatic authentication

**Decision**: Support both JWT (short-lived, user sessions) and API keys (long-lived, automation)

**Rationale**:
- JWTs: Good for web UI sessions (expire automatically)
- API keys: Good for CLI and CI/CD (revocable, long-lived)
- Both use same Bearer token auth mechanism

**Tradeoffs**:
- Complexity of supporting two auth methods
- API keys require storage and revocation management

### Decision 5: Soft Deletes for Instances

**Context**: Instance deletion and potential data recovery

**Decision**: Soft delete instances (mark as deleted, don't remove from DB)

**Rationale**:
- Audit trail (know what instances existed)
- Potential for "undelete" feature (future)
- Compliance and reporting

**Tradeoffs**:
- Database growth over time
- Need for cleanup/archival strategy (future)

## Future Architecture

### Planned Enhancements

#### 1. Event-Driven Architecture

**Current**: Synchronous API calls
**Future**: Event queue for long-running operations

```
API Request → Queue Job → Background Worker → Webhook Notification
```

**Benefits**:
- Non-blocking API responses
- Retry logic for failures
- Better resource utilization

#### 2. Observability

**Metrics** (Prometheus):
- Instance creation rate
- API request latency
- Error rates
- Resource usage per instance

**Tracing** (Jaeger):
- Request flow visualization
- Performance bottleneck identification

**Logging** (Loki):
- Centralized log aggregation
- Structured logging with correlation IDs

#### 3. Multi-Cluster Support

**Current**: Single Kubernetes cluster
**Future**: Manage instances across multiple clusters

```
┌───────────────┐
│  SupaControl  │
└───────┬───────┘
        │
        ├──────► Cluster A (us-west)
        ├──────► Cluster B (us-east)
        └──────► Cluster C (eu-west)
```

**Benefits**:
- Geographic distribution
- Fault tolerance
- Regulatory compliance (data residency)

#### 4. Instance Templates

**Current**: Default Supabase installation
**Future**: Predefined instance configurations

```yaml
templates:
  - name: "small"
    resources: { cpu: "500m", memory: "1Gi" }
  - name: "medium"
    resources: { cpu: "2000m", memory: "4Gi" }
  - name: "large"
    resources: { cpu: "4000m", memory: "8Gi" }
```

#### 5. Cost Tracking

Track resource usage and costs per instance:
- CPU/memory consumption
- Storage usage
- Network transfer
- Cost attribution per customer

#### 6. GitOps Integration

Support declarative instance management:

```yaml
# instances.yaml
apiVersion: supacontrol.io/v1
kind: Instance
metadata:
  name: production-app
spec:
  template: large
  domain: app.example.com
```

Sync with ArgoCD/Flux for GitOps workflow.

---

## Conclusion

SupaControl's architecture is designed for:
- **Simplicity**: Easy to understand and maintain
- **Scalability**: Horizontal scaling and multi-tenancy
- **Security**: Defense in depth, least privilege
- **Extensibility**: Modular design for future enhancements

The architecture balances pragmatism with best practices, focusing on delivering value while maintaining code quality and operational excellence.

For questions or suggestions about the architecture, please open an issue on GitHub.

---

**Document Version**: 1.0
**Last Updated**: November 2025
**Maintained By**: SupaControl Contributors
