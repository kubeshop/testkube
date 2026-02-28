package webhooktemplates

import (
	"testing"

	"github.com/stretchr/testify/require"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestWebhookTemplateTargetMappingRoundTrip(t *testing.T) {
	apiTemplate := testkube.WebhookTemplateCreateRequest{
		Name: "template",
		Target: &testkube.ExecutionTarget{
			Match: map[string][]string{
				"application": []string{"accounting"},
			},
			Not: map[string][]string{
				"region": []string{"deprecated"},
			},
			Replicate: []string{"name"},
		},
	}

	crd := MapAPIToCRD(apiTemplate)
	require.NotNil(t, crd.Spec.Target)
	require.Equal(t, apiTemplate.Target.Match, crd.Spec.Target.Match)
	require.Equal(t, apiTemplate.Target.Not, crd.Spec.Target.Not)
	require.Equal(t, apiTemplate.Target.Replicate, crd.Spec.Target.Replicate)

	mappedBack := MapCRDToAPI(crd)
	require.NotNil(t, mappedBack.Target)
	require.Equal(t, apiTemplate.Target, mappedBack.Target)
}

func TestWebhookTemplateTargetUpdateMapping(t *testing.T) {
	template := executorv1.WebhookTemplate{
		Spec: executorv1.WebhookTemplateSpec{},
	}
	target := testkube.ExecutionTarget{
		Match: map[string][]string{
			"name": []string{"runner-us-east"},
		},
	}
	targetPtr := &target
	request := testkube.WebhookTemplateUpdateRequest{
		Target: &targetPtr,
	}

	updated := MapUpdateToSpec(request, &template)
	require.NotNil(t, updated.Spec.Target)
	require.Equal(t, &commonv1.Target{Match: map[string][]string{"name": []string{"runner-us-east"}}}, updated.Spec.Target)

	specToUpdate := MapSpecToUpdate(updated)
	require.NotNil(t, specToUpdate.Target)
	require.NotNil(t, *specToUpdate.Target)
	require.Equal(t, target.Match, (*specToUpdate.Target).Match)
}
