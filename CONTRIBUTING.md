# Contributing to SupaControl

Thank you for your interest in contributing to SupaControl! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing Guidelines](#testing-guidelines)
- [Code Style](#code-style)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Guidelines](#issue-guidelines)
- [Community](#community)

## Code of Conduct

By participating in this project, you agree to abide by our code of conduct:

### Our Pledge

We are committed to providing a welcoming and inclusive experience for everyone, regardless of:
- Experience level
- Gender identity and expression
- Sexual orientation
- Disability
- Personal appearance
- Body size
- Race or ethnicity
- Age
- Religion or lack thereof
- Technology choices

### Our Standards

**Positive behaviors:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

**Unacceptable behaviors:**
- Trolling, insulting/derogatory comments, and personal attacks
- Public or private harassment
- Publishing others' private information without permission
- Other conduct which could reasonably be considered inappropriate

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by contacting the project maintainers. All complaints will be reviewed and investigated promptly and fairly.

## Getting Started

### Prerequisites for Contributing

Before you start contributing, ensure you have:

**Required:**
- Git installed and configured
- GitHub account
- Basic knowledge of Go and/or React (depending on contribution area)

**Recommended:**
- Go 1.24+ (for backend development)
- Node.js 18+ (for frontend/CLI development)
- Docker (for running PostgreSQL locally)
- kubectl and Helm (for Kubernetes testing)
- Access to a Kubernetes cluster (Minikube, kind, or cloud provider)

### Ways to Contribute

There are many ways to contribute to SupaControl:

1. **Code Contributions**
   - Fix bugs
   - Implement new features
   - Improve performance
   - Refactor code

2. **Documentation**
   - Fix typos and grammatical errors
   - Improve existing documentation
   - Write tutorials and guides
   - Add code examples

3. **Testing**
   - Write unit tests
   - Write integration tests
   - Perform manual testing
   - Report bugs

4. **Design**
   - Improve UI/UX
   - Create mockups for new features
   - Suggest design improvements

5. **Community**
   - Answer questions on GitHub Discussions
   - Help new contributors
   - Triage issues
   - Review pull requests

### Finding Issues to Work On

Looking for a good first issue? Check out:

- [`good first issue`](https://github.com/qubitquilt/SupaControl/labels/good%20first%20issue) - Good for newcomers
- [`help wanted`](https://github.com/qubitquilt/SupaControl/labels/help%20wanted) - Issues we need help with
- [`documentation`](https://github.com/qubitquilt/SupaControl/labels/documentation) - Documentation improvements

**Before starting work:**
1. Check if the issue is already assigned
2. Comment on the issue expressing your interest
3. Wait for maintainer approval (to avoid duplicate work)

## Development Workflow

### 1. Fork the Repository

Click the "Fork" button on GitHub to create your own copy of the repository.

```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/SupaControl.git
cd SupaControl

# Add upstream remote
git remote add upstream https://github.com/qubitquilt/SupaControl.git

# Verify remotes
git remote -v
```

### 2. Create a Branch

Create a new branch for your changes:

```bash
# Fetch latest changes from upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create and switch to a new branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/your-bug-fix
```

**Branch naming conventions:**
- `feature/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation changes
- `refactor/description` - Code refactoring
- `test/description` - Adding or updating tests
- `chore/description` - Maintenance tasks

### 3. Make Your Changes

Follow the guidelines in [Making Changes](#making-changes) section.

### 4. Test Your Changes

Run all tests before committing:

```bash
# Backend tests
cd server && go test ./...

# Frontend tests
cd ui && npm test

# Or run all tests
make test
```

### 5. Commit Your Changes

Follow [Commit Guidelines](#commit-guidelines) for commit messages.

```bash
git add .
git commit -m "feat: add amazing feature"
```

### 6. Push and Create PR

```bash
# Push to your fork
git push origin feature/your-feature-name

# Then create a Pull Request on GitHub
```

### 7. Address Review Feedback

- Respond to all review comments
- Make requested changes
- Push additional commits to the same branch
- Re-request review when ready

## Development Setup

### Backend Setup (Go)

```bash
cd server

# Install dependencies
go mod download

# Set up environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=supacontrol
export DB_PASSWORD=password
export DB_NAME=supacontrol
export JWT_SECRET=your-dev-jwt-secret-at-least-32-chars

# Start PostgreSQL with Docker
docker run --name supacontrol-postgres \
  -e POSTGRES_USER=supacontrol \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=supacontrol \
  -p 5432:5432 \
  -d postgres:14

# Run the server
go run main.go

# Server runs on http://localhost:8091
```

**Useful Commands:**

```bash
# Format code
gofmt -w .

# Lint code
go vet ./...

# Run with race detection
go run -race main.go

# Build binary
go build -o supacontrol

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend Setup (React)

```bash
cd ui

# Install dependencies
npm install

# Start development server
npm run dev

# Development server runs on http://localhost:5173
```

**Useful Commands:**

```bash
# Run tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm run test:coverage

# Lint code
npm run lint

# Build for production
npm run build
```

### CLI Installer Setup (TypeScript)

```bash
cd cli

# Install dependencies
npm install

# Run installer
npm start

# Run tests
npm test

# Build
npm run build
```

### Full Stack Development

For full-stack development, run backend and frontend concurrently:

```bash
# Terminal 1: Backend
cd server && go run main.go

# Terminal 2: Frontend
cd ui && npm run dev
```

Frontend dev server proxies API calls to backend (configured in `ui/vite.config.js`).

## Making Changes

### Backend Changes (Go)

#### Code Structure

- **API handlers**: `server/api/handlers.go`
- **Business logic**: `server/internal/`
- **Database operations**: `server/internal/db/`
- **Kubernetes operations**: `server/internal/k8s/`
- **Shared types**: `pkg/api-types/`

#### Adding a New Endpoint

1. **Define request/response types** in `pkg/api-types/`:

```go
// pkg/api-types/instance.go
type UpdateInstanceRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}
```

2. **Add handler function** in `server/api/handlers.go`:

```go
func (h *Handler) UpdateInstance(c echo.Context) error {
    var req api_types.UpdateInstanceRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    // Business logic here

    return c.JSON(http.StatusOK, instance)
}
```

3. **Register route** in `server/api/router.go`:

```go
api.PUT("/instances/:name", handler.UpdateInstance)
```

4. **Write tests** in `server/api/handlers_test.go`:

```go
func TestUpdateInstance(t *testing.T) {
    // Test implementation
}
```

#### Adding Database Migrations

**Important**: Per ADR-001, instance state is stored in Kubernetes CRDs, NOT PostgreSQL. Only add database migrations for SupaControl's operational data (users, API keys, audit logs, etc.).

1. **Create migration file**: `server/internal/db/migrations/00X_description.sql`

```sql
-- Example: 004_add_audit_log_table.sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    timestamp TIMESTAMP DEFAULT NOW(),
    details JSONB
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
```

2. **Update repository**: Add functions in the appropriate file (e.g., `server/internal/db/audit_logs.go`)

```go
// server/internal/db/audit_logs.go
func (r *AuditLogRepository) CreateLog(ctx context.Context, log *AuditLog) error {
    query := `INSERT INTO audit_logs (user_id, action, resource_type, resource_id, details)
              VALUES ($1, $2, $3, $4, $5)`
    _, err := r.db.ExecContext(ctx, query, log.UserID, log.Action, log.ResourceType, log.ResourceID, log.Details)
    return err
}
```

3. **Test migration**: Restart server to auto-apply, or test manually:

```bash
psql -h localhost -U supacontrol -d supacontrol
# Verify schema changes
\d audit_logs
```

### Frontend Changes (React)

#### Code Structure

- **Pages**: `ui/src/pages/` - Route components
- **Components**: `ui/src/components/` - Shared components (create if needed)
- **API client**: `ui/src/api.js` - API functions
- **App**: `ui/src/App.jsx` - Main app component
- **Styles**: Component-scoped CSS or `App.css`

#### Adding a New Feature

1. **Add API function** in `ui/src/api.js`:

```javascript
export async function updateInstance(name, description) {
    const response = await fetch(`${API_URL}/instances/${name}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${getToken()}`
        },
        body: JSON.stringify({ name, description })
    });

    if (!response.ok) {
        throw new Error('Failed to update instance');
    }

    return response.json();
}
```

2. **Update component** in `ui/src/pages/Dashboard.jsx`:

```javascript
const handleUpdate = async (name, description) => {
    try {
        await updateInstance(name, description);
        // Refresh instance list
        loadInstances();
    } catch (error) {
        console.error('Update failed:', error);
        setError('Failed to update instance');
    }
};
```

3. **Write tests** in `ui/src/pages/Dashboard.test.jsx`:

```javascript
test('updates instance description', async () => {
    const user = userEvent.setup();
    render(<Dashboard />);

    // Test implementation
});
```

#### Testing Your Changes Locally

```bash
# Run all tests (backend and frontend)
make test

# Testing Backend Changes
cd server
...
# Testing Frontend Changes
cd ui
...
```

#### Integration Testing

```bash
# 1. Start backend
cd server && go run main.go

# 2. Start frontend
cd ui && npm run dev

# 3. Test complete flow:
#    - Login
#    - Create instance
#    - View instance
#    - Delete instance
```

## Testing Guidelines

### Test Requirements

**All contributions must include tests:**

- **New features**: Unit tests + integration tests (if applicable)
- **Bug fixes**: Test that reproduces the bug + fix
- **Refactoring**: Existing tests must still pass
- **Documentation**: No tests required

### Writing Good Tests

#### Backend Tests (Go)

Use **table-driven tests** for multiple scenarios:

```go
func TestValidateInstanceName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:    "valid name",
            input:   "my-app",
            wantErr: false,
        },
        {
            name:    "empty name",
            input:   "",
            wantErr: true,
        },
        {
            name:    "too long",
            input:   strings.Repeat("a", 100),
            wantErr: true,
        },
        {
            name:    "uppercase letters",
            input:   "MyApp",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInstanceName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateInstanceName() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Best Practices:**
- Test both success and error cases
- Use descriptive test names
- Use `t.Run()` for subtests
- Clean up resources (defer cleanup)
- Don't test external libraries

#### Frontend Tests (React)

Test **user interactions**, not implementation:

```javascript
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Dashboard from './Dashboard';

test('creates instance when form is submitted', async () => {
    const user = userEvent.setup();
    render(<Dashboard />);

    // User types instance name
    const input = screen.getByLabelText(/instance name/i);
    await user.type(input, 'test-app');

    // User clicks create button
    const button = screen.getByRole('button', { name: /create/i });
    await user.click(button);

    // Instance appears in list
    await waitFor(() => {
        expect(screen.getByText('test-app')).toBeInTheDocument();
    });
});
```

**Best Practices:**
- Use semantic queries (`getByRole`, `getByLabelText`)
- Simulate real user interactions
- Use `waitFor` for async operations
- Mock API calls
- Don't test implementation details

### Test Coverage Goals

We aim for a high level of test coverage across all components to ensure stability and quality.

**Priority areas for improvement:**
1. API handlers
2. Database operations
3. React components
4. Authentication flows

See [TESTING.md](TESTING.md) for comprehensive testing documentation.

## Code Style

### Go Code Style

Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines.

**Formatting:**
```bash
# Format all Go files
gofmt -w .

# Or use goimports (includes gofmt)
goimports -w .
```

**Linting:**
```bash
# Run go vet
go vet ./...

# Run golangci-lint (recommended)
golangci-lint run
```

**Naming Conventions:**
- Exported functions: `PascalCase`
- Unexported functions: `camelCase`
- Interfaces: `InterfaceName` or `InterfaceNameer`
- Constants: `PascalCase` or `SCREAMING_SNAKE_CASE`

**Example:**
```go
// Good
package auth

// Service provides authentication functionality
type Service struct {
    jwtSecret string
}

// GenerateToken creates a new JWT token
func (s *Service) GenerateToken(username string) (string, error) {
    // Implementation
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
    // Implementation
}

// Bad
package auth

type auth_service struct {  // unexported, snake_case
    JWT_SECRET string       // exported, screaming case
}

func generatetoken(u string) string {  // no error handling, poor name
    // Implementation
}
```

**Error Handling:**
```go
// Good - explicit error handling
func CreateInstance(name string) (*Instance, error) {
    if name == "" {
        return nil, fmt.Errorf("name is required")
    }

    instance, err := createInKubernetes(name)
    if err != nil {
        return nil, fmt.Errorf("failed to create instance: %w", err)
    }

    return instance, nil
}

// Bad - ignoring errors
func CreateInstance(name string) *Instance {
    instance, _ := createInKubernetes(name)  // Don't ignore errors!
    return instance
}
```

### JavaScript/React Code Style

**Formatting:**
```bash
# Lint code
npm run lint

# Auto-fix issues
npm run lint -- --fix
```

**Naming Conventions:**
- Components: `PascalCase`
- Functions: `camelCase`
- Constants: `SCREAMING_SNAKE_CASE`
- Files: Match component name

**Component Structure:**
```javascript
// Good - functional component with hooks
import { useState, useEffect } from 'react';
import { listInstances } from '../api';

function Dashboard() {
    const [instances, setInstances] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        loadInstances();
    }, []);

    const loadInstances = async () => {
        try {
            setLoading(true);
            const data = await listInstances();
            setInstances(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

    return (
        <div className="dashboard">
            {instances.map(instance => (
                <div key={instance.id}>{instance.name}</div>
            ))}
        </div>
    );
}

export default Dashboard;
```

**Best Practices:**
- Use functional components with hooks (not class components)
- Keep components small and focused
- Extract reusable logic into custom hooks
- Use meaningful variable names
- Handle loading and error states
- Use PropTypes or TypeScript for type safety

## Commit Guidelines

### Conventional Commits

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

**Format:**
```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements
- `ci`: CI/CD changes

**Scope** (optional):
- `api`: API changes
- `ui`: Frontend changes
- `cli`: CLI installer changes
- `db`: Database changes
- `k8s`: Kubernetes orchestration
- `auth`: Authentication
- `docs`: Documentation

**Examples:**

```bash
# Feature
git commit -m "feat(api): add instance update endpoint"

# Bug fix
git commit -m "fix(ui): correct instance status display"

# Documentation
git commit -m "docs: update installation instructions"

# Breaking change
git commit -m "feat(api)!: change instance creation response format

BREAKING CHANGE: Instance creation now returns full object instead of just ID"
```

### Commit Best Practices

1. **Write clear, descriptive commit messages**
   - Use imperative mood ("add feature" not "added feature")
   - First line: 50 chars or less
   - Body: Wrap at 72 chars

2. **Make atomic commits**
   - One logical change per commit
   - Commit often
   - Each commit should build successfully

3. **Don't commit:**
   - Build artifacts (dist/, coverage/, etc.)
   - Dependencies (node_modules/, vendor/)
   - IDE files (.vscode/, .idea/)
   - Secrets or credentials
   - Large binary files

4. **Use .gitignore**
   - Keep .gitignore up to date
   - Don't commit files that should be ignored

## Pull Request Process

### Before Submitting

1. **Ensure all tests pass:**
   ```bash
   make test
   ```

2. **Update documentation:**
   - README.md (if user-facing changes)
   - Code comments
   - API documentation
   - CHANGELOG.md (for significant changes)

3. **Verify code quality:**
   ```bash
   # Backend
   gofmt -w .
   go vet ./...

   # Frontend
   npm run lint
   ```

4. **Rebase on latest main:**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

5. **Squash commits if needed:**
   ```bash
   git rebase -i HEAD~N  # N = number of commits to squash
   ```

### Submitting a Pull Request

1. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create PR on GitHub:**
   - Use a clear, descriptive title
   - Fill out the PR template completely
   - Link related issues
   - Add screenshots for UI changes
   - Mark as draft if work-in-progress

3. **PR Title Format:**
   ```
   <type>(<scope>): <description>
   ```

   Examples:
   - `feat(api): add instance update endpoint`
   - `fix(ui): correct dashboard layout on mobile`
   - `docs: add troubleshooting guide`

### During Review

1. **Respond to feedback:**
   - Address all review comments
   - Ask questions if unclear
   - Make requested changes
   - Push additional commits

2. **Re-request review:**
   - After addressing feedback
   - Mark conversations as resolved

3. **Keep PR updated:**
   - Rebase on main if conflicts arise
   - Keep CI passing

### After Approval

1. **Maintainer will merge:**
   - PRs are typically squash-merged
   - Commit message uses PR title

2. **Delete your branch:**
   ```bash
   git branch -d feature/your-feature-name
   git push origin --delete feature/your-feature-name
   ```

3. **Update your fork:**
   ```bash
   git checkout main
   git pull upstream main
   git push origin main
   ```

## Issue Guidelines

### Reporting Bugs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md).

**Good bug reports include:**
- Clear description
- Steps to reproduce
- Expected vs. actual behavior
- Environment details (versions, OS, etc.)
- Logs and screenshots
- Minimal reproducible example

### Requesting Features

Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md).

**Good feature requests include:**
- Problem statement
- Proposed solution
- Use cases and benefits
- Implementation ideas
- Willingness to contribute

### Asking Questions

- Check [README.md](README.md) and documentation first
- Search existing issues
- Use [GitHub Discussions](https://github.com/qubitquilt/SupaControl/discussions) for questions
- Be specific and provide context

## Community

### Communication Channels

- **GitHub Issues**: Bug reports, feature requests
- **GitHub Discussions**: Questions, ideas, general discussion
- **Pull Requests**: Code review, technical discussion

### Getting Help

**Stuck? Here's how to get help:**

1. **Check documentation:**
   - [README.md](README.md)
   - [ARCHITECTURE.md](ARCHITECTURE.md)
   - [TESTING.md](TESTING.md)

2. **Search existing issues:**
   - Someone may have had the same problem
   - Solutions may already exist

3. **Ask on GitHub Discussions:**
   - Provide context and details
   - Include relevant code/logs
   - Be patient and respectful

4. **Open an issue:**
   - If you found a bug
   - If documentation is unclear
   - If you need a new feature

### Recognition

We value all contributions! Contributors are:

- Listed in GitHub's contributor graph
- Acknowledged in release notes (for significant contributions)
- Eligible for "Contributor" badge
- Part of our community

## Thank You!

Thank you for contributing to SupaControl! Every contribution, no matter how small, helps make the project better for everyone.

**Questions about contributing?**
Open an issue or start a discussion. We're here to help!

---

**Happy Coding!** 🚀

Made with ❤️ by the SupaControl community
