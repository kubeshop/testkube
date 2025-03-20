package testworkflows

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows/renderer"
	tc "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTestWorkflowExecutionsCmd() *cobra.Command {
	var (
		limit                                  int
		selectors                              []string
		testWorkflowName, actorName, actorType string
		logsOnly                               bool
		tags                                   []string
	)

	cmd := &cobra.Command{
		Use:     "testworkflowexecution [executionID]",
		Aliases: []string{"testworkflowexecutions", "twe", "tw-execution", "twexecution"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Gets TestWorkflow execution details",
		Long:    `Gets TestWorkflow execution details by ID, or list if id is not passed`,

		Run: func(cmd *cobra.Command, args []string) {
			outputFlag := cmd.Flag("output")
			outputType := render.OutputPretty
			if outputFlag != nil {
				outputType = render.OutputType(outputFlag.Value.String())
			}

			outputPretty := outputType == render.OutputPretty

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			err = validateActorType(testkube.TestWorkflowRunningContextActorType(actorType))
			ui.ExitOnError("validatig actor type", err)

			if len(args) == 0 {
				client, _, err := common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				options := tc.FilterTestWorkflowExecutionOptions{
					Selector:    strings.Join(selectors, ","),
					TagSelector: strings.Join(tags, ","),
					ActorName:   actorName,
					ActorType:   testkube.TestWorkflowRunningContextActorType(actorType),
				}
				executions, err := client.ListTestWorkflowExecutions(testWorkflowName, limit, options)
				ui.ExitOnError("getting test workflow executions list", err)
				err = render.List(cmd, testkube.TestWorkflowExecutionSummaries(executions.Results), os.Stdout)
				ui.ExitOnError("rendering list", err)
				return
			}

			executionID := args[0]
			execution, err := client.GetTestWorkflowExecution(executionID)
			ui.ExitOnError("getting recent test workflow execution data id:"+execution.Id, err)
			if !logsOnly {
				err = render.Obj(cmd, execution, os.Stdout, renderer.TestWorkflowExecutionRenderer)
				ui.ExitOnError("rendering obj", err)
			}

			if outputPretty {
				ui.Info("Getting logs for test workflow execution", execution.Id)

				logs, err := client.GetTestWorkflowExecutionLogs(execution.Id)
				ui.ExitOnError("getting logs from test workflow", err)

				sigs := flattenSignatures(execution.Signature)

				printRawLogLines(logs, sigs, execution)
				if !logsOnly {
					render.PrintTestWorkflowExecutionURIs(&execution)
				}
			}
		},
	}

	cmd.Flags().StringVarP(&testWorkflowName, "testworkflow", "w", "", "test workflow name")
	cmd.Flags().IntVar(&limit, "limit", 1000, "max number of records to return")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&logsOnly, "logs-only", false, "show only execution logs")
	cmd.Flags().StringSliceVarP(&tags, "tag", "", nil, "tag key value pair: --tag key1=value1")
	cmd.Flags().StringVarP(&actorName, "actor-name", "", "", "test workflow running context actor name")
	cmd.Flags().StringVarP(&actorType, "actor-type", "", "", "test workflow running context actor type one of cron|testtrigger|user|testworkfow|testworkflowexecution|program")

	return cmd
}

func validateActorType(actorType testkube.TestWorkflowRunningContextActorType) error {
	if actorType == "" {
		return nil
	}

	actorTypes := map[testkube.TestWorkflowRunningContextActorType]struct{}{
		testkube.CRON_TestWorkflowRunningContextActorType:                  {},
		testkube.TESTTRIGGER_TestWorkflowRunningContextActorType:           {},
		testkube.USER_TestWorkflowRunningContextActorType:                  {},
		testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType:          {},
		testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType: {},
		testkube.PROGRAM_TestWorkflowRunningContextActorType:               {},
	}

	if _, ok := actorTypes[actorType]; !ok {
		return fmt.Errorf("please pass one of cron|testtrigger|user|testworkfow|testworkflowexecution|program for actor type")
	}

	return nil
}
