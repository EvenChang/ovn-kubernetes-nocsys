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

var _ = ginkgo.Describe("OVN master FloatingIPClaim Operations", func() {
	const (
		fiNs1Name  = "floatingip-namespace1"
		fiPod1Name = "fi_pod1"
		fiNs2Name  = "floatingip-namespace2"
		fiPod2Name = "fi_pod2"
		fiPod1V4IP = "10.128.0.15"
		fiPod2V4IP = "10.128.0.16"
		fipName    = "floatingipprovider"
		ficName    = "floatingipclaim"

		fiNodeLogicalRouterIPv4     = "100.64.0.2"
		fiNodeLogicalRouterIfAddrV4 = fiNodeLogicalRouterIPv4 + "/29"
	)

	var (
		app     *cli.App
		fakeOvn *FakeOVN
		tExec   *ovntest.FakeExec

		fiPod1Label = map[string]string{"floatingip": "old"}
		fiNs1Label  = map[string]string{"floatingip": "old"}
		fiPod2Label = map[string]string{"floatingip": "new"}
		fiNs2Label  = map[string]string{"floatingip": "new"}
		fiNodeName  = "node"
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

	isFloatingIPOnPod := func(fiNsName string, fiPodName string) func() bool {
		return func() bool {
			tmps, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().List(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			for _, tmp := range tmps.Items {
				if tmp.Spec.PodNamespace == fiNsName && tmp.Spec.Pod == fiPodName {
					return true
				}
			}
			return false
		}
	}

	floatingIPCounter := func(ficName string) func() int {
		return func() int {
			tmps, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().List(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			count := 0
			for _, tmp := range tmps.Items {
				if tmp.Spec.FloatingIPClaim == ficName {
					count += 1
				}
			}
			return count
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

	ginkgo.Context("On Pod UPDATE", func() {
		ginkgo.It("when the floating ip claim stops matching due to the deletion of the Pod tag, the floating ip should be removed", func() {
			app.Action = func(ctx *cli.Context) error {
				fiNodeIPv4 := "192.168.126.202/24"

				fiPod := newPodWithLabels(fiNs1Name, fiPod1Name, fiNodeName, fiPod1V4IP, fiPod1Label)
				fiNs := newNamespaceWithLabels(fiNs1Name, fiNs1Label)

				fiNode := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNodeName,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNodeIPv4, ""),
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
							MatchLabels: fiNs1Label,
						},
						PodSelector: metav1.LabelSelector{
							MatchLabels: fiPod1Label,
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
						Items: []v1.Node{fiNode},
					},
					&v1.NamespaceList{
						Items: []v1.Namespace{*fiNs},
					},
					&v1.PodList{
						Items: []v1.Pod{*fiPod},
					})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node networks"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node options:nat-addresses=router"),
					},
				)

				go fakeOvn.controller.fIPNC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPNodes()
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode.Name)).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeTrue())

				fakeOvn.controller.WatchFloatingIPForClaim()
				gomega.Eventually(isFloatingIPOnPod(fiPod.Namespace, fiPod.Name), 4, 1).Should(gomega.BeTrue())

				fiPod.Labels = map[string]string{}
				_, err := fakeOvn.fakeClient.KubeClient.CoreV1().Pods(fiPod.Namespace).Update(context.TODO(), fiPod, metav1.UpdateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(isFloatingIPOnPod(fiPod.Namespace, fiPod.Name), 4, 1).Should(gomega.BeFalse())
				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("when the floating ip claim stops matching due to pod deletion, the floating ip should be removed", func() {
			app.Action = func(ctx *cli.Context) error {
				fiNodeIPv4 := "192.168.126.202/24"

				fiPod := newPodWithLabels(fiNs1Name, fiPod1Name, fiNodeName, fiPod1V4IP, fiPod1Label)
				fiNs := newNamespaceWithLabels(fiNs1Name, fiNs1Label)

				fiNode := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNodeName,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNodeIPv4, ""),
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
							MatchLabels: fiNs1Label,
						},
						PodSelector: metav1.LabelSelector{
							MatchLabels: fiPod1Label,
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
						Items: []v1.Node{fiNode},
					},
					&v1.NamespaceList{
						Items: []v1.Namespace{*fiNs},
					},
					&v1.PodList{
						Items: []v1.Pod{*fiPod},
					})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node networks"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node options:nat-addresses=router"),
					},
				)

				go fakeOvn.controller.fIPNC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPNodes()
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode.Name)).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeTrue())

				fakeOvn.controller.WatchFloatingIPForClaim()
				gomega.Eventually(isFloatingIPOnPod(fiPod.Namespace, fiPod.Name), 4, 1).Should(gomega.BeTrue())

				err := fakeOvn.fakeClient.KubeClient.CoreV1().Pods(fiPod.Namespace).Delete(context.TODO(), fiPod.Name, metav1.DeleteOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(isFloatingIPOnPod(fiPod.Namespace, fiPod.Name), 4, 1).Should(gomega.BeFalse())
				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("On FloatingIPClaim UPDATE", func() {
		ginkgo.It("when the selector changes, remove the floating ip assigned to the mismatched pod and assign the floating ip to the newly matched pod", func() {
			app.Action = func(ctx *cli.Context) error {
				fiNodeIPv4 := "192.168.126.202/24"

				fiPod1 := newPodWithLabels(fiNs1Name, fiPod1Name, fiNodeName, fiPod1V4IP, fiPod1Label)
				fiNs1 := newNamespaceWithLabels(fiNs1Name, fiNs1Label)
				fiPod2 := newPodWithLabels(fiNs2Name, fiPod2Name, fiNodeName, fiPod2V4IP, fiPod2Label)
				fiNs2 := newNamespaceWithLabels(fiNs2Name, fiNs2Label)

				fiNode := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNodeName,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNodeIPv4, ""),
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

				fip := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fipName,
					},
					Spec: floatingipproviderapi.FloatingIPProviderSpec{
						FloatingIPs: []string{
							"192.168.126.100,192.168.126.110",
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
							MatchLabels: fiNs1Label,
						},
						PodSelector: metav1.LabelSelector{
							MatchLabels: fiPod1Label,
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
						Items: []v1.Node{fiNode},
					},
					&v1.NamespaceList{
						Items: []v1.Namespace{*fiNs1, *fiNs2},
					},
					&v1.PodList{
						Items: []v1.Pod{*fiPod1, *fiPod2},
					})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node networks"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node options:nat-addresses=router"),
					},
				)

				go fakeOvn.controller.fIPNC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPNodes()
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode.Name)).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeTrue())

				fakeOvn.controller.WatchFloatingIPForClaim()
				gomega.Eventually(isFloatingIPOnPod(fiPod1.Namespace, fiPod1.Name), 4, 1).Should(gomega.BeTrue())

				fic.Spec.NamespaceSelector.MatchLabels = fiNs2Label
				fic.Spec.PodSelector.MatchLabels = fiPod2Label
				_, err := fakeOvn.fakeClient.FloatingIPClaimClient.K8sV1().FloatingIPClaims().Update(context.TODO(), &fic, metav1.UpdateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Eventually(isFloatingIPOnPod(fiPod2.Namespace, fiPod2.Name), 4, 1).Should(gomega.BeTrue())
				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("when the address range changes, remove the floating ip that is not in the address range", func() {
			app.Action = func(ctx *cli.Context) error {
				fiNodeIPv4 := "192.168.126.202/24"

				fiPod1 := newPodWithLabels(fiNs1Name, fiPod1Name, fiNodeName, fiPod1V4IP, fiPod1Label)
				fiNs1 := newNamespaceWithLabels(fiNs1Name, fiNs1Label)
				fiPod2 := newPodWithLabels(fiNs2Name, fiPod2Name, fiNodeName, fiPod2V4IP, fiPod2Label)
				fiNs2 := newNamespaceWithLabels(fiNs2Name, fiNs2Label)

				fiNode := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNodeName,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNodeIPv4, ""),
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

				fip := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fipName,
					},
					Spec: floatingipproviderapi.FloatingIPProviderSpec{
						FloatingIPs: []string{
							"192.168.126.100,192.168.126.110",
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
							"192.168.126.107",
						},
						NamespaceSelector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "floatingip",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"old", "new"},
								},
							},
						},
						PodSelector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "floatingip",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"old", "new"},
								},
							},
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
						Items: []v1.Node{fiNode},
					},
					&v1.NamespaceList{
						Items: []v1.Namespace{*fiNs1, *fiNs2},
					},
					&v1.PodList{
						Items: []v1.Pod{*fiPod1, *fiPod2},
					})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node networks"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node options:nat-addresses=router"),
					},
				)

				go fakeOvn.controller.fIPNC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPNodes()
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode.Name)).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeTrue())

				fakeOvn.controller.WatchFloatingIPForClaim()
				gomega.Eventually(floatingIPCounter(fic.Name), 4, 1).Should(gomega.Equal(2))

				fic.Spec.FloatingIPs = []string{"192.168.126.105"}
				_, err := fakeOvn.fakeClient.FloatingIPClaimClient.K8sV1().FloatingIPClaims().Update(context.TODO(), &fic, metav1.UpdateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Eventually(floatingIPCounter(fic.Name), 4, 1).Should(gomega.Equal(1))
				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("when a pod with floating ip is released, the floating ip should be re-assigned", func() {
			app.Action = func(ctx *cli.Context) error {
				fiNodeIPv4 := "192.168.126.202/24"

				fiPod1 := newPodWithLabels(fiNs1Name, fiPod1Name, fiNodeName, fiPod1V4IP, fiPod1Label)
				fiPod2 := newPodWithLabels(fiNs1Name, fiPod2Name, fiNodeName, fiPod2V4IP, fiPod2Label)
				fiNs1 := newNamespaceWithLabels(fiNs1Name, fiNs1Label)

				fiNode := v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fiNodeName,
						Annotations: map[string]string{
							"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", fiNodeIPv4, ""),
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

				fip := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fipName,
					},
					Spec: floatingipproviderapi.FloatingIPProviderSpec{
						FloatingIPs: []string{
							"192.168.126.100,192.168.126.110",
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
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "floatingip",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"old", "new"},
								},
							},
						},
						PodSelector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "floatingip",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"old", "new"},
								},
							},
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
						Items: []v1.Node{fiNode},
					},
					&v1.NamespaceList{
						Items: []v1.Namespace{*fiNs1},
					},
					&v1.PodList{
						Items: []v1.Pod{*fiPod1, *fiPod2},
					})

				fakeOvn.fakeExec.AddFakeCmdsNoOutputNoError(
					[]string{
						fmt.Sprintf("ovn-nbctl --timeout=15 --if-exist get logical_router_port rtoj-GR_node networks"),
						fmt.Sprintf("ovn-nbctl --timeout=15 set logical_switch_port etor-GR_node options:nat-addresses=router"),
					},
				)

				go fakeOvn.controller.fIPNC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPNodes()
				gomega.Eventually(isFloatingIPNodeAssignable(fiNode.Name)).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeTrue())

				fakeOvn.controller.WatchFloatingIPForClaim()
				fiObjs, err := fakeOvn.fakeClient.FloatingIPClient.K8sV1().FloatingIPs().List(context.TODO(), metav1.ListOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(len(fiObjs.Items)).Should(gomega.Equal(1))

				fiObj := fiObjs.Items[0]
				err = fakeOvn.fakeClient.KubeClient.CoreV1().Pods(fiNs1.Name).Delete(context.TODO(), fiObj.Spec.Pod, metav1.DeleteOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				if fiObj.Spec.Pod == fiPod1.Name {
					gomega.Eventually(isFloatingIPOnPod(fiPod2.Namespace, fiPod2.Name), 4, 1).Should(gomega.BeTrue())
				} else {
					gomega.Eventually(isFloatingIPOnPod(fiPod1.Namespace, fiPod1.Name), 4, 1).Should(gomega.BeTrue())
				}
				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
