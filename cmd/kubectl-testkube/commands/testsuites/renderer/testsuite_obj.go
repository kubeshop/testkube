package renderer

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/renderer"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestSuiteRenderer(client client.Client, ui *ui.UI, obj interface{}) error {
	ts, ok := obj.(testkube.TestSuite)
	if !ok {
		return fmt.Errorf("can't use '%T' as testkube.TestSuite in RenderObj for test suite", obj)
	}

	ui.Warn("Name:     ", ts.Name)
	ui.Warn("Namespace:", ts.Namespace)
	if ts.Description != "" {
		ui.NL()
		ui.Warn("Description: ", ts.Description)
	}
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
			ui.Warn("  Name:                        ", ts.ExecutionRequest.Name)
		}

		if len(ts.ExecutionRequest.Variables) > 0 {
			renderer.RenderVariables(ts.ExecutionRequest.Variables)
		}

		if ts.ExecutionRequest.HttpProxy != "" {
			ui.Warn("  Http proxy:                  ", ts.ExecutionRequest.HttpProxy)
		}

		if ts.ExecutionRequest.HttpsProxy != "" {
			ui.Warn("  Https proxy:                 ", ts.ExecutionRequest.HttpsProxy)
		}

		if ts.ExecutionRequest.JobTemplate != "" {
			ui.Warn("  Job template:                ", "\n", ts.ExecutionRequest.JobTemplate)
		}

		if ts.ExecutionRequest.JobTemplateReference != "" {
			ui.Warn("  Job template reference:      ", ts.ExecutionRequest.JobTemplateReference)
		}

		if ts.ExecutionRequest.CronJobTemplate != "" {
			ui.Warn("  Cron job template:           ", "\n", ts.ExecutionRequest.CronJobTemplate)
		}

		if ts.ExecutionRequest.CronJobTemplateReference != "" {
			ui.Warn("  Cron job template reference: ", ts.ExecutionRequest.CronJobTemplateReference)
		}

		if ts.ExecutionRequest.ScraperTemplate != "" {
			ui.Warn("  Scraper template:            ", "\n", ts.ExecutionRequest.ScraperTemplate)
		}

		if ts.ExecutionRequest.ScraperTemplateReference != "" {
			ui.Warn("  Scraper template reference:  ", ts.ExecutionRequest.ScraperTemplateReference)
		}

		if ts.ExecutionRequest.PvcTemplate != "" {
			ui.Warn("  PVC template:                ", "\n", ts.ExecutionRequest.PvcTemplate)
		}

		if ts.ExecutionRequest.PvcTemplateReference != "" {
			ui.Warn("  PVC template reference:      ", ts.ExecutionRequest.PvcTemplateReference)
		}
	}

	batches := append(ts.Before, ts.Steps...)
	batches = append(batches, ts.After...)

	ui.NL()
	ui.Warn("Test batches:", fmt.Sprintf("%d", len(batches)))
	d := [][]string{{"Names", "Stop on failure", "Download artifacts"}}
	for _, batch := range batches {
		var names []string
		for _, step := range batch.Execute {
			names = append(names, step.FullName())
		}

		downloadArtifacts := ""
		if batch.DownloadArtifacts != nil {
			if batch.DownloadArtifacts.AllPreviousSteps {
				downloadArtifacts = "all previous steps"
			} else {
				if len(batch.DownloadArtifacts.PreviousStepNumbers) != 0 {
					downloadArtifacts = fmt.Sprintf("previous step numbers: %v", batch.DownloadArtifacts.PreviousStepNumbers)
				}

				if len(batch.DownloadArtifacts.PreviousTestNames) != 0 {
					downloadArtifacts = fmt.Sprintf("previous test names: %v", batch.DownloadArtifacts.PreviousTestNames)
				}
			}
		}

		d = append(d, []string{
			fmt.Sprintf("[%s]", strings.Join(names, ", ")),
			fmt.Sprintf("%v", batch.StopOnFailure),
			downloadArtifacts,
		})
	}

	ui.Table(ui.NewArrayTable(d), ui.Writer)
	ui.NL()

	return nil

}
