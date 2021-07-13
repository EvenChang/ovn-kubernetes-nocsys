package ovn

import (
	"net"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

type floatingIPNodeController struct {
	ficc *floatingIPClaimController

	client kube.Interface
	queue workqueue.RateLimitingInterface
	workerLoopPeriod time.Duration
}


func (finc *floatingIPNodeController) addFloatingIPNode(obj interface{}) {
	node := obj.(*v1.Node)
	klog.V(5).InfoS("Adding floating ip node", "floatingipnode", klog.KObj(node))

	nodeLabels := node.GetLabels()
	if _, hasLabel := nodeLabels[util.GetNodeFloatingIPLabel()]; hasLabel {
		isReady := finc.isFloatingIPNodeReady(node)
		isReachable := finc.isFloatingIPNodeReachable(node)
		if isReady && isReachable {
			finc.AddNode(node)
			ficObjs, err := finc.client.GetFloatingIPClaims()
			if err != nil {
				finc.queue.Add(node.Name)
				return
			}
			for _, ficObj := range ficObjs.Items {
				if len(ficObj.Status.AssignedIPs) == 0 {
					finc.ficc.queue.Add(ficObj.Name)
				}
			}
		}
	}
}

func (finc *floatingIPNodeController) updateFloatingIPClaim(old, cur interface{}) {
	oldNode := old.(*v1.Node)
	curNode := cur.(*v1.Node)
	klog.V(5).InfoS("Updating floating ip node", "floatingipnode", klog.KObj(oldNode))
	oldLabels := oldNode.GetLabels()
	curLabels := curNode.GetLabels()
	_, oldHasLabel := oldLabels[util.GetNodeFloatingIPLabel()]
	_, curHasLabel := curLabels[util.GetNodeFloatingIPLabel()]
	if !oldHasLabel && !curHasLabel {
		return
	}
	if oldHasLabel && !curHasLabel {
		klog.Infof("Node: %s has been un-labelled, deleting it from floating ip assignment", curNode.Name)
		finc.deleteFloatingIPNode(old)
		return
	}
	isOldReady := finc.isFloatingIPNodeReady(oldNode)
	isCurReady := finc.isFloatingIPNodeReady(curNode)
	isCurReachable := finc.isFloatingIPNodeReachable(curNode)
	if !oldHasLabel && curHasLabel {
		klog.Infof("Node: %s has been labelled, adding it for floating ip assignment", curNode.Name)
		finc.addFloatingIPNode(cur)
		return
	}
	if isOldReady == isCurReady {
		return
	}
	if !isCurReady {
		klog.Warningf("Node: %s is not ready, deleting it from floating ip assignment", curNode.Name)
		finc.deleteFloatingIPNode(cur)
	} else if isCurReady && isCurReachable {
		klog.Infof("Node: %s is ready and reachable, adding it for floating ip assignment", curNode.Name)
		finc.addFloatingIPNode(cur)
	}
}

func (finc *floatingIPNodeController) deleteFloatingIPNode(obj interface{}) {
	node := obj.(*v1.Node)
	klog.V(5).InfoS("Deleting floating ip node", "floatingipnode", klog.KObj(node))
	nodeLabels := node.GetLabels()
	if _, hasLabel := nodeLabels[util.GetNodeFloatingIPLabel()]; hasLabel {
		finc.DeleteNode(node)
		fiObjs, err := finc.client.GetFloatingIPs()
		if err != nil {
			finc.queue.Add(node.Name)
			return
		}
		for _, fiObj := range fiObjs.Items {
			if fiObj.Status.NodeName == node.Name {
				if err = finc.client.DeleteFloatingIP(fiObj.Name); err != nil {
					finc.queue.Add(node.Name)
				} else {
					finc.ficc.queue.AddRateLimited(fiObj.Spec.FloatingIPClaim)
				}
			}
		}
	}
}

func (finc *floatingIPNodeController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer finc.queue.ShutDown()

	klog.Infof("Starting floating ip node controller")
	defer klog.Infof("Shutting down floating ip node controller")

	go wait.Until(finc.worker, finc.workerLoopPeriod, stopCh)

	<-stopCh
}

func (finc *floatingIPNodeController) worker() {
	for finc.processNextWorkItem() {
	}
}

func (finc *floatingIPNodeController) processNextWorkItem() bool {
	eKey, quit := finc.queue.Get()
	if quit {
		return false
	}
	defer finc.queue.Done(eKey)

	err := finc.sync(eKey.(string))
	finc.handleErr(err, eKey)

	return true
}

func (finc *floatingIPNodeController) handleErr(err error, key interface{}) {
	if err == nil {
		finc.queue.Forget(key)
		return
	}

	if finc.queue.NumRequeues(key) < maxRetries {
		klog.V(2).InfoS("Error syncing floating ip node, retrying", "floatingipnode", key, "err", err)
		finc.queue.AddRateLimited(key)
		return
	}

	klog.Warningf("Dropping floatingipnode %q out of the queue: %v", key, err)
	finc.queue.Forget(key)
	utilruntime.HandleError(err)
}

func (finc *floatingIPNodeController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing floating ip node %q. (%v)", key, time.Since(startTime))
	}()

	node, err := finc.client.GetNode(key)
	if err != nil {
		//if !errors.IsNotFound(err)
		if !strings.HasSuffix(err.Error(), "not found") {
			return err
		}

		fiObjs, err := finc.client.GetFloatingIPs()
		if err != nil {
			return err
		}
		for _, fiObj := range fiObjs.Items {
			if fiObj.Status.NodeName == node.Name {
				if err = finc.client.DeleteFloatingIP(fiObj.Name); err != nil {
					finc.queue.AddRateLimited(node.Name)
				} else {
					finc.ficc.queue.AddRateLimited(fiObj.Spec.FloatingIPClaim)
				}
			}
		}
	}

	nodeLabels := node.GetLabels()
	if _, hasLabel := nodeLabels[util.GetNodeFloatingIPLabel()]; hasLabel {
		ficObjs, err := finc.client.GetFloatingIPClaims()
		if err != nil {
			return err
		}
		for _, ficObj := range ficObjs.Items {
			if len(ficObj.Status.AssignedIPs) == 0 {
				finc.ficc.queue.Add(ficObj.Name)
			}
		}
	}

	return nil
}

func (finc *floatingIPNodeController) AddNode(node *v1.Node) {
	util.AddGARP(types.EXTSwitchToGWRouterPrefix + types.GWRouterPrefix + node.Name)

	finc.ficc.nodeMutex.Lock()
	defer finc.ficc.nodeMutex.Unlock()
	if _, ok := finc.ficc.fiOnNodes[node.Name]; !ok {
		finc.ficc.fiOnNodes[node.Name] = 0
	}
}

func (finc *floatingIPNodeController) DeleteNode(node *v1.Node) {
	nodeEgressLabel := util.GetNodeEgressLabel()
	nodeLabels := node.GetLabels()
	if _, hasEgressLabel := nodeLabels[nodeEgressLabel]; !hasEgressLabel {
		util.DeleteGARP(types.EXTSwitchToGWRouterPrefix + types.GWRouterPrefix + node.Name);
	}

	finc.ficc.nodeMutex.Lock()
	defer finc.ficc.nodeMutex.Unlock()
	delete(finc.ficc.fiOnNodes, node.Name)
}

func (finc *floatingIPNodeController) isFloatingIPNodeReady(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}

func (finc *floatingIPNodeController) isFloatingIPNodeReachable(node *v1.Node) bool {
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