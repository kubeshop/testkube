package convert

import (
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// The seven target tables, and the exact column order each serializer below
// writes. These lists and the corresponding writeXxxRow functions must be kept
// in step; the integration test asserts the result round-trips through the
// repository layer, which is what catches a mismatch.
//
// organization_id and environment_id are deliberately absent: OSS never sets
// them (pkg/repository/postgres_factory.go builds the repository without
// WithOrganizationID/WithEnvironmentID), and both columns are NOT NULL with a
// default of the empty string, so omitting them lets the default apply.
// Emitting them through escapeCopyValue would write a NULL sentinel for the
// empty string and violate the NOT NULL constraint.
//
// workflow_name and status are written directly rather than left to
// trg_sync_execution_workflow_name / trg_sync_execution_status. Those triggers
// still fire per row during COPY, but their IS DISTINCT FROM guards then match
// nothing, so each costs an index probe instead of a heap update plus index
// churn.
var (
	executionColumns = []string{
		"id", "group_id", "runner_id", "runner_target", "runner_original_target",
		"name", "namespace", "number", "scheduled_at", "assigned_at", "status_at",
		"test_workflow_execution_name", "disable_webhooks", "tags", "running_context",
		"config_params", "runtime", "silent_mode", "workflow_name", "status",
	}

	signatureColumns = []string{
		"execution_id", "ref", "name", "category", "optional", "negative",
		"parent_id", "sig_order", "id",
	}

	resultColumns = []string{
		"execution_id", "status", "predicted_status", "queued_at", "started_at",
		"finished_at", "duration", "total_duration", "duration_ms", "paused_ms",
		"total_duration_ms", "pauses", "initialization", "steps",
	}

	outputColumns = []string{"execution_id", "ref", "name", "value", "out_order"}

	reportColumns = []string{"execution_id", "ref", "kind", "file", "summary", "rep_order"}

	resourceAggregationColumns = []string{"execution_id", "global", "step"}

	workflowColumns = []string{
		"execution_id", "workflow_type", "name", "namespace", "description",
		"labels", "annotations", "spec", "read_only", "status", "created", "updated",
	}
)

// Workflow snapshot discriminators, matching the UNIQUE contract on
// test_workflows(execution_id, workflow_type).
const (
	workflowTypeWorkflow         = "workflow"
	workflowTypeResolvedWorkflow = "resolved_workflow"
)

// writeExecutionRow serializes the parent row. Callers must have run
// validateExecution first: Name is NOT NULL and escapeCopyValue maps the empty
// string to a NULL sentinel.
func writeExecutionRow(w io.Writer, exec *testkube.TestWorkflowExecution) error {
	runnerTarget, err := toJSONBytes(exec.RunnerTarget)
	if err != nil {
		return err
	}
	runnerOriginalTarget, err := toJSONBytes(exec.RunnerOriginalTarget)
	if err != nil {
		return err
	}
	tags, err := toJSONBytes(exec.Tags)
	if err != nil {
		return err
	}
	runningContext, err := toJSONBytes(exec.RunningContext)
	if err != nil {
		return err
	}
	configParams, err := toJSONBytes(exec.ConfigParams)
	if err != nil {
		return err
	}
	runtime, err := toJSONBytes(exec.Runtime)
	if err != nil {
		return err
	}
	silentMode, err := toJSONBytes(exec.SilentMode)
	if err != nil {
		return err
	}

	// Denormalized columns, sourced from the children they mirror.
	workflowName := copyNull
	if exec.Workflow != nil {
		workflowName = escapeCopyValue(exec.Workflow.Name)
	}
	status := copyNull
	if exec.Result != nil && exec.Result.Status != nil {
		status = escapeCopyValue(string(*exec.Result.Status))
	}

	_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		escapeCopyValue(exec.Id),
		escapeCopyValue(exec.GroupId),
		escapeCopyValue(exec.RunnerId),
		escapeJSONB(runnerTarget),
		escapeJSONB(runnerOriginalTarget),
		escapeCopyValue(exec.Name),
		escapeCopyValue(exec.Namespace),
		exec.Number,
		formatTimestamp(exec.ScheduledAt),
		formatTimestamp(exec.AssignedAt),
		formatTimestamp(exec.StatusAt),
		escapeCopyValue(exec.TestWorkflowExecutionName),
		exec.DisableWebhooks,
		escapeJSONB(tags),
		escapeJSONB(runningContext),
		escapeJSONB(configParams),
		escapeJSONB(runtime),
		escapeJSONB(silentMode),
		workflowName,
		status,
	)
	return err
}

// writeSignatureRows flattens the signature tree in pre-order.
//
// test_workflow_signatures.id is a UUID with a self-referencing parent_id, and
// COPY cannot read generated keys back, so each id is minted here and handed to
// the recursive call as the children's parent. Writing parents before their
// children in the same file is what satisfies the self-FK, since COPY applies
// rows in file order within the batch transaction.
//
// order is a single pre-order counter shared across the whole tree and advanced
// through the pointer, so it stays unique per execution across recursion.
func writeSignatureRows(w io.Writer, executionID string, signatures []testkube.TestWorkflowSignature,
	parentID *string, order *int32) error {
	for i := range signatures {
		sig := &signatures[i]
		*order++

		parent := copyNull
		if parentID != nil {
			parent = *parentID
		}

		id := uuid.NewString()

		_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%t\t%s\t%d\t%s\n",
			escapeCopyValue(executionID),
			escapeCopyValue(sig.Ref),
			escapeCopyValue(sig.Name),
			escapeCopyValue(sig.Category),
			sig.Optional,
			sig.Negative,
			parent,
			*order,
			id,
		)
		if err != nil {
			return err
		}

		if len(sig.Children) > 0 {
			if err := writeSignatureRows(w, executionID, sig.Children, &id, order); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeResultRow(w io.Writer, executionID string, result *testkube.TestWorkflowResult) error {
	pauses, err := toJSONBytes(result.Pauses)
	if err != nil {
		return err
	}
	initialization, err := toJSONBytes(result.Initialization)
	if err != nil {
		return err
	}
	steps, err := toJSONBytes(result.Steps)
	if err != nil {
		return err
	}

	status := copyNull
	if result.Status != nil {
		status = escapeCopyValue(string(*result.Status))
	}
	predictedStatus := copyNull
	if result.PredictedStatus != nil {
		predictedStatus = escapeCopyValue(string(*result.PredictedStatus))
	}

	_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\t%s\n",
		escapeCopyValue(executionID),
		status,
		predictedStatus,
		formatTimestamp(result.QueuedAt),
		formatTimestamp(result.StartedAt),
		formatTimestamp(result.FinishedAt),
		escapeCopyValue(result.Duration),
		escapeCopyValue(result.TotalDuration),
		result.DurationMs,
		result.PausedMs,
		result.TotalDurationMs,
		escapeJSONB(pauses),
		escapeJSONB(initialization),
		escapeJSONB(steps),
	)
	return err
}

// writeOutputRow writes one output reference. order is 1-based to match how the
// repository layer assigns out_order on insert.
func writeOutputRow(w io.Writer, executionID string, output *testkube.TestWorkflowOutput, order int32) error {
	value, err := toJSONBytes(output.Value)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
		escapeCopyValue(executionID),
		escapeCopyValue(output.Ref),
		escapeCopyValue(output.Name),
		escapeJSONB(value),
		order,
	)
	return err
}

func writeReportRow(w io.Writer, executionID string, report *testkube.TestWorkflowReport, order int32) error {
	summary, err := toJSONBytes(report.Summary)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
		escapeCopyValue(executionID),
		escapeCopyValue(report.Ref),
		escapeCopyValue(report.Kind),
		escapeCopyValue(report.File),
		escapeJSONB(summary),
		order,
	)
	return err
}

func writeResourceAggregationRow(w io.Writer, executionID string,
	agg *testkube.TestWorkflowExecutionResourceAggregationsReport) error {
	global, err := toJSONBytes(agg.Global)
	if err != nil {
		return err
	}
	step, err := toJSONBytes(agg.Step)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\t%s\t%s\n",
		escapeCopyValue(executionID),
		escapeJSONB(global),
		escapeJSONB(step),
	)
	return err
}

func writeWorkflowRow(w io.Writer, executionID, workflowType string, workflow *testkube.TestWorkflow) error {
	labels, err := toJSONBytes(workflow.Labels)
	if err != nil {
		return err
	}
	annotations, err := toJSONBytes(workflow.Annotations)
	if err != nil {
		return err
	}
	spec, err := toJSONBytes(workflow.Spec)
	if err != nil {
		return err
	}
	status, err := toJSONBytes(workflow.Status)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
		escapeCopyValue(executionID),
		workflowType,
		escapeCopyValue(workflow.Name),
		escapeCopyValue(workflow.Namespace),
		escapeCopyValue(workflow.Description),
		escapeJSONB(labels),
		escapeJSONB(annotations),
		escapeJSONB(spec),
		workflow.ReadOnly,
		escapeJSONB(status),
		formatTimestamp(workflow.Created),
		formatTimestamp(workflow.Updated),
	)
	return err
}
