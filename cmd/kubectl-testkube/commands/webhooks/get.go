package webhooks

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetWebhookCmd() *cobra.Command {
	var name, namespace string

	cmd := &cobra.Command{
		Use:     "webhooks <webhookName>",
		Aliases: []string{"webhook", "wh"},
		Short:   "Get webhook details",
		Long:    `Get webhook, you can change output format, to get single details pass name as first arg`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _ := common.GetClient(cmd)

			if len(args) > 0 {
				name := args[0]
				webhook, err := client.GetWebhook(namespace, name)
				ui.ExitOnError("getting webhook: "+name, err)
				err = render.Obj(cmd, webhook, os.Stdout)
				ui.ExitOnError("rendering obj", err)
			} else {
				webhooks, err := client.ListWebhooks(namespace)
				ui.ExitOnError("getting webhooks", err)
				err = render.List(cmd, webhooks, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as argument")
	cmd.Flags().StringVarP(&namespace, "namespace", "", "testkube", "Kubernetes namespace")

	return cmd
}
