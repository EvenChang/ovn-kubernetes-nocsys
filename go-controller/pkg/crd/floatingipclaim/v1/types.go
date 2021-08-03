package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type FloatingIPClaimPhase string

// These are the valid statuses of floatingip
const (
	// FloatingIPClaimReady means the floating ip has been creating successful.
	FloatingIPClaimReady FloatingIPClaimPhase = "Ready"
	// FloatingIPClaimNotReady means some errors happened.
	FloatingIPClaimNotReady FloatingIPClaimPhase = "Not Ready"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingipclaim
// +kubebuilder:resource:shortName=fic,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="IPS",type=string,JSONPath=".spec.floatingIPs[*]"
// +kubebuilder:printcolumn:name="ASSIGN",type=string,JSONPath=".status.assignedIPs[*]"
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=".status.phase"
type FloatingIPClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of FloatingIPClaim.
	Spec FloatingIPClaimSpec `json:"spec"`

	// Observed status of FloatingIPClaim. Read-only.
	// +optional
	Status FloatingIPClaimStatus `json:"status,omitempty"`
}

// FloatingIPClaimSpec is a desired state description of FloatingIP.
type FloatingIPClaimSpec struct {
	// Provider is the FloatingIPProvider which provides the floating IPs.
	Provider string `json:"provider"`
	// FloatingIPs
	FloatingIPs []string `json:"floatingIPs"`

	// NamespaceSelector applies the floating IP only to the namespace(s) whose label
	// matches this definition. This field is mandatory.
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector"`
	// PodSelector applies the floating IP only to the pods whose label
	// matches this deifnition. This field is mandatory.
	PodSelector metav1.LabelSelector `json:"podSelector"`
}

type FloatingIPClaimStatus struct {
	// Assigned floating IP
	AssignedIPs []string `json:"assignedIPs"`

	// The phase of a floating ip.
	// +optional
	Phase FloatingIPClaimPhase `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=floatingipclaim
/// FloatingIPClaimList is the list of FloatingIPClaim.
type FloatingIPClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of FloatingIP
	Items []FloatingIPClaim `json:"items"`
}
