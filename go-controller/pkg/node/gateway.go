package node

import (
	"fmt"
	"hash/fnv"
	"net"
	"reflect"
	"strings"
	"sync"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	floatingipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/informer"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	util "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	kapi "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// Gateway responds to Service and Endpoint K8s events
// and programs OVN gateway functionality.
// It may also spawn threads to ensure the flow tables
// are kept in sync
type Gateway interface {
	informer.ServiceAndEndpointsEventHandler
	Init(factory.NodeWatchFactory) error
	Run(<-chan struct{}, *sync.WaitGroup)
	GetGatewayBridgeIface() string
}

type gateway struct {
	// loadBalancerHealthChecker is a health check server for load-balancer type services
	loadBalancerHealthChecker informer.ServiceAndEndpointsEventHandler
	// portClaimWatcher is for reserving ports for virtual IPs allocated by the cluster on the host
	portClaimWatcher informer.ServiceEventHandler
	// nodePortWatcher is used in Shared GW mode to handle nodePort flows in shared OVS bridge
	nodePortWatcher informer.ServiceEventHandler
	// localPortWatcher is used in Local GW mode to handle iptables rules and routes for services
	localPortWatcher informer.ServiceEventHandler
	openflowManager  *openflowManager
	nodeIPManager    *addressManager
	initFunc         func() error
	readyFunc        func() (bool, error)
	nodeName         string
}

func (g *gateway) AddService(svc *kapi.Service) {
	if g.portClaimWatcher != nil {
		g.portClaimWatcher.AddService(svc)
	}
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.AddService(svc)
	}
	if g.nodePortWatcher != nil {
		g.nodePortWatcher.AddService(svc)
	}
	if g.localPortWatcher != nil {
		g.localPortWatcher.AddService(svc)
	}
}

func (g *gateway) UpdateService(old, new *kapi.Service) {
	if g.portClaimWatcher != nil {
		g.portClaimWatcher.UpdateService(old, new)
	}
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.UpdateService(old, new)
	}
	if g.nodePortWatcher != nil {
		g.nodePortWatcher.UpdateService(old, new)
	}
	if g.localPortWatcher != nil {
		g.localPortWatcher.UpdateService(old, new)
	}
}

func (g *gateway) DeleteService(svc *kapi.Service) {
	if g.portClaimWatcher != nil {
		g.portClaimWatcher.DeleteService(svc)
	}
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.DeleteService(svc)
	}
	if g.nodePortWatcher != nil {
		g.nodePortWatcher.DeleteService(svc)
	}
	if g.localPortWatcher != nil {
		g.localPortWatcher.DeleteService(svc)
	}
}

func (g *gateway) SyncServices(objs []interface{}) {
	if g.portClaimWatcher != nil {
		g.portClaimWatcher.SyncServices(objs)
	}
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.SyncServices(objs)
	}
	if g.nodePortWatcher != nil {
		g.nodePortWatcher.SyncServices(objs)
	}
	if g.localPortWatcher != nil {
		g.localPortWatcher.SyncServices(objs)
	}
}

func (g *gateway) AddEndpoints(ep *kapi.Endpoints) {
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.AddEndpoints(ep)
	}
}

func (g *gateway) UpdateEndpoints(old, new *kapi.Endpoints) {
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.UpdateEndpoints(old, new)
	}
}

func (g *gateway) DeleteEndpoints(ep *kapi.Endpoints) {
	if g.loadBalancerHealthChecker != nil {
		g.loadBalancerHealthChecker.DeleteEndpoints(ep)
	}
}

func (g *gateway) AddFloatingIP(fip *floatingipv1.FloatingIP) {
	if strings.Compare(fip.Spec.NodeName, g.nodeName) != 0 {
		klog.V(5).Infof("Skipping Floating IP creating for: %v which is not assigned to this node", fip)
		return
	}

	if !fip.Status.Verified {
		klog.V(5).Infof("Skipping Floating IP creating for: %v which is not verified", fip)
		return
	}
    g.updateFloatingIPFlowCache(fip, true)
    g.openflowManager.requestFlowSync()
}

func (g *gateway) UpdateFloatingIP(old, new *floatingipv1.FloatingIP) {
	if strings.Compare(new.Spec.NodeName, g.nodeName) != 0 {
		klog.V(5).Infof("Skipping Floating IP updating for: %v which is not assigned to this node", new)
		return
	}

    if reflect.DeepEqual(new.Spec.NodeName, old.Spec.NodeName) &&
    	reflect.DeepEqual(new.Spec.Pod, old.Spec.Pod) &&
    	reflect.DeepEqual(new.Spec.FloatingIP, old.Spec.FloatingIP) &&
    	old.Status.Verified == new.Status.Verified {
		klog.V(5).Infof("Skipping Floating IP updating for: %s as change does not apply to any of"+
			".Spec.Node, .Spec.Pod, .Spec.FloatingIP", new.Name)
		return
	}

	if strings.Compare(old.Spec.NodeName, new.Spec.NodeName) != 0 {
		klog.Errorf("Invalid Floating IP update for: %s. When node change, Floating IP should be delete " +
			"and recreate", old.Name)
		return
	}

	if old.Status.Verified {
		g.updateFloatingIPFlowCache(old, false)
	}
    if new.Status.Verified {
    	g.updateFloatingIPFlowCache(new, true)
	}
    g.openflowManager.requestFlowSync()
}

func (g *gateway) SyncFloatingIP(objs []interface{}) {
    for _, floatingIPInterface := range objs {
    	fip, ok := floatingIPInterface.(*floatingipv1.FloatingIP)
    	if !ok {
    		klog.Errorf("Spurious object in sync Floating IP: %v", floatingIPInterface)
    		continue
		}

		if strings.Compare(fip.Spec.NodeName, g.nodeName) != 0 {
			klog.V(5).Infof("Skipping Floating IP syncing for: %v which is not assigned to this node", fip)
			continue
		}

		if !fip.Status.Verified {
			klog.V(5).Infof("Skip Floating IP creating for: %v which is not verified", fip)
			continue
		}

		g.updateFloatingIPFlowCache(fip, true)
	}
	g.openflowManager.requestFlowSync()
}

func (g *gateway) DeleteFloatingIP(fip *floatingipv1.FloatingIP) {
	if strings.Compare(fip.Spec.NodeName, g.nodeName) != 0 {
		klog.V(5).Infof("Skipping Floating IP deleting for: %v which is not assigned to this node", fip)
		return
	}

	if !fip.Status.Verified {
		klog.V(5).Infof("Skipping Floating IP delete for: %v which is not verified", fip)
		return
	}
    g.updateFloatingIPFlowCache(fip, false)
    g.openflowManager.requestFlowSync()
}

func (g *gateway) updateFloatingIPFlowCache(fip *floatingipv1.FloatingIP, add bool) {
	var cookie, key string
	var err error

	spec := fip.Spec
	cookie, err = FloatingIPToCookie(spec.NodeName, spec.Pod, spec.FloatingIP)
	if err != nil {
		klog.Warningf("Unable to generate cookie for FloatingI: %s, %s, %s, error: %v",
			spec.NodeName, spec.Pod, spec.FloatingIP)
		cookie = "0"
	}
	key = strings.Join([]string{"FloatingIP", spec.NodeName, spec.Pod, spec.FloatingIP}, "_")
	if !add {
		g.openflowManager.deleteFlowsByKey(key)
	} else {
		g.openflowManager.updateFlowCacheEntry(key, []string{
			fmt.Sprintf("cookie=%s, priority=50, table=0,ip,in_port=%s,nw_dst=%s actions=output:%s",
				cookie, g.openflowManager.physIntf, spec.FloatingIP, g.openflowManager.ofportPatch),
		})
	}
}

func (g *gateway) Init(wf factory.NodeWatchFactory) error {
	err := g.initFunc()
	if err != nil {
		return err
	}
	wf.AddServiceHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*kapi.Service)
			g.AddService(svc)
		},
		UpdateFunc: func(old, new interface{}) {
			oldSvc := old.(*kapi.Service)
			newSvc := new.(*kapi.Service)
			g.UpdateService(oldSvc, newSvc)
		},
		DeleteFunc: func(obj interface{}) {
			svc := obj.(*kapi.Service)
			g.DeleteService(svc)
		},
	}, g.SyncServices)

	wf.AddEndpointsHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ep := obj.(*kapi.Endpoints)
			g.AddEndpoints(ep)
		},
		UpdateFunc: func(old, new interface{}) {
			oldEp := old.(*kapi.Endpoints)
			newEp := new.(*kapi.Endpoints)
			g.UpdateEndpoints(oldEp, newEp)
		},
		DeleteFunc: func(obj interface{}) {
			ep := obj.(*kapi.Endpoints)
			g.DeleteEndpoints(ep)
		},
	}, nil)

	wf.AddFloatingIPHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fip := obj.(*floatingipv1.FloatingIP)
			g.AddFloatingIP(fip)
		},
		UpdateFunc: func(old, new interface{}) {
			oldFip := old.(*floatingipv1.FloatingIP)
			newFip := new.(*floatingipv1.FloatingIP)
			g.UpdateFloatingIP(oldFip, newFip)
		},
		DeleteFunc: func(obj interface{}) {
			fip := obj.(*floatingipv1.FloatingIP)
			g.DeleteFloatingIP(fip)
		},
	}, g.SyncFloatingIP)
	return nil
}

func (g *gateway) Run(stopChan <-chan struct{}, wg *sync.WaitGroup) {
	if g.nodeIPManager != nil {
		g.nodeIPManager.Run(stopChan)
	}

	if g.openflowManager != nil {
		klog.Info("Spawning Conntrack Rule Check Thread")
		wg.Add(1)
		defer wg.Done()
		g.openflowManager.Run(stopChan)
	}
}

func gatewayInitInternal(nodeName, gwIntf string, subnets []*net.IPNet, gwNextHops []net.IP, nodeAnnotator kube.Annotator) (
	string, string, net.HardwareAddr, []*net.IPNet, error) {

	var bridgeName string
	var uplinkName string
	var brCreated bool
	var err error

	if bridgeName, _, err = util.RunOVSVsctl("--", "port-to-br", gwIntf); err == nil {
		// This is an OVS bridge's internal port
		uplinkName, err = util.GetNicName(bridgeName)
		if err != nil {
			return bridgeName, uplinkName, nil, nil, err
		}
	} else if _, _, err := util.RunOVSVsctl("--", "br-exists", gwIntf); err != nil {
		// This is not a OVS bridge. We need to create a OVS bridge
		// and add cluster.GatewayIntf as a port of that bridge.
		bridgeName, err = util.NicToBridge(gwIntf)
		if err != nil {
			return bridgeName, uplinkName, nil, nil, fmt.Errorf("failed to convert %s to OVS bridge: %v", gwIntf, err)
		}
		uplinkName = gwIntf
		gwIntf = bridgeName
		brCreated = true
	} else {
		// gateway interface is an OVS bridge
		uplinkName, err = getIntfName(gwIntf)
		if err != nil {
			return bridgeName, uplinkName, nil, nil, err
		}
		bridgeName = gwIntf
	}

	// Now, we get IP addresses from OVS bridge. If IP does not exist,
	// error out.
	ips, err := getNetworkInterfaceIPAddresses(gwIntf)
	if err != nil {
		return bridgeName, uplinkName, nil, nil, fmt.Errorf("failed to get interface details for %s (%v)",
			gwIntf, err)
	}
	ifaceID, macAddress, err := bridgedGatewayNodeSetup(nodeName, bridgeName, gwIntf,
		types.PhysicalNetworkName, brCreated)
	if err != nil {
		return bridgeName, uplinkName, nil, nil, fmt.Errorf("failed to set up shared interface gateway: %v", err)
	}

	if config.Gateway.Mode == config.GatewayModeLocal {
		err = setupLocalNodeAccessBridge(nodeName, subnets)
		if err != nil {
			return bridgeName, uplinkName, nil, nil, err
		}
	}
	chassisID, err := util.GetNodeChassisID()
	if err != nil {
		return bridgeName, uplinkName, nil, nil, err
	}

	err = util.SetL3GatewayConfig(nodeAnnotator, &util.L3GatewayConfig{
		Mode:           config.GatewayModeShared,
		ChassisID:      chassisID,
		InterfaceID:    ifaceID,
		MACAddress:     macAddress,
		IPAddresses:    ips,
		NextHops:       gwNextHops,
		NodePortEnable: config.Gateway.NodeportEnable,
		VLANID:         &config.Gateway.VLANID,
	})
	return bridgeName, uplinkName, macAddress, ips, err
}

func gatewayReady(patchPort string) (bool, error) {
	// Get ofport of patchPort
	ofportPatch, _, err := util.RunOVSVsctl("--if-exists", "get", "interface", patchPort, "ofport")
	if err != nil || len(ofportPatch) == 0 {
		return false, nil
	}
	klog.Info("Gateway is ready")
	return true, nil
}

func (g *gateway) GetGatewayBridgeIface() string {
	return g.openflowManager.gwBridge
}

func FloatingIPToCookie(node string, pod string, ip string) (string, error) {
	id := fmt.Sprintf("%s%s%s", node, pod, ip)
	h := fnv.New64a()
	_,err := h.Write([]byte(id))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("0x%x", h.Sum64()), nil
}
