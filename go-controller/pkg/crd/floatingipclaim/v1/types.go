package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingipclaim
// +kubebuilder:resource:shortName=fic,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="FloatingIPs",type=string,JSONPath=".spec.floatingIPs[*]"
// +kubebuilder:printcolumn:name="Nodes",type=string,JSONPath=".status.nodes[*]"
// +kubebuilder:printcolumn:name="Assigned FloatingIPs",type=string,JSONPath=".status.assignedIPs[*]"
type FloatingIPClaim struct {
	metav1.TypeMeta `json:",inline"`
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
	Nodes []string `json:"nodes"`
	// Assigned floating IP
	AssignedIPs []string `json:"assignedIPs"`
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
