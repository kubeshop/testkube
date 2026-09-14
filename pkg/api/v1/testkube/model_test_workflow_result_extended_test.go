package testkube

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/common"
)

func TestHealDuration(t *testing.T) {
	scheduled := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	finished := scheduled.Add(5 * time.Minute)
	passed := PASSED_TestWorkflowStatus

	tests := []struct {
		name           string
		pauses         []TestWorkflowPause
		steps          map[string]TestWorkflowStepResult
		wantPausedMs   int32
		wantDurationMs int32
		wantDuration   string
	}{
		{
			name: "subtracts paused time from duration",
			pauses: []TestWorkflowPause{{
				Ref:       "step1",
				PausedAt:  scheduled.Add(1 * time.Minute),
				ResumedAt: scheduled.Add(3 * time.Minute),
			}},
			steps: map[string]TestWorkflowStepResult{
				"step1": {StartedAt: scheduled, FinishedAt: finished},
			},
			wantPausedMs:   2 * 60 * 1000,
			wantDurationMs: 3 * 60 * 1000,
			wantDuration:   "3m0s",
		},
		{
			name:           "no pauses keeps duration equal to total",
			wantPausedMs:   0,
			wantDurationMs: 5 * 60 * 1000,
			wantDuration:   "5m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TestWorkflowResult{
				Status:     &passed,
				QueuedAt:   scheduled,
				FinishedAt: finished,
				Pauses:     tt.pauses,
				Steps:      tt.steps,
			}
			r.HealDuration(scheduled)

			assert.Equal(t, tt.wantPausedMs, r.PausedMs)
			assert.Equal(t, tt.wantDurationMs, r.DurationMs)
			assert.Equal(t, tt.wantDuration, r.Duration)
			assert.Equal(t, int32(5*60*1000), r.TotalDurationMs)
			assert.Equal(t, "5m0s", r.TotalDuration)
		})
	}
}

func TestTestWorkflowResult_Fatal(t *testing.T) {
	ts := time.Date(2026, 7, 22, 15, 20, 0, 0, time.UTC)

	tests := []struct {
		name             string
		result           *TestWorkflowResult
		err              error
		aborted          bool
		wantStatus       TestWorkflowStatus
		wantInitStatus   TestWorkflowStepStatus
		wantErrorMessage string
		wantErrorReason  string
	}{
		{
			name:             "handles nil error and nil initialization without panicking",
			result:           &TestWorkflowResult{},
			err:              nil,
			aborted:          true,
			wantStatus:       ABORTED_TestWorkflowStatus,
			wantInitStatus:   ABORTED_TestWorkflowStepStatus,
			wantErrorMessage: "fatal error without details",
		},
		{
			name: "keeps existing initialization error message when error is nil",
			result: &TestWorkflowResult{
				Initialization: &TestWorkflowStepResult{ErrorMessage: "original failure"},
			},
			err:              nil,
			aborted:          false,
			wantStatus:       FAILED_TestWorkflowStatus,
			wantInitStatus:   FAILED_TestWorkflowStepStatus,
			wantErrorMessage: "original failure",
		},
		{
			name: "clears the reason code of the message that the error replaces",
			result: &TestWorkflowResult{
				Initialization: &TestWorkflowStepResult{ErrorMessage: "no node can run the pod", ErrorReason: "unschedulable"},
			},
			err:              assert.AnError,
			wantStatus:       FAILED_TestWorkflowStatus,
			wantInitStatus:   FAILED_TestWorkflowStepStatus,
			wantErrorMessage: assert.AnError.Error(),
		},
		{
			name: "stores error message and marks result failed",
			result: &TestWorkflowResult{
				Initialization: &TestWorkflowStepResult{},
			},
			err:              assert.AnError,
			aborted:          false,
			wantStatus:       FAILED_TestWorkflowStatus,
			wantInitStatus:   FAILED_TestWorkflowStepStatus,
			wantErrorMessage: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.result.Fatal(tt.err, tt.aborted, ts)

			assert.Equal(t, tt.wantStatus, *tt.result.Status)
			assert.Equal(t, tt.wantInitStatus, *tt.result.Initialization.Status)
			assert.Equal(t, tt.wantErrorMessage, tt.result.Initialization.ErrorMessage)
			assert.Equal(t, tt.wantErrorReason, tt.result.Initialization.ErrorReason)
			assert.Equal(t, ts, tt.result.QueuedAt)
			assert.Equal(t, ts, tt.result.StartedAt)
			assert.Equal(t, ts, tt.result.FinishedAt)
		})
	}
}

func TestTestWorkflowResult_HealAbortedOrCanceled(t *testing.T) {
	type stepWant struct {
		status  TestWorkflowStepStatus
		message string
	}
	const defaultErrorStr = "Job has been aborted"
	aborted := string(ABORTED_TestWorkflowStatus)
	canceled := string(CANCELED_TestWorkflowStatus)
	step := func(status TestWorkflowStepStatus, message string) TestWorkflowStepResult {
		return TestWorkflowStepResult{Status: common.Ptr(status), ErrorMessage: message}
	}

	tests := []struct {
		name            string
		initialization  TestWorkflowStepResult
		steps           map[string]TestWorkflowStepResult
		sigSequence     []TestWorkflowSignature
		errorStr        string
		terminationCode string
		wantInit        stepWant
		wantSteps       map[string]stepWant
	}{
		{
			name:            "puts a recorded initialization cause after the reason of the cancel",
			initialization:  step(RUNNING_TestWorkflowStepStatus, "the pod cannot be scheduled: 0/1 nodes are available"),
			steps:           map[string]TestWorkflowStepResult{},
			errorStr:        "by the user",
			terminationCode: canceled,
			wantInit:        stepWant{CANCELED_TestWorkflowStepStatus, "The execution has been canceled. (by the user: the pod cannot be scheduled: 0/1 nodes are available)"},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:            "replaces the default message on the initialization step",
			initialization:  step(RUNNING_TestWorkflowStepStatus, defaultErrorStr),
			steps:           map[string]TestWorkflowStepResult{},
			errorStr:        "by the runner",
			terminationCode: aborted,
			wantInit:        stepWant{ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the runner)"},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:            "writes the abort message when the initialization message is empty",
			initialization:  step(QUEUED_TestWorkflowStepStatus, ""),
			steps:           map[string]TestWorkflowStepResult{},
			terminationCode: aborted,
			wantInit:        stepWant{ABORTED_TestWorkflowStepStatus, "The execution has been aborted."},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:            "does not repeat a recorded cause that the caller passes as the reason",
			initialization:  step(RUNNING_TestWorkflowStepStatus, "the image could not be pulled"),
			steps:           map[string]TestWorkflowStepResult{},
			errorStr:        "the image could not be pulled",
			terminationCode: aborted,
			wantInit:        stepWant{ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (the image could not be pulled)"},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:            "removes the end period of the cause and does not add the default reason",
			initialization:  step(RUNNING_TestWorkflowStepStatus, "no node can run the pod: 1 Insufficient cpu."),
			steps:           map[string]TestWorkflowStepResult{},
			errorStr:        defaultErrorStr,
			terminationCode: aborted,
			wantInit:        stepWant{ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (no node can run the pod: 1 Insufficient cpu)"},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:            "keeps the message of an initialization step that an earlier heal canceled",
			initialization:  step(CANCELED_TestWorkflowStepStatus, "The execution has been canceled. (by the user: the image could not be pulled)"),
			steps:           map[string]TestWorkflowStepResult{},
			errorStr:        "by the runner",
			terminationCode: aborted,
			wantInit:        stepWant{ABORTED_TestWorkflowStepStatus, "The execution has been canceled. (by the user: the image could not be pulled)"},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:            "keeps the message of an initialization step that an earlier heal aborted",
			initialization:  step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the user: the image could not be pulled)"),
			steps:           map[string]TestWorkflowStepResult{},
			errorStr:        "The execution has been aborted. (by the user: the image could not be pulled)",
			terminationCode: aborted,
			wantInit:        stepWant{ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the user: the image could not be pulled)"},
			wantSteps:       map[string]stepWant{},
		},
		{
			name:           "keeps finished steps, aborts the first running step and skips later queued steps",
			initialization: step(FAILED_TestWorkflowStepStatus, "init failed"),
			steps: map[string]TestWorkflowStepResult{
				"passed":  step(PASSED_TestWorkflowStepStatus, ""),
				"running": step(RUNNING_TestWorkflowStepStatus, ""),
				"queued":  step(QUEUED_TestWorkflowStepStatus, ""),
			},
			sigSequence:     []TestWorkflowSignature{{Ref: "passed"}, {Ref: "running"}, {Ref: "queued"}},
			errorStr:        "trigger: deleted",
			terminationCode: aborted,
			wantInit:        stepWant{FAILED_TestWorkflowStepStatus, "init failed"},
			wantSteps: map[string]stepWant{
				"passed":  {PASSED_TestWorkflowStepStatus, ""},
				"running": {ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (trigger: deleted)"},
				"queued":  {SKIPPED_TestWorkflowStepStatus, "The execution was aborted before. (trigger: deleted)"},
			},
		},
		{
			name:           "marks the first running step canceled when the code is canceled",
			initialization: step(PASSED_TestWorkflowStepStatus, ""),
			steps: map[string]TestWorkflowStepResult{
				"running": step(RUNNING_TestWorkflowStepStatus, ""),
				"queued":  step(QUEUED_TestWorkflowStepStatus, ""),
			},
			sigSequence:     []TestWorkflowSignature{{Ref: "running"}, {Ref: "queued"}},
			errorStr:        "user: stopped",
			terminationCode: canceled,
			wantInit:        stepWant{PASSED_TestWorkflowStepStatus, ""},
			wantSteps: map[string]stepWant{
				"running": {CANCELED_TestWorkflowStepStatus, "The execution has been canceled. (user: stopped)"},
				"queued":  {SKIPPED_TestWorkflowStepStatus, "The execution was canceled before. (user: stopped)"},
			},
		},
		{
			name:           "keeps a recorded cause of the first step that did not start",
			initialization: step(PASSED_TestWorkflowStepStatus, ""),
			steps: map[string]TestWorkflowStepResult{
				"passed":  step(PASSED_TestWorkflowStepStatus, ""),
				"waiting": step(QUEUED_TestWorkflowStepStatus, "the image could not be pulled"),
				"queued":  step(QUEUED_TestWorkflowStepStatus, ""),
			},
			sigSequence:     []TestWorkflowSignature{{Ref: "passed"}, {Ref: "waiting"}, {Ref: "queued"}},
			errorStr:        "by the runner",
			terminationCode: aborted,
			wantInit:        stepWant{PASSED_TestWorkflowStepStatus, ""},
			wantSteps: map[string]stepWant{
				"passed":  {PASSED_TestWorkflowStepStatus, ""},
				"waiting": {ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the runner: the image could not be pulled)"},
				"queued":  {SKIPPED_TestWorkflowStepStatus, "The execution was aborted before. (by the runner)"},
			},
		},
		{
			name:            "cancels a step that is not in the signature from the termination code, not from the result status",
			initialization:  step(PASSED_TestWorkflowStepStatus, ""),
			steps:           map[string]TestWorkflowStepResult{"unknown": step(RUNNING_TestWorkflowStepStatus, "")},
			errorStr:        "by the user",
			terminationCode: canceled,
			wantInit:        stepWant{PASSED_TestWorkflowStepStatus, ""},
			wantSteps: map[string]stepWant{
				"unknown": {CANCELED_TestWorkflowStepStatus, "The execution was canceled, but we could not determine steps order: by the user"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialization := tt.initialization
			r := &TestWorkflowResult{
				// The result status differs from the termination code, so a case fails if the heal reads the status.
				Status:         common.Ptr(RUNNING_TestWorkflowStatus),
				Initialization: &initialization,
				Steps:          tt.steps,
			}

			r.HealAbortedOrCanceled(tt.sigSequence, tt.errorStr, defaultErrorStr, tt.terminationCode)

			assert.Equal(t, tt.wantInit, stepWant{*r.Initialization.Status, r.Initialization.ErrorMessage})
			gotSteps := make(map[string]stepWant, len(r.Steps))
			for ref, s := range r.Steps {
				gotSteps[ref] = stepWant{*s.Status, s.ErrorMessage}
			}
			assert.Equal(t, tt.wantSteps, gotSteps)
		})
	}
}
