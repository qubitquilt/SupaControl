package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// API Metrics

	// APIRequestsTotal counts total API requests by endpoint, method, and status code
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "supacontrol_api_requests_total",
			Help: "Total number of API requests by endpoint, method, and status code",
		},
		[]string{"endpoint", "method", "status_code"},
	)

	// APIRequestDuration tracks API request duration by endpoint and method
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "supacontrol_api_request_duration_seconds",
			Help:    "Duration of API requests in seconds",
			Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"endpoint", "method"},
	)

	// Instance State Metrics

	// InstancesTotal tracks the total number of instances
	InstancesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "supacontrol_instances_total",
			Help: "Total number of Supabase instances",
		},
	)

	// InstanceStatus tracks instance status by project name and phase
	InstanceStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "supacontrol_instance_status",
			Help: "Instance status by project name and phase (1 = current phase, 0 = not in this phase)",
		},
		[]string{"project_name", "phase"},
	)

	// InstanceCreationDuration tracks time to create instances (from Pending to Running)
	InstanceCreationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "supacontrol_instance_creation_duration_seconds",
			Help:    "Duration of instance creation from Pending to Running phase in seconds",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 900, 1800}, // 10s to 30m
		},
	)

	// Controller/Job Metrics

	// JobStatusTotal counts jobs by operation type and status
	JobStatusTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "supacontrol_job_status_total",
			Help: "Total number of jobs by operation type and final status",
		},
		[]string{"operation", "status"}, // operation: provision/cleanup, status: succeeded/failed
	)

	// JobDuration tracks job execution duration
	JobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "supacontrol_job_duration_seconds",
			Help:    "Duration of job execution in seconds",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 900, 1800}, // 10s to 30m
		},
		[]string{"operation", "status"},
	)

	// ReconciliationTotal counts total reconciliation loops
	ReconciliationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "supacontrol_reconciliation_total",
			Help: "Total number of reconciliation loops by phase",
		},
		[]string{"phase"},
	)

	// ReconciliationDuration tracks reconciliation duration
	ReconciliationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "supacontrol_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation loops in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"phase"},
	)

	// ReconciliationErrorsTotal counts reconciliation errors
	ReconciliationErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "supacontrol_reconciliation_errors_total",
			Help: "Total number of reconciliation errors by phase",
		},
		[]string{"phase"},
	)
)

// SetInstanceStatus sets the status for a specific instance
// This helper ensures only one phase is set to 1, all others to 0
func SetInstanceStatus(projectName, currentPhase string, allPhases []string) {
	for _, phase := range allPhases {
		if phase == currentPhase {
			InstanceStatus.WithLabelValues(projectName, phase).Set(1)
		} else {
			InstanceStatus.WithLabelValues(projectName, phase).Set(0)
		}
	}
}

// DeleteInstanceMetrics removes all metrics for a specific instance
func DeleteInstanceMetrics(projectName string, allPhases []string) {
	for _, phase := range allPhases {
		InstanceStatus.DeleteLabelValues(projectName, phase)
	}
}
