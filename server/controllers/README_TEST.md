# Controller Tests

This directory contains comprehensive tests for the SupabaseInstance controller using controller-runtime's envtest framework.

## Test Coverage

The test suite covers:

1. **Pending → Provisioning Transitions**
   - TestReconcilePending_CreatesProvisioningJob
   - TestFinalizer_AddedOnCreation

2. **Provisioning → InProgress → Running**
   - TestReconcileProvisioning_TransitionsToInProgress
   - TestReconcileProvisioningInProgress_HandlesJobSuccess

3. **Job Failure Scenarios**
   - TestReconcileProvisioningInProgress_HandlesJobFailure
   - TestJobTimeout_HandlesActiveDeadlineSeconds

4. **Deletion and Cleanup**
   - TestReconcileDelete_CreatesCleanupJob
   - TestCleanupViaJob_TransitionsToDeletingInProgress

5. **Owner References and Orphan Prevention**
   - TestJobOwnerReferences_PreventOrphans

6. **Additional Scenarios**
   - TestReconcilePaused_SkipsReconciliation
   - TestReconcileRunning_PeriodicHealthChecks

## Prerequisites

The controller tests use [envtest](https://book.kubebuilder.io/reference/envtest.html) which requires Kubernetes test binaries (etcd and kube-apiserver).

### Installing Test Binaries

#### Option 1: Using setup-envtest (Recommended)

```bash
# Install setup-envtest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# Download test binaries (this will download binaries for your K8s version)
setup-envtest use 1.28.x

# Set the path (the command above will print the path)
export KUBEBUILDER_ASSETS="$(setup-envtest use -p path 1.28.x)"
```

#### Option 2: Manual Installation

Download and extract kubebuilder:

```bash
# For Linux/amd64
curl -L -o kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.12.0/kubebuilder_linux_amd64
chmod +x kubebuilder
sudo mv kubebuilder /usr/local/bin/

# Initialize kubebuilder to download test binaries
kubebuilder init --domain qubitquilt.com --repo github.com/qubitquilt/supacontrol
```

#### Option 3: Using a Makefile target

Add to your Makefile:

```makefile
ENVTEST = $(shell pwd)/bin/setup-envtest
ENVTEST_K8S_VERSION = 1.28.x

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: test-controller
test-controller: envtest ## Run controller tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./controllers/... -coverprofile=coverage.out
```

## Running Tests

### Run All Controller Tests

```bash
# After setting up envtest
cd server
go test -v ./controllers/...
```

### Run Specific Test

```bash
go test -v ./controllers/... -run TestReconcilePending_CreatesProvisioningJob
```

### Run with Coverage

```bash
go test -v ./controllers/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run with Race Detection

```bash
go test -v -race ./controllers/...
```

## Test Architecture

### Test Suite Setup (suite_test.go)

- **TestMain**: Sets up and tears down the envtest environment
- **Helper Functions**:
  - `createTestReconciler()`: Creates a configured reconciler for testing
  - `createBasicInstance(name)`: Creates a basic SupabaseInstance for tests
  - `getInstanceState()`: Retrieves the current state of an instance
  - `cleanupInstance()`: Cleans up test resources after each test
  - `waitForCondition()`: Polls for a condition with timeout

### Test Structure

Each test follows this pattern:

```go
func TestSomething(t *testing.T) {
    ctx := context.Background()
    reconciler := createTestReconciler()

    // 1. Setup: Create instance
    instance := createBasicInstance("test-name")
    err := k8sClient.Create(ctx, instance)
    if err != nil {
        t.Fatalf("Failed to create test instance: %v", err)
    }
    defer cleanupInstance(ctx, t, instance)

    // 2. Act: Reconcile
    req := ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}}
    result, err := reconciler.Reconcile(ctx, req)

    // 3. Assert: Verify behavior
    if err != nil {
        t.Fatalf("Reconcile failed: %v", err)
    }
    // ... more assertions
}
```

## Debugging Tests

### Enable Verbose Logging

```bash
go test -v ./controllers/... -args -zap-log-level=debug
```

### Check Test Isolation

Tests should be isolated and not depend on each other. Each test:
- Creates its own instance with a unique name
- Cleans up resources using `defer cleanupInstance()`
- Uses a fresh context

### Common Issues

**Issue**: Test hangs or times out
- **Cause**: Waiting for reconciliation that never completes
- **Solution**: Check requeue logic and ensure Jobs transition to success/failure state

**Issue**: "resource already exists" errors
- **Cause**: Test cleanup didn't finish
- **Solution**: Ensure unique names per test and proper cleanup in defer

**Issue**: "envtest binaries not found"
- **Cause**: KUBEBUILDER_ASSETS not set
- **Solution**: Follow Prerequisites section above

## CI/CD Integration

### GitHub Actions

Controller tests are **fully configured and running in CI**. The configuration is located in `.github/workflows/ci.yml` and includes:

- Automatic setup of envtest binaries using `setup-envtest`
- Test execution with race detection enabled
- Coverage reporting to Codecov
- Runs on all pushes to `main` and `develop` branches
- Runs on all pull requests

**Local developers** should follow the Prerequisites section above to set up their environment.

### CI Configuration Details

The CI workflow (`test-backend` job) includes:
```yaml
- name: Install envtest binaries for controller tests
  run: |
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    echo "$(go env GOPATH)/bin" >> $GITHUB_PATH

- name: Set up envtest environment
  run: |
    setup-envtest use 1.28.x -p path
    echo "KUBEBUILDER_ASSETS=$(setup-envtest use 1.28.x -p path)" >> $GITHUB_ENV

- name: Run tests with coverage
  run: |
    cd server
    go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

## Test Maintenance

When modifying the controller:

1. **Add new phase**: Add tests for transitions to/from the new phase
2. **Add new status field**: Verify field is set correctly in relevant tests
3. **Change reconciliation logic**: Update affected test assertions
4. **Add new condition**: Test that condition is set with correct status/reason

## Performance Considerations

- Individual tests should complete in < 5 seconds
- Full suite should complete in < 30 seconds
- Use shorter timeouts in tests (e.g., 5s instead of 5m)
- Mock time-consuming operations when possible

## References

- [Kubebuilder Testing Guide](https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html)
- [controller-runtime envtest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
- [Testing Best Practices](https://github.com/kubernetes-sigs/controller-runtime/blob/main/designs/move-cluster-specific-code-out-of-manager.md)
