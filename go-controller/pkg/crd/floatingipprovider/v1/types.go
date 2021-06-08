package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +resource:path=floatingipprovider
// +kubebuilder:resource:shortName=fip,scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="FloatingIPs",type=string,JSONPath=".spec.floatingIPs[*]"
// +kubebuilder:printcolumn:name="Assigned FloatingIP",type=string,JSONPath=".status.items[*].floatingIPs[*]"
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

// The per FloatingIP status, for those floating ip who have been claimed.
type FloatingIPProviderStatusItem struct {
	// floatingIPClaim's name
	FloatingIPClaim string `json:"floatingIPClaim"`
	// floating ip addresses
	FloatingIPs []string `json:"floatingIPs"`
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