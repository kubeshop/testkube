package controlplaneclient

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestChildRunningContextName_MasksQualityLoopParent(t *testing.T) {
	t.Parallel()

	got := childRunningContextName(
		testkube.QUALITYLOOP_TestWorkflowRunningContextActorType,
		"ql-parent-kubeshop-testkube-cloud-api",
	)
	assert.Equal(t, "Git Integration", got,
		"internal ql-parent-* name must not leak into the child's RunningContext.Name")
}

func TestChildRunningContextName_PreservesNonQualityLoopParents(t *testing.T) {
	t.Parallel()

	cases := []testkube.TestWorkflowRunningContextActorType{
		testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType,
		testkube.CRON_TestWorkflowRunningContextActorType,
		testkube.USER_TestWorkflowRunningContextActorType,
		testkube.TESTTRIGGER_TestWorkflowRunningContextActorType,
	}
	for _, parentActor := range cases {
		got := childRunningContextName(parentActor, "user-authored-suite")
		assert.Equal(t, "user-authored-suite", got,
			"non-quality-loop parents keep exposing the parent workflow name")
	}
}
