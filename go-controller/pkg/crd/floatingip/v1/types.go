package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingip
// +kubebuilder:resource:shortName=fi
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FloatingIP struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of FloatingIPClaim.
	Spec FloatingIPSpec `json:"spec"`

	// Observed status of FloatingIP. Read-only.
	// +optional
	Status FloatingIPStatus `json:"status,omitempty"`
}

// FloatingIPSpec is a desired state description of FloatingIP.
type FloatingIPSpec struct {
	// floating ip claim name
	FloatingIPClaim string `json:"floatingIPClaim"`

	// Pod assigned to floating ip
	Pod string `json:"pod"`

	// PodNamespace is namespace of the pod assigned to floating ip
	PodNamespace string `json:"podNamespace"`

	// Node assigned to floating ip
	Node string `json:"node"`

	// Assigned floating ip address
	FloatingIP string `json:"floatingIP"`
}

type FloatingIPStatus struct {
	// The phase of a floating ip.
	// +optional
	Phase FloatingIPPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=FloatingIPPhase"`

	// +optional
	Verified bool `json:"verified,omitempty" protobuf:"bytes,2,opt,name=verified"`
}

type FloatingIPPhase string

// These are the valid statuses of floatingip
const (
	// FloatingIPCreating means the floating ip has been verified, and will be creating
	FloatingIPCreating FloatingIPPhase = "Creating"
	// FloatingIPSucceeded means the floating ip has been creating successful.
	FloatingIPSucceeded FloatingIPPhase = "Succeeded"
	// FloatingIPFailed means some errors happened.
	FloatingIPFailed FloatingIPPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=floatingip
/// FloatingIPList is the list of FloatingIP.
type FloatingIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of FloatingIP
	Items []FloatingIP `json:"items"`
}
