package tests

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteTestsCmd() *cobra.Command {
	var deleteAll bool

	cmd := &cobra.Command{
		Use:     "test [testName]",
		Aliases: []string{"t", "tests"},
		Short:   "Delete Test",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, _ := common.GetClient(cmd)
			namespace := cmd.Flag("namespace").Value.String()

			if deleteAll {
				err := client.DeleteTests()
				ui.ExitOnError("delete all tests from namespace "+namespace, err)
				ui.Success("Succesfully deleted all tests in namespace", namespace)
			} else if len(args) > 0 {
				name := args[0]
				err := client.DeleteTest(name)
				ui.ExitOnError("delete test "+name+" from namespace "+namespace, err)
				ui.Success("Succesfully deleted", name)
			} else {
				ui.Failf("Pass Test name or pass --all flag to delete all")
			}

		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all tests")

	return cmd
}
