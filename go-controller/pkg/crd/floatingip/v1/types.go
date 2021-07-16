package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingip
// +kubebuilder:resource:shortName=fi,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="FloatingIPClaim",type=string,JSONPath=".spec.floatingIPClaim"
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=".spec.podNamespace"
// +kubebuilder:printcolumn:name="Pod",type=string,JSONPath=".spec.pod"
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=".status.nodeName"
// +kubebuilder:printcolumn:name="FloatingIP",type=string,JSONPath=".status.floatingIP"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=".status.phase"
type FloatingIP struct {
	metav1.TypeMeta   `json:",inline"`
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
	// +optional
	PodNamespace string `json:"podNamespace"`
}

type FloatingIPStatus struct {
	// Node assigned to floating ip
	// +optional
	NodeName string `json:"nodeName"`

	// Assigned floating ip address
	// +optional
	FloatingIP string `json:"floatingIP"`

	// Use the host's network
	// +optional
	HostNetwork bool `json:"HostNetWork"`

	// The IP addresses of pod
	// +optional
	PodIPs []string `json:"podIPs"`

	// The phase of a floating ip.
	// +optional
	Phase FloatingIPPhase `json:"phase,omitempty"`
}

type FloatingIPPhase string

// These are the valid statuses of floatingip
const (
	// FloatingIPCreating means the floating ip has been verified, and will be creating
	FloatingIPCreating FloatingIPPhase = "Creating"
	// FloatingIPSucceeded means the floating ip has been creating successful.
	FloatingIPSucceeded FloatingIPPhase = "Success"
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
