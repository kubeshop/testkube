package scripts

import (
	"fmt"
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/runner/output"
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

func printExecutionDetails(execution testkube.Execution) {
	ui.Warn("Type          :", execution.ScriptType)
	ui.Warn("Name          :", execution.ScriptName)
	ui.Warn("Execution ID  :", execution.Id)
	ui.Warn("Execution name:", execution.Name)
	ui.NL()
	ui.NL()
}

func downloadArtifacts(id, dir string, client client.Client) {
	artifacts, err := client.GetExecutionArtifacts(id)
	ui.ExitOnError("getting artifacts ", err)

	err = os.MkdirAll(dir, os.ModePerm)
	ui.ExitOnError("creating dir "+dir, err)

	if len(artifacts) > 0 {
		ui.Info("Getting artifacts", fmt.Sprintf("count = %d", len(artifacts)))
	}
	for _, artifact := range artifacts {
		f, err := client.DownloadFile(id, artifact.Name, dir)
		ui.ExitOnError("downloading file: "+f, err)
		ui.Info("- downloading file " + f)
	}

	ui.NL()
	ui.NL()
}

func watchLogs(id string, client client.Client) {
	ui.Info("Getting pod logs")

	logs, err := client.Logs(id)
	ui.ExitOnError("getting logs from executor", err)

	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.Warn(l.Content)
		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			ui.LogLine(l.String())
		}
	}

	ui.NL()

	// TODO watch for success | error status - in case of connection error on logs watch need fix in 0.8
	for range time.Tick(time.Second) {
		execution, err := client.GetExecution("-", id)
		ui.ExitOnError("get script execution details", err)

		fmt.Print(".")

		if execution.ExecutionResult.IsCompleted() {
			fmt.Println()

			uiShellCommandBlock(id)

			return
		}
	}

	uiShellCommandBlock(id)
}
