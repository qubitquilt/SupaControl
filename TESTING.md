# Testing Guide

This document provides information about the testing strategy, tools, and procedures for the SupaControl project.

## Current Status ⚠️

**IMPORTANT**: SupaControl currently has very low test coverage. The overall project coverage is approximately **6%**. This is a major quality gap that needs to be addressed.

- **Server (Go)**: ~6% coverage.
  - `auth` package is well-tested (~75%).
  - Critical components like the API handlers, database repositories, and the Kubernetes controller are largely untested (0-3% coverage).
- **UI (React)**: ~6% coverage.
  - The `api.js` client is well-tested (~78%).
  - No React components have any test coverage.
- **CLI (TypeScript)**: Utility functions are partially tested, but the interactive components are untested. Some existing tests for the `helm` utility are currently failing.

This document outlines the existing structure and the path forward for improving test coverage and quality.

## How to Run Tests

### Prerequisites

- **Go** (1.24+), **Node.js** (18+), and **npm**.
- For controller tests: Kubernetes `envtest` binaries. See [Controller Testing](#controller-testing) for setup.

### Key Commands

The `Makefile` provides convenient targets for running tests:

```bash
# Run all tests across the entire project
make test

# Run all tests and generate a combined coverage report
make test-coverage

# Run all CI checks locally (linting, testing, building)
make ci
```

## Backend Testing (Go)

The Go backend uses the standard `testing` package and `testify` for assertions.

### Running Backend Tests

```bash
cd server

# Run all backend tests
go test -v ./...

# Run tests with race detection (highly recommended)
go test -v -race ./...

# Generate a coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Priority Areas for Improvement

1.  **Controller Logic (`/server/controllers`)**: This is the highest priority. The reconciliation loop, job management, and status updates are critical and currently untested.
2.  **API Handlers (`/server/api`)**: Test request validation, authentication/authorization middleware, and correct interaction with the Kubernetes client.
3.  **Database Repositories (`/server/internal/db`)**: Test all CRUD operations for users and API keys.
4.  **Kubernetes Client (`/server/internal/k8s`)**: Test the wrappers around the `client-go` and `controller-runtime` clients.

### Controller Testing

The most critical and complex part of the backend is the Kubernetes controller. These tests are located in `server/controllers/` and use the `envtest` framework from `controller-runtime` to simulate a real Kubernetes API server.

**For detailed instructions on setting up the `envtest` environment and running these specific tests, please refer to the dedicated guide:**

➡️ **[server/controllers/README_TEST.md](./server/controllers/README_TEST.md)**

For guidance on writing effective tests, including examples of table-driven tests, please see the [Contributing Guide](./CONTRIBUTING.md#backend-tests-go).

## Frontend Testing (React)

The React UI uses `vitest` as the test runner and `@testing-library/react` for rendering components and simulating user interactions.

### Running Frontend Tests

```bash
cd ui

# Run all UI tests
npm test

# Run tests and generate a coverage report
npm run test:coverage

# Run tests with an interactive UI for debugging
npm run test:ui
```

### Priority Areas for Improvement

1.  **Page Components (`/ui/src/pages`)**:
    - `Dashboard.jsx`: Test instance listing, creation, and deletion flows.
    - `Login.jsx`: Test user authentication, input validation, and error handling.
    - `Settings.jsx`: Test API key management.
2.  **Authentication Flow**: Test the logic for handling JWT tokens, authenticated routes, and redirects.
3.  **Error Handling**: Ensure that API errors are gracefully handled and displayed to the user.

For guidance on writing effective component tests, see the [Contributing Guide](./CONTRIBUTING.md#frontend-tests-react).

## CLI Testing (TypeScript)

The interactive installer CLI also uses `vitest`.

### Running CLI Tests

```bash
cd cli

# Run all CLI tests
npm test
```

### Current Status & Priorities

- The utility functions for generating secrets and checking prerequisites are reasonably well-tested.
- **Component Gaps**: The interactive React Ink components (`ConfigurationWizard.tsx`, `Installation.tsx`, etc.) have no test coverage.

## Continuous Integration (CI)

The CI pipeline is defined in `.github/workflows/ci.yml` and runs automatically on every pull request and push to the `main` and `develop` branches.

The pipeline performs:
1.  **Linting**: `golangci-lint` for Go, `eslint` for React/TypeScript.
2.  **Testing**: Runs `make test` to execute all backend, frontend, and CLI tests.
3.  **Coverage Reporting**: Uploads coverage reports from all three components to Codecov.
4.  **Building**: Ensures all components build successfully.

A pull request cannot be merged unless all CI checks pass.