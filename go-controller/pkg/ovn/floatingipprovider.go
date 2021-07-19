package ovn

import (
	"net"
	"reflect"
	"strings"
	"time"

	floatingipclaimapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	floatingipproviderapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipprovider/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/ovn/floatingipallocator"
	kapi "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	// maxRetries is the number of times a service will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the
	// sequence of delays between successive queuings of a service.
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

type floatingIpProviderController struct {
	ficc *floatingIPClaimController

	client           kube.Interface
	eventRecorder    record.EventRecorder
	queue            workqueue.RateLimitingInterface
	workerLoopPeriod time.Duration
}

func (fipc *floatingIpProviderController) addFloatingIPProvider(obj interface{}) {
	fip := obj.(*floatingipproviderapi.FloatingIPProvider)
	klog.V(5).InfoS("Adding floating ip provider", "floatingipprovider", klog.KRef("", fip.Name))
	fipc.ficc.ficMutex.Lock()
	defer fipc.ficc.ficMutex.Unlock()
	var aps []*floatingipallocator.AddrPair
	for _, addr := range fip.Spec.FloatingIPs {
		ap, err := floatingipallocator.NewAddrPair(addr)
		if err != nil {
			fipc.eventRecorder.Eventf(fip, kapi.EventTypeWarning, "FailedToAddFloatingIPProvider", "Invalid network address %s", addr)
			return
		}
		aps = append(aps, ap)
	}
	for fipName, allocator := range fipc.ficc.allocators {
		for _, ap := range aps {
			if allocator.HasPool(ap.Begin, ap.End) {
				fipc.eventRecorder.Eventf(fip, kapi.EventTypeWarning, "FailedToAddFloatingIPProvider", "Network address conflict %s", fipName)
				return
			}
		}
	}
	if allocator, err := floatingipallocator.NewDistributor(aps); err != nil {
		fipc.eventRecorder.Eventf(fip, kapi.EventTypeWarning, "FailedToCreateFloatingIPClaim", "Failed to create floating ip allocator: %v", err)
		return
	} else {
		fipc.ficc.allocators[fip.Name] = allocator
	}
	fipc.queue.AddRateLimited(fip.Name)
}

func (fipc *floatingIpProviderController) deleteFloatingIPProvider(obj interface{}) {
	fip := obj.(*floatingipproviderapi.FloatingIPProvider)
	klog.V(5).InfoS("Deleting floating ip provider", "floatingipprovider", klog.KRef("", fip.Name))
	{
		fipc.ficc.ficMutex.Lock()
		defer fipc.ficc.ficMutex.Unlock()
		delete(fipc.ficc.allocators, fip.Name)
	}

	ficObjs, err := fipc.client.GetFloatingIPClaims()
	if err != nil {
		klog.V(5).InfoS("Error get floating ip claims", "err", err)
		fipc.queue.Add(fip.Name)
		return
	}
	for _, ficObj := range ficObjs.Items {
		if ficObj.Spec.Provider == fip.Name {
			if err = fipc.client.DeleteFloatingIPClaim(ficObj.Name); err != nil {
				klog.V(5).InfoS("Error delete floating ip claim", "floatingipclaim", ficObj.Name, "err", err)
				fipc.queue.AddRateLimited(fip.Name)
			}
		}
	}
}

func (fipc *floatingIpProviderController) addFloatingIPClaim(obj interface{}) {
	fic := obj.(*floatingipclaimapi.FloatingIPClaim)
	klog.V(5).InfoS("Adding floating ip claim", "floatingipclaim", klog.KRef("", fic.Name))
	defer func() {
		fipc.ficc.queue.AddRateLimited(fic.Name)
		fipc.queue.AddRateLimited(fic.Spec.Provider)
	}()
	{
		fipc.ficc.ficMutex.Lock()
		defer fipc.ficc.ficMutex.Unlock()
		allocator, exist := fipc.ficc.allocators[fic.Spec.Provider]
		if !exist {
			fipc.ficc.eventRecorder.Eventf(fic, kapi.EventTypeWarning, "FailedToAddFloatingIPClaim", "Failed to get floating ip provider allocator %s", fic.Spec.Provider)
			return
		}

		var aps []*floatingipallocator.AddrPair
		for _, addr := range fic.Spec.FloatingIPs {
			ap, err := floatingipallocator.NewAddrPair(addr)
			if err != nil {
				fipc.ficc.eventRecorder.Eventf(fic, kapi.EventTypeWarning, "FailedToAddFloatingIPClaim", "Invalid network address %s", addr)
				return
			}
			aps = append(aps, ap)
		}
		var rollback []*floatingipallocator.AddrPair
		ok := false
		defer func() {
			if !ok {
				for _, ap := range rollback {
					if err := allocator.ReleasePool(ap.Begin, ap.End); err != nil {
						klog.Errorf("It shouldn't happen: %v", err)
					}
				}
			}
		}()
		for _, ap := range aps {
			if err := allocator.AllocatePool(ap.Begin, ap.End); err != nil {
				fipc.ficc.eventRecorder.Eventf(fic, kapi.EventTypeWarning, "FailedToAddFloatingIPClaim", "Network address conflict %v", ap)
				return
			}
			rollback = append(rollback, ap)
		}
		if suballocator, err := floatingipallocator.NewDistributor(aps); err != nil {
			fipc.ficc.eventRecorder.Eventf(fic, kapi.EventTypeWarning, "FailedToInitFloatingIPClaim", "Failed to create floating ip allocator")
			return
		} else {
			fipc.ficc.suballocators[fic.Name] = suballocator
		}
		ok = true
	}

	if !fipc.ficc.addFloatingIPClaim(obj) {
		return
	}
}

func (fipc *floatingIpProviderController) updateFloatingIPClaim(old, cur interface{}) {
	oldFic := old.(*floatingipclaimapi.FloatingIPClaim)
	curFic := cur.(*floatingipclaimapi.FloatingIPClaim)
	klog.V(5).InfoS("Updating floating ip claim", "floatingipclaim", klog.KObj(oldFic))
	if !reflect.DeepEqual(oldFic.Spec, curFic.Spec) {
		if oldFic.Status.Phase == floatingipclaimapi.FloatingIPClaimReady {
			fipc.deleteFloatingIPClaim(oldFic)
		}
		fipc.addFloatingIPClaim(curFic)
	}
}

func (fipc *floatingIpProviderController) deleteFloatingIPClaim(obj interface{}) {
	fic := obj.(*floatingipclaimapi.FloatingIPClaim)
	klog.V(5).InfoS("Deleting floating ip claim", "floatingipclaim", klog.KRef("", fic.Name))
	{
		fipc.ficc.ficMutex.Lock()
		defer fipc.ficc.ficMutex.Unlock()
		if suballocator, exist := fipc.ficc.suballocators[fic.Name]; exist {
			if allocator, exist := fipc.ficc.allocators[fic.Spec.Provider]; exist {
				suballocator.ForEach(func(ip net.IP) {
					if err := allocator.Release(ip); err != nil {
						klog.Errorf("It shouldn't happen")
					}
				})
			}
			delete(fipc.ficc.suballocators, fic.Name)
		}
	}

	fipc.ficc.deleteFloatingIPClaim(obj)
	fipc.queue.AddRateLimited(fic.Spec.Provider)
}

func (fipc *floatingIpProviderController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer fipc.queue.ShutDown()

	klog.Infof("Starting floating ip provider controller")
	defer klog.Infof("Shutting down floating ip provider controller")

	go wait.Until(fipc.worker, fipc.workerLoopPeriod, stopCh)

	<-stopCh
}

func (fipc *floatingIpProviderController) worker() {
	for fipc.processNextWorkItem() {
	}
}

func (fipc *floatingIpProviderController) processNextWorkItem() bool {
	eKey, quit := fipc.queue.Get()
	if quit {
		return false
	}
	defer fipc.queue.Done(eKey)

	err := fipc.sync(eKey.(string))
	fipc.handleErr(err, eKey)

	return true
}

func (fipc *floatingIpProviderController) handleErr(err error, key interface{}) {
	if err == nil {
		fipc.queue.Forget(key)
		return
	}

	if fipc.queue.NumRequeues(key) < maxRetries {
		klog.V(2).InfoS("Error syncing floating ip provider, retrying", "floatingipprovider", key, "err", err)
		fipc.queue.AddRateLimited(key)
		return
	}

	klog.Warningf("Dropping floating ip provider %q out of the queue: %v", key, err)
	fipc.queue.Forget(key)
	utilruntime.HandleError(err)
}

func (fipc *floatingIpProviderController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing floating ip provider %q. (%v)", key, time.Since(startTime))
	}()

	fipObj, err := fipc.client.GetFloatingIPProvider(key)
	if err != nil {
		//if !errors.IsNotFound(err)
		if !strings.HasSuffix(err.Error(), "not found") {
			return err
		}

		{
			fipc.ficc.ficMutex.Lock()
			defer fipc.ficc.ficMutex.Unlock()
			delete(fipc.ficc.allocators, fipObj.Name)
		}

		// floating ip provider has been deleted
		ficObjs, err := fipc.client.GetFloatingIPClaims()
		if err != nil {
			return err
		}
		for _, ficObj := range ficObjs.Items {
			if ficObj.Spec.Provider == fipObj.Name {
				if err = fipc.client.DeleteFloatingIPClaim(ficObj.Name); err != nil {
					klog.V(5).InfoS("Error delete floating ip claim", "floatingipclaim", ficObj.Name, "err", err)
					return err
				}
			}
		}
		return nil
	}

	fip := fipObj.DeepCopy()
	fip.Status.FloatingIPClaims = []string{}
	ficObjs, err := fipc.client.GetFloatingIPClaims()
	if err != nil {
		klog.V(5).InfoS("Error get floating ip claims", "err", err)
		return err
	}
	{
		fipc.ficc.ficMutex.Lock()
		defer fipc.ficc.ficMutex.Unlock()
		if _, ok := fipc.ficc.allocators[fipObj.Name]; ok {
			fip.Status.Phase = floatingipproviderapi.FloatingIPProviderReady
			for _, ficObj := range ficObjs.Items {
				if ficObj.Spec.Provider == fipObj.Name {
					if _, ok := fipc.ficc.suballocators[ficObj.Name]; ok {
						fip.Status.FloatingIPClaims = append(fip.Status.FloatingIPClaims, ficObj.Name)
					}
				}
			}
		} else {
			fip.Status.Phase = floatingipproviderapi.FloatingIPProviderNotReady
		}
	}
	if err = fipc.client.UpdateFloatingIPProvider(fip); err != nil {
		fipc.eventRecorder.Eventf(fipObj, kapi.EventTypeWarning, "FailedToUpdateFloatingIPProvider", "Failed to update status for floating ip provider: %v", err)
		fipc.queue.AddRateLimited(fipObj.Name)
	}

	return nil
}
