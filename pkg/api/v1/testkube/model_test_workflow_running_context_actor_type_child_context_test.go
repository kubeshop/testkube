package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestTestWorkflowRunningContextActorType_ChildRunningContextType(t *testing.T) {
	tests := []struct {
		name   string
		actor  TestWorkflowRunningContextActorType
		wanted cloud.RunningContextType
	}{
		{
			name:   "QUALITYLOOP parent propagates to child",
			actor:  QUALITYLOOP_TestWorkflowRunningContextActorType,
			wanted: cloud.RunningContextType_QUALITYLOOP,
		},
		{
			name:   "USER parent falls back to EXECUTION default",
			actor:  USER_TestWorkflowRunningContextActorType,
			wanted: cloud.RunningContextType_EXECUTION,
		},
		{
			name:   "CRON parent falls back to EXECUTION default",
			actor:  CRON_TestWorkflowRunningContextActorType,
			wanted: cloud.RunningContextType_EXECUTION,
		},
		{
			name:   "TESTTRIGGER parent falls back to EXECUTION default",
			actor:  TESTTRIGGER_TestWorkflowRunningContextActorType,
			wanted: cloud.RunningContextType_EXECUTION,
		},
		{
			name:   "empty actor type falls back to EXECUTION default",
			actor:  "",
			wanted: cloud.RunningContextType_EXECUTION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wanted, tt.actor.ChildRunningContextType())
		})
	}
}

func TestTestWorkflowRunningContextActorType_ChildRunningContextName(t *testing.T) {
	tests := []struct {
		name       string
		actor      TestWorkflowRunningContextActorType
		parentName string
		wanted     string
	}{
		{
			name:       "QUALITYLOOP parent hides internal workflow name",
			actor:      QUALITYLOOP_TestWorkflowRunningContextActorType,
			parentName: "ql-parent-kubeshop-testkube-cloud-api",
			wanted:     "Git Integration",
		},
		{
			name:       "USER parent keeps original workflow name",
			actor:      USER_TestWorkflowRunningContextActorType,
			parentName: "user-authored-suite",
			wanted:     "user-authored-suite",
		},
		{
			name:       "TESTWORKFLOW parent keeps original workflow name",
			actor:      TESTWORKFLOW_TestWorkflowRunningContextActorType,
			parentName: "composite-suite",
			wanted:     "composite-suite",
		},
		{
			name:       "empty actor type keeps original workflow name",
			actor:      "",
			parentName: "some-workflow",
			wanted:     "some-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wanted, tt.actor.ChildRunningContextName(tt.parentName))
		})
	}
}
