package ovn

import (
	"fmt"
	floatingipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	kapi "k8s.io/api/core/v1"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	utilnet "k8s.io/utils/net"
	"net"
	"strings"
	"sync"
)

func (oc *Controller) addFloatingIP(fIP *floatingipv1.FloatingIP) error {
	spec := fIP.Spec
	pod, err := oc.kube.GetPod(spec.PodNamespace, spec.Pod)
	if err != nil {
		return err
	}
	if pod.Spec.HostNetwork {
		return nil
	}

	if err = oc.fIPC.addPodFloatingIP(fIP, pod); err != nil {
		return fmt.Errorf("unable to add pod(%s/%s)'s FloatingIP: %s, err: %v", pod.Namespace, pod.Name, fIP.Name, err)
	}
	return nil
}

func (oc *Controller) deleteFloatingIP(fIP *floatingipv1.FloatingIP) error {
	spec := fIP.Spec
	pod, err := oc.kube.GetPod(spec.PodNamespace, spec.Pod)
	if err != nil {
		return err
	}
	if pod.Spec.HostNetwork {
		return nil
	}

	if err = oc.fIPC.deletePodFloatingIP(fIP, pod); err != nil {
		return fmt.Errorf("unable to delete pod(%s/%s)'s FloatingIP: %s, err: %v", pod.Namespace, pod.Name, fIP.Name, err)
	}

	return nil
}

func (oc *Controller) syncFloatingIPs(fIPs []interface{}) {

}

func (oc *Controller) updateFloatingIPWithRetry(fIP *floatingipv1.FloatingIP) error{
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return oc.kube.UpdateFloatingIP(fIP)
	})
	if retryErr != nil {
		return fmt.Errorf("error in updating status on FloatingIP %s: %v", fIP.Name, retryErr)
	}
	return nil
}

func findOneToOneNatIDs(floatingIPName, podIP, floatingIP string) ([]string, error) {
	natIDs, stderr, err := util.RunOVNNbctl(
		"--format=csv",
		"--data=bare",
		"--no-heading",
		"--columns=_uuid",
		"find",
		"dnat_and_snat",
		fmt.Sprintf("external_ids:name=%s", floatingIPName),
		fmt.Sprintf("logical_ip=\"%s\"", podIP),
		fmt.Sprintf("external_ip=\"%s\"", floatingIP),
		)
	if err != nil {
		return nil, fmt.Errorf("unable to find dnat_and_snat ID, stderr: %s, err: %v", stderr, err)
	}
	if natIDs == "" {
		return nil, nil
	}
	return strings.Split(natIDs, "\n"), nil
}

type floatingIPController struct {
	// Cache used for retrying pods which did not have an IP address when we processed the FloatingIP object
	podRetry sync.Map

	// Cache of gateway join router IPs, usefull since these should not change often
	gatewayIPCache sync.Map
}

func (f *floatingIPController) addPodFloatingIP(fIP *floatingipv1.FloatingIP, pod *kapi.Pod) error {
	if pod.Spec.HostNetwork {
		return nil
	}
	podIPs := getPodIPs(pod)
	if podIPs == nil {
		f.podRetry.Store(getPodKey(pod), true)
		return nil
	}
	if f.needsRetry(pod) {
		f.podRetry.Delete(getPodKey(pod))
	}

	if err := f.createReroutePolicy(podIPs, fIP.Spec, fIP.Name); err != nil {
		return err
	}
	if err := f.createNATRule(podIPs, fIP.Spec, fIP.Name); err != nil {
		return err
	}
	return nil
}

func (f *floatingIPController) deletePodFloatingIP(fIP *floatingipv1.FloatingIP, pod *kapi.Pod) error {
	if pod.Spec.HostNetwork {
		return nil
	}
	podIPs := getPodIPs(pod)
	if podIPs == nil {
		return nil
	}

	if err := f.deleteReroutePolicy(podIPs, fIP.Spec, fIP.Name); err != nil {
		return err
	}
	if err := f.deleteNATRule(podIPs, fIP.Spec, fIP.Name); err != nil {
		return err
	}
	return nil
}

func (f *floatingIPController) createReroutePolicy(podIPs []net.IP, spec floatingipv1.FloatingIPSpec, fIPName string) error {
    isFloatingIPv6 := utilnet.IsIPv6String(spec.FloatingIP)
    gatewayRouterIP, err := f.getGatewayRouterJoinIP(spec.Node, isFloatingIPv6)
    if err!= nil {
    	return fmt.Errorf("unable to retrieve gateway IP for node: %s, err: %v", spec.Node, err)
	}
	for _, podIP := range podIPs {
		var err error
		var stderr, filterOption string
		if isFloatingIPv6 && utilnet.IsIPv6(podIP) {
			filterOption = fmt.Sprintf("ip6.src == %s", podIP.String())
		} else if !isFloatingIPv6 && !utilnet.IsIPv6(podIP) {
			filterOption = fmt.Sprintf("ip4.src == %s", podIP.String())
		}
		policyIDs, err := f.findReroutePolicyIDs(filterOption, fIPName, gatewayRouterIP)
		if err != nil {
			return err
		}
		if policyIDs == nil {
			_, stderr, err = util.RunOVNNbctl(
				"--id=@lr-policy",
				"create",
				"logical_router_policy",
				"action=reroute",
				fmt.Sprintf("match=\"%s\"", filterOption),
				fmt.Sprintf("priority=%v", types.FloatingIPReroutePriority),
				fmt.Sprintf("nexthop=\"%s\"", gatewayRouterIP),
				fmt.Sprintf("external_ids:name=%s", fIPName),
				"--",
				"add",
				"logical_router",
				types.OVNClusterRouter,
				"policies",
				"@lr-policy",
				)
			if err != nil {
				return fmt.Errorf("unable to create logical router policy: %s, stderr: %s, err: %v", spec.FloatingIP, stderr, err)
			}
		}
	}
	return nil
}

func (f *floatingIPController) deleteReroutePolicy(podIPs []net.IP, spec floatingipv1.FloatingIPSpec, fIPName string) error {
	isFloatingIPv6 := utilnet.IsIPv6String(spec.FloatingIP)
	gatewayRouterIP, err := f.getGatewayRouterJoinIP(spec.Node, isFloatingIPv6)
	if err != nil {
		return fmt.Errorf("unable to retrieve gateway IP for node: %s, err: %v", spec.Node, err)
	}
	for _, podIP := range podIPs {
		var filterOption string
		if utilnet.IsIPv6(podIP) && utilnet.IsIPv6String(spec.FloatingIP) {
			filterOption = fmt.Sprintf("ip6.src == %s", podIP.String())
		} else if !utilnet.IsIPv6(podIP) && !utilnet.IsIPv6String(spec.FloatingIP) {
			filterOption = fmt.Sprintf("ip4.src == %s", podIP.String())
		}
		policyIDs, err := f.findReroutePolicyIDs(filterOption, fIPName, gatewayRouterIP)
		if err != nil {
			return err
		}
		for _, policyID := range policyIDs {
			_, stderr, err := util.RunOVNNbctl(
				"remove",
				"logical_router",
				types.OVNClusterRouter,
				"policies",
				policyID,
				)
			if err != nil {
				return fmt.Errorf("unable to remove logicaal router policy: %s, stderr: %s, err: %v", spec.FloatingIP, stderr, err)
			}
		}
	}
	return nil
}

func (f *floatingIPController) createNATRule(podIPs []net.IP, spec floatingipv1.FloatingIPSpec, fIPName string) error {
	for _, podIP := range podIPs {
		if (utilnet.IsIPv6String(spec.FloatingIP) && utilnet.IsIPv6(podIP)) || (!utilnet.IsIPv6String(spec.FloatingIP) && !utilnet.IsIPv6(podIP)) {
			natIDs, err := findOneToOneNatIDs(fIPName, podIP.String(), spec.FloatingIP)
			if err != nil {
				return err
			}
			if natIDs == nil {
				_, stderr, err := util.RunOVNNbctl(
					"--id=@nat",
					"create",
					"nat",
					"type=dnat_and_snat",
					fmt.Sprintf("logical_port=k8s-%s", spec.Node),
					fmt.Sprintf("external_ip=\"%s\"", spec.FloatingIP),
					fmt.Sprintf("logical_ip=\"%s\"", podIP),
					fmt.Sprintf("external_ids:name=%s", fIPName),
					"--",
					"add",
					"logical_router",
					util.GetGatewayRouterFromNode(spec.Node),
					"nat",
					"@nat",
				)
				if err != nil {
					return fmt.Errorf("unable to create one-to-one nat rule, stderr: %s, err: %v", stderr, err)
				}
			}
		}
	}
	return nil
}

func (f *floatingIPController) deleteNATRule(podIPs []net.IP, spec floatingipv1.FloatingIPSpec, fIPName string) error {
	for _, podIP := range podIPs {
		if (utilnet.IsIPv6String(spec.FloatingIP) && utilnet.IsIPv6(podIP)) || (!utilnet.IsIPv6String(spec.FloatingIP) && !utilnet.IsIPv6(podIP)) {
			natIDs, err := findOneToOneNatIDs(fIPName, podIP.String(), spec.FloatingIP)
			if err != nil {
				return err
			}
			for _, natID := range natIDs {
				_, stderr, err := util.RunOVNNbctl(
					"remove",
					"logica_router",
					util.GetGatewayRouterFromNode(spec.Node),
					"nat",
					natID,
					)
				if err != nil {
					return fmt.Errorf("unable to remove nat from logical_router, stderr: %s, err: %v", stderr, err)
				}
			}
		}
	}
	return nil
}

func (f *floatingIPController) needsRetry(pod *kapi.Pod) bool {
	_, retry := f.podRetry.Load(getPodKey(pod))
	return retry
}

func (f *floatingIPController) getGatewayRouterJoinIP(node string, wantsIPv6 bool) (net.IP, error) {
	var gatewayIPs []*net.IPNet
	if item, exists := f.gatewayIPCache.Load(node); exists {
		var ok bool
		if gatewayIPs, ok = item.([]*net.IPNet); !ok {
			return nil, fmt.Errorf("unable to cast node(%s)'s gatewayIP cache item to correct type", node)
		}
	} else {
		err := utilwait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			var err error
			gatewayIPs, err = util.GetLRPAddrs(types.GWRouterToJoinSwitchPrefix + types.GWRouterPrefix + node)
			if err != nil {
				klog.Errorf("Attempt at finding node gateway router network information failed, err: %v", err)
			}
			return err == nil, nil
		})
		if err != nil {
			return nil, err
		}
		f.gatewayIPCache.Store(node, gatewayIPs)
	}

	if gatewayIP, err := util.MatchIPNetFamily(wantsIPv6, gatewayIPs); gatewayIP != nil {
		return gatewayIP.IP, nil
	} else {
		return nil, fmt.Errorf("could not find node %s gateway router: %v", node, err)
	}
}

func (f *floatingIPController) findReroutePolicyIDs(filterOption, fIPName string, gatewayRouterIP net.IP) ([]string, error) {
	policyIDs, stderr, err := util.RunOVNNbctl(
		"--format=csv",
		"--data=bare",
		"--columns=_uuid",
		"find",
		"logical_router_policy",
		fmt.Sprintf("match=\"%s\"", filterOption),
		fmt.Sprintf("priority=%v", types.FloatingIPReroutePriority),
		fmt.Sprintf("external_ids:name=%s", fIPName),
		fmt.Sprintf("nexthop=\"%s\"", gatewayRouterIP),
		)
	if err != nil {
		return nil, fmt.Errorf("unable to find logical router policy for FloatingIP: %s, stderr: %s, err: %v", fIPName, stderr, err)
	}
	if policyIDs == "" {
		return nil, nil
	}
	return strings.Split(policyIDs, "\n"), nil
}

func getPodIPs(pod *kapi.Pod) []net.IP {
	if len(pod.Status.PodIPs) == 0 {
		return nil
	}
	podIPs := []net.IP{}
	for _, podIP := range pod.Status.PodIPs {
		podIPs = append(podIPs, net.ParseIP(podIP.IP))
	}
	return podIPs
}