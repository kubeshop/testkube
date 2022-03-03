package webhooks

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteWebhookCmd() *cobra.Command {
	var name, namespace string

	cmd := &cobra.Command{

		Use:   "delete <webhookName>",
		Short: "Delete webhook",
		Long:  `Delete webhook, pass webhook name which should be deleted`,
		Args:  validator.DNS1123Subdomain,
		Run: func(cmd *cobra.Command, args []string) {
			name = args[0]

			client, _ := common.GetClient(cmd)

			err := client.DeleteWebhook(namespace, name)
			ui.ExitOnError("deleting webhook: "+name, err)

			ui.Success("Webhook deleted")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as first argument")
	cmd.Flags().StringVarP(&namespace, "namespace", "", "", "Kubernetes namespace")

	return cmd
}
