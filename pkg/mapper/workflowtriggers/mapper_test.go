package workflowtriggers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapCRDToAPI_flattensSpecAndPreservesFields(t *testing.T) {
	delay := metav1.Duration{Duration: 30 * time.Second}
	status := workflowtriggersv1.WorkflowTriggerConditionStatusTrue
	crd := &workflowtriggersv1.WorkflowTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "canary",
			Namespace: "prod",
			Labels:    map[string]string{"app": "api"},
		},
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			Disabled: true,
			Watch: &workflowtriggersv1.WorkflowTriggerWatch{
				Resource: workflowtriggersv1.WorkflowTriggerResource{
					Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout",
				},
			},
			When: workflowtriggersv1.WorkflowTriggerWhen{Event: "modified"},
			Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: workflowtriggersv1.FieldOperatorEquals, Value: "3"},
			},
			Wait: &workflowtriggersv1.WorkflowTriggerWait{
				Conditions: &workflowtriggersv1.WorkflowTriggerWaitConditions{
					Items: []workflowtriggersv1.WorkflowTriggerCondition{
						{Type: "Available", Status: &status},
					},
					Timeout: 60,
				},
			},
			Run: workflowtriggersv1.WorkflowTriggerRun{
				Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{Name: "smoke"},
				Delay:    &delay,
			},
		},
	}

	api := MapCRDToAPI(crd)

	assert.Equal(t, "canary", api.Name)
	assert.Equal(t, "prod", api.Namespace)
	assert.True(t, api.Disabled)
	require.NotNil(t, api.Watch)
	assert.Equal(t, "Rollout", api.Watch.Resource.Kind)
	assert.Equal(t, "modified", api.When.Event)
	require.Len(t, api.Match, 1)
	assert.Equal(t, "equals", api.Match[0].Operator)
	require.NotNil(t, api.Wait)
	require.NotNil(t, api.Wait.Conditions)
	assert.Equal(t, int32(60), api.Wait.Conditions.Timeout)
	assert.Equal(t, "True", api.Wait.Conditions.Items[0].Status)
	assert.Equal(t, "smoke", api.Run.Workflow.Name)
	assert.Equal(t, "30s", api.Run.Delay)
}

func TestMapAPIToCRD_wrapsSpecAndParsesDelay(t *testing.T) {
	api := testkube.WorkflowTrigger{
		Name:     "canary",
		Disabled: true,
		When:     testkube.WorkflowTriggerWhen{Event: "created"},
		Match:    []testkube.WorkflowTriggerFieldCondition{{Path: ".spec.image", Operator: "changed"}},
		Run: testkube.WorkflowTriggerRun{
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "smoke"},
			Delay:    "10s",
		},
	}

	crd := MapAPIToCRD(api)

	assert.Equal(t, "canary", crd.Name)
	assert.True(t, crd.Spec.Disabled)
	assert.Equal(t, "created", crd.Spec.When.Event)
	require.Len(t, crd.Spec.Match, 1)
	assert.Equal(t, workflowtriggersv1.FieldOperatorChanged, crd.Spec.Match[0].Operator)
	require.NotNil(t, crd.Spec.Run.Delay)
	assert.Equal(t, 10*time.Second, crd.Spec.Run.Delay.Duration)
}

func TestMapAPIToCRD_invalidDelay_dropsField(t *testing.T) {
	api := testkube.WorkflowTrigger{
		Name: "bad-delay",
		When: testkube.WorkflowTriggerWhen{Event: "created"},
		Run: testkube.WorkflowTriggerRun{
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "smoke"},
			Delay:    "not-a-duration",
		},
	}

	crd := MapAPIToCRD(api)

	assert.Nil(t, crd.Spec.Run.Delay, "invalid delay should be dropped rather than panic")
}

func TestMapListCRDToAPI_handlesEmptyAndMulti(t *testing.T) {
	assert.Nil(t, MapListCRDToAPI(nil))

	list := &workflowtriggersv1.WorkflowTriggerList{Items: []workflowtriggersv1.WorkflowTrigger{
		{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
	}}
	out := MapListCRDToAPI(list)
	require.Len(t, out, 2)
	assert.Equal(t, "a", out[0].Name)
	assert.Equal(t, "b", out[1].Name)
}
