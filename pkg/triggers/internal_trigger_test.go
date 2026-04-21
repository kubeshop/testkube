package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

func TestConvertV1ToInternal(t *testing.T) {
	condStatus := testtriggersv1.TRUE_TestTriggerConditionStatuses
	delay := metav1.Duration{Duration: 10_000_000_000} // 10s

	v1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-trigger",
			Namespace: "testkube",
			Labels:    map[string]string{"team": "platform"},
		},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource: testtriggersv1.TestTriggerResourceDeployment,
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				Name:      "api-server",
				Namespace: "production",
				NameRegex: "api-.*",
			},
			Event: testtriggersv1.TestTriggerEventModified,
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Conditions: []testtriggersv1.TestTriggerCondition{
					{Type_: "Available", Status: &condStatus, Reason: "MinimumReplicasAvailable"},
				},
				Timeout: 60,
			},
			TestSelector: testtriggersv1.TestTriggerSelector{
				Name: "smoke-test",
			},
			ActionParameters: &testtriggersv1.TestTriggerActionParameters{
				Config: map[string]string{"IMAGE": "jsonpath={.spec.template.spec.containers[0].image}"},
				Tags:   map[string]string{"scope": "smoke"},
				Target: &commonv1.Target{
					Match: map[string][]string{"group": {"staging"}},
				},
			},
			ConcurrencyPolicy: testtriggersv1.TestTriggerConcurrencyPolicyReplace,
			Delay:             &delay,
			Disabled:          false,
		},
	}

	it := convertV1ToInternal(v1)

	assert.Equal(t, "deploy-trigger", it.Name)
	assert.Equal(t, "testkube", it.Namespace)
	assert.Equal(t, triggerSourceV1, it.Source)
	assert.Equal(t, "platform", it.Labels["team"])

	// Resource resolved from enum
	assert.Equal(t, "apps", it.ResourceGroup)
	assert.Equal(t, "v1", it.ResourceVersion)
	assert.Equal(t, "Deployment", it.ResourceKind)
	assert.Equal(t, "api-server", it.ResourceName)
	assert.Equal(t, "production", it.ResourceNamespace)

	// Selector
	require.NotNil(t, it.Selector)
	assert.Equal(t, "api-.*", it.Selector.NameRegex)

	// Event
	assert.Equal(t, "modified", it.Event)

	// No field conditions in v1
	assert.Empty(t, it.FieldConditions)

	// Conditions
	require.NotNil(t, it.Conditions)
	assert.Len(t, it.Conditions.Items, 1)
	assert.Equal(t, "Available", it.Conditions.Items[0].Type)
	assert.Equal(t, "True", *it.Conditions.Items[0].Status)
	assert.Equal(t, int32(60), it.Conditions.Timeout)

	// Workflow selector
	assert.Equal(t, "smoke-test", it.WorkflowSelector.Name)

	// Target
	require.NotNil(t, it.Target)
	assert.Equal(t, []string{"staging"}, it.Target.Match["group"])

	// Config/Tags
	assert.Equal(t, "jsonpath={.spec.template.spec.containers[0].image}", it.Config["IMAGE"])
	assert.Equal(t, "smoke", it.Tags["scope"])

	// Concurrency
	assert.Equal(t, "replace", it.ConcurrencyPolicy)

	// Delay
	require.NotNil(t, it.Delay)
	assert.Equal(t, 10_000_000_000, int(it.Delay.Nanoseconds()))
}

func TestConvertV2ToInternal(t *testing.T) {
	condStatus := workflowtriggersv1.WorkflowTriggerConditionStatusTrue
	delay := metav1.Duration{Duration: 5_000_000_000} // 5s

	v2 := &workflowtriggersv1.WorkflowTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-trigger",
			Namespace: "testkube",
			Labels:    map[string]string{"team": "data"},
		},
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			Watch: &workflowtriggersv1.WorkflowTriggerWatch{
				Resource: workflowtriggersv1.WorkflowTriggerResource{
					Group:     "kafka.strimzi.io",
					Version:   "v1beta2",
					Kind:      "KafkaTopic",
					Name:      "orders",
					Namespace: "kafka",
				},
				Selector: &workflowtriggersv1.WorkflowTriggerSelector{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"critical": "true"},
					},
				},
			},
			When: workflowtriggersv1.WorkflowTriggerWhen{
				Event: "created",
			},
			Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
				{
					Path:     ".spec.partitions",
					Operator: workflowtriggersv1.FieldOperatorEquals,
					Value:    "12",
				},
			},
			Wait: &workflowtriggersv1.WorkflowTriggerWait{
				Conditions: &workflowtriggersv1.WorkflowTriggerWaitConditions{
					Items: []workflowtriggersv1.WorkflowTriggerCondition{
						{Type: "Ready", Status: &condStatus},
					},
					Timeout: 120,
				},
			},
			Run: workflowtriggersv1.WorkflowTriggerRun{
				Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{
					Name: "kafka-integration-tests",
				},
				Target: &commonv1.Target{
					Match:     map[string][]string{"group": {"production"}},
					Replicate: []string{"region"},
				},
				Parameters: &workflowtriggersv1.WorkflowTriggerRunParameters{
					Config: map[string]string{"TOPIC": "{{resource.metadata.name}}"},
					Tags:   map[string]string{"scope": "integration"},
				},
				ConcurrencyPolicy: "forbid",
				Delay:             &delay,
			},
		},
	}

	it := convertV2ToInternal(v2)

	assert.Equal(t, "kafka-trigger", it.Name)
	assert.Equal(t, "testkube", it.Namespace)
	assert.Equal(t, triggerSourceV2, it.Source)
	assert.Equal(t, "data", it.Labels["team"])

	// Watch resource
	assert.Equal(t, "kafka.strimzi.io", it.ResourceGroup)
	assert.Equal(t, "v1beta2", it.ResourceVersion)
	assert.Equal(t, "KafkaTopic", it.ResourceKind)
	assert.Equal(t, "orders", it.ResourceName)
	assert.Equal(t, "kafka", it.ResourceNamespace)

	// Selector
	require.NotNil(t, it.Selector)
	require.NotNil(t, it.Selector.LabelSelector)
	assert.Equal(t, "true", it.Selector.LabelSelector.MatchLabels["critical"])

	// When
	assert.Equal(t, "created", it.Event)

	// Match
	require.Len(t, it.FieldConditions, 1)
	assert.Equal(t, ".spec.partitions", it.FieldConditions[0].Path)
	assert.Equal(t, workflowtriggersv1.FieldOperatorEquals, it.FieldConditions[0].Operator)
	assert.Equal(t, "12", it.FieldConditions[0].Value)

	// Wait
	require.NotNil(t, it.Conditions)
	assert.Len(t, it.Conditions.Items, 1)
	assert.Equal(t, "Ready", it.Conditions.Items[0].Type)
	assert.Equal(t, "True", *it.Conditions.Items[0].Status)
	assert.Equal(t, int32(120), it.Conditions.Timeout)

	// Run
	assert.Equal(t, "kafka-integration-tests", it.WorkflowSelector.Name)
	require.NotNil(t, it.Target)
	assert.Equal(t, []string{"production"}, it.Target.Match["group"])
	assert.Equal(t, []string{"region"}, it.Target.Replicate)

	// Parameters
	assert.Equal(t, "{{resource.metadata.name}}", it.Config["TOPIC"])
	assert.Equal(t, "integration", it.Tags["scope"])

	// Concurrency
	assert.Equal(t, "forbid", it.ConcurrencyPolicy)

	// Delay
	require.NotNil(t, it.Delay)
	assert.Equal(t, 5_000_000_000, int(it.Delay.Nanoseconds()))
}

func TestConvertV1ToInternal_ResourceRef(t *testing.T) {
	v1 := &testtriggersv1.TestTrigger{
		Spec: testtriggersv1.TestTriggerSpec{
			Resource: testtriggersv1.TestTriggerResourceDeployment,
			ResourceRef: &testtriggersv1.TestTriggerResourceRef{
				Group:   "kafka.strimzi.io",
				Version: "v1beta2",
				Kind:    "KafkaTopic",
			},
			Event: testtriggersv1.TestTriggerEventCreated,
		},
	}
	it := convertV1ToInternal(v1)
	assert.Equal(t, "kafka.strimzi.io", it.ResourceGroup)
	assert.Equal(t, "v1beta2", it.ResourceVersion)
	assert.Equal(t, "KafkaTopic", it.ResourceKind)
}

func TestConvertV1ToInternal_ResourceRefOnly(t *testing.T) {
	v1 := &testtriggersv1.TestTrigger{
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceRef: &testtriggersv1.TestTriggerResourceRef{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "Rollout",
			},
			Event: testtriggersv1.TestTriggerEventModified,
		},
	}
	it := convertV1ToInternal(v1)
	assert.Equal(t, "argoproj.io", it.ResourceGroup)
	assert.Equal(t, "v1alpha1", it.ResourceVersion)
	assert.Equal(t, "Rollout", it.ResourceKind)
}

func TestConvertV1ToInternal_AllResourceTypes(t *testing.T) {
	tests := map[string]struct {
		resource      testtriggersv1.TestTriggerResource
		expectedKind  string
		expectedGroup string
	}{
		"pod":         {testtriggersv1.TestTriggerResourcePod, "Pod", ""},
		"deployment":  {testtriggersv1.TestTriggerResourceDeployment, "Deployment", "apps"},
		"statefulset": {testtriggersv1.TestTriggerResourceStatefulSet, "StatefulSet", "apps"},
		"daemonset":   {testtriggersv1.TestTriggerResourceDaemonSet, "DaemonSet", "apps"},
		"service":     {testtriggersv1.TestTriggerResourceService, "Service", ""},
		"ingress":     {testtriggersv1.TestTriggerResourceIngress, "Ingress", "networking.k8s.io"},
		"event":       {testtriggersv1.TestTriggerResourceEvent, "Event", ""},
		"configmap":   {testtriggersv1.TestTriggerResourceConfigMap, "ConfigMap", ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			v1 := &testtriggersv1.TestTrigger{
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: tc.resource,
					Event:    testtriggersv1.TestTriggerEventCreated,
				},
			}
			it := convertV1ToInternal(v1)
			assert.Equal(t, tc.expectedKind, it.ResourceKind)
			assert.Equal(t, tc.expectedGroup, it.ResourceGroup)
		})
	}
}
