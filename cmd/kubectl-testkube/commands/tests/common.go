package tests

import (
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func GetClient(cmd *cobra.Command) (client.Client, string) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()

	client, err := client.GetClient(client.ClientType(clientType), namespace)
	ui.ExitOnError("setting up client type", err)

	return client, namespace
}

func printTestExecutionDetails(execution testkube.TestExecution) {
	ui.Warn("Name:", execution.Name)
	ui.Warn("Status:", string(*execution.Status)+"\n")
	ui.Table(execution, os.Stdout)

	ui.NL()
	ui.NL()
}

func uiPrintTestStatus(execution testkube.TestExecution) {
	switch execution.Status {
	case testkube.TestStatusQueued:
		ui.Warn("Test queued for execution")

	case testkube.TestStatusPending:
		ui.Warn("Test execution started")

	case testkube.TestStatusSuccess:
		duration := execution.EndTime.Sub(execution.StartTime)
		ui.Success("Test execution completed with sucess in " + duration.String())

	case testkube.TestStatusError:
		ui.Errf("Test execution failed")
		os.Exit(1)
	}

	ui.NL()
}

func uiShellTestGetCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube tests execution "+id,
	)

	ui.NL()
}

func uiShellTestWatchCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl testkube tests watch "+id,
	)

	ui.NL()
}
