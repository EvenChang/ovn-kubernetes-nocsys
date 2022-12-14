// Code generated by mockery v2.7.5. DO NOT EDIT.

package mocks

import (
	egressfirewallv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressfirewall/v1"
	egressipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1"
	floatingipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	floatingipclaimv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	floatingipproviderv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipprovider/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"
)

// KubeInterface is an autogenerated mock type for the Interface type
type KubeInterface struct {
	mock.Mock
}

// CreateEndpoint provides a mock function with given fields: namespace, ep
func (_m *KubeInterface) CreateEndpoint(namespace string, ep *v1.Endpoints) (*v1.Endpoints, error) {
	ret := _m.Called(namespace, ep)

	var r0 *v1.Endpoints
	if rf, ok := ret.Get(0).(func(string, *v1.Endpoints) *v1.Endpoints); ok {
		r0 = rf(namespace, ep)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Endpoints)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *v1.Endpoints) error); ok {
		r1 = rf(namespace, ep)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Events provides a mock function with given fields:
func (_m *KubeInterface) Events() corev1.EventInterface {
	ret := _m.Called()

	var r0 corev1.EventInterface
	if rf, ok := ret.Get(0).(func() corev1.EventInterface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(corev1.EventInterface)
		}
	}

	return r0
}

// GetAnnotationsOnPod provides a mock function with given fields: namespace, name
func (_m *KubeInterface) GetAnnotationsOnPod(namespace string, name string) (map[string]string, error) {
	ret := _m.Called(namespace, name)

	var r0 map[string]string
	if rf, ok := ret.Get(0).(func(string, string) map[string]string); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEgressFirewalls provides a mock function with given fields:
func (_m *KubeInterface) GetEgressFirewalls() (*egressfirewallv1.EgressFirewallList, error) {
	ret := _m.Called()

	var r0 *egressfirewallv1.EgressFirewallList
	if rf, ok := ret.Get(0).(func() *egressfirewallv1.EgressFirewallList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*egressfirewallv1.EgressFirewallList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEgressIP provides a mock function with given fields: name
func (_m *KubeInterface) GetEgressIP(name string) (*egressipv1.EgressIP, error) {
	ret := _m.Called(name)

	var r0 *egressipv1.EgressIP
	if rf, ok := ret.Get(0).(func(string) *egressipv1.EgressIP); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*egressipv1.EgressIP)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEgressIPs provides a mock function with given fields:
func (_m *KubeInterface) GetEgressIPs() (*egressipv1.EgressIPList, error) {
	ret := _m.Called()

	var r0 *egressipv1.EgressIPList
	if rf, ok := ret.Get(0).(func() *egressipv1.EgressIPList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*egressipv1.EgressIPList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEndpoint provides a mock function with given fields: namespace, name
func (_m *KubeInterface) GetEndpoint(namespace string, name string) (*v1.Endpoints, error) {
	ret := _m.Called(namespace, name)

	var r0 *v1.Endpoints
	if rf, ok := ret.Get(0).(func(string, string) *v1.Endpoints); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Endpoints)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNamespaces provides a mock function with given fields: labelSelector
func (_m *KubeInterface) GetNamespaces(labelSelector metav1.LabelSelector) (*v1.NamespaceList, error) {
	ret := _m.Called(labelSelector)

	var r0 *v1.NamespaceList
	if rf, ok := ret.Get(0).(func(metav1.LabelSelector) *v1.NamespaceList); ok {
		r0 = rf(labelSelector)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.NamespaceList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(metav1.LabelSelector) error); ok {
		r1 = rf(labelSelector)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNode provides a mock function with given fields: name
func (_m *KubeInterface) GetNode(name string) (*v1.Node, error) {
	ret := _m.Called(name)

	var r0 *v1.Node
	if rf, ok := ret.Get(0).(func(string) *v1.Node); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Node)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodes provides a mock function with given fields:
func (_m *KubeInterface) GetNodes() (*v1.NodeList, error) {
	ret := _m.Called()

	var r0 *v1.NodeList
	if rf, ok := ret.Get(0).(func() *v1.NodeList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.NodeList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPods provides a mock function with given fields: namespace, labelSelector
func (_m *KubeInterface) GetPods(namespace string, labelSelector metav1.LabelSelector) (*v1.PodList, error) {
	ret := _m.Called(namespace, labelSelector)

	var r0 *v1.PodList
	if rf, ok := ret.Get(0).(func(string, metav1.LabelSelector) *v1.PodList); ok {
		r0 = rf(namespace, labelSelector)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.PodList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, metav1.LabelSelector) error); ok {
		r1 = rf(namespace, labelSelector)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *KubeInterface) GetPod(namespace, name string) (*v1.Pod, error) {
	ret := _m.Called(namespace, name)

	var r0 *v1.Pod
	if rf, ok := ret.Get(0).(func(string) *v1.Pod); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Pod)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetAnnotationsOnNamespace provides a mock function with given fields: namespace, annotations
func (_m *KubeInterface) SetAnnotationsOnNamespace(namespace *v1.Namespace, annotations map[string]string) error {
	ret := _m.Called(namespace, annotations)

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1.Namespace, map[string]string) error); ok {
		r0 = rf(namespace, annotations)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetAnnotationsOnNode provides a mock function with given fields: node, annotations
func (_m *KubeInterface) SetAnnotationsOnNode(node *v1.Node, annotations map[string]interface{}) error {
	ret := _m.Called(node, annotations)

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1.Node, map[string]interface{}) error); ok {
		r0 = rf(node, annotations)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetAnnotationsOnPod provides a mock function with given fields: namespace, podName, annotations
func (_m *KubeInterface) SetAnnotationsOnPod(namespace string, podName string, annotations map[string]string) error {
	ret := _m.Called(namespace, podName, annotations)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, map[string]string) error); ok {
		r0 = rf(namespace, podName, annotations)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateEgressFirewall provides a mock function with given fields: egressfirewall
func (_m *KubeInterface) UpdateEgressFirewall(egressfirewall *egressfirewallv1.EgressFirewall) error {
	ret := _m.Called(egressfirewall)

	var r0 error
	if rf, ok := ret.Get(0).(func(*egressfirewallv1.EgressFirewall) error); ok {
		r0 = rf(egressfirewall)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateEgressIP provides a mock function with given fields: eIP
func (_m *KubeInterface) UpdateEgressIP(eIP *egressipv1.EgressIP) error {
	ret := _m.Called(eIP)

	var r0 error
	if rf, ok := ret.Get(0).(func(*egressipv1.EgressIP) error); ok {
		r0 = rf(eIP)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateNodeStatus provides a mock function with given fields: node
func (_m *KubeInterface) UpdateNodeStatus(node *v1.Node) error {
	ret := _m.Called(node)

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1.Node) error); ok {
		r0 = rf(node)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *KubeInterface) UpdateFloatingIP(fi *floatingipv1.FloatingIP) error {
	ret := _m.Called(fi)

	var r0 error
	if rf, ok := ret.Get(0).(func(*floatingipv1.FloatingIP) error); ok {
		r0 = rf(fi)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *KubeInterface) UpdateFloatingIPClaim(fic *floatingipclaimv1.FloatingIPClaim) error {
	ret := _m.Called(fic)

	var r0 error
	if rf, ok := ret.Get(0).(func(*floatingipclaimv1.FloatingIPClaim) error); ok {
		r0 = rf(fic)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *KubeInterface) UpdateFloatingIPProvider(fip *floatingipproviderv1.FloatingIPProvider) error {
	ret := _m.Called(fip)

	var r0 error
	if rf, ok := ret.Get(0).(func(*floatingipproviderv1.FloatingIPProvider) error); ok {
		r0 = rf(fip)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *KubeInterface) GetFloatingIP(name string) (*floatingipv1.FloatingIP, error) {
	ret := _m.Called(name)

	var r0 *floatingipv1.FloatingIP
	if rf, ok := ret.Get(0).(func(string) *floatingipv1.FloatingIP); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*floatingipv1.FloatingIP)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *KubeInterface) CreateFloatingIP(fi *floatingipv1.FloatingIP) (*floatingipv1.FloatingIP, error) {
	return fi, nil
}

func (_m *KubeInterface) DeleteFloatingIP(name string) error {
	return nil
}

func (_m *KubeInterface) GetFloatingIPs() (*floatingipv1.FloatingIPList, error) {
	ret := _m.Called()

	var r0 *floatingipv1.FloatingIPList
	if rf, ok := ret.Get(0).(func() *floatingipv1.FloatingIPList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*floatingipv1.FloatingIPList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *KubeInterface) GetFloatingIPClaim(name string) (*floatingipclaimv1.FloatingIPClaim, error) {
	ret := _m.Called(name)

	var r0 *floatingipclaimv1.FloatingIPClaim
	if rf, ok := ret.Get(0).(func(string) *floatingipclaimv1.FloatingIPClaim); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*floatingipclaimv1.FloatingIPClaim)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *KubeInterface) GetFloatingIPClaims() (*floatingipclaimv1.FloatingIPClaimList, error) {
	ret := _m.Called()

	var r0 *floatingipclaimv1.FloatingIPClaimList
	if rf, ok := ret.Get(0).(func() *floatingipclaimv1.FloatingIPClaimList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*floatingipclaimv1.FloatingIPClaimList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *KubeInterface) DeleteFloatingIPClaim(name string) error {
	return nil
}

func (_m *KubeInterface) GetFloatingIPProvider(name string) (*floatingipproviderv1.FloatingIPProvider, error) {
	ret := _m.Called(name)

	var r0 *floatingipproviderv1.FloatingIPProvider
	if rf, ok := ret.Get(0).(func(string) *floatingipproviderv1.FloatingIPProvider); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*floatingipproviderv1.FloatingIPProvider)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *KubeInterface) GetFloatingIPProviders() (*floatingipproviderv1.FloatingIPProviderList, error) {
	ret := _m.Called()

	var r0 *floatingipproviderv1.FloatingIPProviderList
	if rf, ok := ret.Get(0).(func() *floatingipproviderv1.FloatingIPProviderList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*floatingipproviderv1.FloatingIPProviderList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
