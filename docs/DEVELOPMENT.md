# Development Guide

This guide covers local development setup, project structure, and development workflows for SupaControl.

## Table of Contents

- [Prerequisites for Development](#prerequisites-for-development)
- [Project Structure](#project-structure)
- [Backend Development](#backend-development)
- [Frontend Development](#frontend-development)
- [Building](#building)
- [Code Style](#code-style)
- [Local Testing](#local-testing)

## Prerequisites for Development

- **Go** 1.24+
- **Node.js** 18+
- **PostgreSQL** 14+
- **Docker** (optional, for containerized PostgreSQL)
- **Kubernetes cluster** (for integration testing)
- **kubectl** and **helm** (for Kubernetes testing)

## Project Structure

```
/SupaControl
├── /server/              # Go backend
│   ├── /api             # REST API (handlers, routes, middleware)
│   ├── /internal        # Core business logic
│   │   ├── /auth        # JWT & password hashing
│   │   ├── /config      # Configuration management
│   │   ├── /db          # Database layer (sqlx + migrations)
│   │   └── /k8s         # Kubernetes orchestration (Helm + client-go)
│   ├── main.go          # Application entry point
│   └── go.mod           # Go dependencies
│
├── /ui/                 # React frontend
│   ├── /src
│   │   ├── /pages       # Page components (Dashboard, Login, Settings)
│   │   ├── api.js       # API client
│   │   ├── App.jsx      # Main app component
│   │   └── main.jsx     # Entry point
│   ├── package.json     # NPM dependencies
│   └── vite.config.js   # Vite build configuration
│
├── /cli/                # Interactive installer
│   ├── /src
│   │   ├── /components  # Ink React components
│   │   └── /utils       # Helper functions
│   └── package.json
│
├── /pkg/                # Shared packages
│   └── /api-types       # API request/response types
│
├── /charts/             # Helm charts
│   └── /supacontrol     # SupaControl deployment chart
│
├── /.github/            # GitHub Actions CI/CD
│   └── /workflows
│       └── ci.yml       # CI pipeline
│
├── /docs/               # Detailed documentation
│   ├── /adr             # Architecture Decision Records
│   ├── API.md
│   ├── ARCHITECTURE.md
│   └── ...
├── README.md            # This file
├── CONTRIBUTING.md      # Contribution guidelines
├── TESTING.md           # Testing documentation
├── CLAUDE.md            # AI assistant guide
└── LICENSE.md           # MIT License
```

## Backend Development

```bash
cd server

# Install dependencies
go mod download

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=supacontrol
export DB_PASSWORD=password
export DB_NAME=supacontrol
export JWT_SECRET=your-dev-jwt-secret-change-this

# Set KUBECONFIG for local Kubernetes development
# This is crucial if you're not running in-cluster or using a non-default kubeconfig path
export KUBECONFIG=$HOME/.kube/config  # or your custom kubeconfig path

# Option 1: Use Docker for PostgreSQL
docker run --name supacontrol-postgres \
  -e POSTGRES_USER=supacontrol \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=supacontrol \
  -p 5432:5432 \
  -d postgres:14

# Option 2: Use local PostgreSQL
createdb supacontrol

# Run the server (migrations auto-apply on startup)
go run main.go

# Server runs on http://localhost:8091
```

### Kubernetes Configuration for Local Development

When developing locally, SupaControl needs access to a Kubernetes cluster to manage Supabase instances. Configure your environment based on your setup:

**For local clusters (minikube, kind, k3s, Docker Desktop):**
```bash
# Use default kubeconfig location
export KUBECONFIG=$HOME/.kube/config

# Verify connection
kubectl cluster-info
kubectl get nodes
```

**For custom kubeconfig paths:**
```bash
# Point to your custom config
export KUBECONFIG=/path/to/your/kubeconfig.yaml
```

**Troubleshooting:**
- If you see "unable to connect to Kubernetes" errors, check your `KUBECONFIG`.
- Verify your cluster is accessible: `kubectl get nodes`.
- Ensure the controller has the necessary RBAC permissions to create resources.

### Security Considerations (RBAC)

**IMPORTANT**: The project has a critical security advisory regarding RBAC permissions. Read **[ADVISORY-001-provisioner-rbac.md](./security/ADVISORY-001-provisioner-rbac.md)**.

The correct, secure architecture uses a **two-tiered RBAC model**:

1.  **Controller `ClusterRole`**: The main controller has a `ClusterRole` with limited permissions to manage namespaces and the RBAC roles for each instance.
2.  **Provisioner `Role` (Namespace-Scoped)**: For each instance, the controller creates a `Role` and `RoleBinding` that is scoped **only to that instance's namespace**. The provisioning `Job` uses these scoped permissions.

When developing, ensure your changes adhere to this model to maintain security and isolation between tenants. The controller should be responsible for creating the namespace-scoped RBAC, and the provisioning jobs should operate with those limited permissions.

**Backend Code Guidelines:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html) conventions.
- Use `gofmt` for formatting and `go vet` for analysis.
- Add tests for all new features, especially for controller logic.
- Handle errors explicitly and add context.
- Add comments for exported functions and complex logic.

## Frontend Development

```bash
cd ui

# Install dependencies
npm install

# Start development server with hot reload
npm run dev

# Development server runs on http://localhost:5173
# API calls are proxied to backend (see vite.config.js)
```

**Frontend Code Guidelines:**
- Use functional components with hooks
- Follow React best practices
- Use meaningful component/variable names
- Keep components focused and reusable
- Add PropTypes or TypeScript types
- Test user interactions

## Building

```bash
# Build backend binary
cd server
go build -o supacontrol

# Build frontend for production
cd ui
npm run build
# Output: ui/dist/

# Build Docker image (with UI embedded)
cd ui && npm run build && cd ..
docker build -t supacontrol/server:latest .

# Build Helm chart
helm package ./charts/supacontrol
# Output: supacontrol-<version>.tgz
```

## Local Code Quality Checks

SupaControl includes comprehensive local linting and code quality checks that mirror the CI pipeline, helping you catch issues before pushing code.

### Quick Start

```bash
# Run all linting checks
make lint

# Auto-fix lintable issues
make lint-fix

# Run pre-commit checks (comprehensive local CI simulation)
make pre-commit

# Run full CI checks locally
make ci
```

### Available Makefile Targets

| Command | Description |
|---------|-------------|
| `make lint` | Run all linters (Go + UI) |
| `make lint-fix` | Auto-fix lintable issues |
| `make pre-commit` | Run comprehensive pre-flight checks |
| `make format` | Format all code (Go + UI) |
| `make ci` | Run full CI pipeline locally |

### Local Pre-commit Hooks

Enable automatic checks before every commit:

**Option 1: Using pre-commit (Recommended)**
```bash
# Install pre-commit
pip install pre-commit

# Install the hooks
pre-commit install

# Run on all files (first time)
pre-commit run --all-files

# Update hooks
pre-commit autoupdate
```

**Option 2: Manual Pre-commit Script**
```bash
# Run manually before committing
./scripts/pre-commit.sh

# Or add to your shell profile
echo 'alias precommit="./scripts/pre-commit.sh"' >> ~/.zshrc
```

### Go Code Quality (Backend)

**Required Tools:**
- `golangci-lint` - Install with: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

**Run Individual Checks:**
```bash
cd server

# Format code
go fmt ./...

# Check for issues
go vet ./...

# Run comprehensive linting
golangci-lint run

# Auto-fix issues (if supported by linter)
golangci-lint run --fix
```

**Configuration:**
- See `.golangci.yml` for linter settings
- Checks include: govet, errcheck, staticcheck, unused, ineffassign, gocritic, revive

### React/Frontend Code Quality

**Run Individual Checks:**
```bash
cd ui

# Lint code
npm run lint

# Fix auto-fixable issues
npm run lint -- --fix

# Format code
npm run format
```

**Configuration:**
- ESLint configuration in `ui/.eslintrc.json`
- Checks include: React best practices, unused variables, import ordering

### Pre-commit Check Details

The `make pre-commit` target runs:
1. **Go Modules**: Verifies dependencies are tidy and verified
2. **Backend Linting**: Runs go vet and golangci-lint
3. **Frontend Linting**: Runs ESLint
4. **Basic Tests**: Runs unit tests (non-blocking if they fail)
5. **Common Issues**: Checks for TODO/FIXME comments, debug prints

The `scripts/pre-commit.sh` script provides more detailed output and handles missing dependencies gracefully.

### CI Pipeline Alignment

Local checks mirror the CI pipeline stages:

| CI Job | Local Equivalent | Command |
|--------|------------------|---------|
| Backend Tests | `make test` | `make test` |
| Frontend Tests | UI test suite | `make ui-test` |
| Backend Lint | Go linting | `make lint` (Go part) |
| Frontend Lint | ESLint | `make lint` (UI part) |
| Build | Binary build | `make build` |

### Development Workflow

**Before committing code:**
```bash
# 1. Auto-fix lintable issues
make lint-fix

# 2. Run all checks
make pre-commit

# 3. Run tests (if not included in pre-commit)
make test
```

**Before pushing to remote:**
```bash
# Run full CI pipeline locally
make ci

# Or if you have pre-commit hooks enabled
git add .
git commit -m "Your commit message"
# Pre-commit hooks will run automatically
git push origin your-branch
```

### Troubleshooting

**golangci-lint not found:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

**Node.js dependencies missing:**
```bash
cd ui
npm install
```

**Permission denied on pre-commit script:**
```bash
chmod +x scripts/pre-commit.sh
```

**Pre-commit hooks not running:**
```bash
# Check if hooks are installed
cat .git/hooks/pre-commit

# Reinstall hooks
pre-commit install
```

### Code Style Standards

**Go:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use gofmt for formatting
- Add comments for exported functions
- Handle errors explicitly
- Run `go mod tidy` after adding dependencies

**JavaScript/React:**
- Use functional components with hooks
- Follow React best practices
- Use meaningful component/variable names
- Keep components focused and reusable
- Test user interactions



## Local Testing

SupaControl uses comprehensive testing across all components.

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run CI checks (tests + lints + build)
make ci
```

### Backend Tests

```bash
cd server

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Run with race detection
go test -race ./...

# Run specific package
go test ./internal/auth

# Run specific test
go test -run TestHashPassword ./internal/auth
```

### Frontend Tests

```bash
cd ui

# Run tests
npm test

# Run with coverage
npm run test:coverage

# Run in watch mode
npm test -- --watch

# Run with UI
npm run test:ui

# View coverage report
open coverage/index.html
```

### Test Coverage

Maintaining high test coverage is crucial for project stability and quality. We aim for a high level of coverage across all components.

**Priority areas for testing:**
1. API handlers
2. Database operations
3. Kubernetes controller logic
4. React components and user flows
5. Authentication and authorization logic

For detailed information on running tests and our testing strategy, see the [**Comprehensive Testing Guide**](../TESTING.md).

### Writing Tests

**Backend (Table-driven tests):**
```go
func TestValidateInstanceName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid name", "my-app", false},
        {"empty name", "", true},
        {"too long", strings.Repeat("a", 100), true},
        {"uppercase", "MyApp", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInstanceName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Frontend (Component tests):**
```javascript
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Dashboard from './Dashboard';

test('creates instance on form submit', async () => {
    const user = userEvent.setup();
    render(<Dashboard />);

    await user.type(screen.getByLabelText('Instance Name'), 'test-app');
    await user.click(screen.getByText('Create'));

    await waitFor(() => {
        expect(screen.getByText('test-app')).toBeInTheDocument();
    });
});
```

---

**Related Documentation:**
- [Testing Guide](../TESTING.md)
- [Contributing Guide](../CONTRIBUTING.md)
- [Architecture Documentation](../ARCHITECTURE.md)
- [Architecture Decision Records](../docs/adr) - Key architectural decisions and their rationale.
- [CLAUDE.md](../CLAUDE.md) - AI assistant development guide

**Last Updated: November 2025**
