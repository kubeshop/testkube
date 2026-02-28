package webhooks

import (
	"testing"

	"github.com/stretchr/testify/require"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestWebhookTargetMappingRoundTrip(t *testing.T) {
	apiWebhook := testkube.Webhook{
		Name: "webhook",
		Target: &testkube.ExecutionTarget{
			Match: map[string][]string{
				"application": []string{"accounting"},
				"name":        []string{"runner-us-east"},
			},
			Not: map[string][]string{
				"region": []string{"deprecated"},
			},
			Replicate: []string{"name"},
		},
	}

	crd := MapAPIToCRD(apiWebhook)
	require.NotNil(t, crd.Spec.Target)
	require.Equal(t, apiWebhook.Target.Match, crd.Spec.Target.Match)
	require.Equal(t, apiWebhook.Target.Not, crd.Spec.Target.Not)
	require.Equal(t, apiWebhook.Target.Replicate, crd.Spec.Target.Replicate)

	mappedBack := MapCRDToAPI(crd)
	require.NotNil(t, mappedBack.Target)
	require.Equal(t, apiWebhook.Target, mappedBack.Target)
}

func TestWebhookTargetUpdateMapping(t *testing.T) {
	webhook := executorv1.Webhook{
		Spec: executorv1.WebhookSpec{},
	}
	target := testkube.ExecutionTarget{
		Match: map[string][]string{
			"id": []string{"agent-1"},
		},
	}
	targetPtr := &target
	request := testkube.WebhookUpdateRequest{
		Target: &targetPtr,
	}

	updated := MapUpdateToSpec(request, &webhook)
	require.NotNil(t, updated.Spec.Target)
	require.Equal(t, &commonv1.Target{Match: map[string][]string{"id": []string{"agent-1"}}}, updated.Spec.Target)

	specToUpdate := MapSpecToUpdate(updated)
	require.NotNil(t, specToUpdate.Target)
	require.NotNil(t, *specToUpdate.Target)
	require.Equal(t, target.Match, (*specToUpdate.Target).Match)
}
