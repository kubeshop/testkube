package tests

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/app/api/apiutils"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestsCmd() *cobra.Command {
	var deleteAll bool
	var selectors []string

	cmd := &cobra.Command{
		Use:     "test [testName]",
		Aliases: []string{"t", "tests"},
		Short:   "Delete Test",
		Run: func(cmd *cobra.Command, args []string) {

			ignoreNotFound, err := cmd.Flags().GetBool("ignore-not-found")
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			namespace := cmd.Flag("namespace").Value.String()
			if deleteAll {
				err := client.DeleteTests("")
				ui.ExitOnError("delete all tests from namespace "+namespace, err)
				ui.SuccessAndExit("Succesfully deleted all tests in namespace", namespace)
			}

			if len(args) > 0 {
				name := args[0]
				err := client.DeleteTest(name)
				if ignoreNotFound && apiutils.IsNotFound(err) {
					ui.Info("Test '" + name + "' not found, but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("delete test "+name+" from namespace "+namespace, err)
				ui.SuccessAndExit("Succesfully deleted test", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteTests(selector)
				if ignoreNotFound && apiutils.IsNotFound(err) {
					ui.Info("Tests not found for matching selector '" + selector + "', but ignoring since --ignore-not-found was passed")
					ui.SuccessAndExit("Operation completed")
				}
				ui.ExitOnError("deleting tests by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted tests by labels", selector)
			}

			ui.Failf("Pass Test name, --all flag to delete all or labels to delete by labels")
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all tests")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
