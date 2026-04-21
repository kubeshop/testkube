package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

func int32Ptr(i int32) *int32 { return &i }

func newDeployment(name string, replicas int32, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(replicas),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: image},
					},
				},
			},
		},
	}
}

func TestMatchFieldSelector(t *testing.T) {
	oldDeploy := newDeployment("test", 3, "nginx:1.19")
	newDeploy := newDeployment("test", 5, "nginx:2.0")

	tests := map[string]struct {
		conditions []v1.WorkflowTriggerFieldCondition
		obj        any
		oldObj     any
		expected   bool
	}{
		"empty conditions matches everything": {
			conditions: nil,
			obj:        newDeploy,
			expected:   true,
		},

		// equals
		"equals matches when value is equal": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorEquals, Value: "5"},
			},
			obj:      newDeploy,
			expected: true,
		},
		"equals fails when value differs": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorEquals, Value: "3"},
			},
			obj:      newDeploy,
			expected: false,
		},

		// not_equals
		"not_equals matches when value differs": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorNotEquals, Value: "3"},
			},
			obj:      newDeploy,
			expected: true,
		},
		"not_equals fails when value is equal": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorNotEquals, Value: "5"},
			},
			obj:      newDeploy,
			expected: false,
		},

		// exists
		"exists matches when field is present": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorExists},
			},
			obj:      newDeploy,
			expected: true,
		},
		"exists fails when field is absent": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.nonExistent", Operator: v1.FieldOperatorExists},
			},
			obj:      newDeploy,
			expected: false,
		},

		// not_exists
		"not_exists matches when field is absent": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.nonExistent", Operator: v1.FieldOperatorNotExists},
			},
			obj:      newDeploy,
			expected: true,
		},
		"not_exists does not fire on invalid path syntax": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: "+++invalid", Operator: v1.FieldOperatorNotExists},
			},
			obj:      newDeploy,
			expected: false,
		},
		"not_exists fails when field is present": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorNotExists},
			},
			obj:      newDeploy,
			expected: false,
		},

		// changed
		"changed matches when field value differs": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChanged},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: true,
		},
		"changed fails when field value is same": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".metadata.name", Operator: v1.FieldOperatorChanged},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: false,
		},
		"changed fails when old object is nil": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChanged},
			},
			obj:      newDeploy,
			oldObj:   nil,
			expected: false,
		},

		// changed_to
		"changed_to matches when value changed to target": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChangedTo, Value: "5"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: true,
		},
		"changed_to fails when value changed to different target": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChangedTo, Value: "10"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: false,
		},
		"changed_to fails when value did not change": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".metadata.name", Operator: v1.FieldOperatorChangedTo, Value: "test"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: false,
		},

		// changed_from
		"changed_from matches when old value was target": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChangedFrom, Value: "3"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: true,
		},
		"changed_from fails when old value was different": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChangedFrom, Value: "10"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: false,
		},
		"changed_from fires when field removed on new object and old had target": {
			// Regression for greptile P2: previously the handler discarded newErr
			// and compared against "", which silently missed removals when the
			// target value was also "".
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChangedFrom, Value: "3"},
			},
			obj:      map[string]any{"spec": map[string]any{}}, // replicas removed
			oldObj:   oldDeploy,
			expected: true,
		},

		// AND logic
		"multiple conditions all must pass": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChanged},
				{Path: ".spec.replicas", Operator: v1.FieldOperatorNotEquals, Value: "1"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: true,
		},
		"multiple conditions one fails all fail": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.replicas", Operator: v1.FieldOperatorChanged},
				{Path: ".spec.replicas", Operator: v1.FieldOperatorEquals, Value: "3"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: false,
		},

		// image change detection
		"image changed detected": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.template.spec.containers.0.image", Operator: v1.FieldOperatorChanged},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: true,
		},
		"image changed_to specific value": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.template.spec.containers.0.image", Operator: v1.FieldOperatorChangedTo, Value: "nginx:2.0"},
			},
			obj:      newDeploy,
			oldObj:   oldDeploy,
			expected: true,
		},

		// changed on non-scalar fields (arrays, maps)
		"changed detects array modification": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.template.spec.containers", Operator: v1.FieldOperatorChanged},
			},
			obj:      newDeployment("test", 3, "nginx:2.0"),
			oldObj:   newDeployment("test", 3, "nginx:1.19"),
			expected: true,
		},
		"changed on unchanged array": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.template.spec.containers", Operator: v1.FieldOperatorChanged},
			},
			obj:      newDeployment("test", 3, "nginx:1.19"),
			oldObj:   newDeployment("test", 3, "nginx:1.19"),
			expected: false,
		},
		"changed detects map modification": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.config", Operator: v1.FieldOperatorChanged},
			},
			obj:      map[string]interface{}{"spec": map[string]interface{}{"config": map[string]interface{}{"key": "new"}}},
			oldObj:   map[string]interface{}{"spec": map[string]interface{}{"config": map[string]interface{}{"key": "old"}}},
			expected: true,
		},
		"changed on unchanged map": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.config", Operator: v1.FieldOperatorChanged},
			},
			obj:      map[string]interface{}{"spec": map[string]interface{}{"config": map[string]interface{}{"key": "same"}}},
			oldObj:   map[string]interface{}{"spec": map[string]interface{}{"config": map[string]interface{}{"key": "same"}}},
			expected: false,
		},

		// exists/not_exists on non-scalar fields
		"exists works on array field": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.template.spec.containers", Operator: v1.FieldOperatorExists},
			},
			obj:      newDeploy,
			expected: true,
		},
		"exists works on map field": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".metadata.labels", Operator: v1.FieldOperatorExists},
			},
			obj:      map[string]interface{}{"metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "test"}}},
			expected: true,
		},
		"equals on array field fails": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.template.spec.containers", Operator: v1.FieldOperatorEquals, Value: "anything"},
			},
			obj:      newDeploy,
			expected: false,
		},
		"equals on map field fails": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".metadata.labels", Operator: v1.FieldOperatorEquals, Value: "anything"},
			},
			obj:      map[string]interface{}{"metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "test"}}},
			expected: false,
		},
		"not_exists on missing array": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.volumes", Operator: v1.FieldOperatorNotExists},
			},
			obj:      newDeploy,
			expected: true,
		},

		// invalid path
		"invalid path does not panic": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: "+++invalid", Operator: v1.FieldOperatorExists},
			},
			obj:      newDeploy,
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := matchFieldSelector(tc.conditions, tc.obj, tc.oldObj)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEvaluateFieldPath(t *testing.T) {
	deploy := newDeployment("my-app", 3, "nginx:1.19")
	deploy.Labels = map[string]string{"app": "api", "env": "production"}
	deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers,
		corev1.Container{Name: "sidecar", Image: "envoy:1.20"},
	)

	tests := map[string]struct {
		path    string
		obj     any
		wantVal string
		wantErr bool
	}{
		// numbers
		"int32 field": {
			path: ".spec.replicas", obj: deploy, wantVal: "3",
		},

		// strings
		"string field": {
			path: ".metadata.name", obj: deploy, wantVal: "my-app",
		},
		"nested string": {
			path: ".spec.template.spec.containers.0.image", obj: deploy, wantVal: "nginx:1.19",
		},

		// array index
		"second container": {
			path: ".spec.template.spec.containers.1.image", obj: deploy, wantVal: "envoy:1.20",
		},
		"container name": {
			path: ".spec.template.spec.containers.0.name", obj: deploy, wantVal: "app",
		},

		// map access
		"label value": {
			path: ".metadata.labels.app", obj: deploy, wantVal: "api",
		},
		"label value 2": {
			path: ".metadata.labels.env", obj: deploy, wantVal: "production",
		},

		// missing fields
		"missing field errors": {
			path: ".spec.nonExistent", obj: deploy, wantErr: true,
		},
		"missing nested field errors": {
			path: ".spec.template.spec.doesNotExist", obj: deploy, wantErr: true,
		},

		// pointer vs value
		"pointer to struct": {
			path: ".spec.replicas", obj: deploy, wantVal: "3",
		},
		"value struct": {
			path: ".spec.replicas", obj: *deploy, wantVal: "3",
		},

		// unstructured map
		"unstructured int": {
			path:    ".spec.partitions",
			obj:     map[string]interface{}{"spec": map[string]interface{}{"partitions": float64(12)}},
			wantVal: "12",
		},
		"unstructured string": {
			path:    ".metadata.name",
			obj:     map[string]interface{}{"metadata": map[string]interface{}{"name": "my-topic"}},
			wantVal: "my-topic",
		},
		"unstructured bool": {
			path:    ".spec.enabled",
			obj:     map[string]interface{}{"spec": map[string]interface{}{"enabled": true}},
			wantVal: "true",
		},
		"unstructured nested map": {
			path:    ".spec.config.key",
			obj:     map[string]interface{}{"spec": map[string]interface{}{"config": map[string]interface{}{"key": "value"}}},
			wantVal: "value",
		},
		"unstructured missing field errors": {
			path:    ".spec.missing",
			obj:     map[string]interface{}{"spec": map[string]interface{}{}},
			wantErr: true,
		},

		// without leading dot
		"path without leading dot": {
			path: "spec.replicas", obj: deploy, wantVal: "3",
		},

		// arrays and maps error for scalar evaluation
		"array field errors": {
			path:    ".spec.template.spec.containers",
			obj:     map[string]interface{}{"spec": map[string]interface{}{"template": map[string]interface{}{"spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"name": "app"}}}}}},
			wantErr: true,
		},
		"map field errors": {
			path:    ".metadata.labels",
			obj:     map[string]interface{}{"metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "api"}}},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			val, err := evaluateFieldPath(tc.path, tc.obj)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantVal, val)
			}
		})
	}
}

func TestMatchFieldSelectorUnstructured(t *testing.T) {
	oldObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"partitions": float64(6),
		},
		"metadata": map[string]interface{}{
			"name": "my-topic",
		},
	}

	newObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"partitions": float64(12),
		},
		"metadata": map[string]interface{}{
			"name": "my-topic",
		},
	}

	tests := map[string]struct {
		conditions []v1.WorkflowTriggerFieldCondition
		obj        any
		oldObj     any
		expected   bool
	}{
		"equals on unstructured": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.partitions", Operator: v1.FieldOperatorEquals, Value: "12"},
			},
			obj:      newObj,
			expected: true,
		},
		"changed on unstructured": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".spec.partitions", Operator: v1.FieldOperatorChanged},
			},
			obj:      newObj,
			oldObj:   oldObj,
			expected: true,
		},
		"not changed on unstructured": {
			conditions: []v1.WorkflowTriggerFieldCondition{
				{Path: ".metadata.name", Operator: v1.FieldOperatorChanged},
			},
			obj:      newObj,
			oldObj:   oldObj,
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := matchFieldSelector(tc.conditions, tc.obj, tc.oldObj)
			assert.Equal(t, tc.expected, result)
		})
	}
}
