package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SupabaseInstanceSpec defines the desired state of SupabaseInstance
type SupabaseInstanceSpec struct {
	// ProjectName is the unique identifier for this Supabase instance
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	ProjectName string `json:"projectName"`

	// IngressClass specifies the Kubernetes ingress class to use
	// +optional
	IngressClass string `json:"ingressClass,omitempty"`

	// IngressDomain specifies the base domain for instance URLs
	// +optional
	IngressDomain string `json:"ingressDomain,omitempty"`

	// ChartVersion specifies the Supabase Helm chart version to use
	// +optional
	ChartVersion string `json:"chartVersion,omitempty"`

	// Paused indicates whether reconciliation should be paused
	// +optional
	Paused bool `json:"paused,omitempty"`
}

// SupabaseInstancePhase represents the current phase of a SupabaseInstance
// +kubebuilder:validation:Enum=Pending;Provisioning;ProvisioningInProgress;Running;Deleting;DeletingInProgress;Failed
type SupabaseInstancePhase string

const (
	// PhasePending indicates the instance is waiting to be provisioned
	PhasePending SupabaseInstancePhase = "Pending"

	// PhaseProvisioning indicates the provisioning Job has been created
	PhaseProvisioning SupabaseInstancePhase = "Provisioning"

	// PhaseProvisioningInProgress indicates the provisioning Job is actively running
	PhaseProvisioningInProgress SupabaseInstancePhase = "ProvisioningInProgress"

	// PhaseRunning indicates the instance is running and healthy
	PhaseRunning SupabaseInstancePhase = "Running"

	// PhaseDeleting indicates the cleanup Job has been created
	PhaseDeleting SupabaseInstancePhase = "Deleting"

	// PhaseDeletingInProgress indicates the cleanup Job is actively running
	PhaseDeletingInProgress SupabaseInstancePhase = "DeletingInProgress"

	// PhaseFailed indicates the instance has failed
	PhaseFailed SupabaseInstancePhase = "Failed"
)

// AllPhases returns a slice of all possible SupabaseInstance phases as strings.
// This is the canonical source of truth for phase enumeration used by metrics and other components.
func AllPhases() []string {
	return []string{
		string(PhasePending),
		string(PhaseProvisioning),
		string(PhaseProvisioningInProgress),
		string(PhaseRunning),
		string(PhaseDeleting),
		string(PhaseDeletingInProgress),
		string(PhaseFailed),
	}
}

// SupabaseInstanceStatus defines the observed state of SupabaseInstance
type SupabaseInstanceStatus struct {
	// Phase represents the current phase of the instance
	// +optional
	Phase SupabaseInstancePhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the instance's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Namespace is the Kubernetes namespace where the instance is deployed
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// StudioURL is the URL to access the Supabase Studio UI
	// +optional
	StudioURL string `json:"studioUrl,omitempty"`

	// APIURL is the URL to access the Supabase API
	// +optional
	APIURL string `json:"apiUrl,omitempty"`

	// ErrorMessage contains error details if the instance is in Failed phase
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed spec
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastTransitionTime is the last time the phase transitioned
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// HelmReleaseName is the name of the Helm release
	// +optional
	HelmReleaseName string `json:"helmReleaseName,omitempty"`

	// ProvisioningJobName is the name of the current/last provisioning Job
	// +optional
	ProvisioningJobName string `json:"provisioningJobName,omitempty"`

	// CleanupJobName is the name of the current/last cleanup Job
	// +optional
	CleanupJobName string `json:"cleanupJobName,omitempty"`
}

// Condition types for SupabaseInstance
const (
	// ConditionTypeReady indicates whether the instance is ready
	ConditionTypeReady = "Ready"

	// ConditionTypeNamespaceReady indicates whether the namespace is created
	ConditionTypeNamespaceReady = "NamespaceReady"

	// ConditionTypeSecretsReady indicates whether secrets are created
	ConditionTypeSecretsReady = "SecretsReady"

	// ConditionTypeHelmReleaseReady indicates whether the Helm release is ready
	ConditionTypeHelmReleaseReady = "HelmReleaseReady"

	// ConditionTypeIngressReady indicates whether ingress is configured
	ConditionTypeIngressReady = "IngressReady"
)

// SupabaseInstance is the Schema for the supabaseinstances API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=sbi;sbinst
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Studio URL",type=string,JSONPath=`.status.studioUrl`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type SupabaseInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SupabaseInstanceSpec   `json:"spec,omitempty"`
	Status SupabaseInstanceStatus `json:"status,omitempty"`
}

// SupabaseInstanceList contains a list of SupabaseInstance
// +kubebuilder:object:root=true
type SupabaseInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SupabaseInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SupabaseInstance{}, &SupabaseInstanceList{})
}
