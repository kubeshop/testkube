package testworkflowexecutor

import (
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
