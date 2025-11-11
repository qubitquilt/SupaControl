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

- **Go** 1.21+
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

# Or merge multiple configs
export KUBECONFIG=$HOME/.kube/config:$HOME/.kube/config-dev
```

**For in-cluster development:**
- If running SupaControl inside a Kubernetes pod, KUBECONFIG is not needed
- The application will automatically use in-cluster service account credentials
- See [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment patterns

**Troubleshooting:**
- If you see "unable to connect to Kubernetes" errors, check your KUBECONFIG
- Verify your cluster is accessible: `kubectl get nodes`
- Ensure proper RBAC permissions (see [charts/supacontrol/templates/rbac.yaml](../charts/supacontrol/templates/rbac.yaml))

**Backend Code Guidelines:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html) conventions
- Use `gofmt` for formatting
- Run `go vet` before committing
- Add tests for new features
- Handle errors explicitly
- Add comments for exported functions

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

## Code Style

**Go:**
```bash
# Format code
gofmt -w .

# Lint code
go vet ./...

# Run with golangci-lint (recommended)
golangci-lint run
```

**JavaScript/React:**
```bash
cd ui

# Lint code
npm run lint

# Fix auto-fixable issues
npm run lint -- --fix
```

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

**Current Status (November 2024):**
- Backend: 6.3% coverage (goal: 70%+)
- Frontend: 5.87% coverage (goal: 70%+)
- Critical paths need improvement

**Priority areas for testing:**
1. API handlers
2. Database operations
3. Kubernetes orchestration
4. React components
5. Authentication flows

See [TESTING.md](../TESTING.md) for comprehensive testing documentation.

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
- [CLAUDE.md](../CLAUDE.md) - AI assistant development guide

**Last Updated: November 2025**
