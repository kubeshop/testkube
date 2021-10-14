package commands

import (
	"fmt"
	"runtime"

	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDashboardCmd() *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Open testkube dashboard",
		Long:  `Open testkube dashboard`,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true
			ui.Logo()

			dashboardAddress := fmt.Sprintf("%slocalhost:%d", DashboardURI, ApiServerPort)
			ui.Success("The dashboard is accessible here:", dashboardAddress)
			ui.Success("Port forwarding is started to make possible accessing the test result, in order to stop it hit Ctrl+C")
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
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "s", "testkube", "namespace where the testkube is installed")
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
