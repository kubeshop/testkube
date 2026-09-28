package renderer

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	tclcmd "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cmd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func PrintTestWorkflowExecution(cmd *cobra.Command, w io.Writer, execution testkube.TestWorkflowExecution) error {
	outputFlag := cmd.Flag("output")
	outputType := render.OutputPretty
	if outputFlag != nil {
		outputType = render.OutputType(outputFlag.Value.String())
	}

	switch outputType {
	case render.OutputPretty:
		printPrettyOutput(ui.NewUI(ui.Verbose, w), execution)
	case render.OutputYAML:
		return render.RenderYaml(execution, w)
	case render.OutputJSON:
		return render.RenderJSON(execution, w)
	case render.OutputGoTemplate:
		tpl := cmd.Flag("go-template").Value.String()
		return render.RenderGoTemplate(execution, w, tpl)
	default:
		return render.RenderYaml(execution, w)
	}

	return nil
}

func TestWorkflowExecutionRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.TestWorkflowExecution)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestWorkflowExecution in RenderObj for test workflow execution", obj)
	}

	printPrettyOutput(ui, execution)
	return nil
}

func printPrettyOutput(ui *ui.UI, execution testkube.TestWorkflowExecution) {
	ui.Info("Test Workflow Execution:")

	if execution.Workflow != nil {
		ui.Warn("Name:                ", execution.Workflow.Name)
	} else {
		ui.NL()
		ui.Err(errors.New("incomplete execution data received from API: missing Workflow field"))
		ui.Hint("check your API endpoint configuration (HTTP vs HTTPS) and API server accessibility")
	}

	if execution.Id != "" {
		ui.Warn("Execution ID:        ", execution.Id)
		ui.Warn("Execution name:      ", execution.Name)
		ui.Warn("Execution namespace: ", execution.Namespace)
		if execution.Number != 0 {
			ui.Warn("Execution number:    ", fmt.Sprintf("%d", execution.Number))
		}
		ui.Warn("Requested at:        ", execution.ScheduledAt.String())
		ui.Warn("Disabled webhooks:   ", fmt.Sprint(execution.DisableWebhooks))
		if len(execution.Tags) > 0 {
			ui.NL()
			ui.Warn("Tags:                ", testkube.MapToString(execution.Tags))
		}
		// Pro edition only (tcl protected code)
		tclcmd.PrintRunningContext(ui, execution)
		if execution.Result != nil && execution.Result.Status != nil {
			ui.Warn("Status:              ", string(*execution.Result.Status))
			// The object says which layer ended the execution, so the reader learns the kind of
			// outcome before the words of the message. The words stay neutral, because a cancel by
			// a person is an outcome and not a failure.
			if details := execution.Result.StatusDetails; details != nil {
				ui.Warn("Status type:         ", details.Type_)
				ui.Warn("Reason:              ", details.Reason)
				if details.Step != "" {
					ui.Warn("Step:                ", details.Step)
				}
				if details.Message != "" {
					ui.Warn("Message:             ", details.Message)
				}
				if details.Actor != "" {
					ui.Warn("Stopped by:          ", details.Actor)
				}
				if details.User != nil && details.User.Name != "" {
					ui.Warn("Canceled by:         ", details.User.Name)
				}
			}
			if !execution.Result.QueuedAt.IsZero() {
				ui.Warn("Queued at:           ", execution.Result.QueuedAt.String())
			}
			if !execution.Result.StartedAt.IsZero() {
				ui.Warn("Started at:          ", execution.Result.StartedAt.String())
			}
			if !execution.Result.FinishedAt.IsZero() {
				ui.Warn("Finished at:         ", execution.Result.FinishedAt.String())
				ui.Warn("Duration:            ", execution.Result.Duration)
			}
		}
	}

	if execution.Result != nil && execution.Result.Initialization != nil && execution.Result.Initialization.ErrorMessage != "" {
		ui.NL()
		ui.Err(errors.New(execution.Result.Initialization.ErrorMessage))
	}
}
