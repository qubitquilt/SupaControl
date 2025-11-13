#!/bin/bash

# Pre-commit hook script for SupaControl
# This script runs the same linting and checks as the CI pipeline
# to catch issues locally before pushing code

set -e

echo "🔍 Running SupaControl pre-commit checks..."
echo "================================================"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if we're in the right directory
if [ ! -f "Makefile" ] || [ ! -d "server" ] || [ ! -d "ui" ]; then
    print_error "This script must be run from the SupaControl root directory"
    exit 1
fi

# Track overall success
OVERALL_SUCCESS=true

# Backend checks
echo ""
echo "🔧 Backend Checks (Go)"
echo "----------------------"

# Check if Go is available
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    OVERALL_SUCCESS=false
else
    print_status "Go found: $(go version)"
fi

# Check if golangci-lint is available
if ! command -v golangci-lint &> /dev/null; then
    print_warning "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
else
    print_status "golangci-lint found: $(golangci-lint version)"
fi

# Run Go module checks
echo ""
echo "Checking Go modules..."
if (cd server && go mod tidy -v); then
    print_status "Go modules are tidy"
else
    print_error "Go modules need to be tidied. Run 'cd server && go mod tidy'"
    OVERALL_SUCCESS=false
fi

if (cd server && go mod verify); then
    print_status "Go modules verify successfully"
else
    print_error "Go module verification failed"
    OVERALL_SUCCESS=false
fi

# Run go vet
echo ""
echo "Running go vet..."
if (cd server && go vet ./...); then
    print_status "go vet passed"
else
    print_error "go vet found issues"
    OVERALL_SUCCESS=false
fi

# Run golangci-lint if available
if command -v golangci-lint &> /dev/null; then
    echo ""
    echo "Running golangci-lint..."
    if (cd server && golangci-lint run --timeout=5m); then
        print_status "golangci-lint passed"
    else
        print_error "golangci-lint found issues"
        OVERALL_SUCCESS=false
    fi
fi

# Frontend checks
echo ""
echo "🎨 Frontend Checks (React/Node.js)"
echo "-----------------------------------"

# Check if Node.js is available
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed or not in PATH"
    OVERALL_SUCCESS=false
else
    print_status "Node.js found: $(node --version)"
fi

# Check if npm is available
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed or not in PATH"
    OVERALL_SUCCESS=false
else
    print_status "npm found: $(npm --version)"
fi

# Check if we're in the right directory for UI
if [ ! -f "ui/package.json" ]; then
    print_error "UI package.json not found"
    OVERALL_SUCCESS=false
else
    print_status "UI package.json found"
fi

# Install UI dependencies if needed
echo ""
echo "Checking UI dependencies..."
if [ ! -d "ui/node_modules" ]; then
    echo "Installing UI dependencies..."
    if (cd ui && npm install --silent); then
        print_status "UI dependencies installed"
    else
        print_error "Failed to install UI dependencies"
        OVERALL_SUCCESS=false
    fi
else
    print_status "UI dependencies already installed"
fi

# Run ESLint
echo ""
echo "Running ESLint..."
if (cd ui && npm run lint); then
    print_status "ESLint passed"
else
    print_error "ESLint found issues"
    echo ""
    echo "To fix auto-fixable issues, run: cd ui && npm run lint -- --fix"
    OVERALL_SUCCESS=false
fi

# Check for common issues
echo ""
echo "🔍 General Checks"
echo "-----------------"

# Check for TODO/FIXME comments (development check)
TODO_COUNT=$(grep -r "TODO\|FIXME" server/ ui/ --include="*.go" --include="*.js" --include="*.jsx" 2>/dev/null | grep -c . || echo "0")
if [ "$TODO_COUNT" -gt 0 ] 2>/dev/null; then
    print_warning "Found $TODO_COUNT TODO/FIXME comments (consider addressing before committing)"
    echo "Search with: grep -r 'TODO\|FIXME' server/ ui/"
else
    print_status "No TODO/FIXME comments found"
fi

# Check for debug prints
DEBUG_COUNT=$(grep -r "fmt\.Print\|print\|log\.Print" server/ --include="*.go" | grep -c -v "_test.go" 2>/dev/null || echo "0")
if [ "$DEBUG_COUNT" -gt 0 ] 2>/dev/null; then
    print_warning "Found potential debug prints in server code (check before committing)"
    echo "Search with: grep -r 'fmt\.Print\|print\|log\.Print' server/ | grep -v '_test.go'"
else
    print_status "No debug prints found"
fi

# Final summary
echo ""
echo "================================================"
if [ "$OVERALL_SUCCESS" = true ]; then
    print_status "All pre-commit checks passed! ✅"
    echo ""
    echo "Your code is ready to commit. Don't forget to run tests with 'make test' before pushing!"
    exit 0
else
    print_error "Some pre-commit checks failed! ❌"
    echo ""
    echo "Please fix the issues above before committing."
    echo ""
    echo "Common fixes:"
    echo "  • Run 'make lint-fix' to auto-fix some issues"
    echo "  • Run 'cd ui && npm run lint -- --fix' to fix frontend lint issues"
    echo "  • Run 'make ci' to run full CI checks locally"
    exit 1
fi
