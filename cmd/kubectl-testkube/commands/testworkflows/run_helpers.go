package testworkflows

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// RunWorkflowByName triggers an execution of the given TestWorkflow using the
// client bound to the command. It prints a short summary on success so users
// can follow up with `testkube get twe <id>` or `kubectl testkube watch twe <id>`.
//
// This is intentionally minimal (no --watch, artifacts, logs streaming) so
// shared callers like `testkube marketplace install` stay focused; the
// dedicated `testkube run testworkflow` command remains the tool of choice
// for richer execution flows.
func RunWorkflowByName(cmd *cobra.Command, name string) {
	client, _, err := common.GetClient(cmd)
	ui.ExitOnError("getting client", err)

	execution, err := client.ExecuteTestWorkflow(name, testkube.TestWorkflowExecutionRequest{})
	ui.ExitOnError(fmt.Sprintf("starting test workflow %q", name), err)

	ui.Success("Test workflow execution started", execution.Name)
	if execution.Id != "" {
		ui.Info("Execution ID:", execution.Id)
		ui.Info("Watch live:", fmt.Sprintf("kubectl testkube watch twe %s", execution.Id))
	}
}
