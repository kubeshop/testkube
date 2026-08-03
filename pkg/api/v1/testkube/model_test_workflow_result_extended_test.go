package testkube

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, ts, tt.result.QueuedAt)
			assert.Equal(t, ts, tt.result.StartedAt)
			assert.Equal(t, ts, tt.result.FinishedAt)
		})
	}
}
