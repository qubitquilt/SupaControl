# Testing Guide

This document provides comprehensive information about testing SupaControl.

## Current Status ⚠️

**IMPORTANT**: As of November 2024, SupaControl has low test coverage:
- **Server (Go)**: 6.3% coverage
- **UI (React)**: 5.87% coverage
- **CLI (TypeScript)**: Limited utility testing, component tests needed

The 70% coverage goals below are aspirational and not yet achieved.

## Overview

SupaControl uses a multi-layered testing strategy:

- **Unit Tests**: Test individual functions and components in isolation
- **Integration Tests**: Test interactions between components
- **Coverage Reports**: Track code coverage across the codebase
- **CI/CD**: Automated testing on every push and pull request

## Test Coverage Goals

**Current Reality (November 2024)**:
- **Server (Go)**: 6.3% coverage (goal: 70%+)
- **Frontend (React)**: 5.87% coverage (goal: 70%+)
- **CLI (TypeScript)**: Utility functions tested, components need coverage

**Target Goals** (to be achieved):
- **Backend (Go)**: 70%+ coverage
- **Frontend (React)**: 70%+ coverage
- **Critical Paths**: 90%+ coverage (auth, orchestration, API)

**Priority Areas for Testing**:
1. API handlers (currently 0% coverage)
2. React UI components (currently 0% coverage)
3. Database operations (currently 0% coverage)
4. Kubernetes controller (currently 0% coverage)

## Running Tests

### Quick Start

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run CI checks (tests, lints, build)
make ci
```

### Individual Component Testing

```bash
# Server tests only
cd server && go test -v ./...

# UI tests only
cd ui && npm test

# CLI tests only (currently failing due to implementation mismatches)
cd cli && npm test
```

## Backend Testing (Go)

### Current Test Coverage

```
server/internal/auth     - 74.5% coverage ✅
server/internal/config   - 55.0% coverage ⚠️
server/internal/k8s      - 2.9% coverage ❌
server/api              - 0.0% coverage ❌
server/controllers      - 0.0% coverage ❌
server/internal/db      - 0.0% coverage ❌
```

### Structure

```
server/
├── internal/
│   ├── auth/
│   │   ├── auth.go         (tested ✅)
│   │   └── auth_test.go
│   ├── config/
│   │   ├── config.go       (tested ⚠️)
│   │   └── config_test.go
│   ├── db/
│   │   ├── db.go          (untested ❌)
│   │   ├── api_keys.go    (untested ❌)
│   │   └── instances.go   (untested ❌)
│   └── k8s/
│       ├── k8s.go         (minimal testing ❌)
│       ├── crclient.go    (untested ❌)
│       ├── orchestrator.go (untested ❌)
│       └── k8s_test.go
├── api/
│   ├── handlers.go        (untested ❌)
│   ├── router.go          (untested ❌)
│   └── middleware.go      (untested ❌)
├── controllers/
│   └── supabaseinstance_controller.go (untested ❌)
└── main.go                (untested ❌)
```

### Running Backend Tests

```bash
cd server

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test ./internal/auth/...

# Run specific test
go test -run TestHashPassword ./internal/auth/
```

### Writing Backend Tests

Example test structure:

```go
package mypackage

import (
    "testing"
)

func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "expected",
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Categories

#### ✅ Implemented Unit Tests

- Authentication functions (hashing, JWT generation) - 74.5% coverage
- Secret generation utilities - 2.9% coverage
- Configuration loading - 55% coverage

#### ❌ Missing Unit Tests

- API request/response handling
- Database operations
- Kubernetes client operations
- Controller reconciliation logic

#### 🚧 Integration Tests (TODO)

- Database operations
- API endpoint flows
- Kubernetes client operations
- End-to-end user workflows

### Mocking

For external dependencies:

```go
// Mock Kubernetes client
type MockK8sClient struct {
    CreateNamespaceFunc func(ctx context.Context, name string) error
}

func (m *MockK8sClient) CreateNamespace(ctx context.Context, name string) error {
    if m.CreateNamespaceFunc != nil {
        return m.CreateNamespaceFunc(ctx, name)
    }
    return nil
}
```

## Frontend Testing (React)

### Current Test Coverage

```
ui/src/api.js           - 78% coverage ✅
ui/src/pages/Dashboard.jsx - 0% coverage ❌
ui/src/pages/Login.jsx  - 0% coverage ❌
ui/src/pages/Settings.jsx - 0% coverage ❌
ui/src/App.jsx          - 0% coverage ❌
ui/src/main.jsx         - 0% coverage ❌
```

### Structure

```
ui/
├── src/
│   ├── api.js          (tested ✅)
│   ├── api.test.js
│   ├── App.jsx         (untested ❌)
│   ├── main.jsx        (untested ❌)
│   ├── pages/
│   │   ├── Dashboard.jsx (untested ❌)
│   │   ├── Login.jsx   (untested ❌)
│   │   └── Settings.jsx (untested ❌)
│   └── test/
│       └── setup.js
└── vitest.config.js
```

### Running Frontend Tests

```bash
cd ui

# Install dependencies
npm install

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Run tests in watch mode
npm test -- --watch

# Run tests with UI
npm run test:ui
```

### Writing Frontend Tests

Example component test:

```javascript
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import MyComponent from './MyComponent';

describe('MyComponent', () => {
  it('renders correctly', () => {
    render(<MyComponent title="Test" />);
    expect(screen.getByText('Test')).toBeInTheDocument();
  });

  it('handles click events', async () => {
    const { user } = render(<MyComponent />);
    const button = screen.getByRole('button');
    await user.click(button);
    expect(screen.getByText('Clicked')).toBeInTheDocument();
  });
});
```

### Testing Tools

- **Vitest**: Fast unit test framework
- **React Testing Library**: Test React components
- **@testing-library/user-event**: Simulate user interactions
- **jsdom**: DOM environment for tests

### Test Coverage

Coverage reports include:
- Line coverage
- Branch coverage
- Function coverage
- Statement coverage

View coverage:
```bash
npm run test:coverage
open coverage/index.html
```

## CLI Testing (TypeScript/React)

### Current Status

**Utilities (Tested ✅)**:
- `cli/src/utils/helm.test.ts` - 16 tests (3 failing due to implementation mismatches)
- `cli/src/utils/prerequisites.test.ts` - 10 tests
- `cli/src/utils/secrets.test.ts` - 21 tests

**Components (Untested ❌)**:
- `cli/src/components/ConfigurationWizard.tsx` - 0% coverage
- `cli/src/components/Installation.tsx` - 0% coverage
- `cli/src/components/PrerequisitesCheck.tsx` - 0% coverage

### Structure

```
cli/
├── src/
│   ├── cli.tsx          (excluded from coverage)
│   ├── components/
│   │   ├── ConfigurationWizard.tsx (untested ❌)
│   │   ├── Installation.tsx (untested ❌)
│   │   ├── PrerequisitesCheck.tsx (untested ❌)
│   │   ├── Welcome.tsx   (excluded from coverage)
│   │   └── Complete.tsx  (excluded from coverage)
│   └── utils/
│       ├── helm.test.ts  (tested ⚠️)
│       ├── helm.ts       (utility functions)
│       ├── prerequisites.test.ts (tested ✅)
│       ├── prerequisites.ts (utility functions)
│       ├── secrets.test.ts (tested ✅)
│       └── secrets.ts    (utility functions)
└── vitest.config.ts
```

### Running CLI Tests

```bash
cd cli

# Install dependencies
npm install

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Run tests in watch mode
npm test -- --watch
```

## Continuous Integration

### GitHub Actions

Our CI pipeline (`.github/workflows/ci.yml`) runs on:
- Push to `main` or `develop`
- Pull requests to `main` or `develop`

#### Pipeline Steps

1. **Backend Tests**
   - Run Go tests with race detection
   - Generate coverage report
   - Upload to Codecov

2. **Frontend Tests**
   - Run Vitest tests
   - Generate coverage report
   - Upload to Codecov

3. **Linting**
   - Go: `go vet` and `golangci-lint`
   - React: ESLint

4. **Build**
   - Build backend binary
   - Build frontend assets
   - Upload artifacts

### Coverage Reporting

Coverage reports are automatically uploaded to [Codecov](https://codecov.io) and include:

- Overall project coverage
- Per-file coverage
- Coverage diff for PRs
- Historical trends

View coverage: `https://codecov.io/gh/qubitquilt/SupaControl`

## Best Practices

### General

1. **Write tests first** (TDD) when possible
2. **Test behavior, not implementation**
3. **Keep tests simple and focused**
4. **Use descriptive test names**
5. **Don't test external libraries**
6. **Mock external dependencies**

### Backend (Go)

1. **Use table-driven tests** for multiple scenarios
2. **Test error cases** thoroughly
3. **Use `t.Parallel()`** for independent tests
4. **Clean up resources** in tests
5. **Use subtests** with `t.Run()`

### Frontend (React)

1. **Test user interactions**, not implementation details
2. **Use semantic queries** (`getByRole`, `getByLabelText`)
3. **Avoid testing styles** directly
4. **Test async behavior** properly with `waitFor`
5. **Mock API calls** consistently

### CLI (TypeScript/React)

1. **Test component logic**, not UI rendering
2. **Mock external commands** (helm, kubectl, git)
3. **Test error handling** and edge cases
4. **Use realistic test data** and scenarios

## Coverage Reports

### Viewing Coverage

#### Backend

```bash
cd server
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### Frontend

```bash
cd ui
npm run test:coverage
open coverage/index.html
```

#### CLI

```bash
cd cli
npm run test:coverage
```

#### Combined

```bash
make test-coverage
open coverage/coverage.html
```

### Coverage Badges

Add to your README:

```markdown
[![codecov](https://codecov.io/gh/qubitquilt/SupaControl/branch/main/graph/badge.svg)](https://codecov.io/gh/qubitquilt/SupaControl)
```

## Troubleshooting

### Tests Failing Locally

1. **Clear caches**:
   ```bash
   go clean -testcache
   rm -rf ui/node_modules
   rm -rf cli/node_modules
   ```

2. **Update dependencies**:
   ```bash
   cd server && go mod download
   cd ui && npm install
   cd cli && npm install
   ```

3. **Check environment**:
   ```bash
   go version  # Should be 1.21+
   node --version  # Should be 18+
   ```

### CLI Test Issues

**Current CLI Test Failures**:
- `helm.test.ts` tests failing due to implementation mismatches
- Tests expect different execa call signatures
- Need to update test expectations to match current helm utility implementation

**Solution**: Update test expectations in `cli/src/utils/helm.test.ts`

### CI Failures

1. **Check logs** in GitHub Actions
2. **Reproduce locally** with `make ci`
3. **Verify Go/Node versions** match CI
4. **Check for flaky tests**

### Low Coverage

1. **Current status**: 5.87-6.3% coverage across components
2. **Focus on critical business logic first**:
   - API handlers and endpoints
   - React UI components (Dashboard, Login, Settings)
   - Database operations
   - Kubernetes controller logic
3. **Use coverage tools** to find gaps
4. **Prioritize user-facing functionality**

## Immediate Action Items

### Phase 1: Fix Current Issues (1-2 days)
- [ ] Fix CLI test failures in `helm.test.ts`
- [ ] Update coverage expectations to match current reality
- [ ] Add basic API handler tests

### Phase 2: Critical Path Coverage (1-2 weeks)
- [ ] Add React component tests (Dashboard, Login, Settings)
- [ ] Add API endpoint tests with authentication
- [ ] Add database operation tests
- [ ] Add Kubernetes controller tests

### Phase 3: Integration & E2E (2-3 weeks)
- [ ] End-to-end user workflow tests
- [ ] Integration tests between components
- [ ] Performance and load testing
- [ ] Security testing

## Future Improvements

- [ ] E2E tests with Playwright
- [ ] API integration tests
- [ ] Kubernetes integration tests (kind)
- [ ] Performance/load tests
- [ ] Security tests (SAST/DAST)
- [ ] Contract tests for API
- [ ] Mutation testing
- [ ] Visual regression testing for UI

## Resources

- [Go Testing](https://golang.org/pkg/testing/)
- [Vitest Documentation](https://vitest.dev/)
- [React Testing Library](https://testing-library.com/react)
- [Testing Best Practices](https://testingjavascript.com/)
- [Codecov Documentation](https://docs.codecov.com/)

---

**Current Reality Check**: The project needs significant testing investment. The current 5.87% coverage indicates most business logic and user-facing functionality lacks proper test coverage. Focus on testing the most critical user journeys first.

**Remember**: Good tests are the foundation of maintainable code!
