.PHONY: help build run test test-coverage clean docker-build docker-push install-deps ui-build

# Default target
help:
	@echo "SupaControl - Makefile commands"
	@echo ""
	@echo "  make build         - Build the Go backend"
	@echo "  make run           - Run the server locally"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make ui-build      - Build the React frontend"
	@echo "  make ui-dev        - Run UI development server"
	@echo "  make ui-test       - Run UI tests"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-push   - Push Docker image to registry"
	@echo "  make install-deps  - Install all dependencies"
	@echo "  make lint          - Run linters"
	@echo "  make format        - Format code"
	@echo "  make ci            - Run CI checks (tests, lints, build)"

# Build the backend
build:
	@echo "Building SupaControl server..."
	cd server && go build -o supacontrol main.go

# Run the server
run:
	@echo "Running SupaControl server..."
	cd server && go run main.go

# Run tests
test:
	@echo "Running tests..."
	cd server && go test -v ./...
	cd ui && npm test

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	cd server && go test -v -coverprofile=../coverage/coverage.out -covermode=atomic ./...
	cd server && go tool cover -html=../coverage/coverage.out -o ../coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"
	@echo ""
	@echo "Coverage summary:"
	cd server && go tool cover -func=../coverage/coverage.out | grep total

# Run UI tests
ui-test:
	@echo "Running UI tests..."
	cd ui && npm test -- --coverage

# CI target - runs all checks
ci: lint test-coverage build
	@echo "✓ All CI checks passed!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f server/supacontrol
	rm -rf ui/dist
	rm -rf ui/node_modules
	rm -rf coverage
	rm -rf ui/coverage
	rm -rf cli/dist
	rm -rf cli/node_modules

# Build UI
ui-build:
	@echo "Building React frontend..."
	cd ui && npm install && npm run build

# Run UI development server
ui-dev:
	@echo "Starting UI development server..."
	cd ui && npm run dev

# Build Docker image
docker-build: ui-build
	@echo "Building Docker image..."
	docker build -t supacontrol/server:latest .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push supacontrol/server:latest

# Install dependencies
install-deps:
	@echo "Installing Go dependencies..."
	cd server && go mod download
	cd pkg/api-types && go mod download
	@echo "Installing Node.js dependencies..."
	cd ui && npm install

# Run linters
lint:
	@echo "Running Go linters..."
	cd server && go vet ./...
	cd server && golangci-lint run
	@echo "Running UI linters..."
	cd ui && npm run lint

# Format code
format:
	@echo "Formatting Go code..."
	cd server && go fmt ./...
	@echo "Formatting UI code..."
	cd ui && npm run format

# Run migrations (requires database)
migrate:
	@echo "Running database migrations..."
	# Migrations are auto-applied on server startup

# Development setup
dev-setup: install-deps
	@echo "Development environment setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Set up PostgreSQL database"
	@echo "  2. Copy .env.example to .env and configure"
	@echo "  3. Run 'make run' to start the server"
	@echo "  4. Run 'make ui-dev' in another terminal for UI development"
