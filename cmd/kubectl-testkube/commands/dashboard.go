package commands

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

const maxPortNumber = 65535

// NewDashboardCmd is a method to create new dashboard command
func NewDashboardCmd() *cobra.Command {
	var (
		useGlobalDashboard bool
	)

	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d", "open-dashboard"},
		Short:   "Open testkube dashboard",
		Long:    `Open testkube dashboard`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.APIServerName == "" {
				cfg.APIServerName = config.APIServerName
			}

			if cfg.APIServerPort == 0 {
				cfg.APIServerPort = config.APIServerPort
			}

			if cfg.DashboardName == "" {
				cfg.DashboardName = config.DashboardName
			}

			if cfg.DashboardPort == 0 {
				cfg.DashboardPort = config.DashboardPort
			}

			namespace := cmd.Flag("namespace").Value.String()

			if cfg.ContextType == config.ContextTypeCloud {
				openCloudDashboard(cfg)

			} else {
				openLocalDashboard(cfg, useGlobalDashboard, namespace)
			}

		},
	}

	cmd.Flags().BoolVar(&useGlobalDashboard, "use-global-dashboard", false, "use global dashboard for viewing testkube results")
	return cmd
}

func openCloudDashboard(cfg config.Data) {
	// open browser
	uri := fmt.Sprintf("%s/organization/%s/environment/%s", cfg.CloudContext.UiUri, cfg.CloudContext.OrganizationId, cfg.CloudContext.EnvironmentId)
	err := open.Run(uri)
	ui.PrintOnError("openning dashboard", err)
}

func openLocalDashboard(cfg config.Data, useGlobalDashboard bool, namespace string) {

	dashboardLocalPort, err := getDashboardLocalPort(cfg.APIServerPort)
	ui.PrintOnError("checking dashboard port", err)

	uri := fmt.Sprintf("http://localhost:%d", dashboardLocalPort)
	if useGlobalDashboard {
		uri = DashboardURI
	}

	apiURI := fmt.Sprintf("localhost:%d/%s", cfg.APIServerPort, ApiVersion)
	dashboardAddress := fmt.Sprintf("%s/apiEndpoint?apiEndpoint=%s", uri, apiURI)
	apiAddress := fmt.Sprintf("http://%s", apiURI)

	var commandsToKill []*exec.Cmd

	// kill background port-forwarded processes
	defer func() {
		for _, command := range commandsToKill {
			if command != nil {
				err := command.Process.Kill()
				ui.PrintOnError("killing command: "+command.String(), err)
			}
		}
	}()

	// if not global dasboard - we'll try to port-forward current cluster API
	if !useGlobalDashboard {
		command, err := asyncPortForward(namespace, cfg.DashboardName, dashboardLocalPort, cfg.DashboardPort)
		ui.PrintOnError("port forwarding dashboard endpoint", err)
		commandsToKill = append(commandsToKill, command)
	}

	command, err := asyncPortForward(namespace, cfg.APIServerName, cfg.APIServerPort, cfg.APIServerPort)
	ui.PrintOnError("port forwarding api endpoint", err)
	commandsToKill = append(commandsToKill, command)

	// check for api and dasboard to be ready
	ready, err := readinessCheck(apiAddress, dashboardAddress)
	ui.PrintOnError("checking readiness of services", err)
	ui.Debug("Endpoints readiness", fmt.Sprintf("%v", ready))

	// open browser
	err = open.Run(dashboardAddress)
	ui.PrintOnError("openning dashboard", err)

	// wait for Ctrl/Cmd + c signal to clear everything
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ui.NL()
	ui.Success("The dashboard is accessible here:", dashboardAddress)
	ui.Success("The API is accessible here:", apiAddress+"/info")
	ui.Success("Port forwarding is started for the test results endpoint, hit Ctrl+c (or Cmd+c) to stop")

	s := <-c
	fmt.Println("Got signal:", s)
}

func readinessCheck(apiURI, dashboardURI string) (bool, error) {
	const readinessCheckTimeout = 30 * time.Second
	client := http.NewClient()

	ticker := time.NewTicker(readinessCheckTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			return false, fmt.Errorf("timed-out waiting for dashboard and api")
		default:
			apiResp, err := client.Get(apiURI + "/info")
			if err != nil {
				continue
			}
			dashboardResp, err := client.Get(dashboardURI)
			if err != nil {
				continue
			}

			if apiResp.StatusCode < 400 && dashboardResp.StatusCode < 400 {
				return true, nil
			}
		}
	}
}

func asyncPortForward(namespace, deploymentName string, localPort, clusterPort int) (command *exec.Cmd, err error) {
	fullDeploymentName := fmt.Sprintf("deployment/%s", deploymentName)
	ports := fmt.Sprintf("%d:%d", localPort, clusterPort)
	return process.ExecuteAsync("kubectl", "port-forward", "--namespace", namespace, fullDeploymentName, ports)
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
