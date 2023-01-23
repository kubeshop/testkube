package tests

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests/renderer"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetExecutionCmd() *cobra.Command {
	var (
		selectors []string
		testID    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:     "execution [executionID][executionName]",
		Aliases: []string{"executions", "e"},
		Short:   "Lists or gets test executions",
		Long:    `Getting list of execution for given test name or recent executions if there is no test name passed`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)

			if len(args) == 1 {
				executionID := args[0]
				execution, err := client.GetExecution(executionID)
				ui.ExitOnError("getting test execution: "+executionID, err)

				err = render.Obj(cmd, execution, os.Stdout, renderer.ExecutionRenderer)
				ui.ExitOnError("rendering execution", err)
			} else {
				executions, err := client.ListExecutions(testID, limit, strings.Join(selectors, ","))
				ui.ExitOnError("Getting executions for test: "+testID, err)
				err = render.List(cmd, executions, os.Stdout)
				ui.ExitOnError("rendering", err)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&testID, "test", "", "", "test id")
	cmd.Flags().IntVarP(&limit, "limit", "", 10, "records limit")

	return cmd
}
