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
			Type_: common.Ptr(testkube.GITINTEGRATION_TestWorkflowRunningContextActorType),
			Name:  "git-provider-github",
		},
	}

	rc, untrustedUser := GetNewRunningContext(legacy, nil)

	require.NotNil(t, rc)
	assert.Equal(t, cloud.RunningContextType_GITINTEGRATION, rc.Type)
	assert.Equal(t, "git-provider-github", rc.Name)
	assert.Nil(t, untrustedUser)
}

func TestGetLegacyRunningContext_GitIntegrationType(t *testing.T) {
	req := &cloud.ScheduleRequest{
		RunningContext: &cloud.RunningContext{
			Type: cloud.RunningContextType_GITINTEGRATION,
			Name: "git-provider-github",
		},
	}

	rc := GetLegacyRunningContext(req)

	require.NotNil(t, rc)
	require.NotNil(t, rc.Actor)
	require.NotNil(t, rc.Actor.Type_)
	assert.Equal(t, testkube.GITINTEGRATION_TestWorkflowRunningContextActorType, *rc.Actor.Type_)
	assert.Equal(t, "git-provider-github", rc.Actor.Name)
	require.NotNil(t, rc.Interface_)
	require.NotNil(t, rc.Interface_.Type_)
	assert.Equal(t, testkube.CICD_TestWorkflowRunningContextInterfaceType, *rc.Interface_.Type_)
}
