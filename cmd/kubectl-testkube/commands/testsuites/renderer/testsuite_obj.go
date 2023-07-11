package renderer

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestSuiteRenderer(ui *ui.UI, obj interface{}) error {
	ts, ok := obj.(testkube.TestSuite)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestSuite in RenderObj for test suite", obj)
	}

	ui.Warn("Name:     ", ts.Name)
	ui.Warn("Namespace:", ts.Namespace)
	if len(ts.Labels) > 0 {
		ui.NL()
		ui.Warn("Labels:   ", testkube.MapToString(ts.Labels))
	}
	if ts.Schedule != "" {
		ui.NL()
		ui.Warn("Schedule: ", ts.Schedule)
	}

	if ts.ExecutionRequest != nil {
		ui.Warn("Execution request: ")
		if ts.ExecutionRequest.Name != "" {
			ui.Warn("  Name:              ", ts.ExecutionRequest.Name)
		}

		if len(ts.ExecutionRequest.Variables) > 0 {
			renderer.RenderVariables(ts.ExecutionRequest.Variables)
		}

		if ts.ExecutionRequest.HttpProxy != "" {
			ui.Warn("  Http proxy:        ", ts.ExecutionRequest.HttpProxy)
		}

		if ts.ExecutionRequest.HttpsProxy != "" {
			ui.Warn("  Https proxy:       ", ts.ExecutionRequest.HttpsProxy)
		}

		if ts.ExecutionRequest.CronJobTemplate != "" {
			ui.Warn("  Cron job template: ", ts.ExecutionRequest.CronJobTemplate)
		}
	}

	batches := append(ts.Before, ts.Steps...)
	batches = append(batches, ts.After...)

	ui.NL()
	ui.Warn("Test batches:", fmt.Sprintf("%d", len(batches)))
	d := [][]string{{"Names", "Stop on failure"}}
	for _, batch := range batches {
		var names []string
		for _, step := range batch.Execute {
			names = append(names, step.FullName())
		}

		d = append(d, []string{
			fmt.Sprintf("[%s]", strings.Join(names, ", ")),
			fmt.Sprintf("%v", batch.StopOnFailure),
		})
	}

	ui.Table(ui.NewArrayTable(d), ui.Writer)
	ui.NL()

	return nil

}
