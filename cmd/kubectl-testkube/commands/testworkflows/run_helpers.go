package testworkflows

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// RunWorkflowByName triggers an execution of the given TestWorkflow using the
// client bound to the command. It prints a short summary on success and, when
// follow is true, streams live logs until the execution completes (mirroring
// `testkube run testworkflow -f`). The process exits with the execution's
// exit code when following.
//
// This is intentionally minimal (no service/parallel-step filtering or
// artifact download) so shared callers like `testkube marketplace install`
// stay focused; the dedicated `testkube run testworkflow` command remains
// the tool of choice for richer execution flows.
func RunWorkflowByName(cmd *cobra.Command, name string, follow bool) {
	client, _, err := common.GetClient(cmd)
	ui.ExitOnError("getting client", err)

	execution, err := client.ExecuteTestWorkflow(name, testkube.TestWorkflowExecutionRequest{})
	ui.ExitOnError(fmt.Sprintf("starting test workflow %q", name), err)

	ui.Success("Test workflow execution started", execution.Name)
	if execution.Id == "" {
		return
	}
	ui.Info("Execution ID:", execution.Id)

	if !follow {
		ui.Info("Watch live:", fmt.Sprintf("kubectl testkube watch twe %s", execution.Id))
		return
	}

	ui.NL()
	exitCode := uiWatch(execution, nil, 0, nil, 0, client)
	ui.NL()

	if refreshed, err := client.GetTestWorkflowExecution(execution.Id); err == nil {
		render.PrintTestWorkflowExecutionURIs(&refreshed)
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
