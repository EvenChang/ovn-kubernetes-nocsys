package ovn

import (
	"fmt"
	"github.com/google/uuid"
	"net"
	"reflect"
	"sync"

	floatingipapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	floatingipclaimapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/ovn/floatingipallocator"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	fipOnPodAnnotationName = "k8s.ovn.org/floatingipclaim"
)

func (oc *Controller) addFloatingIPClaim(fic *floatingipclaimapi.FloatingIPClaim) error {
	klog.V(5).Infof("floating ip claim: %s about to be added", fic.Name)
	oc.fIPCC.operLock.Lock()
	defer oc.fIPCC.operLock.Unlock()

	runtime := &floatingIPClaimRuntime{
		ipAllocator: nil,
		podLock: &sync.Mutex{},
		podHandlers: make(map[string]factory.Handler),
	}
	if ipAllocator, err := floatingipallocator.NewAllocatorRange(fic.Spec.FloatingIPs); err != nil {
		return err
	} else {
		runtime.ipAllocator = ipAllocator
		oc.fIPCC.runtimes[fic.Name] = runtime
	}

	ns, err := metav1.LabelSelectorAsSelector(&fic.Spec.NamespaceSelector)
	if err != nil {
		return fmt.Errorf("invalid namespace selector on floating ip claim %s: %v", fic.Name, err)
	}
	handler := oc.watchFactory.AddFilteredNamespaceHandler("", ns,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				namespace := obj.(*v1.Namespace)
				klog.V(5).Infof("floating ip claim: namespace %s/%s about to be added", fic.Name, namespace.Name)
				if err := oc.watchPodsInClaim(runtime, fic, namespace); err != nil {
					klog.Errorf("error: unable to add pod handler for floating ip claim: %s, err: %v", fic.Name, err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				namespace := obj.(*v1.Namespace)
				klog.V(5).Infof("floating ip claim: namespace %s/%s about to be removed", fic.Name, namespace.Name)
				oc.unwatchPodsInClaim(runtime, namespace)
				if err := oc.deleteFipOnPodsInClaim(fic, namespace); err != nil {
					klog.Errorf("error: unable to delete floating ip for floating ip claim: %s err: %v", fic.Name, err)
				}
			},
		}, nil)
	oc.fIPCC.namespaceHandlers[fic.Name] = *handler

	return nil
}

func (oc *Controller) deleteFloatingIPClaim(fic *floatingipclaimapi.FloatingIPClaim) error {
	klog.V(5).Infof("floating ip claim: %s about to be removed", fic.Name)
	oc.fIPCC.operLock.Lock()
	defer oc.fIPCC.operLock.Unlock()

	if handler, exist := oc.fIPCC.namespaceHandlers[fic.Name]; exist {
		oc.watchFactory.RemoveNamespaceHandler(&handler)
		delete(oc.fIPCC.namespaceHandlers, fic.Name)
	}

	if runtime, exist := oc.fIPCC.runtimes[fic.Name]; exist {
		if namespaces, err := oc.kube.GetNamespaces(fic.Spec.NamespaceSelector); err != nil {
			klog.Errorf("error: unable to list namespaces for floating ip claim: %s, err: %v", fic.Name, err)
		} else {
			for _, namespace := range namespaces.Items {
				oc.unwatchPodsInClaim(runtime, &namespace)
			}
		}
		delete(oc.fIPCC.runtimes, fic.Name)
	}

	if err := oc.deleteFipsInClaim(fic); err != nil {
		klog.Errorf("error: unable to delete namespace handler for floating ip claim: %s, err: %v", fic.Name, err)
		return err
	}
	return nil
}

func (oc *Controller) addFloatingIPToClaim(ficName string, fi *floatingipapi.FloatingIP) error {
	klog.V(5).Infof("floating ip claim: floating ip %s about to be added", fi.Name)
	runtime := oc.fIPCC.GetRuntime(ficName)

	oc.fIPCC.fipOnPodLock.Lock()
	defer oc.fIPCC.fipOnPodLock.Unlock()
	if runtime == nil {
		klog.Errorf("unknow floating ip add action after claim deleted")
		return nil
	}
	if err := oc.verifyFipInClaim(runtime, fi); err != nil {
		return err
	}

	// update status field
	if ficObj, err := oc.kube.GetFloatingIPClaim(fi.Spec.FloatingIPClaim); err != nil {
		klog.Errorf("floating ip claim: unable to get floating ip claim: %s err: %s", fi.Spec.FloatingIPClaim, err)
	} else {
		obj := ficObj.DeepCopy()
		skip := false
		for _, status := range obj.Status.Items {
			if status.Node == fi.Spec.NodeName {
				for _, ip := range status.AssignedIPs {
					if ip == fi.Spec.FloatingIP {
						skip = true
					}
				}
				if !skip {
					status.AssignedIPs = append(status.AssignedIPs, fi.Spec.FloatingIP)
					skip = true
				}
			}
		}
		if !skip {
			obj.Status.Items = append(obj.Status.Items, floatingipclaimapi.FloatingIPClaimStatusItem{
				Node:        fi.Spec.NodeName,
				AssignedIPs: []string{fi.Spec.FloatingIP},
			})
		}
		if err := oc.kube.UpdateFloatingIPClaim(obj); err != nil {
			klog.Errorf("floating ip claim: unable to update floating ip claim status: %s err: %s", fi.Spec.FloatingIPClaim, err)
		}
	}
	return nil
}

func (oc *Controller) deleteFloatingIPInClaim(ficName string, fi *floatingipapi.FloatingIP) error {
	klog.V(5).Infof("floating ip claim: floating ip %s about to be removed", fi.Name)
	// release resources
	if fi.Spec.NodeName != "" {
		oc.fIPNC.Release(fi.Spec.NodeName)
	}
	runtime := oc.fIPCC.GetRuntime(ficName)
	if runtime != nil && fi.Spec.FloatingIP != "" {
		runtime.ipAllocator.Release(net.ParseIP(fi.Spec.FloatingIP))

		if ficObj, err := oc.kube.GetFloatingIPClaim(fi.Spec.FloatingIPClaim); err != nil {
			klog.Errorf("floating ip claim: unable to get floating ip claim: %s err: %s", fi.Spec.FloatingIPClaim, err)
		} else {
			obj := ficObj.DeepCopy()
			skip := true
			for _, status := range obj.Status.Items {
				if status.Node == fi.Spec.NodeName {
					for i, ip := range status.AssignedIPs {
						if ip == fi.Spec.FloatingIP {
							status.AssignedIPs = append(status.AssignedIPs[:i], status.AssignedIPs[i + 1:]...)
							skip = false
						}
					}
				}
			}
			if !skip {
				if err := oc.kube.UpdateFloatingIPClaim(obj); err != nil {
					klog.Errorf("floating ip claim: unable to update floating ip claim status: %s err: %s", fi.Spec.FloatingIPClaim, err)
				}
			}
		}
	}

	oc.fIPCC.fipOnPodLock.Lock()
	defer oc.fIPCC.fipOnPodLock.Unlock()
	id := PodID{
		name: fi.Spec.Pod,
		namespace: fi.Spec.PodNamespace,
	}
	fipId := FipID{
		fipName: fi.Name,
		ficName: ficName,
	}
	if _, exist := oc.fIPCC.fipOnPodCache[id]; exist {
		if err := oc.kube.UnsetAnnotationsOnPod(id.namespace, id.name, []string{fipOnPodAnnotationName}); err != nil {
			klog.V(5).Infof("floating ip claim: unable to unmark pod: %s/%s err: %s", id.namespace, id.name, err)
		}
		delete(oc.fIPCC.fipOnPodCache, id)
	} else {
		delete(oc.fIPCC.fipOnPodPendingCache, fipId)
	}

	return oc.retryFipInClaim(&fipId, &id)
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * helper
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// monitor the pods that match the floating IP claim selector, and annotate the pods
func (oc *Controller) watchPodsInClaim(runtime *floatingIPClaimRuntime, fic *floatingipclaimapi.FloatingIPClaim, namespace *v1.Namespace) error {
	runtime.podLock.Lock()
	defer runtime.podLock.Unlock()
	sel, err := metav1.LabelSelectorAsSelector(&fic.Spec.PodSelector)
	if err != nil {
		return fmt.Errorf("invalid podSelector on floating ip claim %s: %v", fic.Name, err)
	}
	handler := oc.watchFactory.AddFilteredPodHandler(namespace.Name, sel,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				klog.V(5).Infof("floating ip claim: %s has matched on pod: %s in namespace: %s", fic.Name, pod.Name, namespace.Name)
				fip := &floatingipapi.FloatingIP{
					ObjectMeta: metav1.ObjectMeta{
						Name: uuid.New().String(),
						/*OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(fic, floatingipclaimapi.SchemeGroupVersion.WithKind("FloatingIPClaim")),
						},*/
					},
					Spec: floatingipapi.FloatingIPSpec{
						Pod: pod.Name,
						PodNamespace: pod.Namespace,
						FloatingIPClaim: fic.Name,
					},
				}
				if _, err := oc.kube.CreateFloatingIP(fip); err != nil {
					klog.Errorf("error: unable to add floating ip: %s to floating ip claim: %s, err: %v", fip.Name, fic.Name, err)
				}
			},
		}, nil)
	runtime.podHandlers[namespace.Name] = *handler
	return nil
}

// Stop monitoring pod in the namespace
func (oc *Controller) unwatchPodsInClaim(runtime *floatingIPClaimRuntime, namespace *v1.Namespace) {
	runtime.podLock.Lock()
	defer runtime.podLock.Unlock()
	if handler, exist := runtime.podHandlers[namespace.Name]; exist {
		oc.watchFactory.RemovePodHandler(&handler)
		delete(runtime.podHandlers, namespace.Name)
	}
}

// delete the floating ip crd associated with pod in the namespace that belonging to floating ip claim
func (oc *Controller) deleteFipOnPodsInClaim(fic *floatingipclaimapi.FloatingIPClaim, namespace *v1.Namespace) error {
	klog.V(5).Infof("floating ip claim: floating ip associated with the pod under the namespace %s/%s about to be removed", fic.Name, namespace.Name)
	fiObjs, err := oc.kube.GetFloatingIPs()
	if err != nil {
		klog.Errorf("floating ip claim: unable to list floating ip under namespace: %s err: %v", namespace.Name, err)
		return err
	}
	for _, fiObj := range fiObjs.Items {
		if fiObj.Spec.FloatingIPClaim == fic.Name && fiObj.Spec.PodNamespace == namespace.Name {
			if err := oc.kube.DeleteFloatingIP(fiObj.Name); err != nil {
				klog.Errorf("floating ip claim: unable to remove floating ip %s from floating ip claim: %s err: %v", fiObj.Name, fic.Name, err)
			}
		}
	}
	return nil
}

// delete all floating ip crd belong to floating ip claim
func (oc *Controller) deleteFipsInClaim(fic *floatingipclaimapi.FloatingIPClaim) error {
	klog.V(5).Infof("floating ip claim: floating ip under the floating ip claim %s about to be removed", fic.Name)
	fiObjs, err := oc.kube.GetFloatingIPs()
	if err != nil {
		klog.Errorf("floating ip claim: unable to list floating ip under floating ip claim: %s err: %v", fic.Name, err)
		return err
	}
	for _, fiObj := range fiObjs.Items {
		if fiObj.Spec.FloatingIPClaim == fic.Name {
			if err := oc.kube.DeleteFloatingIP(fiObj.Name); err != nil {
				klog.Errorf("floating ip claim: unable to remove floating ip %s from floating ip claim: %s err: %v", fiObj.Name, fic.Name, err)
			}
		}
	}
	return nil
}

// function must be locked(oc.fIPCC.fipOnPodLock) before calling
func (oc *Controller) verifyFipInClaim(runtime *floatingIPClaimRuntime, fi *floatingipapi.FloatingIP) error {
	ok := false
	pod, err := oc.kube.GetPod(fi.Spec.PodNamespace, fi.Spec.Pod)
	if err != nil {
		return err
	}
	id := PodID{pod.Name, pod.Namespace}
	fipId := FipID{fi.Name, fi.Spec.FloatingIPClaim}
	defer func() {
		// set annotation to track automatically created and non automatically created floating ip crd
		if !ok {
			oc.fIPCC.fipOnPodPendingCache[fipId] = id
			if err := oc.kube.SetAnnotationsOnPod(pod.Namespace, pod.Name, map[string]string{fipOnPodAnnotationName: ""}); err != nil {
				klog.Errorf("floating ip claim: unable to mark pod %s/%s for floating ip claim: %s err: %s", id.namespace, id.name, fipId.ficName)
			}
		} else {
			oc.fIPCC.fipOnPodCache[id] = fipId
			delete(oc.fIPCC.fipOnPodPendingCache, fipId)
			if err := oc.kube.SetAnnotationsOnPod(pod.Namespace, pod.Name, map[string]string{fipOnPodAnnotationName: fi.Spec.FloatingIPClaim}); err != nil {
				klog.Errorf("floating ip claim: unable to mark pod %s/%s for floating ip claim: %s err: %s", id.namespace, id.name, fipId.ficName)
			}
		}
	}()
	// pod already has another floating ip crd
	if _, exist := oc.fIPCC.fipOnPodCache[id]; exist {
		return nil
	}

	// pod network not ready
	if len(pod.Status.PodIPs) == 0 {
		return nil
	}

	// allocate network resources
	ip, err := runtime.ipAllocator.AllocateNext()
	if err != nil {
		return err
	}
	defer func() {
		if !ok {
			runtime.ipAllocator.Release(ip)
		}
	}()

	node := oc.fIPNC.Allocate()
	if node == "" {
		return nil
	}
	defer func() {
		if !ok {
			oc.fIPNC.Release(node)
		}
	}()

	// update floating ip crd
	fiObj := fi.DeepCopy()
	fiObj.Spec.FloatingIP = ip.To4().String()
	fiObj.Spec.NodeName = node
	fiObj.Status.Verified = true
	if err := oc.kube.UpdateFloatingIP(fiObj); err != nil {
		return err
	}
	ok = true
	return nil
}

func (oc *Controller) retryFipInClaim(fipId *FipID, id *PodID) error {
	for k, v := range oc.fIPCC.fipOnPodPendingCache {
		if runtime := oc.fIPCC.GetRuntime(k.ficName); runtime == nil {
			delete(oc.fIPCC.fipOnPodPendingCache, k)
		} else {
			if fiObj, err := oc.kube.GetFloatingIP(k.fipName); err != nil {
				klog.Errorf("floating ip claim: unable to list floating ip: %s err: %v", k.fipName, err)
			} else {
				if err := oc.verifyFipInClaim(runtime, fiObj); err != nil {
					klog.Errorf("floating ip claim: unable to verify floating ip: %s err: %v", k.fipName, err)
				} else if (fipId != nil && reflect.DeepEqual(k, fipId)) || (id != nil && reflect.DeepEqual(v, id)) {
					break
				}
			}
		}
	}
	return nil
}

type PodID struct {
	name string
	namespace string
}

type FipID struct {
	fipName string
	ficName string
}

type floatingIPClaimRuntime struct {
	// network address allocator
	ipAllocator floatingipallocator.Interface

	// Pod level lock
	podLock *sync.Mutex

	// annotate pods that meet claim requirements
	podHandlers map[string]factory.Handler
}

type floatingIPClaimController struct {
	// floating ip claim level lock
	operLock *sync.RWMutex

	// claim runtime information
	runtimes map[string]*floatingIPClaimRuntime

	// monitoring namespace that meets claim requirements
	namespaceHandlers map[string]factory.Handler

	// floating ip crd level lock
	fipOnPodLock *sync.Mutex

	// Record the mapping between pod and confirmed floating ip crd.
	// A pod can correspond to multiple floating ip crds, but only one floating ip crd can be confirmed
	fipOnPodCache map[PodID]FipID

	// The mapping between pod and unconfirmed floating ip crd is that the resource allocation fails and needs to be retried
	// etc. Pod network is not ready、Pod has confirmed floating ip crd and there is no idle network address
	fipOnPodPendingCache map[FipID]PodID
}

func (ficc *floatingIPClaimController) GetRuntime(ficName string) *floatingIPClaimRuntime {
	ficc.operLock.RLock()
	defer ficc.operLock.RUnlock()
	if runtime, ok := ficc.runtimes[ficName]; ok {
		return runtime
	}
	return nil
}