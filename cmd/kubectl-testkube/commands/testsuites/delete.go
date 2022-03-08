package testsuites

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteTestSuiteCmd() *cobra.Command {
	var deleteAll bool

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"ts"},
		Short:   "Delete test suite",
		Long:    `Delete test suite by name`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			client, _ := common.GetClient(cmd)
			namespace := cmd.Flag("namespace").Value.String()

			if deleteAll {
				err := client.DeleteTestSuites(namespace)
				ui.ExitOnError("delete all tests from namespace "+namespace, err)
				ui.Success("Succesfully deleted all test suites in namespace", namespace)
			} else if len(args) > 0 {
				name := args[0]
				err := client.DeleteTestSuite(name, namespace)
				ui.ExitOnError("delete test suite "+name+" from namespace "+namespace, err)
				ui.Success("Succesfully deleted", name)
			} else {
				ui.Failf("Pass TestSuite name or pass --all flag to delete all")
			}
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all tests")

	return cmd
}
