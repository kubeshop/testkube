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
