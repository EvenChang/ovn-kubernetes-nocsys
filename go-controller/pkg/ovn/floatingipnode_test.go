package ovn

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	floatingipclaimapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	floatingipproviderapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipprovider/v1"
	ovntest "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/testing"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("OVN master FloatingIPNode Operations", func() {
	const (
		fiNsName  = "floatingip-namespace"
		fiPodName = "fi_pod"
		fiPodV4IP = "10.128.0.15"
		fipName   = "floatingipprovider"
		ficName   = "floatingipclaim"
	)

	var (
		app     *cli.App
		fakeOvn *FakeOVN
		tExec   *ovntest.FakeExec

		fiPodLabel  = map[string]string{"floatingip": "needed"}
		fiNsLabel   = map[string]string{"floatingip": "needed"}
		fiNode1Name = "node1"
		fiNode2Name = "node2"
	)

	isFloatingIPProviderReady := func(fipName string) func() bool {
		return func() bool {
			tmp, err := fakeOvn.fakeClient.FloatingIPProviderClient.K8sV1().FloatingIPProviders().Get(context.TODO(), fipName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			return tmp.Status.Phase == floatingipproviderapi.FloatingIPProviderReady
		}
	}

	isFloatingIPClaimReady := func(ficName string) func() bool {
		return func() bool {
			tmp, err := fakeOvn.fakeClient.FloatingIPClaimClient.K8sV1().FloatingIPClaims().Get(context.TODO(), ficName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			return tmp.Status.Phase == floatingipclaimapi.FloatingIPClaimReady
		}
	}

	isFloatingIPNodeAssignable := func(fiNode string) func() bool {
		return func() bool {
			fakeOvn.controller.fIPNC.ficc.nodeMutex.Lock()
			defer fakeOvn.controller.fIPNC.ficc.nodeMutex.Unlock()
			_, exists := fakeOvn.controller.fIPNC.ficc.fiOnNodes[fiNode]
			return exists
		}
	}

	isFloatingIPOnNode := func(fiPod string, fiNode string) func() bool {
		return func() bool {
			tmps, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().List(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			for _, tmp := range tmps.Items {
				if tmp.Spec.Pod == fiPod {
					if tmp.Status.NodeName == fiNode {
						return true
					} else {
						return false
					}
				}
			}
			return false
		}
	}

	ginkgo.BeforeEach(func() {
		// Restore global default values before each testcase
		config.PrepareTestConfig()
		config.OVNKubernetesFeature.EnableFloatingIP = true

		app = cli.NewApp()
		app.Name = "test"
		app.Flags = config.Flags

		tExec = ovntest.NewLooseCompareFakeExec()
		fakeOvn = NewFakeOVN(tExec)

	})

	ginkgo.AfterEach(func() {
		fakeOvn.shutdown()
	})

	ginkgo.Context("On node UPDATE", func() {
		ginkgo.It("should re-assign FloatingIPs and perform proper OVN transactions when pod is created after node floating ip label switch", func() {
			app.Action = func(ctx *cli.Context) error {
				fiNode1IPv4 := "192.168.126.202/24"
				fiNode2IPv4 := "192.168.126.51/24"

				fiPod := newPodWithLabels(fiNsName, fiPodName, fiNode1Name, fiPodV4IP, fiPodLabel)
				fiNs := newNamespaceWithLabels(fiNsName, fiNsLabel)

				fiNode1 := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNode1Name,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNode1IPv4, ""),
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
				fiNode2 := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNode2Name,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNode2IPv4, ""),
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

				fip := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fipName,
					},
					Spec: floatingipproviderapi.FloatingIPProviderSpec{
						FloatingIPs: []string{
							"192.168.126.100,192.168.126.110",
							"192.168.126.120",
						},
						VpcSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
					},
				}

				fic := floatingipclaimapi.FloatingIPClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: ficName,
					},
					Spec: floatingipclaimapi.FloatingIPClaimSpec{
						Provider: fipName,
						FloatingIPs: []string{
							"192.168.126.105",
						},
						NamespaceSelector: metav1.LabelSelector{
							MatchLabels: fiNsLabel,
						},
						PodSelector: metav1.LabelSelector{
							MatchLabels: fiPodLabel,
						},
					},
				}

				fakeOvn.start(ctx,
					&floatingipproviderapi.FloatingIPProviderList{
						Items: []floatingipproviderapi.FloatingIPProvider{fip},
					},
					&floatingipclaimapi.FloatingIPClaimList{
						Items: []floatingipclaimapi.FloatingIPClaim{fic},
					},
					&v1.NodeList{
						Items: []v1.Node{fiNode1, fiNode2},
					},
					&v1.NamespaceList{
						Items: []v1.Namespace{*fiNs},
					},
					&v1.PodList{
						Items: []v1.Pod{*fiPod},
					})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node1 networks"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node1 options:nat-addresses=router"),
					},
				)

				go fakeOvn.controller.fIPNC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPNodes()
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode1.Name)).Should(gomega.BeTrue())
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode2.Name)).Should(gomega.BeFalse())

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeTrue())

				fakeOvn.controller.WatchFloatingIPForClaim()
				gomega.Eventually(isFloatingIPOnNode(fiPod.Name, fiNode1.Name), 4, 1).Should(gomega.BeTrue())

				fiNode1.Labels = map[string]string{}
				fiNode2.Labels = map[string]string{
					"k8s.ovn.org/floatingip-assignable": "",
				}

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 remove logical_switch_port etor-GR_node1 options nat-addresses=router"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node2 options:nat-addresses=router"),
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node2 networks"),
					},
				)

				_, err := fakeOvn.fakeClient.KubeClient.CoreV1().Nodes().Update(context.TODO(), &fiNode1, metav1.UpdateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, err = fakeOvn.fakeClient.KubeClient.CoreV1().Nodes().Update(context.TODO(), &fiNode2, metav1.UpdateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Eventually(isFloatingIPOnNode(fiPod.Name, fiNode2.Name), 4, 1).Should(gomega.BeTrue())
				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
