package testworkflows

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows/renderer"
	testkubecfg "github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	tclcmd "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cmd"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewReRunTestWorkflowExecutionCmd() *cobra.Command {
	var (
		watchEnabled             bool
		downloadArtifactsEnabled bool
		downloadDir              string
		format                   string
		masks                    []string
		serviceName              string
		parallelStepName         string
		serviceIndex             int
		parallelStepIndex        int
	)

	cmd := &cobra.Command{
		Use:     "testworkflowexecution [id]",
		Aliases: []string{"testworkflowexecutions", "twe"},
		Short:   "ReRun test workflow execution",
		Args:    validator.ExecutionName,

		Run: func(cmd *cobra.Command, args []string) {
			outputFlag := cmd.Flag("output")
			outputType := render.OutputPretty
			if outputFlag != nil {
				outputType = render.OutputType(outputFlag.Value.String())
			}

			outputPretty := outputType == render.OutputPretty
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			executionID := args[0]
			execution, err := client.GetTestWorkflowExecution(executionID)
			ui.ExitOnError("get test workflow execution failed", err)

			runContext := telemetry.GetCliRunContext()
			interfaceType := testkube.CICD_TestWorkflowRunningContextInterfaceType
			if runContext == "others|local" {
				runContext = ""
				interfaceType = testkube.CLI_TestWorkflowRunningContextInterfaceType
			}

			cfg, err := testkubecfg.Load()
			ui.ExitOnError("loading config file", err)
			ui.NL()

			var runningContext *testkube.TestWorkflowRunningContext
			// Pro edition only (tcl protected code)
			if cfg.ContextType == testkubecfg.ContextTypeCloud {
				runningContext = tclcmd.GetRunningContext(runContext, cfg.CloudContext.ApiKey, interfaceType)
			}

			name := ""
			if execution.Workflow != nil {
				name = execution.Workflow.Name
			}

			execution, err = client.ReRunTestWorkflowExecution(name, execution.Id, runningContext)
			if err != nil {
				// User friendly Open Source operation error
				errMessage := err.Error()
				if strings.Contains(errMessage, constants.OpenSourceOperationErrorMessage) {
					startp := strings.LastIndex(errMessage, apiErrorMessage)
					endp := strings.Index(errMessage, constants.OpenSourceOperationErrorMessage)
					if startp != -1 && endp != -1 {
						startp += len(apiErrorMessage)
						operation := ""
						if endp > startp {
							operation = strings.TrimSpace(errMessage[startp:endp])
						}

						err = errors.New(operation + " " + constants.OpenSourceOperationErrorMessage)
					}
				}
			}

			ui.ExitOnError("rerun test workflow execution "+executionID+" from namespace "+namespace, err)

			go func() {
				<-cmd.Context().Done()
				if errors.Is(cmd.Context().Err(), context.Canceled) {
					os.Exit(0)
				}
			}()

			err = renderer.PrintTestWorkflowExecution(cmd, os.Stdout, execution)
			ui.ExitOnError("render test workflow execution", err)

			var exitCode = 0
			if outputPretty {
				ui.NL()
				if !execution.FailedToInitialize() {
					if watchEnabled {
						var pServiceName, pParallelStepName *string
						if cmd.Flag("service-name").Changed || cmd.Flag("service-index").Changed {
							pServiceName = &serviceName
						}
						if cmd.Flag("parallel-step-name").Changed || cmd.Flag("parallel-step-index").Changed {
							pParallelStepName = &parallelStepName
						}

						exitCode = uiWatch(execution, pServiceName, serviceIndex, pParallelStepName, parallelStepIndex, client)
						ui.NL()
						if downloadArtifactsEnabled {
							tests.DownloadTestWorkflowArtifacts(execution.Id, downloadDir, format, masks, client, outputPretty)
						}
					} else {
						uiShellWatchExecution(execution.Id)
					}
				}

				execution, err = client.GetTestWorkflowExecution(execution.Id)
				ui.ExitOnError("get test workflow execution failed", err)

				render.PrintTestWorkflowExecutionURIs(&execution)
				uiShellGetExecution(execution.Id)
			}

			if exitCode != 0 {
				os.Exit(exitCode)
			}
		},
	}

	cmd.Flags().BoolVarP(&watchEnabled, "watch", "f", false, "watch for changes after start")
	cmd.Flags().StringVar(&downloadDir, "download-dir", "artifacts", "download dir")
	cmd.Flags().BoolVarP(&downloadArtifactsEnabled, "download-artifacts", "d", false, "download artifacts automatically")
	cmd.Flags().StringVar(&format, "format", "folder", "data format for storing files, one of folder|archive")
	cmd.Flags().StringArrayVarP(&masks, "mask", "", []string{}, "regexp to filter downloaded files, single or comma separated, like report/.* or .*\\.json,.*\\.js$")
	cmd.Flags().StringVar(&serviceName, "service-name", "", "test workflow service name")
	cmd.Flags().IntVar(&serviceIndex, "service-index", 0, "test workflow service index starting from 0")
	cmd.Flags().StringVar(&parallelStepName, "parallel-step-name", "", "test workflow parallel step name or reference")
	cmd.Flags().IntVar(&parallelStepIndex, "parallel-step-index", 0, "test workflow parallel step index starting from 0")

	return cmd
}
