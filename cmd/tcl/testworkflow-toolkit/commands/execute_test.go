// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
)

// The spec of an entry stays unresolved until its operation starts, so that its
// configuration can read an execution scheduled before it. This checks that the
// deferred resolution actually reaches the nested config values.
func TestDeferredSpecFinalization(t *testing.T) {
	spec := &testworkflowsv1.StepExecuteWorkflow{
		Name: "consumer",
		Config: map[string]testworkflowsv1.ConfigValue{
			"token": `{{ execution("p").outputs.token }}`,
		},
	}

	registry := executiondata.NewRegistry()
	machine := executiondata.NewMachine(executiondata.MachineOptions{Registry: registry})

	t.Run("fails while the referenced execution has not run", func(t *testing.T) {
		workflow := *spec.DeepCopy()
		err := expressions.Finalize(&workflow, machine)
		assert.ErrorContains(t, err, `unknown execution "p"`)
	})

	t.Run("resolves once the execution is registered", func(t *testing.T) {
		registry.Add(executiondata.Execution{
			Id:       "exec-1",
			Workflow: "producer",
			Alias:    "p",
			Outputs:  map[string]string{"token": "abc123"},
		})

		workflow := *spec.DeepCopy()
		require.NoError(t, expressions.Finalize(&workflow, machine))
		assert.Equal(t, testworkflowsv1.ConfigValue("abc123"), workflow.Config["token"])
	})
}

// An output the producer withheld resolves to a marker instead of the value it was
// meant to carry. Handing that to another workflow has to fail, rather than configure
// it with the marker - or, worse, with nothing at all.
//
// This pins the contract: a sensitive output is not exchanged between executions, and
// asking for one is an error naming the channels that can carry it. Turning this back
// into an empty value would be a silent regression, which is what the marker exists to
// prevent.
func TestWithheldOutputStopsTheExecution(t *testing.T) {
	spec := &testworkflowsv1.StepExecuteWorkflow{
		Name: "consumer",
		Config: map[string]testworkflowsv1.ConfigValue{
			"token": `{{ execution("p").outputs.token }}`,
		},
	}

	registry := executiondata.NewRegistry()
	registry.Add(executiondata.Execution{
		Id:       "exec-1",
		Workflow: "producer",
		Alias:    "p",
		Outputs:  map[string]string{"token": executiondata.WithheldMarker("producer", "token")},
	})
	machine := executiondata.NewMachine(executiondata.MachineOptions{Registry: registry})

	workflow := *spec.DeepCopy()
	require.NoError(t, expressions.Finalize(&workflow, machine),
		"the marker is an ordinary value, so resolution itself succeeds")

	markers := executiondata.WithheldMarkersIn(&workflow)
	require.Len(t, markers, 1, "the marker reaches the configuration of the scheduled execution")
	assert.NotEmpty(t, workflow.Config["token"].String(),
		"the consumer must not resolve a withheld output to an empty value")

	err := executiondata.WithheldError("this execution", markers)
	assert.ErrorContains(t, err, "was not published outside the workflow that produced it")
	assert.ErrorContains(t, err, "output token of workflow producer",
		"the error names which output of which workflow to stop relying on")
	assert.ErrorContains(t, err, "read_artifact()",
		"the error names a channel that can carry the value instead")
}

func TestClaimExecutionRefs(t *testing.T) {
	t.Run("an entry is addressed by its workflow name", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "", []string{"producer"}))
		assert.Equal(t, map[string]string{"producer": "producer"}, claimed)
	})

	t.Run("an alias takes precedence over the workflow name", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "p", []string{"producer"}))
		assert.Equal(t, map[string]string{"p": "producer"}, claimed)

		require.NoError(t, claimExecutionRefs(claimed, "", []string{"producer"}),
			"the unaliased entry still claims the workflow name, so both stay addressable")
	})

	t.Run("a selector claims every workflow it matched", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "", []string{"a", "b", "c"}))
		assert.Equal(t, map[string]string{"a": "a", "b": "a", "c": "a"}, claimed)
	})

	t.Run("an aliased selector claims one reference for the whole group", func(t *testing.T) {
		// The matched workflows form a single group addressed by execution("all", i),
		// so the alias must not be claimed once per match.
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "all", []string{"a", "b", "c"}))
		assert.Equal(t, map[string]string{"all": "a"}, claimed)
	})

	t.Run("rejects the same workflow listed twice without an alias", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "", []string{"producer"}))

		err := claimExecutionRefs(claimed, "", []string{"producer"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `duplicated execution reference "producer"`)
		assert.Contains(t, err.Error(), "set a unique 'as'")
	})

	t.Run("rejects two entries sharing an alias", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "shared", []string{"producer"}))

		err := claimExecutionRefs(claimed, "shared", []string{"consumer"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `already used by "producer"`)
	})

	t.Run("rejects two selectors overlapping on a workflow", func(t *testing.T) {
		claimed := map[string]string{}
		require.NoError(t, claimExecutionRefs(claimed, "", []string{"a", "b"}))

		err := claimExecutionRefs(claimed, "", []string{"b", "c"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `duplicated execution reference "b"`)
	})
}

func TestExecutionRecorder_Complete(t *testing.T) {
	// complete prints an output instruction, and that instruction needs a toolkit configuration.
	t.Setenv("TK_CFG", "{}")
	t.Setenv("TK_REF", "rparent")

	tests := []struct {
		name             string
		result           *testkube.TestWorkflowResult
		wantStatus       string
		wantMessage      string
		wantStepErrors   map[string]string
		wantStepAttempts map[string]int64
	}{
		{
			name: "records the messages of a failed execution",
			result: &testkube.TestWorkflowResult{
				Status:         common.Ptr(testkube.FAILED_TestWorkflowStatus),
				Initialization: &testkube.TestWorkflowStepResult{ErrorMessage: "the pod cannot be scheduled"},
				Steps:          map[string]testkube.TestWorkflowStepResult{"rstep1": {ErrorMessage: "the step timed out", Attempts: 3}},
			},
			wantStatus:       "failed",
			wantMessage:      "the pod cannot be scheduled",
			wantStepErrors:   map[string]string{"rstep1": "the step timed out"},
			wantStepAttempts: map[string]int64{"rstep1": 3},
		},
		{
			name: "records no messages for an execution that passed",
			result: &testkube.TestWorkflowResult{
				Status:         common.Ptr(testkube.PASSED_TestWorkflowStatus),
				Initialization: &testkube.TestWorkflowStepResult{},
				Steps:          map[string]testkube.TestWorkflowStepResult{"rstep1": {}},
			},
			wantStatus: "passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := executiondata.NewRegistry()
			recorder := newExecutionRecorder(registry)
			exec := testkube.TestWorkflowExecution{Id: "exec-1", Name: "child-1"}
			entry := recorder.schedule("child", "child-workflow", exec)

			exec.Result = tt.result
			recorder.complete(entry, exec)

			got, ok, err := registry.Lookup("child", 0)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, executiondata.Execution{
				Id:           "exec-1",
				Name:         "child-1",
				Workflow:     "child-workflow",
				Alias:        "child",
				Status:       tt.wantStatus,
				Outputs:      map[string]string{},
				ErrorMessage: tt.wantMessage,
				StepErrors:   tt.wantStepErrors,
				StepAttempts: tt.wantStepAttempts,
			}, got)

			// Later steps rebuild the registry from the JSON of the published group.
			raw, err := json.Marshal(registry.Group("child"))
			require.NoError(t, err)
			var decoded []executiondata.Execution
			require.NoError(t, json.Unmarshal(raw, &decoded))
			require.Len(t, decoded, 1)
			assert.Equal(t, got.ErrorMessage, decoded[0].ErrorMessage)
			assert.Equal(t, got.StepErrors, decoded[0].StepErrors)
			assert.Equal(t, got.StepAttempts, decoded[0].StepAttempts)
		})
	}
}

func TestFailureSummary(t *testing.T) {
	passed := func(name string) executionOutcome { return executionOutcome{name: name} }
	tests := []struct {
		name    string
		results []operationResult
		want    string
	}{
		{
			name: "returns an empty summary when every execution passed",
			results: []operationResult{
				{outcomes: []executionOutcome{passed("a-1")}},
				{outcomes: []executionOutcome{passed("b-1")}},
			},
			want: "",
		},
		{
			name: "names the failed executions of all entries with their status",
			results: []operationResult{
				{outcomes: []executionOutcome{passed("a-1"), {name: "a-2", err: errors.New("failed")}}},
				{outcomes: []executionOutcome{passed("b-1"), passed("b-2")}},
				{outcomes: []executionOutcome{{name: "c-1", err: errors.New("aborted")}}},
				notScheduled("consumer", errors.New("computing execution: unknown execution \"p\"")),
			},
			want: "3 of 6 executions failed: a-2 (failed), c-1 (aborted), consumer (computing execution: unknown execution \"p\")",
		},
		{
			name: "adds a failure that is not about one execution after the executions",
			results: []operationResult{
				{outcomes: []executionOutcome{{name: "a-1", err: errors.New("failed")}}, err: errors.New("fetching artifacts: fetch.0: not found")},
			},
			want: "1 of 1 executions failed: a-1 (failed); fetching artifacts: fetch.0: not found",
		},
		{
			name: "returns only the other failure when every execution passed",
			results: []operationResult{
				{outcomes: []executionOutcome{passed("a-1")}, err: errors.New("fetching artifacts: fetch.0: not found")},
			},
			want: "fetching artifacts: fetch.0: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, failureSummary(tt.results))
		})
	}
}
