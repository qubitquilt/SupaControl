# Contributing to SupaControl

Thank you for your interest in contributing to SupaControl! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Submitting Changes](#submitting-changes)
- [Coding Standards](#coding-standards)
- [Testing](#testing)

## Code of Conduct

By participating in this project, you agree to:
- Be respectful and inclusive
- Accept constructive criticism gracefully
- Focus on what's best for the community
- Show empathy towards other community members

## Getting Started

1. **Fork the Repository**
   ```bash
   # Click the "Fork" button on GitHub
   git clone https://github.com/YOUR-USERNAME/SupaControl.git
   cd SupaControl
   ```

2. **Add Upstream Remote**
   ```bash
   git remote add upstream https://github.com/qubitquilt/SupaControl.git
   ```

3. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

## Development Setup

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Docker (optional)
- Kubernetes cluster (for integration testing)

### Backend Setup

```bash
cd server

# Install dependencies
go mod download

# Set up environment
cp ../.env.example ../.env
# Edit .env with your values

# Run the server
go run main.go
```

### Frontend Setup

```bash
cd ui

# Install dependencies
npm install

# Start development server
npm run dev
```

### Database Setup

```bash
# Start PostgreSQL with Docker
docker run --name supacontrol-postgres \
  -e POSTGRES_USER=supacontrol \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=supacontrol \
  -p 5432:5432 \
  -d postgres:14

# Migrations run automatically on server startup
```

## Making Changes

### Backend Changes

1. **Code Structure**
   - API handlers go in `server/api/`
   - Business logic in `server/internal/`
   - Shared types in `pkg/api-types/`

2. **Adding New Endpoints**
   ```go
   // Add to server/api/handlers.go
   func (h *Handler) YourNewHandler(c echo.Context) error {
       // Implementation
   }

   // Register in server/api/router.go
   api.GET("/your-endpoint", handler.YourNewHandler)
   ```

3. **Database Changes**
   - Add migration in `server/internal/db/migrations/`
   - Number sequentially: `003_your_change.sql`
   - Update repository functions in `server/internal/db/`

### Frontend Changes

1. **Component Structure**
   - Pages in `ui/src/pages/`
   - Shared components in `ui/src/components/`
   - API calls in `ui/src/api.js`

2. **Styling**
   - Use CSS modules or component-scoped CSS
   - Follow existing patterns
   - Use CSS variables from `App.css`

### Testing Your Changes

```bash
# Backend tests
cd server
go test ./...

# Frontend tests
cd ui
npm test

# Build test
make docker-build
```

## Submitting Changes

1. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

   Use conventional commit messages:
   - `feat:` New feature
   - `fix:` Bug fix
   - `docs:` Documentation only
   - `style:` Code style changes
   - `refactor:` Code refactoring
   - `test:` Adding tests
   - `chore:` Maintenance tasks

2. **Keep Your Fork Updated**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

3. **Push to Your Fork**
   ```bash
   git push origin feature/your-feature-name
   ```

4. **Create Pull Request**
   - Go to GitHub and create a PR
   - Describe your changes clearly
   - Link any related issues
   - Ensure CI passes

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `go vet` to catch common mistakes
- Add comments for exported functions
- Handle errors explicitly

```go
// Good
func CreateInstance(name string) (*Instance, error) {
    if name == "" {
        return nil, fmt.Errorf("name is required")
    }
    // ...
}

// Bad
func createInstance(name string) *Instance {
    // No error handling
}
```

### JavaScript/React Code

- Use functional components with hooks
- Follow React best practices
- Use meaningful variable names
- Add PropTypes or TypeScript types
- Keep components focused and small

```jsx
// Good
function InstanceCard({ instance, onDelete }) {
    return (
        <div className="instance-card">
            {/* Component JSX */}
        </div>
    );
}

// Bad - Too many responsibilities
function Dashboard() {
    // Hundreds of lines mixing logic and UI
}
```

### Database

- Use migrations for schema changes
- Never modify existing migrations
- Write both up and down migrations
- Index foreign keys and frequently queried columns

## Testing

### Unit Tests

```go
// server/internal/auth/auth_test.go
func TestHashPassword(t *testing.T) {
    service := NewService("test-secret")
    hash, err := service.HashPassword("password123")

    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }

    if hash == "" {
        t.Error("Hash should not be empty")
    }
}
```

### Integration Tests

Test API endpoints with real database:

```go
func TestCreateInstance(t *testing.T) {
    // Setup test database
    // Create test request
    // Assert response
}
```

## Pull Request Checklist

Before submitting your PR, ensure:

- [ ] Code follows project style guidelines
- [ ] All tests pass
- [ ] New features have tests
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] No unnecessary dependencies added
- [ ] PR description explains the changes

## Questions?

If you have questions:
- Open an issue for discussion
- Check existing issues and PRs
- Read the README thoroughly

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to SupaControl! 🎉
