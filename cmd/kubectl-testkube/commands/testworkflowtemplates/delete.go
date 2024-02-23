package testworkflowtemplates

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestWorkflowTemplateCmd() *cobra.Command {
	var deleteAll bool
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testworkflowtemplate [name]",
		Aliases: []string{"testworkflowtemplates", "twt"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Delete test workflow templates",

		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) == 0 {
				if len(selectors) > 0 {
					selector := strings.Join(selectors, ",")
					err = client.DeleteTestWorkflowTemplates(selector)
					ui.ExitOnError("deleting test workflow templates by labels: "+selector, err)
					ui.SuccessAndExit("Successfully deleted test workflow templates by labels", selector)
				} else if deleteAll {
					err = client.DeleteTestWorkflowTemplates("")
					ui.ExitOnError("delete all test workflow templates from namespace "+namespace, err)
					ui.SuccessAndExit("Successfully deleted all test workflow templates in namespace", namespace)
				} else {
					ui.Failf("Pass test workflow template name, --all flag to delete all or labels to delete by labels")
				}
				return
			}

			name := args[0]
			err = client.DeleteTestWorkflowTemplate(name)
			ui.ExitOnError("delete test workflow template "+name+" from namespace "+namespace, err)
			ui.SuccessAndExit("Successfully deleted test workflow template", name)
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all test workflow templates")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
