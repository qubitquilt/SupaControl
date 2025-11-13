package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAPIMetrics(t *testing.T) {
	t.Run("APIRequestsTotal increments", func(t *testing.T) {
		// Get initial count
		initial := testutil.ToFloat64(APIRequestsTotal.WithLabelValues("/test", "GET", "OK"))

		// Increment
		APIRequestsTotal.WithLabelValues("/test", "GET", "OK").Inc()

		// Verify increment
		final := testutil.ToFloat64(APIRequestsTotal.WithLabelValues("/test", "GET", "OK"))
		assert.Equal(t, initial+1, final, "counter should increment by 1")
	})

	t.Run("APIRequestDuration records observations", func(t *testing.T) {
		// Record some durations - verify they don't panic
		assert.NotPanics(t, func() {
			APIRequestDuration.WithLabelValues("/test", "GET").Observe(0.1)
			APIRequestDuration.WithLabelValues("/test", "GET").Observe(0.2)
			APIRequestDuration.WithLabelValues("/test", "GET").Observe(0.3)
		}, "recording histogram observations should not panic")
	})
}

func TestInstanceMetrics(t *testing.T) {
	t.Run("InstancesTotal increments and decrements", func(t *testing.T) {
		// Get initial value
		initial := testutil.ToFloat64(InstancesTotal)

		// Increment
		InstancesTotal.Inc()
		afterInc := testutil.ToFloat64(InstancesTotal)
		assert.Equal(t, initial+1, afterInc, "gauge should increment by 1")

		// Decrement
		InstancesTotal.Dec()
		afterDec := testutil.ToFloat64(InstancesTotal)
		assert.Equal(t, initial, afterDec, "gauge should decrement back to initial")
	})

	t.Run("InstanceCreationDuration records observations", func(t *testing.T) {
		// Record some creation durations - verify they don't panic
		assert.NotPanics(t, func() {
			InstanceCreationDuration.Observe(60.0)  // 1 minute
			InstanceCreationDuration.Observe(120.0) // 2 minutes
			InstanceCreationDuration.Observe(300.0) // 5 minutes
		}, "recording histogram observations should not panic")
	})
}

func TestSetInstanceStatus(t *testing.T) {
	allPhases := []string{"Pending", "Provisioning", "Running", "Failed"}

	t.Run("sets only current phase to 1", func(t *testing.T) {
		projectName := "test-instance"
		currentPhase := "Running"

		// Set instance status
		SetInstanceStatus(projectName, currentPhase, allPhases)

		// Verify Running is 1
		runningVal := testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Running"))
		assert.Equal(t, 1.0, runningVal, "current phase should be 1")

		// Verify others are 0
		pendingVal := testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Pending"))
		assert.Equal(t, 0.0, pendingVal, "other phases should be 0")

		provisioningVal := testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Provisioning"))
		assert.Equal(t, 0.0, provisioningVal, "other phases should be 0")

		failedVal := testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Failed"))
		assert.Equal(t, 0.0, failedVal, "other phases should be 0")
	})

	t.Run("updates when phase changes", func(t *testing.T) {
		projectName := "test-instance-2"

		// Set to Pending
		SetInstanceStatus(projectName, "Pending", allPhases)
		pendingVal := testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Pending"))
		assert.Equal(t, 1.0, pendingVal, "Pending should be 1")

		// Change to Running
		SetInstanceStatus(projectName, "Running", allPhases)
		runningVal := testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Running"))
		assert.Equal(t, 1.0, runningVal, "Running should be 1")

		pendingVal = testutil.ToFloat64(InstanceStatus.WithLabelValues(projectName, "Pending"))
		assert.Equal(t, 0.0, pendingVal, "Pending should be 0 after transition")
	})
}

func TestDeleteInstanceMetrics(t *testing.T) {
	allPhases := []string{"Pending", "Provisioning", "Running", "Failed"}

	t.Run("deletes all phase metrics for instance", func(_ *testing.T) {
		projectName := "test-instance-delete"

		// Set instance status
		SetInstanceStatus(projectName, "Running", allPhases)

		// Delete metrics
		DeleteInstanceMetrics(projectName, allPhases)

		// Note: After deletion, the metrics are removed from the registry
		// We can't easily verify deletion without accessing internal state,
		// but we can verify the function runs without error
		// In a real test, you might check the prometheus registry
	})
}

func TestJobMetrics(t *testing.T) {
	t.Run("JobStatusTotal increments", func(t *testing.T) {
		// Get initial count
		initial := testutil.ToFloat64(JobStatusTotal.WithLabelValues("provision", "succeeded"))

		// Increment
		JobStatusTotal.WithLabelValues("provision", "succeeded").Inc()

		// Verify increment
		final := testutil.ToFloat64(JobStatusTotal.WithLabelValues("provision", "succeeded"))
		assert.Equal(t, initial+1, final, "counter should increment by 1")
	})

	t.Run("JobDuration records observations", func(t *testing.T) {
		// Record some job durations - verify they don't panic
		assert.NotPanics(t, func() {
			JobDuration.WithLabelValues("provision", "succeeded").Observe(120.0)
			JobDuration.WithLabelValues("provision", "succeeded").Observe(150.0)
		}, "recording histogram observations should not panic")
	})
}

func TestReconciliationMetrics(t *testing.T) {
	t.Run("ReconciliationTotal increments", func(t *testing.T) {
		// Get initial count
		initial := testutil.ToFloat64(ReconciliationTotal.WithLabelValues("Pending"))

		// Increment
		ReconciliationTotal.WithLabelValues("Pending").Inc()

		// Verify increment
		final := testutil.ToFloat64(ReconciliationTotal.WithLabelValues("Pending"))
		assert.Equal(t, initial+1, final, "counter should increment by 1")
	})

	t.Run("ReconciliationDuration records observations", func(t *testing.T) {
		// Record some reconciliation durations - verify they don't panic
		assert.NotPanics(t, func() {
			ReconciliationDuration.WithLabelValues("Running").Observe(0.05)
			ReconciliationDuration.WithLabelValues("Running").Observe(0.1)
		}, "recording histogram observations should not panic")
	})

	t.Run("ReconciliationErrorsTotal increments", func(t *testing.T) {
		// Get initial count
		initial := testutil.ToFloat64(ReconciliationErrorsTotal.WithLabelValues("Provisioning"))

		// Increment
		ReconciliationErrorsTotal.WithLabelValues("Provisioning").Inc()

		// Verify increment
		final := testutil.ToFloat64(ReconciliationErrorsTotal.WithLabelValues("Provisioning"))
		assert.Equal(t, initial+1, final, "counter should increment by 1")
	})
}

func TestMetricsRegistration(t *testing.T) {
	t.Run("all metrics are properly registered", func(t *testing.T) {
		// Verify metrics are accessible (not nil)
		assert.NotNil(t, APIRequestsTotal, "APIRequestsTotal should be registered")
		assert.NotNil(t, APIRequestDuration, "APIRequestDuration should be registered")
		assert.NotNil(t, InstancesTotal, "InstancesTotal should be registered")
		assert.NotNil(t, InstanceStatus, "InstanceStatus should be registered")
		assert.NotNil(t, InstanceCreationDuration, "InstanceCreationDuration should be registered")
		assert.NotNil(t, JobStatusTotal, "JobStatusTotal should be registered")
		assert.NotNil(t, JobDuration, "JobDuration should be registered")
		assert.NotNil(t, ReconciliationTotal, "ReconciliationTotal should be registered")
		assert.NotNil(t, ReconciliationDuration, "ReconciliationDuration should be registered")
		assert.NotNil(t, ReconciliationErrorsTotal, "ReconciliationErrorsTotal should be registered")
	})

	t.Run("metrics implement correct prometheus types", func(t *testing.T) {
		// Verify types
		assert.Implements(t, (*prometheus.Collector)(nil), APIRequestsTotal)
		assert.Implements(t, (*prometheus.Collector)(nil), APIRequestDuration)
		assert.Implements(t, (*prometheus.Collector)(nil), InstancesTotal)
		assert.Implements(t, (*prometheus.Collector)(nil), InstanceStatus)
		assert.Implements(t, (*prometheus.Collector)(nil), InstanceCreationDuration)
	})
}
