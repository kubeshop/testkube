package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"

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
		Use:   "dashboard",
		Short: "Open testkube dashboard",
		Long:  `Open testkube dashboard`,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true
			ui.Logo()

			uri := fmt.Sprintf("http://localhost:%d", DashboardLocalPort)
			if useGlobalDashboard {
				uri = DashboardURI
			}

			dashboardAddress := fmt.Sprintf("%s/apiEndpoint?apiEndpoint=localhost:%d/%s", uri, ApiServerPort, ApiVersion)
			ui.Success("The dashboard is accessible here:", dashboardAddress)
			ui.Success("Port forwarding is started for the test results endpoint, hit Ctrl+C to stop")

			var (
				wg      sync.WaitGroup
				command *exec.Cmd
				err     error
			)

			if !useGlobalDashboard {
				wg.Add(1)
				go func() {
					defer wg.Done()
					command, err = process.ExecuteAsync("kubectl", "port-forward",
						"--namespace", namespace,
						fmt.Sprintf("deployment/%s", DashboardName),
						fmt.Sprintf("%d:%d", DashboardLocalPort, DashboardPort))
				}()

				wg.Wait()
				ui.ExitOnError("port forwarding dashboard endpoint", err)
			}

			openCmd, err := getOpenCommand()
			if err == nil {
				_, err = process.Execute(openCmd, dashboardAddress)
				ui.PrintOnError("openning dashboard", err)
			}

			_, err = process.Execute("kubectl", "port-forward",
				"--namespace", namespace,
				fmt.Sprintf("deployment/%s", ApiServerName),
				fmt.Sprintf("%d:%d", ApiServerPort, ApiServerPort))
			ui.PrintOnError("port forwarding results endpoint", err)

			if command != nil {
				err = command.Process.Kill()
				ui.ExitOnError("process killing", err)
			}
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "s", "testkube", "namespace where the testkube is installed")
	cmd.Flags().BoolVar(&useGlobalDashboard, "use-global-dashboard", false, "use global dashboard for viewing testkube results")
	return cmd
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
		return "", fmt.Errorf("Unsupported OS")
	}
}
