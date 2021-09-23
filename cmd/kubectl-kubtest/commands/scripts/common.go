package scripts

import (
	"github.com/kubeshop/kubtest/pkg/api/client"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func GetClient(cmd *cobra.Command) (client.Client, string) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()

	client, err := client.GetClient(client.ClientType(clientType), namespace)
	ui.ExitOnError("setting up client type", err)
	return client, namespace
}

func PrintExecutionDetails(execution kubtest.Execution) {
	ui.Warn("Type          :", execution.ScriptType)
	ui.Warn("Name          :", execution.ScriptName)
	ui.Warn("Execution ID  :", execution.Id)
	ui.Warn("Execution name:", execution.Name)
	ui.NL()
}
