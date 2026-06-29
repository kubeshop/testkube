package testtriggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
)

func TestMapResourceFieldRefAPIToKube_InvalidDivisorDefaultsToOne(t *testing.T) {
	mapped := commonmapper.MapResourceFieldRefAPIToKube(&testkube.ResourceFieldRef{
		ContainerName: "app",
		Resource:      "limits.cpu",
		Divisor:       "not-a-quantity",
	})

	assert.Equal(t, "1", mapped.Divisor.String())
}

func TestMapResourceFieldRefAPIToKube_ValidDivisorIsPreserved(t *testing.T) {
	mapped := commonmapper.MapResourceFieldRefAPIToKube(&testkube.ResourceFieldRef{
		ContainerName: "app",
		Resource:      "limits.cpu",
		Divisor:       "1m",
	})

	assert.Equal(t, "1m", mapped.Divisor.String())
}

func TestMapTestTriggerUpsertRequestToTestTriggerCRD_AllowsNilSelectors(t *testing.T) {
	crd := MapTestTriggerUpsertRequestToTestTriggerCRD(testkube.TestTriggerUpsertRequest{
		Name:      "content-trigger",
		Namespace: "testkube",
		Event:     "modified",
	})

	assert.Equal(t, "content-trigger", crd.Name)
	assert.Equal(t, "testkube", crd.Namespace)
	assert.Equal(t, "", crd.Spec.ResourceSelector.Name)
	assert.Equal(t, "", crd.Spec.TestSelector.Name)
}

func TestMapTestTriggerUpsertRequestToTestTriggerCRDWithExistingMeta_AllowsNilSelectors(t *testing.T) {
	crd := MapTestTriggerUpsertRequestToTestTriggerCRDWithExistingMeta(
		testkube.TestTriggerUpsertRequest{
			Name:      "content-trigger",
			Namespace: "testkube",
			Event:     "modified",
		},
		metav1.ObjectMeta{
			Name:      "content-trigger",
			Namespace: "testkube",
		},
	)

	assert.Equal(t, "content-trigger", crd.Name)
	assert.Equal(t, "testkube", crd.Namespace)
	assert.Equal(t, "", crd.Spec.ResourceSelector.Name)
	assert.Equal(t, "", crd.Spec.TestSelector.Name)
}

func TestMapContentGitPullRequestToCRD_NilInput(t *testing.T) {
	result := mapContentGitPullRequestToCRD(nil)
	assert.Nil(t, result)
}

func TestMapContentGitPullRequestToCRD_CopiesAllFields(t *testing.T) {
	pr := &testkube.TestTriggerContentGitPullRequest{
		Types:          []string{"opened", "synchronize"},
		Branches:       []string{"main", "release/*"},
		BranchesIgnore: []string{"release/legacy-*"},
	}
	result := mapContentGitPullRequestToCRD(pr)
	assert.NotNil(t, result)
	assert.Equal(t, []string{"opened", "synchronize"}, result.Types)
	assert.Equal(t, []string{"main", "release/*"}, result.Branches)
	assert.Equal(t, []string{"release/legacy-*"}, result.BranchesIgnore)
}
