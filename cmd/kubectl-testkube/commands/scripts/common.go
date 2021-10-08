package scripts

import (
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

func PrintExecutionDetails(execution testkube.Execution) {
	ui.Warn("Type          :", execution.ScriptType)
	ui.Warn("Name          :", execution.ScriptName)
	ui.Warn("Execution ID  :", execution.Id)
	ui.Warn("Execution name:", execution.Name)
	ui.NL()
}
