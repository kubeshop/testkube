package scripts

import (
	"github.com/kubeshop/kubtest/pkg/api/client"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func GetClient(cmd *cobra.Command) client.Client {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()

	client, err := client.GetClient(client.ClientType(clientType), namespace)
	ui.ExitOnError("setting up client type", err)
	return client
}
