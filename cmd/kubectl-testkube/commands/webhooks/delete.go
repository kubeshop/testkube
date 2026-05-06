package webhooks

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteWebhookCmd() *cobra.Command {
	var name string
	var selectors []string

	cmd := &cobra.Command{
		Use:     "webhook <webhookName>",
		Aliases: []string{"wh"},
		Short:   "Delete webhook",
		Long:    `Delete webhook, pass webhook name which should be deleted`,
		Run: func(cmd *cobra.Command, args []string) {
			ignoreNotFound, err := cmd.Flags().GetBool("ignore-not-found")
			ui.ExitOnError("reading flag ignore-not-found", err)

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name = args[0]
				err := client.DeleteWebhook(name)
				if ignoreNotFound && apiclient.IsNotFound(err) {
					ui.Info("Webhook '" + name + "' not found, but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting webhook: "+name, err)
				ui.SuccessAndExit("Succesfully deleted webhook", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteWebhooks(selector)
				if ignoreNotFound && apiclient.IsNotFound(err) {
					ui.Info("Webhook not found for matching selector '" + selector + "', but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting webhooks by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted webhooks by labels", selector)
			}

			ui.Failf("Pass Webhook name or labels to delete by labels")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as first argument")
	cmd.Flags().MarkShorthandDeprecated("name", "please use --name instead")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
