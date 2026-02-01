package formatters

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// formattedWorkflow is a compact representation of a workflow for MCP responses.
type formattedWorkflow struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
	Updated     time.Time         `json:"updated,omitempty"`
	Health      *formattedHealth  `json:"health,omitempty"`
	Latest      *formattedLatest  `json:"latest,omitempty"`
}

// formattedHealth contains workflow health metrics.
type formattedHealth struct {
	PassRate      float64 `json:"passRate"`
	FlipRate      float64 `json:"flipRate"`
	OverallHealth float64 `json:"overallHealth"`
}

// formattedLatest contains the latest execution summary.
type formattedLatest struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Number      int32     `json:"number,omitempty"`
	Status      string    `json:"status,omitempty"`
	ScheduledAt time.Time `json:"scheduledAt,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// FormatListWorkflows parses a raw API response (JSON or YAML) containing
// []testkube.TestWorkflowWithExecutionSummary and returns a compact JSON
// representation with only essential fields.
func FormatListWorkflows(raw string) (string, error) {
	workflows, isEmpty, err := ParseJSON[[]testkube.TestWorkflowWithExecutionSummary](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "[]", nil
	}

	formatted := make([]formattedWorkflow, 0, len(workflows))
	for _, wf := range workflows {
		f := formattedWorkflow{}

		if wf.Workflow != nil {
			f.Name = wf.Workflow.Name
			f.Namespace = wf.Workflow.Namespace
			f.Description = wf.Workflow.Description
			f.Labels = wf.Workflow.Labels
			f.Created = wf.Workflow.Created
			f.Updated = wf.Workflow.Updated

			if wf.Workflow.Status != nil && wf.Workflow.Status.Health != nil {
				f.Health = &formattedHealth{
					PassRate:      wf.Workflow.Status.Health.PassRate,
					FlipRate:      wf.Workflow.Status.Health.FlipRate,
					OverallHealth: wf.Workflow.Status.Health.OverallHealth,
				}
			}
		}

		if wf.LatestExecution != nil {
			f.Latest = &formattedLatest{
				ID:          wf.LatestExecution.Id,
				Name:        wf.LatestExecution.Name,
				Number:      wf.LatestExecution.Number,
				ScheduledAt: wf.LatestExecution.ScheduledAt,
			}

			if wf.LatestExecution.Result != nil {
				if wf.LatestExecution.Result.Status != nil {
					f.Latest.Status = string(*wf.LatestExecution.Result.Status)
				}
				f.Latest.Duration = wf.LatestExecution.Result.Duration
			}
		}

		formatted = append(formatted, f)
	}

	return FormatJSON(formatted)
}
