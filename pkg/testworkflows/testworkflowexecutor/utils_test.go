package testworkflowexecutor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestGetNewRunningContext_GitIntegrationActor(t *testing.T) {
	legacy := &testkube.TestWorkflowRunningContext{
		Actor: &testkube.TestWorkflowRunningContextActor{
			Type_: common.Ptr(testkube.QUALITYLOOP_TestWorkflowRunningContextActorType),
			Name:  "git-provider-github",
		},
	}

	rc, untrustedUser := GetNewRunningContext(legacy, nil)

	require.NotNil(t, rc)
	assert.Equal(t, cloud.RunningContextType_QUALITYLOOP, rc.Type)
	assert.Equal(t, "git-provider-github", rc.Name)
	assert.Nil(t, untrustedUser)
}

func TestGetLegacyRunningContext_QualityLoopProtoMapsToGitIntegrationActor(t *testing.T) {
	req := &cloud.ScheduleRequest{
		RunningContext: &cloud.RunningContext{
			Type: cloud.RunningContextType_QUALITYLOOP,
			Name: "git-provider-github",
		},
	}

	rc := GetLegacyRunningContext(req)

	require.NotNil(t, rc)
	require.NotNil(t, rc.Actor)
	require.NotNil(t, rc.Actor.Type_)
	assert.Equal(t, testkube.QUALITYLOOP_TestWorkflowRunningContextActorType, *rc.Actor.Type_)
	assert.Equal(t, "git-provider-github", rc.Actor.Name)
	assert.Empty(t, rc.Actor.ExecutionId, "root QUALITYLOOP execution should not carry a parent link")
	assert.Empty(t, rc.Actor.ExecutionPath)
	require.NotNil(t, rc.Interface_)
	require.NotNil(t, rc.Interface_.Type_)
	assert.Equal(t, testkube.CICD_TestWorkflowRunningContextInterfaceType, *rc.Interface_.Type_)
}

func TestGetLegacyRunningContext_QualityLoopWithParentExecutionIdsPopulatesChainLink(t *testing.T) {
	req := &cloud.ScheduleRequest{
		RunningContext: &cloud.RunningContext{
			Type: cloud.RunningContextType_QUALITYLOOP,
			Name: "git-provider-github",
		},
		ParentExecutionIds: []string{"root-exec-id", "parent-exec-id"},
	}

	rc := GetLegacyRunningContext(req)

	require.NotNil(t, rc)
	require.NotNil(t, rc.Actor)
	require.NotNil(t, rc.Actor.Type_)
	assert.Equal(t, testkube.QUALITYLOOP_TestWorkflowRunningContextActorType, *rc.Actor.Type_)
	assert.Equal(t, "parent-exec-id", rc.Actor.ExecutionId,
		"a chained QUALITYLOOP child must inherit the parent link so ListChildExecutions and cascade abort keep working")
	assert.Equal(t, "root-exec-id/parent-exec-id", rc.Actor.ExecutionPath)
}

// The cap lives in the shared request validation so that both the open source
// scheduler and the control plane's enforce it. A limit only one of them applies
// is a limit that does not exist, and the control plane had none.
func TestValidateRerunPolicy(t *testing.T) {
	t.Run("an ordinary selection is carried", func(t *testing.T) {
		require.NoError(t, ValidateRerunPolicy(&cloud.RerunPolicy{
			ExecutionId: "exec-1",
			OnlyFailed:  true,
			TestCases:   []string{"tests.a::test_one", "tests.b::test_two"},
		}))
	})

	t.Run("no policy at all", func(t *testing.T) {
		require.NoError(t, ValidateRerunPolicy(nil))
	})

	t.Run("a reference without names is fine", func(t *testing.T) {
		// The selection is resolved in the pod from the referenced execution's
		// report, so naming nothing here is the normal case.
		require.NoError(t, ValidateRerunPolicy(&cloud.RerunPolicy{ExecutionId: "exec-1", OnlyFailed: true}))
	})

	t.Run("too many test cases is refused, not truncated", func(t *testing.T) {
		// Truncating would be the dangerous option: a caller who asked for 5000
		// specific tests and silently got 1000 would read the result as those
		// tests having passed.
		cases := make([]string, MaxRerunTestCases+1)
		for i := range cases {
			cases[i] = "t"
		}

		err := ValidateRerunPolicy(&cloud.RerunPolicy{TestCases: cases})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "more than the 1000 that can be carried")
		assert.Contains(t, err.Error(), "testCases.select",
			"the error has to name the way that does work")
	})

	t.Run("too many bytes is refused", func(t *testing.T) {
		// Few enough names to pass the count check, but far too long: the list
		// is inlined into the config the pod reads, which has its own limit.
		cases := []string{strings.Repeat("x", MaxRerunTestCaseBytes+1)}

		err := ValidateRerunPolicy(&cloud.RerunPolicy{TestCases: cases})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bytes of test case names")
	})

	t.Run("exactly at the limits is allowed", func(t *testing.T) {
		cases := make([]string, MaxRerunTestCases)
		for i := range cases {
			cases[i] = "t"
		}
		require.NoError(t, ValidateRerunPolicy(&cloud.RerunPolicy{TestCases: cases}))
	})
}

// A request carrying an oversized selection is refused by the shared entry
// point, not only by the helper - that is what makes the control plane get it.
func TestValidateExecutionRequestBoundsTheRerunSelection(t *testing.T) {
	cases := make([]string, MaxRerunTestCases+1)
	for i := range cases {
		cases[i] = "t"
	}

	err := ValidateExecutionRequest(&cloud.ScheduleRequest{
		Executions: []*cloud.ScheduleExecution{{
			Selector: &cloud.ScheduleResourceSelector{Name: "wf"},
		}},
		Rerun: &cloud.RerunPolicy{TestCases: cases},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more than the 1000 that can be carried")
}
