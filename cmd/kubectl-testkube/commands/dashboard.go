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
	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d", "open-dashboard", "ui"},
		Short:   "Open Testkube Pro dashboard",
		Long:    `Open Testkube Pro dashboard`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.ContextType != config.ContextTypeCloud {
				isEnyterpriseInstalled, _ := k8sclient.IsPodOfServiceRunning(context.Background(), cfg.EnterpriseNamespace, config.EnterpriseUiName)
				if isEnyterpriseInstalled {
					openOnPremDashboard(cmd, cfg)
				} else {
					ui.Warn("As of 1.17 the dashboard is no longer included with Testkube Core OSS - please refer to https://bit.ly/tk-dashboard for more info")
				}
			} else {
				openCloudDashboard(cfg)
			}
		},
	}

	return cmd
}

func openCloudDashboard(cfg config.Data) {
	// open browser
	uri := fmt.Sprintf("%s/organization/%s/environment/%s", cfg.CloudContext.UiUri, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId)
	err := open.Run(uri)
	ui.PrintOnError("openning dashboard", err)
}

func openOnPremDashboard(cmd *cobra.Command, cfg config.Data) {
	uiLocalPort, err := getDashboardLocalPort(config.EnterpriseApiForwardingPort)
	ui.PrintOnError("getting an ui forwarding available port", err)
	uri := fmt.Sprintf("http://localhost:%d", uiLocalPort)

	ctx, cancel := context.WithCancel(context.Background())
	err = k8sclient.PortForward(ctx, cfg.EnterpriseNamespace, config.EnterpriseApiName, config.EnterpriseApiPort, config.EnterpriseApiForwardingPort)
	ui.PrintOnError("port forwarding api", err)
	err = k8sclient.PortForward(ctx, cfg.EnterpriseNamespace, config.EnterpriseUiName, config.EnterpriseUiPort, uiLocalPort)
	ui.PrintOnError("port forwarding ui", err)
	err = k8sclient.PortForward(ctx, cfg.EnterpriseNamespace, config.EnterpriseDexName, config.EnterpriseDexPort, config.EnterpriseDexForwardingPort)
	ui.PrintOnError("port forwarding dex", err)

	err = open.Run(uri)
	ui.ExitOnError("openning dashboard in browser", err)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ui.NL()
	ui.Success("The on prem ui is accessible here:", uri)
	ui.Success("Port forwarding is started for the needed components, hit Ctrl+c (or Cmd+c) to stop")
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
