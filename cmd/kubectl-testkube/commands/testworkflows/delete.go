package testworkflows

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestWorkflowCmd() *cobra.Command {
	var deleteAll bool
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testworkflow [name]",
		Aliases: []string{"testworkflows", "tw"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Delete test workflows",

		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) == 0 {
				if len(selectors) > 0 {
					selector := strings.Join(selectors, ",")
					err = client.DeleteTestWorkflows(selector)
					ui.ExitOnError("deleting test workflows by labels: "+selector, err)
					ui.SuccessAndExit("Successfully deleted test workflows by labels", selector)
				} else if deleteAll {
					err = client.DeleteTestWorkflows("")
					ui.ExitOnError("delete all test workflows from namespace "+namespace, err)
					ui.SuccessAndExit("Successfully deleted all test workflows in namespace", namespace)
				} else {
					ui.Failf("Pass test workflow name, --all flag to delete all or labels to delete by labels")
				}
				return
			}

			name := args[0]
			err = client.DeleteTestWorkflow(name)
			ui.ExitOnError("delete test workflow "+name+" from namespace "+namespace, err)
			ui.SuccessAndExit("Successfully deleted test workflow", name)
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all test workflows")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
