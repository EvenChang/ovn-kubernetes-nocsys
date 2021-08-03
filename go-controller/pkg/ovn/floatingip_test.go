package ovn

import (
	"context"
	"fmt"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	floatingip "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	ovntest "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/testing"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	policyUuid = "f34a9f1e-3290-434e-ae6c-1abdec8f97e9"
	natUuid    = "576e1c56-9ed3-4485-b093-68196d8f0ea5"
)

const (
	fipName      = "floatingip"
	fipNamespace = "floatingip-namespace"
	podIPv4      = "10.129.0.16"
	floatingIPv4 = "192.168.200.6"
)

func newFloatingIPNode(name string, annotations map[string]string, labels map[string]string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        node1Name,
			Annotations: annotations,
			Labels:      labels,
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func newFloatingIP(podName, podNamespace, podIP, node, floatingIP string) *floatingip.FloatingIP {
	return &floatingip.FloatingIP{
		ObjectMeta: newObjectMeta(fipName, ""),
		Spec: floatingip.FloatingIPSpec{
			FloatingIPClaim: "claim",
			Pod:             "pod",
			PodNamespace:    "default",
		},
		Status: floatingip.FloatingIPStatus{
			NodeName:    node,
			FloatingIP:  floatingIP,
			HostNetwork: false,
			PodIPs:      []string{podIP},
		},
	}
}

var _ = Describe("OVN master FloatingIP Operations", func() {
	var (
		app     *cli.App
		fakeOvn *FakeOVN
		tExec   *ovntest.FakeExec
	)

	isFloatingIPAssignableNode := func(nodeName string) func() bool {
		return func() bool {
			fakeOvn.controller.fIPNC.ficc.nodeMutex.Lock()
			defer fakeOvn.controller.fIPNC.ficc.nodeMutex.Unlock()
			if item, exists := fakeOvn.controller.fIPNC.ficc.fiOnNodes[nodeName]; exists {
				return item == 0
			}
			return false
		}
	}

	getGatewayRouterJoinIP := func(node string, wantsIPv6 bool) func() (string, error) {
		return func() (string, error) {
			var item interface{}
			var gatewayIPs []*net.IPNet
			var exists, ok bool

			if item, exists = fakeOvn.controller.fIPC.gatewayIPCache.Load(node); !exists {
				return "", fmt.Errorf("gateway ip cache does not have node(%s)'s router join ip", node)
			}

			if gatewayIPs, ok = item.([]*net.IPNet); !ok {
				return "", fmt.Errorf("unable to cast node(%s)'s gateway IP cache item to IPNet list", node)
			}

			if gatewayIP, _ := util.MatchIPNetFamily(wantsIPv6, gatewayIPs); gatewayIP != nil {
				return gatewayIP.IP.String(), nil
			} else {
				return "", fmt.Errorf("could not find node(%s)'s gateway router join ip", node)
			}
		}
	}

	BeforeEach(func() {
		// Restore global default values before each testcase
		config.PrepareTestConfig()
		config.OVNKubernetesFeature.EnableFloatingIP = true

		app = cli.NewApp()
		app.Name = "test"
		app.Flags = config.Flags

		tExec = ovntest.NewLooseCompareFakeExec()
		fakeOvn = NewFakeOVN(tExec)
	})

	AfterEach(func() {
		fakeOvn.shutdown()
	})

	Context("On node UPDATE", func() {

		It("should perform proper OVN transactions", func() {
			app.Action = func(ctx *cli.Context) error {

				node1IPv4 := "192.168.200.180/24"
				node2IPv4 := "192.168.200.181/24"

				node1 := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: node1Name,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", node1IPv4, ""),
						},
						Labels: map[string]string{
							"k8s.ovn.org/floatingip-assignable": "",
						},
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
				node2 := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: node2Name,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", node2IPv4, ""),
						},
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}

				fakeOvn.start(ctx, &v1.NodeList{Items: []v1.Node{node1, node2}})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node1 options:nat-addresses=router"),
					},
				)

				fakeOvn.controller.WatchFloatingIPNodes()
				Eventually(isFloatingIPAssignableNode(node1.Name)).Should(BeTrue())
				Eventually(isFloatingIPAssignableNode(node2.Name)).Should(BeFalse())

				node1.Labels = map[string]string{}
				node2.Labels = map[string]string{
					"k8s.ovn.org/floatingip-assignable": "",
				}

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 remove logical_switch_port etor-GR_node1 options nat-addresses=router"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node2 options:nat-addresses=router"),
					},
				)

				_, err := fakeOvn.fakeClient.KubeClient.CoreV1().Nodes().Update(context.TODO(), &node1, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				_, err = fakeOvn.fakeClient.KubeClient.CoreV1().Nodes().Update(context.TODO(), &node2, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				Eventually(isFloatingIPAssignableNode(node1.Name)).Should(BeTrue())
				Eventually(isFloatingIPAssignableNode(node2.Name)).Should(BeFalse())
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc)

				return nil
			}

			err := app.Run([]string{app.Name})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("On floating ip CREATE/UPDATE/DELETE for IPv4", func() {

		It("should perform proper OVN transactions when floating ip is created", func() {
			app.Action = func(ctx *cli.Context) error {

				nodeIPv4 := "192.168.200.180/24"
				annotations := map[string]string{
					"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", nodeIPv4, ""),
				}
				labels := map[string]string{"k8s.ovn.org/floatingip-assignable": ""}

				node := *newFloatingIPNode(node1Name, annotations, labels)
				pod := *newPod("default", "pod", node1Name, podIPv4)
				floatingIP := newFloatingIP("pod", "default", podIPv4, node1Name, floatingIPv4)

				fakeOvn.start(ctx,
					&v1.NodeList{Items: []v1.Node{node}},
					&v1.PodList{Items: []v1.Pod{pod}},
				)

				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd:    fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_%s networks", node1Name),
						Output: nodeLogicalRouterIfAddrV4,
					},
				)
				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --if-exist get logical_switch_port %s options:nat-addresses",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
						Output: "",
					},
				)

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find logical_router_policy match=\"ip4.src == %s\" priority=%v external_ids:name=%s nexthop=\"%s\"",
							podIPv4,
							types.FloatingIPReroutePriority,
							fipName,
							nodeLogicalRouterIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@lr-policy create logical_router_policy action=reroute match=\"ip4.src == %s\" priority=%v nexthop=\"%s\" external_ids:name=%s -- add logical_router ovn_cluster_router policies @lr-policy",
							podIPv4,
							types.FloatingIPReroutePriority,
							nodeLogicalRouterIPv4,
							fipName,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find nat external_ids:name=%s logical_ip=\"%s\" external_ip=\"%s\"",
							fipName,
							podIPv4,
							floatingIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@nat create nat type=dnat_and_snat logical_port=k8s-%s external_ip=\"%s\" logical_ip=\"%s\" external_ids:name=%s -- add logical_router %s nat @nat",
							node1Name,
							floatingIPv4,
							podIPv4,
							fipName,
							util.GetGatewayRouterFromNode(node1Name),
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 set logical_switch_port %s options:nat-addresses=router",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
					},
				)
				fakeOvn.controller.WatchFloatingIP()

				_, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().Create(context.TODO(), floatingIP, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Eventually(getGatewayRouterJoinIP(node1Name, false)).Should(Equal(nodeLogicalRouterIPv4))
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc())

				return nil
			}

			err := app.Run([]string{app.Name})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform proper OVN transactions when floating ip is deleted", func() {
			app.Action = func(ctx *cli.Context) error {

				nodeIPv4 := "192.168.200.180/24"
				annotations := map[string]string{
					"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", nodeIPv4, ""),
				}
				labels := map[string]string{"k8s.ovn.org/floatingip-assignable": ""}

				node := *newFloatingIPNode(node1Name, annotations, labels)
				pod := *newPod("default", "pod", node1Name, podIPv4)
				floatingIP := *newFloatingIP("pod", "default", podIPv4, node1Name, floatingIPv4)

				fakeOvn.start(ctx,
					&v1.NodeList{Items: []v1.Node{node}},
					&v1.PodList{Items: []v1.Pod{pod}},
					&floatingip.FloatingIPList{Items: []floatingip.FloatingIP{floatingIP}},
				)

				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd:    fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_%s networks", node1Name),
						Output: nodeLogicalRouterIfAddrV4,
					},
				)
				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --if-exist get logical_switch_port %s options:nat-addresses",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
						Output: "",
					},
				)

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find logical_router_policy match=\"ip4.src == %s\" priority=%v external_ids:name=%s nexthop=\"%s\"",
							podIPv4,
							types.FloatingIPReroutePriority,
							fipName,
							nodeLogicalRouterIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@lr-policy create logical_router_policy action=reroute match=\"ip4.src == %s\" priority=%v nexthop=\"%s\" external_ids:name=%s -- add logical_router ovn_cluster_router policies @lr-policy",
							podIPv4,
							types.FloatingIPReroutePriority,
							nodeLogicalRouterIPv4,
							fipName,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find nat external_ids:name=%s logical_ip=\"%s\" external_ip=\"%s\"",
							fipName,
							podIPv4,
							floatingIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@nat create nat type=dnat_and_snat logical_port=k8s-%s external_ip=\"%s\" logical_ip=\"%s\" external_ids:name=%s -- add logical_router %s nat @nat",
							node1Name,
							floatingIPv4,
							podIPv4,
							fipName,
							util.GetGatewayRouterFromNode(node1Name),
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 set logical_switch_port %s options:nat-addresses=router",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
					},
				)

				fakeOvn.controller.WatchFloatingIP()
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected, 2).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc())

				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find logical_router_policy match=\"ip4.src == %s\" priority=%v external_ids:name=%s nexthop=\"%s\"",
							podIPv4,
							types.FloatingIPReroutePriority,
							fipName,
							nodeLogicalRouterIPv4,
						),
						Output: policyUuid,
					},
				)
				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find nat external_ids:name=%s logical_ip=\"%s\" external_ip=\"%s\"",
							fipName,
							podIPv4,
							floatingIPv4,
						),
						Output: natUuid,
					},
				)
				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 remove logical_router %s policies %s", types.OVNClusterRouter, policyUuid),
						fmt.Sprintf("ovn-nbctl --timeout=15 remove logical_router %s nat %s", util.GetGatewayRouterFromNode(node1Name), natUuid),
					},
				)

				err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().Delete(context.TODO(), fipName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
				Eventually(getGatewayRouterJoinIP(node1Name, false)).Should(Equal(nodeLogicalRouterIPv4))
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected, 2).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc())

				return nil
			}

			err := app.Run([]string{app.Name})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform proper OVN transactions when floating ip is updated", func() {
			app.Action = func(ctx *cli.Context) error {

				newFloatingIPv4 := "192.168.200.7"
				nodeIPv4 := "192.168.200.180/24"
				annotations := map[string]string{
					"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", nodeIPv4, ""),
				}
				labels := map[string]string{"k8s.ovn.org/floatingip-assignable": ""}

				node1 := *newFloatingIPNode(node1Name, annotations, labels)
				pod := *newPod("default", "pod", node1Name, podIPv4)
				floatingIP := *newFloatingIP("pod", "default", podIPv4, node1Name, floatingIPv4)

				fakeOvn.start(ctx,
					&v1.NodeList{Items: []v1.Node{node1}},
					&v1.PodList{Items: []v1.Pod{pod}},
					&floatingip.FloatingIPList{Items: []floatingip.FloatingIP{floatingIP}},
				)

				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd:    fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_%s networks", node1Name),
						Output: nodeLogicalRouterIfAddrV4,
					},
				)
				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --if-exist get logical_switch_port %s options:nat-addresses",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
						Output: "",
					},
				)
				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find logical_router_policy match=\"ip4.src == %s\" priority=%v external_ids:name=%s nexthop=\"%s\"",
							podIPv4,
							types.FloatingIPReroutePriority,
							fipName,
							nodeLogicalRouterIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@lr-policy create logical_router_policy action=reroute match=\"ip4.src == %s\" priority=%v nexthop=\"%s\" external_ids:name=%s -- add logical_router ovn_cluster_router policies @lr-policy",
							podIPv4,
							types.FloatingIPReroutePriority,
							nodeLogicalRouterIPv4,
							fipName,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find nat external_ids:name=%s logical_ip=\"%s\" external_ip=\"%s\"",
							fipName,
							podIPv4,
							floatingIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@nat create nat type=dnat_and_snat logical_port=k8s-%s external_ip=\"%s\" logical_ip=\"%s\" external_ids:name=%s -- add logical_router %s nat @nat",
							node1Name,
							floatingIPv4,
							podIPv4,
							fipName,
							util.GetGatewayRouterFromNode(node1Name),
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 set logical_switch_port %s options:nat-addresses=router",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
					},
				)

				fakeOvn.controller.WatchFloatingIP()
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected, 2).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc())

				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find logical_router_policy match=\"ip4.src == %s\" priority=%v external_ids:name=%s nexthop=\"%s\"",
							podIPv4,
							types.FloatingIPReroutePriority,
							fipName,
							nodeLogicalRouterIPv4,
						),
						Output: policyUuid,
					},
				)
				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find nat external_ids:name=%s logical_ip=\"%s\" external_ip=\"%s\"",
							fipName,
							podIPv4,
							floatingIPv4,
						),
						Output: natUuid,
					},
				)
				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 remove logical_router %s policies %s", types.OVNClusterRouter, policyUuid),
						fmt.Sprintf("ovn-nbctl --timeout=15 remove logical_router %s nat %s", util.GetGatewayRouterFromNode(node1Name), natUuid),
					},
				)

				fakeOvn.fakeExec.AddFakeCmd(
					&ovntest.ExpectedCmd{
						Cmd: fmt.Sprintf(
							"ovn-nbctl --timeout=15 --if-exist get logical_switch_port %s options:nat-addresses",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
						Output: "",
					},
				)
				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find logical_router_policy match=\"ip4.src == %s\" priority=%v external_ids:name=%s nexthop=\"%s\"",
							podIPv4,
							types.FloatingIPReroutePriority,
							fipName,
							nodeLogicalRouterIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@lr-policy create logical_router_policy action=reroute match=\"ip4.src == %s\" priority=%v nexthop=\"%s\" external_ids:name=%s -- add logical_router ovn_cluster_router policies @lr-policy",
							podIPv4,
							types.FloatingIPReroutePriority,
							nodeLogicalRouterIPv4,
							fipName,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --format=csv --data=bare --no-heading --columns=_uuid find nat external_ids:name=%s logical_ip=\"%s\" external_ip=\"%s\"",
							fipName,
							podIPv4,
							newFloatingIPv4,
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 --id=@nat create nat type=dnat_and_snat logical_port=k8s-%s external_ip=\"%s\" logical_ip=\"%s\" external_ids:name=%s -- add logical_router %s nat @nat",
							node1Name,
							newFloatingIPv4,
							podIPv4,
							fipName,
							util.GetGatewayRouterFromNode(node1Name),
						),
						fmt.Sprintf(
							"ovn-nbctl --timeout=15 set logical_switch_port %s options:nat-addresses=router",
							types.EXTSwitchToGWRouterPrefix+types.GWRouterPrefix+node1Name,
						),
					},
				)

				floatingIP.Status.FloatingIP = newFloatingIPv4
				_, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().Update(context.TODO(), &floatingIP, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected, 2).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc())

				return nil
			}

			err := app.Run([]string{app.Name})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not perform OVN transactions when pod's network is host", func() {
			app.Action = func(ctx *cli.Context) error {

				nodeIPv4 := "192.168.200.180/24"
				annotations := map[string]string{
					"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", nodeIPv4, ""),
				}
				labels := map[string]string{"k8s.ovn.org/floatingip-assignable": ""}

				node1 := *newFloatingIPNode(node1Name, annotations, labels)
				pod := *newPod("default", "pod", node1Name, "")
				pod.Spec.HostNetwork = true

				fakeOvn.start(ctx,
					&v1.NodeList{Items: []v1.Node{node1}},
					&v1.PodList{Items: []v1.Pod{pod}},
				)

				fakeOvn.controller.WatchFloatingIP()

				floatingIP := floatingip.FloatingIP{
					ObjectMeta: newObjectMeta(fipName, ""),
					Spec: floatingip.FloatingIPSpec{
						FloatingIPClaim: "claim",
						Pod:             "pod",
						PodNamespace:    "default",
					},
					Status: floatingip.FloatingIPStatus{
						NodeName:    node1Name,
						FloatingIP:  "",
						HostNetwork: true,
						PodIPs:      []string{},
					},
				}

				_, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().Create(context.TODO(), &floatingIP, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Eventually(fakeOvn.fakeExec.CalledMatchesExpected()).Should(BeTrue(), fakeOvn.fakeExec.ErrorDesc())

				return nil
			}

			err := app.Run([]string{app.Name})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
