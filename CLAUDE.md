# CLAUDE.md - AI Assistant Guide for SupaControl

This document provides guidance for AI assistants (like Claude) working with the SupaControl codebase. It includes architecture overview, development patterns, and common workflows.

## Project Overview

**SupaControl** is a self-hosted management platform for orchestrating multi-tenant Supabase instances on Kubernetes. It provides:
- REST API for programmatic control
- React-based web dashboard
- CLI installer for deployment
- Kubernetes-native orchestration using Helm

### Key Technologies
- **Backend**: Go (Echo framework), PostgreSQL
- **Frontend**: React + Vite
- **Orchestration**: Kubernetes client-go, Helm v3 SDK
- **Infrastructure**: Docker, Helm charts, K8s

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      SupaControl                         │
├─────────────────────────────────────────────────────────┤
│  Web UI (React) ──► REST API (Echo/Go)                  │
│                          │                               │
│                    Controller (K8s Operator)            │
│                   watches SupabaseInstance CRDs         │
│                          │                               │
│                    PostgreSQL DB                         │
│                (Users, API Keys, Audit Logs)            │
└─────────────────────────────────────────────────────────┘
                          │
                  Kubernetes API
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   Instance1         Instance2         Instance3
   (supa-app1)      (supa-app2)      (supa-app3)

Note: Instance state stored in SupabaseInstance CRDs, not PostgreSQL (ADR-001)
```

## Code Organization

### Directory Structure

```
/SupaControl
├── /server/              # Go backend
│   ├── /api             # REST API routes and handlers
│   │   ├── handlers.go  # HTTP request handlers
│   │   ├── router.go    # Route definitions
│   │   └── middleware.go # Auth, logging, CORS
│   ├── /internal        # Core business logic
│   │   ├── /auth        # JWT & password hashing (74.5% tested)
│   │   ├── /config      # Environment configuration (55% tested)
│   │   ├── /db          # Database layer (sqlx) - 0% coverage
│   │   │   ├── db.go           # Connection & migrations
│   │   │   └── api_keys.go     # API key CRUD
│   │   └── /k8s         # K8s orchestration (2.9% tested)
│   │       ├── k8s.go          # K8s client wrapper
│   │       ├── orchestrator.go # Helm operations (legacy)
│   │       └── crclient.go     # CRD client (instance CRUD)
│   ├── main.go          # Application entry point
│   └── go.mod           # Go dependencies
│
├── /ui/                 # React frontend (5.87% tested)
│   ├── /src
│   │   ├── /pages       # Route components
│   │   │   ├── Dashboard.jsx  # Instance list/management
│   │   │   ├── Login.jsx      # Auth page
│   │   │   └── Settings.jsx   # API key management
│   │   ├── api.js       # API client (78% tested)
│   │   ├── App.jsx      # Main app component
│   │   └── main.jsx     # Entry point
│   └── vite.config.js   # Build configuration
│
├── /cli/                # Interactive installer (TypeScript/React Ink)
│   ├── /src
│   │   ├── /components  # UI components (untested)
│   │   │   ├── ConfigurationWizard.tsx
│   │   │   ├── Installation.tsx
│   │   │   └── PrerequisitesCheck.tsx
│   │   └── /utils       # Utility functions (partially tested)
│   │       ├── helm.ts         # Helm operations
│   │       ├── prerequisites.ts # K8s/Helm checks
│   │       └── secrets.ts      # Secret generation
│   └── cli.tsx          # CLI entry point
│
├── /charts/             # Helm chart
│   └── /supacontrol     # SupaControl deployment chart
│
├── README.md            # User documentation
├── CONTRIBUTING.md      # Contribution guide
├── TESTING.md           # Testing guide (current coverage: ~6%)
└── CLAUDE.md            # This file
```

## Key Components

### 1. API Handlers (`server/api/handlers.go`)

**Purpose**: HTTP request handling for all API endpoints

**Key Functions**:
- `Login()` - POST /api/v1/auth/login (public)
- `GetMe()` - GET /api/v1/auth/me (authenticated)
- `CreateAPIKey()` - POST /api/v1/auth/api-keys (authenticated)
- `ListAPIKeys()` - GET /api/v1/auth/api-keys (authenticated)
- `RevokeAPIKey()` - DELETE /api/v1/auth/api-keys/:id (authenticated)
- `ListInstances()` - GET /api/v1/instances (authenticated)
- `CreateInstance()` - POST /api/v1/instances (authenticated)
- `GetInstance()` - GET /api/v1/instances/:name (authenticated)
- `DeleteInstance()` - DELETE /api/v1/instances/:name (authenticated)

**Important**: All endpoints except `/healthz` and `/api/v1/auth/login` require Bearer token authentication.

### 2. Authentication Service (`server/internal/auth/auth.go`)

**Purpose**: JWT generation/validation and password hashing

**Key Functions**:
- `GenerateToken(username string) (string, error)` - Creates JWT
- `ValidateToken(tokenString string) (*Claims, error)` - Validates JWT
- `HashPassword(password string) (string, error)` - bcrypt hashing
- `CheckPassword(hash, password string) error` - Verifies password

**Testing**: 74.5% coverage - well tested

### 3. Database Layer (`server/internal/db/`)

**Purpose**: SupaControl operational data persistence (users, API keys)

**IMPORTANT**: Per ADR-001, instance state is stored in Kubernetes CRDs, NOT PostgreSQL. The `instances` table was removed by migration 003 (`server/internal/db/migrations/003_remove_instances_table.sql`) as it was never used; all instance operations use CRDs via `server/internal/k8s/crclient.go`. See ADR-001: `docs/adr/001-crd-as-single-source-of-truth.md`.

**Files**:
- `db.go` - Connection, migrations, health checks
- `api_keys.go` - CRUD for API keys

**Schema**:
```sql
-- users table
id, username, password_hash, role, created_at, updated_at

-- api_keys table
id, name, key_hash, user_id, created_at, expires_at, last_used
```

**Instance State** (NOT in PostgreSQL):
- Stored as SupabaseInstance CRDs in Kubernetes
- Accessed via `server/internal/k8s/crclient.go`
- See ADR-001: `docs/adr/001-crd-as-single-source-of-truth.md`

**Migrations**: Located in `server/internal/db/migrations/`, auto-applied on startup

**Testing**: 0% coverage - NEEDS TESTS

### 4. Kubernetes Controller (`server/controllers/supabaseinstance_controller.go`)

**Purpose**: Kubernetes operator that reconciles SupabaseInstance CRDs

**Architecture**:
- API handlers create/delete SupabaseInstance CRDs (`server/api/handlers.go` uses `crClient`)
- Controller watches CRDs and reconciles desired state
- Controller manages namespace, secrets, Helm releases, ingresses

**Reconciliation Phases**:
- `Pending` → `Provisioning` → `Running` (success path)
- Any phase → `Failed` (error path)
- `Deleting` (cleanup with finalizer)

**Pattern**: Each instance gets its own namespace `supa-{name}`

**Testing**: 0% coverage - NEEDS TESTS (use controller-runtime envtest)

### 5. React Dashboard (`ui/src/pages/Dashboard.jsx`)

**Purpose**: Web UI for instance management

**Features**:
- List all instances
- Create new instances
- View instance status
- Delete instances
- API key management (Settings page)

**State Management**: React hooks (useState, useEffect)
**API Client**: `ui/src/api.js` (78% tested)

**Testing**: 0% coverage - NEEDS TESTS

### 6. CLI Installer (`cli/src/`)

**Purpose**: Interactive Kubernetes deployment wizard

**Flow**:
1. Welcome screen
2. Prerequisites check (kubectl, helm, K8s connection)
3. Configuration wizard (domain, ingress, secrets)
4. Installation (helm install)
5. Completion summary

**Testing**: Utils tested, components untested

## Development Workflows

### Adding a New API Endpoint

1. **Define request/response types** in `server/api/v1alpha1/supabaseinstance_types.go` if CRD-related
2. **Add handler function** in `server/api/handlers.go`
3. **Register route** in `server/api/router.go`
4. **Add database operations** in `server/internal/db/` if needed
5. **Write tests** in `server/api/handlers_test.go`
6. **Update frontend** in `ui/src/api.js` and relevant page

Example:
```go
// 1. Add handler using crClient for CRD operations
func (h *Handler) GetInstanceLogs(c echo.Context) error {
    name := c.Param("name")
    logs, err := h.k8s.GetLogs(c.Request().Context(), name)  // Uses k8s client wrapper
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    return c.JSON(http.StatusOK, logs)
}

// 2. Register route
api.GET("/instances/:name/logs", handler.GetInstanceLogs)
```

**Note**: For CRD operations, use `server/internal/k8s/crclient.go` (aligns with ADR-001). For job-based provisioning (ADR-002), reference `docs/adr/002-job-based-provisioning-pattern.md` when implementing long-running operations like Helm installs.

### Adding Database Migrations

1. **Create new migration file**: `server/internal/db/migrations/00X_description.sql`
2. **Number sequentially**: 001, 002, 003, etc.
3. **Write SQL**: Use `CREATE TABLE IF NOT EXISTS` patterns
4. **Test locally**: Restart server to auto-apply
5. **Update repository functions**: Add CRUD operations in `server/internal/db/`

Example migration:
```sql
-- 004_add_instance_metadata.sql
ALTER TABLE instances ADD COLUMN metadata JSONB DEFAULT '{}';
CREATE INDEX idx_instances_metadata ON instances USING GIN (metadata);
```

### Testing Changes

**Backend**:
```bash
cd server
go test -v ./...                    # Run all tests
go test -coverprofile=coverage.out ./...  # With coverage
go test -race ./...                 # Race detection
```

**Frontend**:
```bash
cd ui
npm test                            # Run tests
npm run test:coverage               # With coverage
npm run test:ui                     # Interactive UI
```

**CLI**:
```bash
cd cli
npm test                            # Run tests
npm start                           # Test installer locally
```

**All Components**:
```bash
make test                           # Run all tests
make test-coverage                  # Generate coverage report
make ci                             # Full CI checks
```

### Building and Deploying

**Local Development**:
```bash
# Backend
cd server && go run main.go

# Frontend (dev server)
cd ui && npm run dev

# Frontend (build for production)
cd ui && npm run build
```

**Docker**:
```bash
# Build backend image
docker build -t supacontrol/server:latest -f server/Dockerfile .

# Build with UI included
cd ui && npm run build && cd ..
docker build -t supacontrol/server:latest .
```

**Helm Deployment**:
```bash
# Using CLI installer (recommended)
cd cli && npm start

# Manual installation
helm install supacontrol ./charts/supacontrol -f values.yaml

# Upgrade
helm upgrade supacontrol ./charts/supacontrol -f values.yaml
```

## Important Patterns and Conventions

### Error Handling

**Go Backend**:
- Always return errors explicitly
- Use `fmt.Errorf()` for error wrapping
- HTTP handlers return `echo.NewHTTPError(statusCode, message)`
- Log errors before returning

Example:
```go
func (s *Service) GetInstance(ctx context.Context, name string) (*Instance, error) {
    if name == "" {
        return nil, fmt.Errorf("instance name is required")
    }

    instance, err := s.db.GetInstanceByName(ctx, name)
    if err != nil {
        log.Printf("failed to get instance %s: %v", name, err)
        return nil, fmt.Errorf("database error: %w", err)
    }

    return instance, nil
}
```

**React Frontend**:
- Use try/catch for async operations
- Display user-friendly error messages
- Log errors to console for debugging

Example:
```javascript
try {
    const instances = await api.listInstances();
    setInstances(instances);
} catch (error) {
    console.error('Failed to load instances:', error);
    setError('Failed to load instances. Please try again.');
}
```

### Authentication Flow

1. User logs in via `/api/v1/auth/login`
2. Backend validates credentials and returns JWT
3. Frontend stores JWT in localStorage
4. All subsequent requests include `Authorization: Bearer <token>`
5. Backend validates JWT on protected endpoints
6. API keys can be created via `/api/v1/auth/api-keys` (also use Bearer auth)

### Instance Lifecycle

1. **Create**: POST /api/v1/instances
   - Validates name
   - Creates SupabaseInstance CRD in Kubernetes
   - Controller watches CRD and provisions resources
   - Controller creates namespace `supa-{name}`, secrets, and Helm release

2. **Monitor**: GET /api/v1/instances/:name
   - Queries SupabaseInstance CRD for status
   - Returns deployment state from CRD status

3. **Delete**: DELETE /api/v1/instances/:name
   - Deletes SupabaseInstance CRD
   - Controller finalizer ensures cleanup (Helm release, namespace)
   - CRD removed from Kubernetes

### Security Considerations

**When Making Changes**:
- Never commit secrets or credentials
- Use parameterized queries (sqlx handles this)
- Validate all user inputs
- Use HTTPS in production (configure ingress)
- Don't log sensitive data (passwords, tokens)
- Check for SQL injection, XSS, CSRF vulnerabilities

**OWASP Top 10**:
- A1 (Injection): Use sqlx prepared statements
- A2 (Auth): JWT validation on all protected routes
- A3 (Data Exposure): Don't return password hashes
- A5 (Access Control): Verify user owns resources
- A7 (XSS): React escapes by default, but sanitize innerHTML
- A8 (Deserialization): Validate JSON inputs

## Common Tasks

### Task: Add New Instance Field

**Example**: Add "description" field to instances

**Note**: Per ADR-001, instance state is stored in SupabaseInstance CRDs, not PostgreSQL.

1. **Update CRD spec**:
```go
// server/api/v1alpha1/supabaseinstance_types.go
type SupabaseInstanceSpec struct {
    ProjectName string `json:"projectName"`
    Description string `json:"description,omitempty"`  // NEW
    // ... other fields
}
```

2. **Update API types** if needed for request/response

3. **Update handler to pass description to CRD**:
```go
// server/api/handlers.go - in CreateInstance handler
instance := &v1.SupabaseInstance{
    Spec: v1.SupabaseInstanceSpec{
        ProjectName: req.Name,
        Description: req.Description,  // NEW
        // ... other fields
    },
}
if err := h.crClient.CreateSupabaseInstance(ctx, instance); err != nil {  // Use crClient
    // handle error
}
```

4. **Update frontend**:
```javascript
// ui/src/pages/Dashboard.jsx
<input
    type="text"
    placeholder="Description"
    value={description}
    onChange={(e) => setDescription(e.target.value)}
/>
```

### Task: Add New API Endpoint

**Example**: Get instance logs

1. **Add K8s function** if needed in `server/internal/k8s/k8s.go` or use existing

2. **Add handler**:
```go
// server/api/handlers.go
func (h *Handler) GetInstanceLogs(c echo.Context) error {
    name := c.Param("name")
    logs, err := h.k8s.GetLogs(c.Request().Context(), name)  // Uses k8s client
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    return c.JSON(http.StatusOK, map[string]string{"logs": logs})
}
```

3. **Register route**:
```go
// server/api/router.go
api.GET("/instances/:name/logs", handler.GetInstanceLogs)
```

4. **Add frontend function**:
```javascript
// ui/src/api.js
export async function getInstanceLogs(name) {
    const response = await fetch(`${API_URL}/instances/${name}/logs`, {
        headers: { 'Authorization': `Bearer ${getToken()}` }
    });
    if (!response.ok) throw new Error('Failed to get logs');
    return response.json();
}
```

### Task: Fix Test Coverage

**Priority Areas** (current coverage ~6%):
1. API handlers (0% coverage)
2. Database operations (0% coverage)
3. React components (0% coverage)
4. K8s orchestrator (2.9% coverage)

**Example: Add API handler test**:
```go
// server/api/handlers_test.go
func TestListInstances(t *testing.T) {
    // Setup
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
    req.Header.Set(echo.HeaderAuthorization, "Bearer "+validToken)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Test
    err := handler.ListInstances(c)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
}
```

## Testing Strategy

### Current State (as of November 2025)

**Coverage**:
- Backend (Go): ~6% overall
  - auth: 74.5% ✅
  - config: 55% ⚠️
  - k8s: 2.9% ❌
  - db: 0% ❌
  - api: 0% ❌
- Frontend (React): 5.87% overall
  - api.js: 78% ✅
  - components: 0% ❌
- CLI (TypeScript): Utilities tested, components untested

**What Needs Testing**:
1. API endpoints (auth, CRUD operations)
2. Database operations (migrations, queries)
3. React components (Dashboard, Login, Settings)
4. K8s orchestration (namespace creation, Helm operations)
5. Error handling and edge cases

### Writing Good Tests

**Backend (Go) - Table-driven tests**:
```go
func TestValidateInstanceName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
        wantErr bool
    }{
        {"valid name", "my-app", true, false},
        {"empty name", "", false, true},
        {"too long", strings.Repeat("a", 100), false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateInstanceName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Frontend (React) - Component tests**:
```javascript
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Dashboard from './Dashboard';

test('displays instances after loading', async () => {
    render(<Dashboard />);

    await waitFor(() => {
        expect(screen.getByText('my-app')).toBeInTheDocument();
    });
});

test('creates new instance on form submit', async () => {
    const user = userEvent.setup();
    render(<Dashboard />);

    await user.type(screen.getByLabelText('Name'), 'new-app');
    await user.click(screen.getByText('Create'));

    await waitFor(() => {
        expect(screen.getByText('new-app')).toBeInTheDocument();
    });
});
```

## Gotchas and Important Notes

### 1. Database Migrations

- Migrations run automatically on server startup
- Never modify existing migrations - create new ones
- Use sequential numbering (001, 002, 003...)
- Test migrations on fresh database before committing
- Recent change: Migration 003 removed the unused `instances` table (per ADR-001)

### 2. Kubernetes Permissions

- SupaControl needs cluster-wide permissions (ClusterRole)
- Required: namespace create/delete, secrets, configmaps, deployments
- RBAC configured in Helm chart (`charts/supacontrol/templates/`)

### 3. Instance Naming

- Must be valid DNS label (lowercase, alphanumeric, hyphens)
- Maximum 63 characters (K8s limit)
- Namespace is `supa-{name}` (prefix added automatically)

### 4. JWT Secret

- MUST be set via environment variable or config
- Never commit JWT_SECRET to git
- Change default in production
- Invalidating secret invalidates all tokens

### 5. API Authentication

- `/healthz` and `/api/v1/auth/login` are public
- All other endpoints require `Authorization: Bearer <token>`
- API keys and JWT tokens both work as Bearer tokens
- Frontend stores token in localStorage

### 6. Development Environment

- Backend requires PostgreSQL running
- Frontend dev server proxies API calls (see `vite.config.js`)
- Use `make test` to run all tests before committing
- CI runs on push to main/develop branches

### 7. Helm Chart Versioning

- Supabase Helm chart version configurable
- Default chart repo: https://supabase-community.github.io/supabase-kubernetes
- Version updates may break compatibility

### 8. Test Coverage

- Current coverage is LOW (~6%)
- Focus on critical paths: auth, instance CRUD, API handlers
- Use `make test-coverage` to check coverage
- Aim for 70%+ coverage on new code
- Recent additions: Controller test suite in `server/controllers/suite_test.go`; Metrics module in `server/internal/metrics/`

### 9. Recent Architectural Changes

- **ADR-001**: CRDs as single source of truth - instance state managed via `server/internal/k8s/crclient.go`, no PostgreSQL table
- **ADR-002**: Job-based provisioning pattern - long-running operations (Helm install/uninstall) delegated to Kubernetes Jobs for better reliability and observability (`docs/adr/002-job-based-provisioning-pattern.md`)
- Use `crclient.go` for all CRD operations; `orchestrator.go` is legacy for direct Helm calls

## CI/CD Pipeline

### GitHub Actions (`.github/workflows/ci.yml`)

**Triggers**: Push to main/develop, PRs to main/develop

**Steps**:
1. Backend tests (Go)
   - Run tests with race detection
   - Generate coverage report
   - Upload to Codecov
2. Frontend tests (React)
   - Run Vitest tests
   - Generate coverage report
   - Upload to Codecov
3. Linting
   - Go: `go vet`, `golangci-lint`
   - React: ESLint
4. Build
   - Build backend binary
   - Build frontend assets
   - Upload artifacts

**Coverage Badge**: See README.md

## Resources

### Documentation
- **User Guide**: README.md
- **Contributing**: CONTRIBUTING.md
- **Testing**: TESTING.md
- **CLI Installer**: cli/README.md

### External Docs
- [Echo Framework](https://echo.labstack.com/)
- [React](https://react.dev/)
- [Kubernetes Client-Go](https://github.com/kubernetes/client-go)
- [Helm Go SDK](https://helm.sh/docs/topics/advanced/)
- [sqlx](https://github.com/jmoiron/sqlx)

### Project Links
- **GitHub**: https://github.com/qubitquilt/SupaControl
- **CLI Tool**: https://github.com/qubitquilt/supactl
- **Issues**: https://github.com/qubitquilt/SupaControl/issues

## Quick Reference

### Common Commands

```bash
# Development
make test                 # Run all tests
make test-coverage        # Generate coverage report
make ci                   # Run CI checks locally
cd server && go run main.go  # Run backend
cd ui && npm run dev      # Run frontend dev server

# Testing
cd server && go test -v ./...           # Backend tests
cd ui && npm test                       # Frontend tests
cd cli && npm test                      # CLI tests
go test -coverprofile=coverage.out ./... # Backend coverage
npm run test:coverage                   # Frontend coverage

# Building
docker build -t supacontrol/server:latest .  # Build Docker image
cd ui && npm run build                       # Build frontend
helm package ./charts/supacontrol            # Package Helm chart

# Deployment
cd cli && npm start                      # Interactive installer
helm install supacontrol ./charts/supacontrol -f values.yaml  # Manual install
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DB_HOST` | PostgreSQL host | Yes |
| `DB_PORT` | PostgreSQL port | Yes |
| `DB_USER` | Database username | Yes |
| `DB_PASSWORD` | Database password | Yes |
| `DB_NAME` | Database name | Yes |
| `JWT_SECRET` | JWT signing secret | Yes |
| `SERVER_PORT` | HTTP server port | No (default: 8091) |
| `KUBECONFIG` | Path to kubeconfig | No (in-cluster) |
| `DEFAULT_INGRESS_CLASS` | Ingress class | No (default: nginx) |
| `DEFAULT_INGRESS_DOMAIN` | Base domain | No (default: supabase.example.com) |

### API Endpoints Quick Reference

```
# Public
GET  /healthz                          # Health check

# Authentication
POST /api/v1/auth/login                # Login
GET  /api/v1/auth/me                   # Get current user

# API Keys
POST   /api/v1/auth/api-keys           # Create API key
GET    /api/v1/auth/api-keys           # List API keys
DELETE /api/v1/auth/api-keys/:id       # Revoke API key

# Instances
GET    /api/v1/instances               # List instances
POST   /api/v1/instances               # Create instance
GET    /api/v1/instances/:name         # Get instance
DELETE /api/v1/instances/:name         # Delete instance
```

---

**Last Updated**: November 2025
**Maintained By**: SupaControl Contributors
**Questions?**: Open an issue on GitHub
