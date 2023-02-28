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
			dashboardLocalPort, err := getDashboardLocalPort()
			ui.PrintOnError("checking dashboard port", err)

			uri := fmt.Sprintf("http://localhost:%d", dashboardLocalPort)
			if useGlobalDashboard {
				uri = DashboardURI
			}

			apiURI := fmt.Sprintf("localhost:%d/%s", ApiServerPort, ApiVersion)
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

			namespace := cmd.Flag("namespace").Value.String()
			// if not global dasboard - we'll try to port-forward current cluster API
			if !useGlobalDashboard {
				command, err := asyncPortForward(namespace, DashboardName, dashboardLocalPort, DashboardPort)
				ui.PrintOnError("port forwarding dashboard endpoint", err)
				commandsToKill = append(commandsToKill, command)
			}

			command, err := asyncPortForward(namespace, ApiServerName, ApiServerPort, ApiServerPort)
			ui.PrintOnError("port forwarding api endpoint", err)
			commandsToKill = append(commandsToKill, command)

			// check for api and dasboard to be ready
			ready, err := readinessCheck(apiAddress, dashboardAddress)
			ui.PrintOnError("checking readiness of services", err)
			ui.Debug("Endpoints readiness", fmt.Sprintf("%v", ready))

			// open browser
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
		},
	}

	cmd.Flags().BoolVar(&useGlobalDashboard, "use-global-dashboard", false, "use global dashboard for viewing testkube results")
	return cmd
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

func getDashboardLocalPort() (int, error) {
	for port := DashboardLocalPort; port <= maxPortNumber; port++ {
		if port == ApiServerPort {
			continue
		}

		if localPortCheck(port) == nil {
			return port, nil
		}
	}

	return 0, errors.New("no available local port")
}
