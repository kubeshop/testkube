package testsuites

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteTestSuiteCmd() *cobra.Command {
	var deleteAll bool
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"ts"},
		Short:   "Delete test suite",
		Long:    `Delete test suite by name`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			namespace := cmd.Flag("namespace").Value.String()
			if deleteAll {
				err := client.DeleteTestSuites("")
				ui.ExitOnError("delete all tests from namespace "+namespace, err)
				ui.SuccessAndExit("Succesfully deleted all test suites in namespace", namespace)
			}

			if len(args) > 0 {
				name := args[0]
				err := client.DeleteTestSuite(name)
				ui.ExitOnError("delete test suite "+name+" from namespace "+namespace, err)
				ui.SuccessAndExit("Succesfully deleted test suite", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteTestSuites(selector)
				ui.ExitOnError("deleting test suites by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted test suites by labels", selector)
			}

			ui.Failf("Pass TestSuite name, --all flag to delete all or labels to delete by labels")
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all tests")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
