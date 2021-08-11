package scripts

import (
	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

func GetClient(cmd *cobra.Command) client.Client {
	clientType, err := cmd.Flags().GetString("client")
	ui.ExitOnError("getting client type", err)
	client, err := client.GetClient(client.ClientType(clientType))
	ui.ExitOnError("setting up client type", err)
	return client
}
