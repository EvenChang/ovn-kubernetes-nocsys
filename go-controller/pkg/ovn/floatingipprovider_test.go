package ovn

import (
	"context"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	floatingipclaimapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	floatingipproviderapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipprovider/v1"
	ovntest "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/testing"
	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("OVN master FloatingIPProvider Operations", func() {
	const (
		fip1Name = "floatingipprovider1"
		fip2Name = "floatingipprovider2"
		ficName  = "floatingipclaim"
	)

	var (
		app     *cli.App
		fakeOvn *FakeOVN
		tExec   *ovntest.FakeExec
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

	ginkgo.Context("On FloatingIPProvider UPDATE", func() {
		ginkgo.It("if the address conflicts with the address of an existing object, the status is marked as not ready", func() {
			app.Action = func(ctx *cli.Context) error {
				fip1 := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fip1Name,
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

				fip2 := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fip2Name,
					},
					Spec: floatingipproviderapi.FloatingIPProviderSpec{
						FloatingIPs: []string{
							"192.168.126.105",
						},
						VpcSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
					},
				}

				fakeOvn.start(ctx,
					&floatingipproviderapi.FloatingIPProviderList{
						Items: []floatingipproviderapi.FloatingIPProvider{fip1},
					})

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip1.Name), 4, 1).Should(gomega.BeTrue())

				_, err := fakeOvn.fakeClient.FloatingIPProviderClient.K8sV1().FloatingIPProviders().Create(context.TODO(), &fip2, metav1.CreateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Eventually(isFloatingIPProviderReady(fip2.Name), 4, 1).Should(gomega.BeFalse())

				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("if the provider does not exist, mark the status of the claim object as not ready", func() {
			app.Action = func(ctx *cli.Context) error {
				fic := floatingipclaimapi.FloatingIPClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: ficName,
					},
					Spec: floatingipclaimapi.FloatingIPClaimSpec{
						Provider: fip1Name,
						FloatingIPs: []string{
							"192.168.126.105",
						},
						NamespaceSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
						PodSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
					},
				}

				fakeOvn.start(ctx,
					&floatingipclaimapi.FloatingIPClaimList{
						Items: []floatingipclaimapi.FloatingIPClaim{fic},
					})

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeFalse())

				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("If the requested address conflicts with the provider, mark the status of the claim object as not ready", func() {
			app.Action = func(ctx *cli.Context) error {
				fip := floatingipproviderapi.FloatingIPProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: fip1Name,
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
						Provider: fip1Name,
						FloatingIPs: []string{
							"192.168.126.50",
						},
						NamespaceSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
						PodSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{},
						},
					},
				}

				fakeOvn.start(ctx,
					&floatingipproviderapi.FloatingIPProviderList{
						Items: []floatingipproviderapi.FloatingIPProvider{fip},
					},
					&floatingipclaimapi.FloatingIPClaimList{
						Items: []floatingipclaimapi.FloatingIPClaim{fic},
					})

				go fakeOvn.controller.fIPPC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPProvider()
				gomega.Eventually(isFloatingIPProviderReady(fip.Name), 4, 1).Should(gomega.BeTrue())

				go fakeOvn.controller.fIPCC.Run(fakeOvn.stopChan)
				fakeOvn.controller.WatchFloatingIPClaim()
				gomega.Eventually(isFloatingIPClaimReady(fic.Name), 4, 1).Should(gomega.BeFalse())

				return nil
			}

			err := app.Run([]string{app.Name})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
