package testsources

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestSourceCmd() *cobra.Command {
	var name string
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testsource <testSourceName>",
		Aliases: []string{"testsources", "tsc"},
		Short:   "Delete test source",
		Long:    `Delete test source, pass test source name which should be deleted`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name = args[0]
				err := client.DeleteTestSource(name)
				ui.ExitOnError("deleting test source: "+name, err)
				ui.SuccessAndExit("Succesfully deleted test source", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteTestSources(selector)
				ui.ExitOnError("deleting test sources by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted test sources by labels", selector)
			}

			ui.Failf("Pass TestSource name or labels to delete by labels")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test source name, you can also pass it as first argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
