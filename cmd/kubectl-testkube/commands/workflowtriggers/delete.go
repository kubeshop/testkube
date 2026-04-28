package workflowtriggers

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteWorkflowTriggerCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "workflowtrigger [name]",
		Aliases: []string{"workflowtriggers", "wt"},
		Short:   "Delete a WorkflowTrigger (v2) by name, or bulk-delete by selector",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name := args[0]
				err := client.DeleteWorkflowTrigger(name)
				ui.ExitOnError("deleting workflow trigger: "+name, err)
				ui.Success("deleted", name)
				return
			}

			selector := strings.Join(selectors, ",")
			if selector == "" {
				ui.Failf("either name argument or --label selector is required")
			}
			err = client.DeleteWorkflowTriggers(selector)
			ui.ExitOnError("deleting workflow triggers", err)
			ui.Success("deleted workflow triggers matching", selector)
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label selector for bulk delete, e.g. --label app=api")

	return cmd
}
