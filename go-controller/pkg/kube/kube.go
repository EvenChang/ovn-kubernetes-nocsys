package kube

import (
	"context"
	"encoding/json"
	"k8s.io/klog/v2"

	egressfirewall "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressfirewall/v1"
	egressfirewallclientset "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressfirewall/v1/apis/clientset/versioned"
	egressipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1"
	egressipclientset "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/clientset/versioned"
	floatingipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	floatingipclientset "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1/apis/clientset/versioned"
	floatingipclaimv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	floatingipclaimclientset "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1/apis/clientset/versioned"
	kapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kv1core "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Interface represents the exported methods for dealing with getting/setting
// kubernetes resources
type Interface interface {
	SetAnnotationsOnPod(namespace, podName string, annotations map[string]string) error
	SetAnnotationsOnNode(node *kapi.Node, annotations map[string]interface{}) error
	SetAnnotationsOnNamespace(namespace *kapi.Namespace, annotations map[string]string) error
	UnsetAnnotationsOnPod(namespace, podName string, annotations []string) error
	UpdateEgressFirewall(egressfirewall *egressfirewall.EgressFirewall) error
	UpdateEgressIP(eIP *egressipv1.EgressIP) error
	UpdateFloatingIP(fIP *floatingipv1.FloatingIP) error
	UpdateFloatingIPClaim(fic *floatingipclaimv1.FloatingIPClaim) error
	UpdateFloatingIPStatus(fIP *floatingipv1.FloatingIP) error
	UpdateNodeStatus(node *kapi.Node) error
	GetAnnotationsOnPod(namespace, name string) (map[string]string, error)
	GetNodes() (*kapi.NodeList, error)
	GetEgressIP(name string) (*egressipv1.EgressIP, error)
	GetEgressIPs() (*egressipv1.EgressIPList, error)
	GetEgressFirewalls() (*egressfirewall.EgressFirewallList, error)
	GetFloatingIP(name string) (*floatingipv1.FloatingIP, error)
	CreateFloatingIP(fi *floatingipv1.FloatingIP) (*floatingipv1.FloatingIP, error)
	DeleteFloatingIP(name string) error
	GetFloatingIPs() (*floatingipv1.FloatingIPList, error)
	GetFloatingIPClaim(name string) (*floatingipclaimv1.FloatingIPClaim, error)
	GetFloatingIPClaims() (*floatingipclaimv1.FloatingIPClaimList, error)
	GetNamespaces(labelSelector metav1.LabelSelector) (*kapi.NamespaceList, error)
	GetPods(namespace string, labelSelector metav1.LabelSelector) (*kapi.PodList, error)
	GetPod(namespace, name string) (*kapi.Pod, error)
	GetNode(name string) (*kapi.Node, error)
	GetEndpoint(namespace, name string) (*kapi.Endpoints, error)
	CreateEndpoint(namespace string, ep *kapi.Endpoints) (*kapi.Endpoints, error)
	Events() kv1core.EventInterface
}

// Kube is the structure object upon which the Interface is implemented
type Kube struct {
	KClient              kubernetes.Interface
	EIPClient            egressipclientset.Interface
	EgressFirewallClient egressfirewallclientset.Interface
	FIPClient            floatingipclientset.Interface
	FIPCClient           floatingipclaimclientset.Interface
}

// SetAnnotationsOnPod takes the pod object and map of key/value string pairs to set as annotations
func (k *Kube) SetAnnotationsOnPod(namespace, podName string, annotations map[string]string) error {
	var err error
	var patchData []byte
	patch := struct {
		Metadata map[string]interface{} `json:"metadata"`
	}{
		Metadata: map[string]interface{}{
			"annotations": annotations,
		},
	}

	podDesc := namespace + "/" + podName
	klog.Infof("Setting annotations %v on pod %s", annotations, podDesc)
	patchData, err = json.Marshal(&patch)
	if err != nil {
		klog.Errorf("Error in setting annotations on pod %s: %v", podDesc, err)
		return err
	}

	_, err = k.KClient.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		klog.Errorf("Error in setting annotation on pod %s: %v", podDesc, err)
	}
	return err
}

// SetAnnotationsOnNode takes the node object and map of key/value string pairs to set as annotations
func (k *Kube) SetAnnotationsOnNode(node *kapi.Node, annotations map[string]interface{}) error {
	var err error
	var patchData []byte
	patch := struct {
		Metadata map[string]interface{} `json:"metadata"`
	}{
		Metadata: map[string]interface{}{
			"annotations": annotations,
		},
	}

	klog.Infof("Setting annotations %v on node %s", annotations, node.Name)
	patchData, err = json.Marshal(&patch)
	if err != nil {
		klog.Errorf("Error in setting annotations on node %s: %v", node.Name, err)
		return err
	}

	_, err = k.KClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		klog.Errorf("Error in setting annotation on node %s: %v", node.Name, err)
	}
	return err
}

// SetAnnotationsOnNamespace takes the namespace object and map of key/value string pairs to set as annotations
func (k *Kube) SetAnnotationsOnNamespace(namespace *kapi.Namespace, annotations map[string]string) error {
	var err error
	var patchData []byte
	patch := struct {
		Metadata map[string]interface{} `json:"metadata"`
	}{
		Metadata: map[string]interface{}{
			"annotations": annotations,
		},
	}

	klog.Infof("Setting annotations %v on namespace %s", annotations, namespace.Name)
	patchData, err = json.Marshal(&patch)
	if err != nil {
		klog.Errorf("Error in setting annotations on namespace %s: %v", namespace.Name, err)
		return err
	}

	_, err = k.KClient.CoreV1().Namespaces().Patch(context.TODO(), namespace.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		klog.Errorf("Error in setting annotation on namespace %s: %v", namespace.Name, err)
	}
	return err
}

func (k *Kube) UnsetAnnotationsOnPod(namespace, podName string, annotations []string) error {
	var err error
	var patchData []byte
	patch := struct {
		Metadata map[string]interface{} `json:"metadata"`
	}{
		Metadata: map[string]interface{}{
			"annotations": func() map[string]interface{} {
				v := make(map[string]interface{})
				for _, annotation := range annotations {
					v[annotation] = nil
				}
				return v
			}() ,
		},
	}

	podDesc := namespace + "/" + podName
	klog.Infof("Setting annotations %v on pod %s", annotations, podDesc)
	patchData, err = json.Marshal(&patch)
	if err != nil {
		klog.Errorf("Error in setting annotations on pod %s: %v", podDesc, err)
		return err
	}

	if _, err = k.KClient.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.MergePatchType, patchData, metav1.PatchOptions{}); err != nil {
		klog.Errorf("Error in setting annotation on pod %s: %v", podDesc, err)
	}
	return err
}

// UpdateEgressFirewall updates the EgressFirewall with the provided EgressFirewall data
func (k *Kube) UpdateEgressFirewall(egressfirewall *egressfirewall.EgressFirewall) error {
	klog.Infof("Updating status on EgressFirewall %s in namespace %s", egressfirewall.Name, egressfirewall.Namespace)
	_, err := k.EgressFirewallClient.K8sV1().EgressFirewalls(egressfirewall.Namespace).Update(context.TODO(), egressfirewall, metav1.UpdateOptions{})
	return err
}

// UpdateEgressIP updates the EgressIP with the provided EgressIP data
func (k *Kube) UpdateEgressIP(eIP *egressipv1.EgressIP) error {
	klog.Infof("Updating status on EgressIP %s", eIP.Name)
	_, err := k.EIPClient.K8sV1().EgressIPs().Update(context.TODO(), eIP, metav1.UpdateOptions{})
	return err
}

// UpdateFloatingIP updates the FloatingIP with the provided FloatingIP status data
func (k *Kube) UpdateFloatingIP(fIP *floatingipv1.FloatingIP) error {
	klog.Infof("Updating status on FloatingIP %s", fIP.Name)
	_, err := k.FIPClient.K8sV1().FloatingIPs().Update(context.TODO(), fIP, metav1.UpdateOptions{})
	return err
}

func (k *Kube) UpdateFloatingIPClaim(fic *floatingipclaimv1.FloatingIPClaim) error {
	klog.Infof("Updating status on FloatingIPClaim %s", fic.Name)
	_, err := k.FIPCClient.K8sV1().FloatingIPClaims().Update(context.TODO(), fic, metav1.UpdateOptions{})
	return err
}

func (k *Kube) UpdateFloatingIPStatus(fIP *floatingipv1.FloatingIP) error {
	klog.Infof("Updating status on FloatingIP %s", fIP.Name)
	_, err := k.FIPClient.K8sV1().FloatingIPs().UpdateStatus(context.TODO(), fIP, metav1.UpdateOptions{})
	return err
}

// UpdateNodeStatus takes the node object and sets the provided update status
func (k *Kube) UpdateNodeStatus(node *kapi.Node) error {
	klog.Infof("Updating status on node %s", node.Name)
	_, err := k.KClient.CoreV1().Nodes().UpdateStatus(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Error in updating status on node %s: %v", node.Name, err)
	}
	return err
}

// GetAnnotationsOnPod obtains the pod annotations from kubernetes apiserver, given the name and namespace
func (k *Kube) GetAnnotationsOnPod(namespace, name string) (map[string]string, error) {
	pod, err := k.KClient.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod.ObjectMeta.Annotations, nil
}

// GetNamespaces returns the list of all Namespace objects matching the labelSelector
func (k *Kube) GetNamespaces(labelSelector metav1.LabelSelector) (*kapi.NamespaceList, error) {
	return k.KClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
}

// GetPods returns the list of all Pod objects in a namespace matching the labelSelector
func (k *Kube) GetPods(namespace string, labelSelector metav1.LabelSelector) (*kapi.PodList, error) {
	return k.KClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
}

// GetNodes returns the list of all Node objects from kubernetes
func (k *Kube) GetNodes() (*kapi.NodeList, error) {
	return k.KClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
}

// GetNode returns the pod resource from kubernetes apiserver, given its namespace and name
func (k *Kube) GetPod(namespace, name string) (*kapi.Pod, error) {
	return k.KClient.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// GetNode returns the Node resource from kubernetes apiserver, given its name
func (k *Kube) GetNode(name string) (*kapi.Node, error) {
	return k.KClient.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
}

// GetEgressIP returns the EgressIP object from kubernetes
func (k *Kube) GetEgressIP(name string) (*egressipv1.EgressIP, error) {
	return k.EIPClient.K8sV1().EgressIPs().Get(context.TODO(), name, metav1.GetOptions{})
}

// GetEgressIPs returns the list of all EgressIP objects from kubernetes
func (k *Kube) GetEgressIPs() (*egressipv1.EgressIPList, error) {
	return k.EIPClient.K8sV1().EgressIPs().List(context.TODO(), metav1.ListOptions{})
}

// GetEgressFirewalls returns the list of all EgressFirewall objects from kubernetes
func (k *Kube) GetEgressFirewalls() (*egressfirewall.EgressFirewallList, error) {
	return k.EgressFirewallClient.K8sV1().EgressFirewalls(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
}

func (k *Kube) GetFloatingIP(name string) (*floatingipv1.FloatingIP, error) {
	return k.FIPClient.K8sV1().FloatingIPs().Get(context.TODO(), name, metav1.GetOptions{})
}

func (k *Kube) CreateFloatingIP(fi *floatingipv1.FloatingIP) (*floatingipv1.FloatingIP, error) {
	return k.FIPClient.K8sV1().FloatingIPs().Create(context.TODO(), fi, metav1.CreateOptions{})
}

func (k *Kube) DeleteFloatingIP(name string) error {
	return k.FIPClient.K8sV1().FloatingIPs().Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (k *Kube) GetFloatingIPs() (*floatingipv1.FloatingIPList, error) {
	return k.FIPClient.K8sV1().FloatingIPs().List(context.TODO(), metav1.ListOptions{})
}

func (k *Kube) GetFloatingIPClaim(name string) (*floatingipclaimv1.FloatingIPClaim, error) {
	return k.FIPCClient.K8sV1().FloatingIPClaims().Get(context.TODO(), name, metav1.GetOptions{})
}

func (k *Kube) GetFloatingIPClaims() (*floatingipclaimv1.FloatingIPClaimList, error) {
	return k.FIPCClient.K8sV1().FloatingIPClaims().List(context.TODO(), metav1.ListOptions{})
}

// GetEndpoint returns the Endpoints resource
func (k *Kube) GetEndpoint(namespace, name string) (*kapi.Endpoints, error) {
	return k.KClient.CoreV1().Endpoints(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// CreateEndpoint creates the Endpoints resource
func (k *Kube) CreateEndpoint(namespace string, ep *kapi.Endpoints) (*kapi.Endpoints, error) {
	return k.KClient.CoreV1().Endpoints(namespace).Create(context.TODO(), ep, metav1.CreateOptions{})
}

// Events returns events to use when creating an EventSinkImpl
func (k *Kube) Events() kv1core.EventInterface {
	return k.KClient.CoreV1().Events("")
}
