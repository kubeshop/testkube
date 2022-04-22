package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// NewDashboardCmd is a method to create new dashboard command
func NewDashboardCmd() *cobra.Command {
	var (
		namespace          string
		useGlobalDashboard bool
	)

	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"d", "open-dashboard"},
		Short:   "Open testkube dashboard",
		Long:    `Open testkube dashboard`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			uri := fmt.Sprintf("http://localhost:%d", DashboardLocalPort)
			if useGlobalDashboard {
				uri = DashboardURI
			}

			apiURI := fmt.Sprintf("localhost:%d/%s", ApiServerPort, ApiVersion)
			dashboardAddress := fmt.Sprintf("%s/apiEndpoint?apiEndpoint=%s", uri, apiURI)
			apiAddress := fmt.Sprintf("http://%s", apiURI)

			var (
				commandsToKill []*exec.Cmd
				err            error
			)

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
				command, err := asyncPortForward(namespace, DashboardName, DashboardLocalPort, DashboardPort)
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
			openCmd, err := getOpenCommand()
			if err == nil {
				_, err = process.Execute(openCmd, dashboardAddress)
				ui.PrintOnError("openning dashboard", err)
			}

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

	cmd.Flags().StringVarP(&namespace, "namespace", "s", "testkube", "namespace where the testkube is installed")
	cmd.Flags().BoolVar(&useGlobalDashboard, "use-global-dashboard", false, "use global dashboard for viewing testkube results")
	return cmd
}

func readinessCheck(apiURI, dashboardURI string) (bool, error) {
	const readinessCheckTimeout = 30 * time.Second
	client := http.NewClient()

	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		time.Sleep(readinessCheckTimeout)
		ticker.Stop()
	}()

	for range ticker.C {
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

	return false, fmt.Errorf("timed-out waiting for dashboard and api")
}

func getOpenCommand() (string, error) {
	os := runtime.GOOS
	switch os {
	case "windows":
		return "start", nil
	case "darwin":
		return "open", nil
	case "linux":
		return "xdg-open", nil
	default:
		return "", fmt.Errorf("unsupported OS")
	}
}

func asyncPortForward(namespace, deploymentName string, localPort, clusterPort int) (command *exec.Cmd, err error) {
	fullDeploymentName := fmt.Sprintf("deployment/%s", deploymentName)
	ports := fmt.Sprintf("%d:%d", localPort, clusterPort)
	return process.ExecuteAsync("kubectl", "port-forward", "--namespace", namespace, fullDeploymentName, ports)
}
