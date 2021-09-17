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

func PrintScriptExecutionDetails(scriptExecution kubtest.ScriptExecution) {
	ui.Warn("Type          :", scriptExecution.ScriptType)
	ui.Warn("Name          :", scriptExecution.ScriptName)
	ui.Warn("Execution ID  :", scriptExecution.Execution.Id)
	ui.Warn("Execution name:", scriptExecution.Name)
	ui.NL()
}
