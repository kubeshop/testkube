package scheduling

import (
	"k8s.io/utils/ptr"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
)

func mapPgTestWorkflowExecutionPartial(exec sqlc.TestWorkflowExecution, execResult sqlc.TestWorkflowResult) testkube.TestWorkflowExecution {
	return testkube.TestWorkflowExecution{
		Id:                        exec.ID,
		GroupId:                   exec.GroupID.String,
		RunnerId:                  exec.RunnerID.String,
		Name:                      exec.Name,
		Namespace:                 exec.Namespace.String,
		Number:                    exec.Number.Int32,
		ScheduledAt:               exec.ScheduledAt.Time,
		AssignedAt:                exec.AssignedAt.Time,
		StatusAt:                  exec.StatusAt.Time,
		TestWorkflowExecutionName: exec.TestWorkflowExecutionName.String,
		RunnerTarget:              exec.RunnerTarget,
		RunnerOriginalTarget:      exec.RunnerOriginalTarget,
		DisableWebhooks:           exec.DisableWebhooks.Bool,
		Tags:                      exec.Tags,
		RunningContext:            exec.RunningContext,
		ConfigParams:              exec.ConfigParams,
		Runtime:                   exec.Runtime,

		// Fields stored in other tables
		Result: common.Ptr(mapPgTestWorkflowResult(execResult)),

		// Fields stored in other tables you will have to fetch separately
		Signature:            nil,
		Output:               nil,
		Reports:              nil,
		ResourceAggregations: nil,
		Workflow:             nil,
		ResolvedWorkflow:     nil,

		// Unimplemented!?
		SilentMode: nil,
	}
}

func mapPgTestWorkflowExecution(exec sqlc.TestWorkflowExecution, result sqlc.TestWorkflowResult, workflow sqlc.TestWorkflow, resolvedWorkflow sqlc.TestWorkflow, signatures []sqlc.TestWorkflowSignature, reports []sqlc.TestWorkflowReport, outputs []sqlc.TestWorkflowOutput, aggregation sqlc.TestWorkflowResourceAggregation) testkube.TestWorkflowExecution {
	return testkube.TestWorkflowExecution{
		Id:                        exec.ID,
		GroupId:                   exec.GroupID.String,
		RunnerId:                  exec.RunnerID.String,
		Name:                      exec.Name,
		Namespace:                 exec.Namespace.String,
		Number:                    exec.Number.Int32,
		ScheduledAt:               exec.ScheduledAt.Time,
		AssignedAt:                exec.AssignedAt.Time,
		StatusAt:                  exec.StatusAt.Time,
		TestWorkflowExecutionName: exec.TestWorkflowExecutionName.String,
		RunnerTarget:              exec.RunnerTarget,
		RunnerOriginalTarget:      exec.RunnerOriginalTarget,
		DisableWebhooks:           exec.DisableWebhooks.Bool,
		Tags:                      exec.Tags,
		RunningContext:            exec.RunningContext,
		ConfigParams:              exec.ConfigParams,
		Runtime:                   exec.Runtime,

		Result:               common.Ptr(mapPgTestWorkflowResult(result)),
		Signature:            mapPgTestWorkflowAllSignatures(signatures),
		Output:               mapPgTestWorkflowAllOutputs(outputs),
		Reports:              mapPgTestWorkflowAllReports(reports),
		ResourceAggregations: common.Ptr(mapPgTestWorkflowResourceResourceAggregation(aggregation)),
		Workflow:             common.Ptr(mapPgTestWorkflow(workflow)),
		ResolvedWorkflow:     common.Ptr(mapPgTestWorkflow(resolvedWorkflow)),

		SilentMode: nil, // TODO: Unimplemented!?
	}
}

func mapPgTestWorkflowResult(r sqlc.TestWorkflowResult) testkube.TestWorkflowResult {
	return testkube.TestWorkflowResult{
		Status:          ptr.To(testkube.TestWorkflowStatus(r.Status.String)),
		PredictedStatus: ptr.To(testkube.TestWorkflowStatus(r.PredictedStatus.String)),
		QueuedAt:        r.QueuedAt.Time,
		StartedAt:       r.StartedAt.Time,
		FinishedAt:      r.FinishedAt.Time,
		Duration:        r.Duration.String,
		TotalDuration:   r.TotalDuration.String,
		DurationMs:      r.DurationMs.Int32,
		PausedMs:        r.PausedMs.Int32,
		TotalDurationMs: r.TotalDurationMs.Int32,
		Pauses:          r.Pauses,
		Initialization:  r.Initialization,
		Steps:           r.Steps,
	}
}

func mapPgTestWorkflow(r sqlc.TestWorkflow) testkube.TestWorkflow {
	return testkube.TestWorkflow{
		Name:        r.Name.String,
		Created:     r.Created.Time,
		Updated:     r.Updated.Time,
		Namespace:   r.Namespace.String,
		Description: r.Description.String,
		ReadOnly:    r.ReadOnly.Bool,
		Labels:      r.Labels,
		Annotations: r.Annotations,
		Spec:        r.Spec,
		Status:      r.Status,
	}
}

// mapPgTestWorkflowAllSignatures transforms flat list of signatures with parent IDs into tree structure with children
func mapPgTestWorkflowAllSignatures(signatures []sqlc.TestWorkflowSignature) []testkube.TestWorkflowSignature {
	if len(signatures) == 0 {
		return []testkube.TestWorkflowSignature{}
	}

	// Create a map for fast lookup by ID
	signatureMap := make(map[string]*testkube.TestWorkflowSignature)

	// First pass: create all signature objects and populate the map
	for _, sig := range signatures {
		signature := &testkube.TestWorkflowSignature{
			Ref:      sig.Ref.String,
			Name:     sig.Name.String,
			Category: sig.Category.String,
			Optional: sig.Optional.Bool,
			Negative: sig.Negative.Bool,
			Children: []testkube.TestWorkflowSignature{},
		}
		signatureMap[sig.ID.String()] = signature
	}

	// Second pass: build the tree by adding children to their parents
	var roots []testkube.TestWorkflowSignature
	for _, sig := range signatures {
		signature := signatureMap[sig.ID.String()]

		// If this signature has a parent, add it as a child to the parent
		if sig.ParentID.Valid {
			parent, exists := signatureMap[sig.ParentID.String()]
			if exists {
				parent.Children = append(parent.Children, *signature)
			}
		} else {
			// No parent means this is a root node
			roots = append(roots, *signature)
		}
	}

	return roots
}

func mapPgTestWorkflowAllOutputs(outputs []sqlc.TestWorkflowOutput) []testkube.TestWorkflowOutput {
	var result []testkube.TestWorkflowOutput
	for _, output := range outputs {
		result = append(result, mapPgTestWorkflowOutput(output))
	}
	return result
}

func mapPgTestWorkflowOutput(output sqlc.TestWorkflowOutput) testkube.TestWorkflowOutput {
	return testkube.TestWorkflowOutput{
		Ref:   output.Ref.String,
		Name:  output.Name.String,
		Value: output.Value,
	}
}

func mapPgTestWorkflowAllReports(reports []sqlc.TestWorkflowReport) []testkube.TestWorkflowReport {
	var result []testkube.TestWorkflowReport
	for _, report := range reports {
		result = append(result, mapPgTestWorkflowReport(report))
	}
	return result
}

func mapPgTestWorkflowReport(report sqlc.TestWorkflowReport) testkube.TestWorkflowReport {
	return testkube.TestWorkflowReport{
		Ref:     report.Ref.String,
		Kind:    report.Kind.String,
		File:    report.File.String,
		Summary: report.Summary,
	}
}

func mapPgTestWorkflowResourceResourceAggregation(r sqlc.TestWorkflowResourceAggregation) testkube.TestWorkflowExecutionResourceAggregationsReport {
	return testkube.TestWorkflowExecutionResourceAggregationsReport{
		Global: r.Global,
		Step:   r.Step,
	}
}
