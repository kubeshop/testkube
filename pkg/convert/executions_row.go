package convert

import (
	"io"
	"strconv"

	"github.com/google/uuid"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// The seven target tables, and the exact column order each serializer below
// writes. Each writeXxxRow function names the column of each value, and
// writeCopyRow refuses a row whose names differ from the list here.
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
		"lineage_base_id", "lineage_root_id", "lineage_attempt",
	}

	signatureColumns = []string{
		"execution_id", "ref", "step_id", "name", "category", "optional", "negative",
		"parent_id", "sig_order", "id",
	}

	resultColumns = []string{
		"execution_id", "status", "predicted_status", "queued_at", "started_at",
		"finished_at", "duration", "total_duration", "duration_ms", "paused_ms",
		"total_duration_ms", "pauses", "initialization", "steps", "status_details",
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

	// Lineage is stored as three scalars, projected the way the repository's
	// lineageBaseID/lineageRootID/lineageAttempt do: a document written before
	// lineage existed carries none and leaves all three NULL, which
	// EffectiveLineage() reads back as an original run. A document that has one
	// keeps an empty base or root as NULL but always records the attempt.
	lineageBaseID, lineageRootID, lineageAttempt := copyNull, copyNull, copyNull
	if exec.Lineage != nil {
		lineageBaseID = escapeCopyValue(exec.Lineage.BaseId)
		lineageRootID = escapeCopyValue(exec.Lineage.RootId)
		lineageAttempt = formatInt(exec.Lineage.Attempt)
	}

	return writeCopyRow(w, executionColumns, []copyField{
		{"id", escapeCopyValue(exec.Id)},
		{"group_id", escapeCopyValue(exec.GroupId)},
		{"runner_id", escapeCopyValue(exec.RunnerId)},
		{"runner_target", escapeJSONB(runnerTarget)},
		{"runner_original_target", escapeJSONB(runnerOriginalTarget)},
		{"name", escapeCopyValue(exec.Name)},
		{"namespace", escapeCopyValue(exec.Namespace)},
		{"number", formatInt(exec.Number)},
		{"scheduled_at", formatTimestamp(exec.ScheduledAt)},
		{"assigned_at", formatTimestamp(exec.AssignedAt)},
		{"status_at", formatTimestamp(exec.StatusAt)},
		{"test_workflow_execution_name", escapeCopyValue(exec.TestWorkflowExecutionName)},
		{"disable_webhooks", strconv.FormatBool(exec.DisableWebhooks)},
		{"tags", escapeJSONB(tags)},
		{"running_context", escapeJSONB(runningContext)},
		{"config_params", escapeJSONB(configParams)},
		{"runtime", escapeJSONB(runtime)},
		{"silent_mode", escapeJSONB(silentMode)},
		{"workflow_name", workflowName},
		{"status", status},
		{"lineage_base_id", lineageBaseID},
		{"lineage_root_id", lineageRootID},
		{"lineage_attempt", lineageAttempt},
	})
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
		(*order)++
		parent := copyNull
		if parentID != nil {
			parent = *parentID
		}

		id := uuid.NewString()

		err := writeCopyRow(w, signatureColumns, []copyField{
			{"execution_id", escapeCopyValue(executionID)},
			{"ref", escapeCopyValue(sig.Ref)},
			{"step_id", escapeCopyValue(sig.Id)},
			{"name", escapeCopyValue(sig.Name)},
			{"category", escapeCopyValue(sig.Category)},
			{"optional", strconv.FormatBool(sig.Optional)},
			{"negative", strconv.FormatBool(sig.Negative)},
			{"parent_id", parent},
			{"sig_order", formatInt(*order)},
			{"id", id},
		})
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
	// A nil *TestWorkflowStatusDetails marshals to the JSON literal null, which
	// escapeJSONB turns into SQL NULL - the column's meaning for a passed run.
	statusDetails, err := toJSONBytes(result.StatusDetails)
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

	return writeCopyRow(w, resultColumns, []copyField{
		{"execution_id", escapeCopyValue(executionID)},
		{"status", status},
		{"predicted_status", predictedStatus},
		{"queued_at", formatTimestamp(result.QueuedAt)},
		{"started_at", formatTimestamp(result.StartedAt)},
		{"finished_at", formatTimestamp(result.FinishedAt)},
		{"duration", escapeCopyValue(result.Duration)},
		{"total_duration", escapeCopyValue(result.TotalDuration)},
		{"duration_ms", formatInt(result.DurationMs)},
		{"paused_ms", formatInt(result.PausedMs)},
		{"total_duration_ms", formatInt(result.TotalDurationMs)},
		{"pauses", escapeJSONB(pauses)},
		{"initialization", escapeJSONB(initialization)},
		{"steps", escapeJSONB(steps)},
		{"status_details", escapeJSONB(statusDetails)},
	})
}

// writeOutputRow writes one output reference. order is 1-based to match how the
// repository layer assigns out_order on insert.
func writeOutputRow(w io.Writer, executionID string, output *testkube.TestWorkflowOutput, order int32) error {
	value, err := toJSONBytes(output.Value)
	if err != nil {
		return err
	}

	return writeCopyRow(w, outputColumns, []copyField{
		{"execution_id", escapeCopyValue(executionID)},
		{"ref", escapeCopyValue(output.Ref)},
		{"name", escapeCopyValue(output.Name)},
		{"value", escapeJSONB(value)},
		{"out_order", formatInt(order)},
	})
}

func writeReportRow(w io.Writer, executionID string, report *testkube.TestWorkflowReport, order int32) error {
	summary, err := toJSONBytes(report.Summary)
	if err != nil {
		return err
	}

	return writeCopyRow(w, reportColumns, []copyField{
		{"execution_id", escapeCopyValue(executionID)},
		{"ref", escapeCopyValue(report.Ref)},
		{"kind", escapeCopyValue(report.Kind)},
		{"file", escapeCopyValue(report.File)},
		{"summary", escapeJSONB(summary)},
		{"rep_order", formatInt(order)},
	})
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

	return writeCopyRow(w, resourceAggregationColumns, []copyField{
		{"execution_id", escapeCopyValue(executionID)},
		{"global", escapeJSONB(global)},
		{"step", escapeJSONB(step)},
	})
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

	return writeCopyRow(w, workflowColumns, []copyField{
		{"execution_id", escapeCopyValue(executionID)},
		{"workflow_type", workflowType},
		{"name", escapeCopyValue(workflow.Name)},
		{"namespace", escapeCopyValue(workflow.Namespace)},
		{"description", escapeCopyValue(workflow.Description)},
		{"labels", escapeJSONB(labels)},
		{"annotations", escapeJSONB(annotations)},
		{"spec", escapeJSONB(spec)},
		{"read_only", strconv.FormatBool(workflow.ReadOnly)},
		{"status", escapeJSONB(status)},
		{"created", formatTimestamp(workflow.Created)},
		{"updated", formatTimestamp(workflow.Updated)},
	})
}
