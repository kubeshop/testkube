package webhooks

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteWebhookCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "delete <webhookName>",
		Short: "Gets webhookdetails",
		Long:  `Gets webhook, you can change output format`,
		Args:  validator.WebhookName,
		Run: func(cmd *cobra.Command, args []string) {
			name = args[0]

			client, _ := common.GetClient(cmd)

			err := client.DeleteWebhook(name)
			ui.ExitOnError("deleting webhook: "+name, err)

			ui.Success("Webhook deleted")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as first argument")

	return cmd
}
