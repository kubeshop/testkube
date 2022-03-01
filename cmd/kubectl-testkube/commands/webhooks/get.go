package webhooks

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetWebhookCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "get <webhookName>",
		Short: "Gets webhookdetails",
		Long:  `Gets webhook, you can change output format`,
		Args:  validator.WebhookName,
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			client, _ := common.GetClient(cmd)
			webhook, err := client.GetWebhook(name)
			ui.ExitOnError("getting webhook: "+name, err)

			render := GetWebhookRenderer(cmd)
			err = render.Render(webhook, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as argument")

	return cmd
}
