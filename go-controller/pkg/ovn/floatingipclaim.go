package ovn

import (
	"fmt"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	floatingipapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	floatingipclaimapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/ovn/floatingipallocator"
	kapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type PodID struct {
	name string
	namespace string
}

type floatingIPClaimController struct {
	client kube.Interface
	watchFactory     *factory.WatchFactory
	eventRecorder    record.EventRecorder

	queue workqueue.RateLimitingInterface
	workerLoopPeriod time.Duration

	ficMutex *sync.RWMutex
	allocators map[string]floatingipallocator.Interface // floating ip alloctor for provider
	suballocators map[string]floatingipallocator.Interface // floating ip alloctor for floating ip claim

	nodeMutex *sync.Mutex
	fiOnNodes map[string]int // store floating ip node

	nsMutex *sync.Mutex
	nsHandlers map[string]factory.Handler // store namespace watch for floating ip claim

	podMutex *sync.Mutex
	podHanders map[string]factory.Handler // store pod(under namespace) watch for floating ip claim
}

func (ficc *floatingIPClaimController) addFloatingIPClaim(obj interface{}) bool {
	fic := obj.(*floatingipclaimapi.FloatingIPClaim)
	klog.V(5).InfoS("Adding floating ip claim", "floatingipclaim", klog.KObj(fic))
	ficc.nsMutex.Lock()
	defer ficc.nsMutex.Unlock()
	sel, err := metav1.LabelSelectorAsSelector(&fic.Spec.NamespaceSelector)
	if err != nil {
		ficc.eventRecorder.Eventf(fic, kapi.EventTypeWarning, "FailedToInitFloatingIPClaim", "Failed to add namespace observer for floating ip claim %s", fic.Name)
		return false
	}
	handler := ficc.watchFactory.AddFilteredNamespaceHandler("", sel,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				namespace := obj.(*kapi.Namespace)
				klog.V(5).Infof("Adding namespace for floating ip claim", "floatingipclaim", klog.KObj(fic), "namespace", klog.KObj(namespace))
				ficc.podMutex.Lock()
				defer ficc.podMutex.Unlock()
				sel, err = metav1.LabelSelectorAsSelector(&fic.Spec.PodSelector)
				if err != nil {
					ficc.eventRecorder.Eventf(fic, kapi.EventTypeWarning, "FailedToInitFloatingIPClaim", "Failed to add pod observer for floating ip claim %s", fic.Name)
					return
				}
				h := ficc.watchFactory.AddFilteredPodHandler(namespace.Name, sel,
					cache.ResourceEventHandlerFuncs{
						AddFunc: func(obj interface{}) {
							ficc.queue.AddRateLimited(fic.Name)
						},
						UpdateFunc: func(old, cur interface{}) {
							ficc.queue.AddRateLimited(fic.Name)
						},
						DeleteFunc: func(obj interface{}) {
							ficc.queue.AddRateLimited(fic.Name)
						},
					}, nil)
				ficc.podHanders[fmt.Sprintf("%s/%s", fic.Name, namespace.Name)] = *h
			},
			DeleteFunc: func(obj interface{}) {
				namespace := obj.(*kapi.Namespace)
				klog.V(5).Infof("Deleting namespace from floating ip claim", "floatingipclaim", klog.KObj(fic), "namespace", klog.KObj(namespace))
				key := fmt.Sprintf("%s/%s", fic.Name, namespace.Name)
				ficc.podMutex.Lock()
				defer ficc.podMutex.Unlock()
				if h, exist := ficc.podHanders[key]; exist {
					ficc.watchFactory.RemovePodHandler(&h)
					delete(ficc.podHanders, key)
				}
				ficc.queue.Add(fic.Name)
			},
		}, nil)
	ficc.nsHandlers[fic.Name] = *handler
	return true
}

func (ficc *floatingIPClaimController) deleteFloatingIPClaim(obj interface{}) {
	fic := obj.(*floatingipclaimapi.FloatingIPClaim)
	klog.V(5).InfoS("Deleting floating ip claim", "floatingipclaim", klog.KObj(fic))
	ficc.nsMutex.Lock()
	defer ficc.nsMutex.Unlock()
	if handler, exist := ficc.nsHandlers[fic.Name]; exist {
		ficc.watchFactory.RemoveNamespaceHandler(&handler)
		delete(ficc.nsHandlers, fic.Name)

		ficc.podMutex.Lock()
		defer ficc.podMutex.Unlock()
		for key, h := range ficc.podHanders {
			parts := strings.Split(key, "/")
			if parts[0] == fic.Name {
				ficc.watchFactory.RemovePodHandler(&h)
				delete(ficc.podHanders, key)
			}
		}
	}
	ficc.queue.Add(fic.Name)
}

func (ficc *floatingIPClaimController) deleteFloatingIP(obj interface{}) {
	fi := obj.(*floatingipapi.FloatingIP)
	klog.V(5).InfoS("Deleting floating ip", "floatingip", klog.KObj(fi))
	ficc.ficMutex.Lock()
	defer ficc.ficMutex.Unlock()
	suballocator, ok := ficc.suballocators[fi.Spec.FloatingIPClaim]
	if ok {
		if fi.Status.FloatingIP != "" {
			suballocator.Release(net.ParseIP(fi.Status.FloatingIP))
		}
		if fi.Status.NodeName != "" {
			ficc.Release(fi.Status.NodeName)
		}
		if suballocator.Free() > 0 {
			ficc.queue.AddRateLimited(fi.Spec.FloatingIPClaim)
		}
	}
}

func (ficc *floatingIPClaimController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer ficc.queue.ShutDown()

	klog.Infof("Starting floating ip claim controller")
	defer klog.Infof("Shutting down floating ip claim controller")

	go wait.Until(ficc.worker, ficc.workerLoopPeriod, stopCh)

	<-stopCh
}

func (ficc *floatingIPClaimController) worker() {
	for ficc.processNextWorkItem() {
	}
}

func (ficc *floatingIPClaimController) processNextWorkItem() bool {
	eKey, quit := ficc.queue.Get()
	if quit {
		return false
	}
	defer ficc.queue.Done(eKey)

	err := ficc.sync(eKey.(string))
	ficc.handleErr(err, eKey)

	return true
}

func (ficc *floatingIPClaimController) handleErr(err error, key interface{}) {
	if err == nil {
		ficc.queue.Forget(key)
		return
	}

	if ficc.queue.NumRequeues(key) < maxRetries {
		klog.V(2).InfoS("Error syncing floating ip claim, retrying", "floatingipclaim", key, "err", err)
		ficc.queue.AddRateLimited(key)
		return
	}

	klog.Warningf("Dropping floating ip claim %q out of the queue: %v", key, err)
	ficc.queue.Forget(key)
	utilruntime.HandleError(err)
}

func (ficc *floatingIPClaimController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing floating ip claim %q. (%v)", key, time.Since(startTime))
	}()

	ficObj, err := ficc.client.GetFloatingIPClaim(key)
	if err != nil {
		//if !errors.IsNotFound(err)
		if !strings.HasSuffix(err.Error(), "not found") {
			return err
		}

		// floating ip claim has been deleted
		fiObjs, err := ficc.client.GetFloatingIPs()
		if err != nil {
			return err
		}
		for _, fiObj := range fiObjs.Items {
			if fiObj.Spec.FloatingIPClaim == ficObj.Name {
				if err := ficc.client.DeleteFloatingIP(fiObj.Name); err != nil {
					return err
				}
			}
		}
		return nil
	}

	var allocator floatingipallocator.Interface
	{
		ficc.ficMutex.RLock()
		defer ficc.ficMutex.RUnlock()
		allocator = ficc.suballocators[ficObj.Name]
		if allocator == nil {
			// floating ip claim not ready
			fiObjs, err := ficc.client.GetFloatingIPs()
			if err != nil {
				return err
			}
			for _, fiObj := range fiObjs.Items {
				if fiObj.Spec.FloatingIPClaim == ficObj.Name {
					if err := ficc.client.DeleteFloatingIP(fiObj.Name); err != nil {
						return err
					}
				}
			}
			return nil
		}
	}

	klog.V(5).Infof("About to update fips for floating ip claim %q", key)
	// Gets the floating ip object owned by the floating ip claim
	fiObjs, err := ficc.client.GetFloatingIPs()
	if err != nil {
		return err
	}
	fiOnPods := make(map[PodID]floatingipapi.FloatingIP)
	for _, fiObj := range fiObjs.Items {
		if fiObj.Spec.FloatingIPClaim == ficObj.Name {
			fiOnPods[PodID{fiObj.Spec.Pod, fiObj.Spec.PodNamespace}] = fiObj
		}
	}

	// Gets the pod selected by the floating ip claim
	filteredPods := make(map[PodID]kapi.Pod)
	if namespaces, err := ficc.client.GetNamespaces(ficObj.Spec.NamespaceSelector); err != nil {
		return err
	} else {
		for _, namespace := range namespaces.Items {
			if pods, err := ficc.client.GetPods(namespace.Name, ficObj.Spec.PodSelector); err != nil {
				return err
			} else {
				for _, pod := range pods.Items {
					filteredPods[PodID{pod.Name, pod.Namespace}] = pod
				}
			}
		}
	}

	fic := ficObj.DeepCopy()
	fic.Status.AssignedIPs = []string{}
	fic.Status.Phase = floatingipclaimapi.FloatingIPClaimReady
	defer func() {
		if err = ficc.client.UpdateFloatingIPClaim(fic); err != nil {
			ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeWarning, "FailedToUpdateFloatingIPClaim", "Failed to update status for floating ip claim: %v", err)
			ficc.queue.AddRateLimited(fic.Name)
		}
	}()
	// Clean up floating ip
	if allocator.Used() == 0 {
		// The spec of floating ip claim changes and the ip allocator recreates the scene
		for id, fiOnPod := range fiOnPods {
			if fiOnPod.Status.FloatingIP != "" {
				ip := net.ParseIP(fiOnPod.Status.FloatingIP)
				if allocator.Has(ip) {
					if err = allocator.Allocate(ip); err == nil {
						fic.Status.AssignedIPs = append(fic.Status.AssignedIPs, fiOnPod.Status.FloatingIP)
						continue
					}
				}
			}
			if err = ficc.client.DeleteFloatingIP(fiOnPod.Name); err != nil {
				//if !errors.IsNotFound(err)
				if !strings.HasSuffix(err.Error(), "not found") {
					ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeWarning, "FailedToDeleteFloatingIP", "Failed to delete fip from floating ip claim %s: %v", ficObj.Name, err)
					return err
				}
			}
			delete(fiOnPods, id)
		}
	} else {
		// Handle the add and delete of pod objects
		var fiOnPods2del []floatingipapi.FloatingIP
		for id, fiOnPod := range fiOnPods {
			if _, ok := filteredPods[id]; !ok {
				// Pod has been deleted
				fiOnPods2del = append(fiOnPods2del, fiOnPod)
				delete(fiOnPods, id)
				continue
			}

			if fiOnPod.Status.NodeName != "" {
				// floating ip label removed
				if !ficc.HasNode(fiOnPod.Status.NodeName) {
					fiOnPods2del = append(fiOnPods2del, fiOnPod)
					delete(fiOnPods, id)
					continue
				}
			}
			if fiOnPod.Status.FloatingIP != "" {
				fic.Status.AssignedIPs = append(fic.Status.AssignedIPs, fiOnPod.Status.FloatingIP)
			}
		}
		for _, fiOnPod := range fiOnPods2del {
			err = ficc.client.DeleteFloatingIP(fiOnPod.Name)
			if err != nil {
				//if !errors.IsNotFound(err)
				if !strings.HasSuffix(err.Error(), "not found") {
					ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeWarning, "FailedToDeleteFloatingIP", "Failed to delete fip from floating ip claim %s: %v", ficObj.Name, err)
					return err
				}
			}
			if fiOnPod.Status.FloatingIP != "" {
				allocator.Release(net.ParseIP(fiOnPod.Status.FloatingIP))
			}
			if fiOnPod.Status.NodeName != "" {
				ficc.Release(fiOnPod.Status.NodeName)
			}
		}
	}

	// Need to create floating ip for pod
	var fiOnPods2crt []kapi.Pod
	for id, pod := range filteredPods {
		if _, ok := fiOnPods[id]; !ok {
			fiOnPods2crt = append(fiOnPods2crt, pod)
		}
	}
	for _, pod := range fiOnPods2crt {
		var ip net.IP
		ok := false
		ip, err = allocator.AllocateNext()
		if err != nil {
			ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeNormal, "FailedToCreateFloatingIP", "No free floating ip for pod %s/%s", pod.Namespace, pod.Name)
			return nil
		}
		defer func() {
			if !ok {
				allocator.Release(ip)
			}
		}()
		node := ficc.Allocate()
		if node == "" {
			ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeNormal, "FailedToCreateFloatingIP", "No free floating ip node for pod %s/%s", pod.Namespace, pod.Name)
			return nil
		}
		defer func() {
			if !ok {
				ficc.Release(node)
			}
		}()

		if len(pod.Status.PodIPs) == 0 {
			ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeNormal, "FailedToCreateFloatingIP", "Pod %s/%s has not been assigned ip address", pod.Namespace, pod.Name)
			continue
		}
		fip := &floatingipapi.FloatingIP{
			ObjectMeta: metav1.ObjectMeta{
				Name: uuid.New().String(),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(ficObj, floatingipclaimapi.SchemeGroupVersion.WithKind("FloatingIPClaim")),
				},
			},
			Spec:       floatingipapi.FloatingIPSpec{
				FloatingIPClaim: ficObj.Name,
				Pod:             pod.Name,
				PodNamespace:    pod.Namespace,
			},
			Status:     floatingipapi.FloatingIPStatus{
				NodeName:    node,
				FloatingIP:  ip.To4().String(),
				HostNetwork: pod.Spec.HostNetwork,
				Phase:       floatingipapi.FloatingIPSucceeded,
			},
		}
		for _, PodIP := range pod.Status.PodIPs {
			fip.Status.PodIPs = append(fip.Status.PodIPs, PodIP.IP)
		}
		_, err = ficc.client.CreateFloatingIP(fip)
		if err != nil {
			ficc.eventRecorder.Eventf(ficObj, kapi.EventTypeWarning, "FailedToCreateFloatingIP", "Failed to create fip on Pod %s/%s: %v", pod.Namespace, pod.Name, err)
			ficc.queue.AddRateLimited(ficObj.Name)
			continue
		}
		fic.Status.AssignedIPs = append(fic.Status.AssignedIPs, ip.To4().String())
		ok = true
	}

	return nil
}

func (ficc *floatingIPClaimController) Allocate() string {
	ficc.nodeMutex.Lock()
	defer ficc.nodeMutex.Unlock()

	min := 0
	node := ""
	for name, count := range ficc.fiOnNodes {
		if min == 0 || min > count {
			min = count
			node = name
		}
	}
	if node != "" {
		ficc.fiOnNodes[node] += 1
	}
	return node
}

func (ficc *floatingIPClaimController) AllocateOnNode(nodeName string) bool {
	ficc.nodeMutex.Lock()
	defer ficc.nodeMutex.Unlock()

	if _, ok := ficc.fiOnNodes[nodeName]; ok {
		ficc.fiOnNodes[nodeName] += 1
		return true
	}
	return false
}

func (ficc *floatingIPClaimController) Release(nodeName string) {
	ficc.nodeMutex.Lock()
	defer ficc.nodeMutex.Unlock()

	if _, ok := ficc.fiOnNodes[nodeName]; ok {
		if ficc.fiOnNodes[nodeName] > 0 {
			ficc.fiOnNodes[nodeName] -= 1
		}
	}
}

func (ficc *floatingIPClaimController) HasNode(nodeName string) bool {
	ficc.nodeMutex.Lock()
	defer ficc.nodeMutex.Unlock()

	_, ok := ficc.fiOnNodes[nodeName]
	return ok
}