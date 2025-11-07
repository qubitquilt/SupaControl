# Testing Guide

This document provides comprehensive information about testing SupaControl.

## Overview

SupaControl uses a multi-layered testing strategy:

- **Unit Tests**: Test individual functions and components in isolation
- **Integration Tests**: Test interactions between components
- **Coverage Reports**: Track code coverage across the codebase
- **CI/CD**: Automated testing on every push and pull request

## Test Coverage Goals

We aim for the following coverage targets:

- **Backend (Go)**: 70%+ coverage
- **Frontend (React)**: 70%+ coverage
- **Critical Paths**: 90%+ coverage (auth, orchestration, API)

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

## Backend Testing (Go)

### Structure

```
server/
├── internal/
│   ├── auth/
│   │   ├── auth.go
│   │   └── auth_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── k8s/
│       ├── k8s.go
│       └── k8s_test.go
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

#### Unit Tests

Test individual functions:
- Authentication functions (hashing, JWT generation)
- Secret generation
- Configuration loading
- API request/response handling

#### Integration Tests

Test component interactions (TODO):
- Database operations
- API endpoint flows
- Kubernetes client operations

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

### Structure

```
ui/
├── src/
│   ├── api.js
│   ├── api.test.js
│   ├── components/
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
   ```

2. **Update dependencies**:
   ```bash
   cd server && go mod download
   cd ui && npm install
   ```

3. **Check environment**:
   ```bash
   go version  # Should be 1.21+
   node --version  # Should be 18+
   ```

### CI Failures

1. **Check logs** in GitHub Actions
2. **Reproduce locally** with `make ci`
3. **Verify Go/Node versions** match CI
4. **Check for flaky tests**

### Low Coverage

1. **Identify uncovered lines**: View coverage report
2. **Add tests** for critical paths first
3. **Focus on business logic**, not boilerplate
4. **Use coverage tools** to find gaps

## Future Improvements

- [ ] E2E tests with Playwright
- [ ] API integration tests
- [ ] Kubernetes integration tests (kind)
- [ ] Performance/load tests
- [ ] Security tests (SAST/DAST)
- [ ] Contract tests for API
- [ ] Mutation testing

## Resources

- [Go Testing](https://golang.org/pkg/testing/)
- [Vitest Documentation](https://vitest.dev/)
- [React Testing Library](https://testing-library.com/react)
- [Testing Best Practices](https://testingjavascript.com/)
- [Codecov Documentation](https://docs.codecov.com/)

---

**Remember**: Good tests are the foundation of maintainable code!
