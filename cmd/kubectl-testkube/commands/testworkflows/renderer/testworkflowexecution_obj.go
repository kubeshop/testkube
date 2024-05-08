package renderer

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestWorkflowExecutionRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	execution, ok := obj.(testkube.TestWorkflowExecution)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestWorkflowExecution in RenderObj for test workflow execution", obj)
	}

	ui.Info("Test Workflow Execution:")
	ui.Warn("Name:                ", execution.Workflow.Name)
	if execution.Id != "" {
		ui.Warn("Execution ID:        ", execution.Id)
		ui.Warn("Execution name:      ", execution.Name)
		ui.Warn("Execution namespace: ", execution.Namespace)
		if execution.Number != 0 {
			ui.Warn("Execution number:    ", fmt.Sprintf("%d", execution.Number))
		}
		ui.Warn("Requested at:        ", execution.ScheduledAt.String())
		if execution.Result != nil && execution.Result.Status != nil {
			ui.Warn("Status:              ", string(*execution.Result.Status))
			if !execution.Result.QueuedAt.IsZero() {
				ui.Warn("Queued at:          ", execution.Result.QueuedAt.String())
			}
			if !execution.Result.StartedAt.IsZero() {
				ui.Warn("Started at:          ", execution.Result.StartedAt.String())
			}
			if !execution.Result.FinishedAt.IsZero() {
				ui.Warn("Finished at:         ", execution.Result.FinishedAt.String())
				ui.Warn("Duration:            ", execution.Result.FinishedAt.Sub(execution.Result.QueuedAt).String())
			}
		}
	}

	if execution.Result != nil && execution.Result.Initialization != nil && execution.Result.Initialization.ErrorMessage != "" {
		ui.NL()
		ui.Err(errors.New(execution.Result.Initialization.ErrorMessage))
	}

	return nil

}
