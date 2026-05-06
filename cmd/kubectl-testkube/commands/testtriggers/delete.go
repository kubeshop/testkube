package testtriggers

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestTriggerCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testtrigger [name]",
		Aliases: []string{"testtriggers", "tt"},
		Short:   "Delete a TestTrigger by name, or bulk-delete by selector",
		Run: func(cmd *cobra.Command, args []string) {
			ignoreNotFound, err := cmd.Flags().GetBool("ignore-not-found")
			ui.ExitOnError("reading flag ignore-not-found", err)

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name := args[0]
				err := client.DeleteTestTrigger(name)
				if ignoreNotFound && apiclient.IsNotFound(err) {
					ui.Info("TestTrigger '" + name + "' not found, but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting test trigger: "+name, err)
				ui.Success("deleted", name)
				return
			}

			selector := strings.Join(selectors, ",")
			if selector == "" {
				ui.Failf("either name argument or --label selector is required")
			}
			err = client.DeleteTestTriggers(selector)
			if ignoreNotFound && apiclient.IsNotFound(err) {
				ui.Info("TestTrigger not found for matching selector '" + selector + "', but ignoring since --ignore-not-found was passed")
				ui.SuccessAndExit("Operation completed")
			}
			ui.ExitOnError("deleting test triggers", err)
			ui.Success("deleted test triggers matching", selector)
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label selector for bulk delete, e.g. --label app=api")

	return cmd
}
