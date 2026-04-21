package testtriggers

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestTriggerCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testtrigger [name]",
		Aliases: []string{"testtriggers", "tt"},
		Short:   "Delete a TestTrigger by name, or bulk-delete by selector",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name := args[0]
				err := client.DeleteTestTrigger(name)
				ui.ExitOnError("deleting test trigger: "+name, err)
				ui.Success("deleted", name)
				return
			}

			selector := strings.Join(selectors, ",")
			if selector == "" {
				ui.Failf("either name argument or --label selector is required")
			}
			err = client.DeleteTestTriggers(selector)
			ui.ExitOnError("deleting test triggers", err)
			ui.Success("deleted test triggers matching", selector)
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label selector for bulk delete, e.g. --label app=api")

	return cmd
}
