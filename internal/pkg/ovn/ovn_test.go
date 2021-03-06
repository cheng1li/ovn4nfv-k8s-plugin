package ovn

import (
	"fmt"
	"testing"

	"github.com/urfave/cli"
	fakeexec "k8s.io/utils/exec/testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ovn4nfv-k8s-plugin/internal/pkg/config"
	ovntest "ovn4nfv-k8s-plugin/internal/pkg/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOvn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OVN/Pod Test Suite")
}

var _ = AfterSuite(func() {
})

var _ = Describe("Add logical Port", func() {
	var app *cli.App

	BeforeEach(func() {
		app = cli.NewApp()
		app.Name = "test"
		app.Flags = config.Flags

	})

	It("tests Pod", func() {
		app.Action = func(ctx *cli.Context) error {
			const (
				gwIP         string = "10.1.1.1"
				gwCIDR       string = gwIP + "/24"
				netName      string = "ovn-prot-net"
				portName     string = "_ok_net0"
				macIPAddress string = "0a:00:00:00:00:01 192.168.1.3"
			)
			fakeCmds := ovntest.AddFakeCmd(nil, &ovntest.ExpectedCmd{
				Cmd:    "ovn-nbctl --timeout=15 --data=bare --no-heading --columns=name find logical_switch " + "name=" + netName,
				Output: netName,
			})
			fakeCmds = ovntest.AddFakeCmdsNoOutputNoError(fakeCmds, []string{
				"ovn-nbctl --timeout=15 --wait=sb -- --may-exist lsp-add " + netName + " " + portName + " -- lsp-set-addresses " + portName + " dynamic -- set logical_switch_port " + portName + " external-ids:namespace= external-ids:logical_switch=" + netName + " external-ids:pod=true",
			})

			fakeCmds = ovntest.AddFakeCmd(fakeCmds, &ovntest.ExpectedCmd{
				Cmd:    "ovn-nbctl --timeout=15 get logical_switch_port " + portName + " dynamic_addresses",
				Output: macIPAddress,
			})
			fakeCmds = ovntest.AddFakeCmd(fakeCmds, &ovntest.ExpectedCmd{
				Cmd:    "ovn-nbctl --timeout=15 --if-exists get logical_switch " + netName + " external_ids:gateway_ip",
				Output: gwCIDR,
			})

			fexec := &fakeexec.FakeExec{
				CommandScript: fakeCmds,
				LookPathFunc: func(file string) (string, error) {
					return fmt.Sprintf("/fake-bin/%s", file), nil
				},
			}
			oldSetupOvnUtils := SetupOvnUtils
			// as we are exiting, revert ConfigureInterface back  at end of function
			defer func() { SetupOvnUtils = oldSetupOvnUtils }()
			SetupOvnUtils = func() error {
				return nil
			}
			ovnController, err := NewOvnController(fexec)
			Expect(err).NotTo(HaveOccurred())

			var (
				okPod = v1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ok",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "by-name",
							},
							{},
						},
					},
				}
			)
			a := []map[string]interface{}{{"name": "ovn-prot-net", "interface": "net0"}}
			ovnController.AddLogicalPorts(&okPod, a)
			Expect(fexec.CommandCalls).To(Equal(len(fakeCmds)))

			return nil
		}

		err := app.Run([]string{app.Name})
		Expect(err).NotTo(HaveOccurred())
	})

	It("tests Pod provider", func() {
		app.Action = func(ctx *cli.Context) error {
			const (
				gwIP         string = "10.1.1.1"
				gwCIDR       string = gwIP + "/24"
				netName      string = "ovn-prot-net"
				portName     string = "_ok_net0"
				macIPAddress string = "0a:00:00:00:00:01 192.168.1.3/24"
			)
			fakeCmds := ovntest.AddFakeCmd(nil, &ovntest.ExpectedCmd{
				Cmd:    "ovn-nbctl --timeout=15 --data=bare --no-heading --columns=name find logical_switch " + "name=" + netName,
				Output: netName,
			})

			fakeCmds = ovntest.AddFakeCmdsNoOutputNoError(fakeCmds, []string{
				"ovn-nbctl --timeout=15 --may-exist lsp-add " + netName + " " + portName + " -- lsp-set-addresses " + portName + " " + macIPAddress + " -- --if-exists clear logical_switch_port " + portName + " dynamic_addresses" + " -- set logical_switch_port " + portName + " external-ids:namespace= external-ids:logical_switch=" + netName + " external-ids:pod=true",
			})

			fakeCmds = ovntest.AddFakeCmd(fakeCmds, &ovntest.ExpectedCmd{
				Cmd:    "ovn-nbctl --timeout=15 get logical_switch_port " + portName + " addresses",
				Output: macIPAddress,
			})

			fakeCmds = ovntest.AddFakeCmd(fakeCmds, &ovntest.ExpectedCmd{
				Cmd:    "ovn-nbctl --timeout=15 --if-exists get logical_switch " + netName + " external_ids:gateway_ip",
				Output: gwCIDR,
			})

			fexec := &fakeexec.FakeExec{
				CommandScript: fakeCmds,
				LookPathFunc: func(file string) (string, error) {
					return fmt.Sprintf("/fake-bin/%s", file), nil
				},
			}
			oldSetupOvnUtils := SetupOvnUtils
			// as we are exiting, revert ConfigureInterface back  at end of function
			defer func() { SetupOvnUtils = oldSetupOvnUtils }()
			SetupOvnUtils = func() error {
				return nil
			}
			ovnController, err := NewOvnController(fexec)
			Expect(err).NotTo(HaveOccurred())
			var (
				okPod = v1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ok",
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "by-name",
							},
							{},
						},
					},
				}
			)
			a := []map[string]interface{}{{"name": "ovn-prot-net", "interface": "net0", "netType": "provider", "ipAddress": "192.168.1.3/24", "macAddress": "0a:00:00:00:00:01"}}
			ovnController.AddLogicalPorts(&okPod, a)
			Expect(fexec.CommandCalls).To(Equal(len(fakeCmds)))

			return nil
		}

		err := app.Run([]string{app.Name})
		Expect(err).NotTo(HaveOccurred())
	})

})
