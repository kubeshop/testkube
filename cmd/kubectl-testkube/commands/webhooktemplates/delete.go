package webhooktemplates

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteWebhookTemplateCmd() *cobra.Command {
	var name string
	var selectors []string

	cmd := &cobra.Command{
		Use:     "webhooktemplate <webhookTemplateName>",
		Aliases: []string{"wht"},
		Short:   "Delete webhook template",
		Long:    `Delete webhook template, pass webhook template name which should be deleted`,
		Run: func(cmd *cobra.Command, args []string) {
			ignoreNotFound, _ := cmd.Flags().GetBool("ignore-not-found")
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name = args[0]
				err := client.DeleteWebhookTemplate(name)
				if ignoreNotFound && apiutils.IsNotFound(err) {
					ui.Info("Webhooktemplate '" + name + "' not found, but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting webhook template: "+name, err)
				ui.SuccessAndExit("Succesfully deleted webhook template", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteWebhookTemplates(selector)
				if ignoreNotFound && apiutils.IsNotFound(err) {
					ui.Info("Webhooktemplate not found for matching selector '" + selector + "', but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting webhook templates by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted webhook templates by labels", selector)
			}

			ui.Failf("Pass Webhook Template name or labels to delete by labels")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook template name, you can also pass it as first argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
