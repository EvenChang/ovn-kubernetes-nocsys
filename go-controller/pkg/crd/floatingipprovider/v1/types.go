package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingipprovider
// +kubebuilder:resource:shortName=fipp,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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
	// The list of assigned floating IPs and their corresponding FloatingIP.
	Items []FloatingIPProviderStatusItem `json:"items"`
}

// The per FloatingIP status, for those floating ip who have been assigned.
type FloatingIPProviderStatusItem struct {
	// FloatingIP's name
	FloatingIPName string `json:"floatingIPName"`
	// floating ip addresses
	IPs []string `json:"floatingIP"`
}

// FloatingIPProviderSpec is a desired state description o FloatingIPProvider.
type FloatingIPProviderSpec struct {
	// FloatingIPs is the list of floating IP addresses requested. Can be IPv4 and/or IPv6.
	// This field is mandatory.
	FloatingIPs []string `json:"floatingIPs"`
	// NamespaceSelector applies the floating IP only to the namespace(s) whose label
	// matches this definition. This field is mandatory.
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector"`
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