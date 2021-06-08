package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +resource:path=floatingip
// +kubebuilder:resource:shortName=fi
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FloatingIP struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of FloatingIPClaim.
	Spec FloatingIPSpec `json:"spec"`
}

// FloatingIPSpec is a desired state description of FloatingIP.
type FloatingIPSpec struct {
	// floating ip claim name
	FloatingIPClaim string `json:"floatingIPClaim"`

	// Pod assigned to floating ip
	Pod string `json:"pod"`

	// Node assigned to floating ip
	Node string `json:"node"`

	// Assigned floating ip address
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
