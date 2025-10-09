package templates

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTemplateCmd() *cobra.Command {
	var name string
	var selectors []string

	cmd := &cobra.Command{
		Use:     "template <templateName>",
		Aliases: []string{"tp"},
		Short:   "Delete a template.",
		Long:    `Delete a template and pass the template name to be deleted.`,
		Run: func(cmd *cobra.Command, args []string) {
			ignoreNotFound, err := cmd.Flags().GetBool("ignore-not-found")
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name = args[0]
				err := client.DeleteTemplate(name)
				if ignoreNotFound && apiutils.IsNotFound(err) {
					ui.Info("Template '" + name + "' not found, but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting template: "+name, err)
				ui.SuccessAndExit("Succesfully deleted template", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteTemplates(selector)
				if ignoreNotFound && apiutils.IsNotFound(err) {
					ui.Info("Template not found for matching selector '" + selector + "', but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting templates by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted templates by labels", selector)
			}

			ui.Failf("Pass Template name or labels to delete by labels")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique template name, you can also pass it as first argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
