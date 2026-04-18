package workflowtriggers

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetWorkflowTriggerCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "workflowtrigger <name>",
		Aliases: []string{"workflowtriggers", "wt"},
		Short:   "Get WorkflowTrigger (v2) details",
		Long:    `Get a single WorkflowTrigger by name, or list all matching ones. Use --label to filter.`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name := args[0]
				trigger, err := client.GetWorkflowTrigger(name)
				ui.ExitOnError("getting workflow trigger: "+name, err)

				err = render.Obj(cmd, trigger, os.Stdout)
				ui.ExitOnError("rendering obj", err)
				return
			}

			triggers, err := client.ListWorkflowTriggers(strings.Join(selectors, ","))
			ui.ExitOnError("listing workflow triggers", err)

			err = render.List(cmd, triggers, os.Stdout)
			ui.ExitOnError("rendering list", err)
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label selector, e.g. --label app=api")

	return cmd
}
