package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type FloatingIPProviderPhase string

// These are the valid statuses of floatingip
const (
	// FloatingIPSucceeded means the floating ip has been creating successful.
	FloatingIPProviderReady FloatingIPProviderPhase = "Ready"
	// FloatingIPFailed means some errors happened.
	FloatingIPProviderNotReady FloatingIPProviderPhase = "Not Ready"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingipprovider
// +kubebuilder:resource:shortName=fip,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="CLAIMS",type=string,JSONPath=".status.floatingIPClaims[*]"
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=".status.phase"
type FloatingIPProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of FloatingIPProvider.
	Spec FloatingIPProviderSpec `json:"spec"`
	// Observed status of FloatingIPProvider. Read-only
	// +optional
	Status FloatingIPProviderStatus `json:"status,omitempty"`
}

type FloatingIPProviderStatus struct {
	FloatingIPClaims []string `json:"floatingIPClaims"`

	// The phase of a floating ip.
	// +optional
	Phase FloatingIPProviderPhase `json:"phase,omitempty"`
}

// FloatingIPProviderSpec is a desired state description o FloatingIPProvider.
type FloatingIPProviderSpec struct {
	// FloatingIPs is the list of floating IP addresses requested. Can be IPv4 and/or IPv6.
	// This field is mandatory.
	FloatingIPs []string `json:"floatingIPs"`
	// VpcSelector applies the floating IP only to the tenant(s) whose label
	// matches this definition. This field is mandatory.
	VpcSelector metav1.LabelSelector `json:"vpcSelector"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=floatingipprovider
// FloatingIPProviderList is the list of FloatingIPProvider
type FloatingIPProviderList struct {
	metav1.TypeMeta `json;",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of FloatingIPProvider.
	Items []FloatingIPProvider `json:"items"`
}
