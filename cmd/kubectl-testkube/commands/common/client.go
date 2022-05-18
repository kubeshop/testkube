package common

import (
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func GetClient(cmd *cobra.Command) (client.Client, string) {
	clientType := cmd.Flag("client").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	apiURI := cmd.Flag("api-uri").Value.String()

	client, err := client.GetClient(client.ClientType(clientType), namespace, apiURI)
	ui.ExitOnError("setting up client type", err)

	return client, namespace
}
