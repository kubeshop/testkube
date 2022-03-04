package testsuites

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewTestSuiteExecutionsCmd() *cobra.Command {
	var (
		limit     int
		selectors []string
	)

	cmd := &cobra.Command{
		Use:     "testsuiteexecutions [testSuiteName]",
		Aliases: []string{"tse", "TestSuiteExecutions"},
		Short:   "Gets test suites executions list",
		Long:    `Gets test suites executions list, can be filtered by test name`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			var testSuiteName string
			if len(args) > 0 {
				testSuiteName = args[0]
			}

			client, _ := common.GetClient(cmd)

			executions, err := client.ListTestSuiteExecutions(testSuiteName, limit, strings.Join(selectors, ","))
			ui.ExitOnError("getting test suites executions list", err)

			ui.Table(executions, os.Stdout)
			ui.NL()

		},
	}

	cmd.Flags().IntVar(&limit, "limit", 1000, "max number of records to return")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
