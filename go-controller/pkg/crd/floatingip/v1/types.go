package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingip
// +kubebuilder:resource:shortName=fip,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FloatingIP struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of FloatingIP.
	Spec FloatingIPSpec `json:"spec"`

	// Observed status of FloatingIP. Read-only.
	// +optional
	Status FloatingIPStatus `json:"status,omitempty"`
}

// FloatingIPSpec is a desired state description of FloatingIP.
type FloatingIPSpec struct {
	// Provider is the FloatingIPProvider which provides the floating IPs.
	Provider string `json:"provider"`
	// FloatingIPs
	FloatingIPs []string `json:"floatingIPs"`
	// Limit is the maximum number of floating IPs that can be used.
	Limit int `json:"limit"`

	// PodSelector applies the floating IP only to the pods whose label
	// matches this deifnition.
	// This field is mandatory.
	PodSelector metav1.LabelSelector `json:"podSelector"`
}

type FloatingIPStatus struct {
	// The list of assigned floating IPs and their corresponding pod/node assignment.
	Items []FloatingIPStatusItem `json:"items"`
}

// The per pod status, for those floating IPs who have been assigned.
type FloatingIPStatusItem struct {
	// Assigned pod name
	Pod string `json:"pod"`
	// Assigned node name
	Node string `json:"node"`
    // Assigned floating IP
	FloatingIP string `json:"floatingIP"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=floatingip
/// FloatingIPList is the list of FloatingIP.
type FloatingIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of FloatingIP
	Items []FloatingIP `json:"items"`
}
