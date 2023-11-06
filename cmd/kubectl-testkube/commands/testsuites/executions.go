package testsuites

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites/renderer"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewTestSuiteExecutionCmd() *cobra.Command {
	var (
		limit         int
		selectors     []string
		testSuiteName string
	)

	cmd := &cobra.Command{
		Use:     "testsuiteexecution [executionID]",
		Aliases: []string{"testsuiteexecutions", "tse", "ts-execution", "tsexecution"},
		Short:   "Gets TestSuite Execution details",
		Long:    `Gets TestSuite Execution details by ID, or list if id is not passed`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				executionID := args[0]
				execution, err := client.GetTestSuiteExecution(executionID)
				ui.ExitOnError("getting recent test suite execution data id:"+execution.Id, err)
				err = render.Obj(cmd, execution, os.Stdout, renderer.TestSuiteExecutionRenderer)
				ui.ExitOnError("rendering obj", err)
			} else {
				client, _, err := common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				executions, err := client.ListTestSuiteExecutions(testSuiteName, limit,
					strings.Join(selectors, ","))
				ui.ExitOnError("getting test suites executions list", err)
				err = render.List(cmd, executions, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}

		},
	}

	cmd.Flags().StringVar(&testSuiteName, "test-suite", "", "test suite name")
	cmd.Flags().IntVar(&limit, "limit", 1000, "max number of records to return")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
