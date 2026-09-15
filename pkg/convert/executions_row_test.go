package convert

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// splitRow returns the COPY fields of a single serialized row, asserting it is
// newline-terminated.
func splitRow(t *testing.T, raw string) []string {
	t.Helper()
	require.True(t, strings.HasSuffix(raw, "\n"), "row must be newline-terminated")
	line := strings.TrimSuffix(raw, "\n")
	require.NotContains(t, line, "\n", "a single row must not contain an embedded newline")
	return strings.Split(line, "\t")
}

func statusPtr(s testkube.TestWorkflowStatus) *testkube.TestWorkflowStatus { return &s }

func TestWriteExecutionRowFieldCountMatchesColumns(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeExecutionRow(&buf, &testkube.TestWorkflowExecution{
		Id:   "exec-1",
		Name: "wf-1",
	}))

	fields := splitRow(t, buf.String())
	assert.Len(t, fields, len(executionColumns),
		"serialized field count must match the executionColumns list")
}

func TestWriteExecutionRowMinimal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeExecutionRow(&buf, &testkube.TestWorkflowExecution{
		Id:   "exec-1",
		Name: "wf-1",
	}))

	fields := splitRow(t, buf.String())
	byName := zip(t, executionColumns, fields)

	assert.Equal(t, "exec-1", byName["id"])
	assert.Equal(t, "wf-1", byName["name"])
	assert.Equal(t, "0", byName["number"])
	assert.Equal(t, "false", byName["disable_webhooks"])

	// Absent optional values must all become SQL NULL rather than empty strings
	// or zero timestamps.
	for _, col := range []string{
		"group_id", "runner_id", "runner_target", "runner_original_target", "namespace",
		"scheduled_at", "assigned_at", "status_at", "test_workflow_execution_name",
		"tags", "running_context", "config_params", "runtime", "silent_mode",
		"workflow_name", "status",
	} {
		assert.Equal(t, `\N`, byName[col], "column %s should be NULL when unset", col)
	}
}

func TestWriteExecutionRowFullyPopulated(t *testing.T) {
	t.Parallel()

	scheduled := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	exec := &testkube.TestWorkflowExecution{
		Id:                        "exec-42",
		Name:                      "wf-run-42",
		Namespace:                 "testkube",
		Number:                    42,
		GroupId:                   "group-1",
		RunnerId:                  "runner-1",
		TestWorkflowExecutionName: "scheduled-run",
		DisableWebhooks:           true,
		ScheduledAt:               scheduled,
		AssignedAt:                scheduled.Add(time.Second),
		StatusAt:                  scheduled.Add(2 * time.Second),
		Tags:                      map[string]string{"team": "platform"},
		ConfigParams:              map[string]testkube.TestWorkflowExecutionConfigValue{"k": {Value: "v"}},
		Workflow:                  &testkube.TestWorkflow{Name: "my-workflow"},
		Result:                    &testkube.TestWorkflowResult{Status: statusPtr(testkube.PASSED_TestWorkflowStatus)},
	}

	var buf bytes.Buffer
	require.NoError(t, writeExecutionRow(&buf, exec))

	byName := zip(t, executionColumns, splitRow(t, buf.String()))

	assert.Equal(t, "exec-42", byName["id"])
	assert.Equal(t, "wf-run-42", byName["name"])
	assert.Equal(t, "testkube", byName["namespace"])
	assert.Equal(t, "42", byName["number"])
	assert.Equal(t, "group-1", byName["group_id"])
	assert.Equal(t, "runner-1", byName["runner_id"])
	assert.Equal(t, "true", byName["disable_webhooks"])
	assert.Equal(t, "2026-03-01 10:00:00+00", byName["scheduled_at"])
	assert.Equal(t, `{"team":"platform"}`, byName["tags"])
	assert.Contains(t, byName["config_params"], `"k"`)

	// The denormalized columns are written here rather than left to the
	// sync triggers.
	assert.Equal(t, "my-workflow", byName["workflow_name"])
	assert.Equal(t, "passed", byName["status"])
}

// A tab or newline anywhere in the data would otherwise split or terminate the
// COPY row and shift every following column.
func TestWriteExecutionRowEscapesDelimiters(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeExecutionRow(&buf, &testkube.TestWorkflowExecution{
		Id:        "exec-1",
		Name:      "wf\twith\ttabs",
		Namespace: "ns\nwith\nnewlines",
		Tags:      map[string]string{"note": "a\tb\nc"},
	}))

	fields := splitRow(t, buf.String())
	require.Len(t, fields, len(executionColumns),
		"embedded delimiters must not add fields")

	byName := zip(t, executionColumns, fields)
	assert.Equal(t, `wf\twith\ttabs`, byName["name"])
	assert.Equal(t, `ns\nwith\nnewlines`, byName["namespace"])

	// JSONB values are escaped twice, and both layers are needed. encoding/json
	// turns the tab into the two characters backslash-t, then the COPY escaping
	// doubles that backslash so COPY does not read it as its own escape. On the
	// way in PostgreSQL undoes the outer layer, leaving the JSON escape intact
	// for the JSONB parser.
	assert.Contains(t, byName["tags"], `a\\tb\\nc`)
}

// Dot-escaped map keys are stored escaped in both backends, so the migrator must
// pass them through untouched.
func TestWriteExecutionRowPreservesEscapedDots(t *testing.T) {
	t.Parallel()

	const escapedKey = "app．kubernetes．io/name"

	var buf bytes.Buffer
	require.NoError(t, writeExecutionRow(&buf, &testkube.TestWorkflowExecution{
		Id:   "exec-1",
		Name: "wf-1",
		Tags: map[string]string{escapedKey: "testkube"},
	}))

	byName := zip(t, executionColumns, splitRow(t, buf.String()))
	assert.Contains(t, byName["tags"], escapedKey,
		"escaped dots must survive as stored, without unescaping")
}

func TestWriteSignatureRowsTree(t *testing.T) {
	t.Parallel()

	signatures := []testkube.TestWorkflowSignature{{
		Ref:  "root",
		Name: "Root",
		Children: []testkube.TestWorkflowSignature{
			{
				Ref:      "child-a",
				Name:     "Child A",
				Optional: true,
				Children: []testkube.TestWorkflowSignature{
					{Ref: "grandchild", Name: "Grandchild", Negative: true},
				},
			},
			{Ref: "child-b", Name: "Child B"},
		},
	}}

	var buf bytes.Buffer
	order := int32(0)
	require.NoError(t, writeSignatureRows(&buf, "exec-1", signatures, nil, &order))

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, 4, "every node in the tree must produce a row")
	assert.Equal(t, int32(4), order, "the shared counter must end at the node count")

	rows := make([]map[string]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		require.Len(t, fields, len(signatureColumns))
		rows = append(rows, zip(t, signatureColumns, fields))
	}

	// Pre-order traversal, with sig_order dense and 1-based.
	assert.Equal(t, []string{"root", "child-a", "grandchild", "child-b"},
		[]string{rows[0]["ref"], rows[1]["ref"], rows[2]["ref"], rows[3]["ref"]})
	for i, row := range rows {
		assert.Equal(t, strconv.Itoa(i+1), row["sig_order"])
		assert.Equal(t, "exec-1", row["execution_id"])
		_, err := uuid.Parse(row["id"])
		assert.NoError(t, err, "each signature id must be a UUID")
	}

	// parent_id must reference the id written for the actual parent, and a root
	// node must have none. COPY applies rows in file order, so a parent always
	// precedes its children and the self-referencing foreign key resolves.
	assert.Equal(t, `\N`, rows[0]["parent_id"], "root has no parent")
	assert.Equal(t, rows[0]["id"], rows[1]["parent_id"], "child-a hangs off root")
	assert.Equal(t, rows[1]["id"], rows[2]["parent_id"], "grandchild hangs off child-a")
	assert.Equal(t, rows[0]["id"], rows[3]["parent_id"], "child-b hangs off root")

	assert.Equal(t, "true", rows[1]["optional"])
	assert.Equal(t, "true", rows[2]["negative"])
}

func TestWriteSignatureRowsEmpty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	order := int32(0)
	require.NoError(t, writeSignatureRows(&buf, "exec-1", nil, nil, &order))
	assert.Zero(t, buf.Len(), "no signatures must produce no rows")
	assert.Equal(t, int32(0), order)
}

func TestWriteResultRow(t *testing.T) {
	t.Parallel()

	queued := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	result := &testkube.TestWorkflowResult{
		Status:          statusPtr(testkube.FAILED_TestWorkflowStatus),
		PredictedStatus: statusPtr(testkube.FAILED_TestWorkflowStatus),
		QueuedAt:        queued,
		StartedAt:       queued.Add(time.Second),
		FinishedAt:      queued.Add(11 * time.Second),
		Duration:        "10s",
		TotalDuration:   "11s",
		DurationMs:      10000,
		TotalDurationMs: 11000,
		PausedMs:        0,
	}

	var buf bytes.Buffer
	require.NoError(t, writeResultRow(&buf, "exec-1", result))

	fields := splitRow(t, buf.String())
	require.Len(t, fields, len(resultColumns))
	byName := zip(t, resultColumns, fields)

	assert.Equal(t, "exec-1", byName["execution_id"])
	assert.Equal(t, "failed", byName["status"])
	assert.Equal(t, "failed", byName["predicted_status"])
	assert.Equal(t, "10s", byName["duration"])
	assert.Equal(t, "10000", byName["duration_ms"])
	assert.Equal(t, "11000", byName["total_duration_ms"])
	assert.Equal(t, "0", byName["paused_ms"])
	assert.Equal(t, "2026-03-01 10:00:00+00", byName["queued_at"])
}

func TestWriteResultRowNilStatuses(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeResultRow(&buf, "exec-1", &testkube.TestWorkflowResult{}))

	byName := zip(t, resultColumns, splitRow(t, buf.String()))
	assert.Equal(t, `\N`, byName["status"])
	assert.Equal(t, `\N`, byName["predicted_status"])
	assert.Equal(t, `\N`, byName["queued_at"])
}

func TestWriteOutputAndReportRows(t *testing.T) {
	t.Parallel()

	t.Run("output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		require.NoError(t, writeOutputRow(&buf, "exec-1", &testkube.TestWorkflowOutput{
			Ref:   "step-1",
			Name:  "artifacts",
			Value: map[string]interface{}{"path": "/data/report.xml"},
		}, 1))

		byName := zip(t, outputColumns, splitRow(t, buf.String()))
		assert.Equal(t, "exec-1", byName["execution_id"])
		assert.Equal(t, "step-1", byName["ref"])
		assert.Equal(t, "artifacts", byName["name"])
		assert.Equal(t, "1", byName["out_order"], "out_order is 1-based")
		assert.Contains(t, byName["value"], "/data/report.xml")
	})

	t.Run("report", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		require.NoError(t, writeReportRow(&buf, "exec-1", &testkube.TestWorkflowReport{
			Ref:  "step-1",
			Kind: "junit",
			File: "report.xml",
		}, 3))

		byName := zip(t, reportColumns, splitRow(t, buf.String()))
		assert.Equal(t, "junit", byName["kind"])
		assert.Equal(t, "report.xml", byName["file"])
		assert.Equal(t, "3", byName["rep_order"])
		assert.Equal(t, `\N`, byName["summary"], "an absent summary must be NULL")
	})
}

func TestWriteWorkflowRow(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC)
	workflow := &testkube.TestWorkflow{
		Name:        "my-workflow",
		Namespace:   "testkube",
		Description: "does a thing",
		Labels:      map[string]string{"tier": "smoke"},
		ReadOnly:    true,
		Created:     created,
	}

	for _, workflowType := range []string{workflowTypeWorkflow, workflowTypeResolvedWorkflow} {
		workflowType := workflowType
		t.Run(workflowType, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			require.NoError(t, writeWorkflowRow(&buf, "exec-1", workflowType, workflow))

			fields := splitRow(t, buf.String())
			require.Len(t, fields, len(workflowColumns))
			byName := zip(t, workflowColumns, fields)

			assert.Equal(t, workflowType, byName["workflow_type"])
			assert.Equal(t, "my-workflow", byName["name"])
			assert.Equal(t, "does a thing", byName["description"])
			assert.Equal(t, "true", byName["read_only"])
			assert.Equal(t, `{"tier":"smoke"}`, byName["labels"])
			assert.Equal(t, "2026-01-15 08:30:00+00", byName["created"])
			assert.Equal(t, `\N`, byName["updated"])
		})
	}
}

func TestWriteResourceAggregationRow(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeResourceAggregationRow(&buf, "exec-1",
		&testkube.TestWorkflowExecutionResourceAggregationsReport{}))

	fields := splitRow(t, buf.String())
	require.Len(t, fields, len(resourceAggregationColumns))
	assert.Equal(t, "exec-1", fields[0])
}

// validateExecution guards the NOT NULL columns, because escapeCopyValue turns
// an empty string into a NULL sentinel that would abort the whole batch rather
// than one row.
func TestValidateExecution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		execution testkube.TestWorkflowExecution
		wantErr   string
	}{
		{
			name:      "valid",
			execution: testkube.TestWorkflowExecution{Id: "exec-1", Name: "wf-1"},
		},
		{
			name:      "missing id",
			execution: testkube.TestWorkflowExecution{Name: "wf-1"},
			wantErr:   "has no id",
		},
		{
			name:      "missing name",
			execution: testkube.TestWorkflowExecution{Id: "exec-1"},
			wantErr:   "has no name",
		},
		{
			name: "workflow snapshot without a name",
			execution: testkube.TestWorkflowExecution{
				Id: "exec-1", Name: "wf-1", Workflow: &testkube.TestWorkflow{},
			},
			wantErr: "workflow snapshot with no name",
		},
		{
			name: "resolved workflow snapshot without a name",
			execution: testkube.TestWorkflowExecution{
				Id: "exec-1", Name: "wf-1", ResolvedWorkflow: &testkube.TestWorkflow{},
			},
			wantErr: "resolved workflow snapshot with no name",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateExecution(&tt.execution)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// zip pairs a column list with the fields of a serialized row.
func zip(t *testing.T, columns, fields []string) map[string]string {
	t.Helper()
	require.Len(t, fields, len(columns), "field count must match column count")

	out := make(map[string]string, len(columns))
	for i, col := range columns {
		out[col] = fields[i]
	}
	return out
}
