package commonmapper

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestSchedulerPolicyTargetMappings(t *testing.T) {
	policy := testkube.ONLY_WHEN_MATCHES_SchedulerPolicy
	apiTarget := testkube.ExecutionTarget{
		SchedulerPolicy: &policy,
		Match:           map[string][]string{"environment": {"production"}},
	}

	kubeTarget := MapTargetApiToKube(apiTarget)
	assert.Equal(t, commonv1.SchedulerPolicyOnlyWhenMatches, kubeTarget.SchedulerPolicy)
	assert.Equal(t, apiTarget, MapTargetKubeToAPI(kubeTarget))

	grpcTarget := MapTargetApiToGrpc(&apiTarget)
	assert.Equal(t, string(policy), grpcTarget.SchedulerPolicy)
	assert.True(t, proto.Equal(grpcTarget, MapTargetKubeToGrpc(&kubeTarget)))

	encoded, err := proto.Marshal(grpcTarget)
	require.NoError(t, err)
	decoded := &cloud.ExecutionTarget{}
	require.NoError(t, proto.Unmarshal(encoded, decoded))
	assert.True(t, proto.Equal(grpcTarget, decoded))
}

func TestEmptySchedulerPolicyIsOmitted(t *testing.T) {
	encoded, err := json.Marshal(testkube.ExecutionTarget{Match: map[string][]string{"name": {"runner"}}})
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "schedulerPolicy")
	var oldPayload testkube.ExecutionTarget
	require.NoError(t, json.Unmarshal(encoded, &oldPayload))
	assert.Nil(t, oldPayload.SchedulerPolicy)
	assert.Nil(t, MapTargetKubeToAPI(commonv1.Target{}).SchedulerPolicy)
	protoBytes, err := proto.Marshal(&cloud.ExecutionTarget{})
	require.NoError(t, err)
	assert.Empty(t, protoBytes)
}
