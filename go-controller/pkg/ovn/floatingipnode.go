package ovn

import (
	"github.com/google/uuid"
	floatingipapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func (oc *Controller) addFloatingIPNode(node *v1.Node) {
	klog.V(5).Infof("floating ip node: %s about to be added", node)
	oc.fIPCC.fipOnPodLock.Lock()
	defer oc.fIPCC.fipOnPodLock.Unlock()
	oc.retryFipInClaim(nil, nil)
}

func (oc *Controller) deleteFloatingIPNode(node *v1.Node) {
	klog.V(5).Infof("floating ip node: %s about to be removed", node)
	fiObjs, err := oc.kube.GetFloatingIPs()
	if err != nil {
		klog.Errorf("floating ip node: unable to list floating ip on node: %s err: %v", node.Name, err)
		return
	}
	for _, fiObj := range fiObjs.Items {
		if fiObj.Status.NodeName == node.Name {
			if err := oc.kube.DeleteFloatingIP(fiObj.Name); err != nil {
				klog.Errorf("floating ip claim: unable to remove floating ip %s on node: %s err: %v", fiObj.Name, node.Name, err)
			}
			obj := &floatingipapi.FloatingIP{
				ObjectMeta: metav1.ObjectMeta{
					Name: uuid.New().String(),
					/*OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(fic, floatingipclaimapi.SchemeGroupVersion.WithKind("FloatingIPClaim")),
					},*/
				},
				Spec: floatingipapi.FloatingIPSpec{
					Pod: fiObj.Spec.Pod,
					PodNamespace: fiObj.Spec.PodNamespace,
					FloatingIPClaim: fiObj.Spec.FloatingIPClaim,
				},
			}
			if _, err := oc.kube.CreateFloatingIP(obj); err != nil {
				klog.Errorf("floating ip node: unable to create retry floating ip %s", obj.Name)
			}
		}
	}
}

func (oc *Controller) isFloatingIPNodeReady(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}

func (oc *Controller) isFloatingIPNodeReachable(node *v1.Node) bool {
	reachable := false
	v4IfAddr, _, err := util.ParseNodePrimaryIfAddr(node)
	if err != nil {
		klog.Errorf("unable to use node for floating ip assignment, err: %v", err)
		return reachable
	}

	if v4IfAddr != "" {
		v4IP, _, err := net.ParseCIDR(v4IfAddr)
		if err != nil {
			klog.Errorf("Unable to resolve network address, err: %v", err)
			return reachable
		}
		reachable = dialer.dial(v4IP)
	}
	return reachable
}

type floatingIPNodeController struct {
	lock *sync.Mutex
	allocations map[string]int
}

func (finc *floatingIPNodeController) AddNode(node *v1.Node) {
	util.AddGARP(types.EXTSwitchToGWRouterPrefix + types.GWRouterPrefix + node.Name)

	finc.lock.Lock()
	defer finc.lock.Unlock()

	if _, ok := finc.allocations[node.Name]; !ok {
		finc.allocations[node.Name] = 0
	}
}

func (finc *floatingIPNodeController) DeleteNode(node *v1.Node) {
	nodeEgressLabel := util.GetNodeEgressLabel()
	nodeLabels := node.GetLabels()
	if _, hasEgressLabel := nodeLabels[nodeEgressLabel]; !hasEgressLabel {
		util.DeleteGARP(types.EXTSwitchToGWRouterPrefix + types.GWRouterPrefix + node.Name);
	}

	finc.lock.Lock()
	defer finc.lock.Unlock()

	delete(finc.allocations, node.Name)
}

func (finc *floatingIPNodeController) Allocate() string {
	finc.lock.Lock()
	defer finc.lock.Unlock()

	min := 0
	node := ""
	for name, count := range finc.allocations {
		if min == 0 || min > count {
			min = count
			node = name
		}
	}
	if node != "" {
		finc.allocations[node] += 1
	}
	return node
}

func (finc *floatingIPNodeController) AllocateOnNode(nodeName string) bool {
	finc.lock.Lock()
	defer finc.lock.Unlock()

	if _, ok := finc.allocations[nodeName]; ok {
		finc.allocations[nodeName] += 1
		return true
	}
	return false
}

func (finc *floatingIPNodeController) Release(nodeName string) {
	finc.lock.Lock()
	defer finc.lock.Unlock()

	if _, ok := finc.allocations[nodeName]; ok {
		if finc.allocations[nodeName] > 0 {
			finc.allocations[nodeName] -= 1
		}
	}
}