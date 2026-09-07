package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	tclcmd "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cmd"
	"github.com/kubeshop/testkube/pkg/testworkflows"
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

	printTestCaseResults(ui, execution)

	if execution.Result != nil && execution.Result.Initialization != nil && execution.Result.Initialization.ErrorMessage != "" {
		ui.NL()
		ui.Err(errors.New(execution.Result.Initialization.ErrorMessage))
	}
}

// printTestCaseResults reports what each step's test report said.
//
// A step that passed because its failures were muted is indistinguishable from
// one that had nothing to fail unless the counts are shown, and the whole point
// of muting is that a failure is tolerated rather than hidden. Steps without a
// testCases policy contribute nothing, so the section is absent for the
// workflows that do not use the feature.
func printTestCaseResults(ui *ui.UI, execution testkube.TestWorkflowExecution) {
	if execution.Result == nil || len(execution.Result.Steps) == 0 {
		return
	}

	// Walking the signatures rather than the results map keeps the steps in the
	// order the workflow declares them, which the map does not preserve.
	printed := false
	for _, signature := range testworkflows.FlattenSignatures(execution.Signature) {
		step, ok := execution.Result.Steps[signature.Ref]
		if !ok || step.TestResults == nil || step.TestResults.Tests == 0 {
			continue
		}
		if !printed {
			ui.NL()
			ui.Info("Test cases:")
			printed = true
		}
		printStepTestResults(ui, signature.Label(), step.TestResults)
	}
}

func printStepTestResults(ui *ui.UI, name string, results *testkube.TestWorkflowStepTestResults) {
	summary := fmt.Sprintf("%d/%d passed", results.Passed, results.Tests)
	if results.Muted > 0 {
		summary += fmt.Sprintf(", %d muted", results.Muted)
	}
	if results.Unexpected > 0 {
		summary += fmt.Sprintf(", %d unexpected", results.Unexpected)
	}
	if results.Skipped > 0 {
		summary += fmt.Sprintf(", %d skipped", results.Skipped)
	}
	ui.Warn(fmt.Sprintf("  %s:", name), summary)

	switch {
	case results.IdentitiesIncomplete:
		// The reason mute did not apply, which otherwise looks like mute being
		// broken rather than the report being unusable.
		ui.Warn("    note:", fmt.Sprintf("the report describes %d test cases it does not name, so mute was not applied",
			results.Unrepresented))
	case results.RequirementApplied && results.Tolerated:
		ui.Warn("    note:", "within the pass requirement")
	case results.RequirementApplied:
		ui.Warn("    note:", "short of the pass requirement")
	}

	if len(results.UnusedMutePatterns) > 0 {
		// Dead quarantine config. Naming it is what stops mute lists outliving
		// the bugs they were written for.
		ui.Warn("    unused mute patterns:", strings.Join(results.UnusedMutePatterns, ", "))
	}
}
