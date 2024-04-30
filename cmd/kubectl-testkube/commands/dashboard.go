package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/ui"
)

const maxPortNumber = 65535

// NewDashboardCmd is a method to create new dashboard command
func NewDashboardCmd() *cobra.Command {
	var namespace string
	var verbose bool

	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d", "open-dashboard", "ui"},
		Short:   "Open Testkube dashboard",
		Long:    `Open Testkube dashboard`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if namespace == "" {
				namespace = cfg.Namespace
			}

			if cfg.ContextType != config.ContextTypeCloud {
				isDashboardRunning, _ := k8sclient.IsPodOfServiceRunning(context.Background(), cfg.Namespace, config.EnterpriseUiName)
				if isDashboardRunning {
					openOnPremDashboard(cmd, cfg, verbose)
				} else {
					ui.Warn("No dashboard found. Is it running in the " + namespace + " namespace?")
				}
			} else {
				openCloudDashboard(cfg)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "", false, "show additional debug messages")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to install "+demoInstallationName)

	return cmd
}

func openCloudDashboard(cfg config.Data) {
	// open browser
	uri := fmt.Sprintf("%s/organization/%s/environment/%s", cfg.CloudContext.UiUri, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId)
	err := open.Run(uri)
	ui.PrintOnError("openning dashboard", err)
}

func openOnPremDashboard(cmd *cobra.Command, cfg config.Data, verbose bool) {
	uiLocalPort, err := getDashboardLocalPort(config.EnterpriseApiForwardingPort)
	ui.PrintOnError("getting an ui forwarding available port", err)
	uri := fmt.Sprintf("http://localhost:%d", uiLocalPort)

	ctx, cancel := context.WithCancel(context.Background())
	err = k8sclient.PortForward(ctx, cfg.Namespace, config.EnterpriseApiName, config.EnterpriseApiPort, config.EnterpriseApiForwardingPort, verbose)
	ui.PrintOnError("port forwarding api", err)
	err = k8sclient.PortForward(ctx, cfg.Namespace, config.EnterpriseUiName, config.EnterpriseUiPort, uiLocalPort, verbose)
	ui.PrintOnError("port forwarding ui", err)
	err = k8sclient.PortForward(ctx, cfg.Namespace, config.EnterpriseDexName, config.EnterpriseDexPort, config.EnterpriseDexForwardingPort, verbose)
	ui.PrintOnError("port forwarding dex", err)

	err = open.Run(uri)
	ui.ExitOnError("opening dashboard in browser", err)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ui.Success("The dashboard is accessible here:", uri)
	ui.Success("Port forwarding the necessary services, hit Ctrl+c (or Cmd+c) to stop")
	<-c
	cancel()

}

func localPortCheck(port int) error {
	ln, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return err
	}

	ln.Close()
	return nil
}

func getDashboardLocalPort(apiServerPort int) (int, error) {
	for port := DashboardLocalPort; port <= maxPortNumber; port++ {
		if port == apiServerPort {
			continue
		}

		if localPortCheck(port) == nil {
			return port, nil
		}
	}

	return 0, errors.New("no available local port")
}
