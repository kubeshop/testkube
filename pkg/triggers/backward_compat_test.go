package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
	"github.com/kubeshop/testkube/pkg/log"
)

// TestBackwardCompat_V1TriggersMatchIdentically verifies that existing v1
// TestTrigger matching behavior is preserved after the internalTrigger refactor.
func TestBackwardCompat_V1TriggersMatchIdentically(t *testing.T) {
	tests := map[string]struct {
		trigger  *testtriggersv1.TestTrigger
		event    *watcherEvent
		shouldFire bool
	}{
		"deployment modified matches": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t1", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     testtriggersv1.TestTriggerEventModified,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "production",
				eventType: "modified",
			},
			shouldFire: true,
		},
		"deployment created matches": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t2", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "testkube",
					},
					Event:     testtriggersv1.TestTriggerEventCreated,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "testkube",
				eventType: "created",
			},
			shouldFire: true,
		},
		"wrong resource type does not match": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t3", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     testtriggersv1.TestTriggerEventModified,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "pod",
				name:      "api-server",
				Namespace: "production",
				eventType: "modified",
			},
			shouldFire: false,
		},
		"wrong name does not match": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t4", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     testtriggersv1.TestTriggerEventModified,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "other-service",
				Namespace: "production",
				eventType: "modified",
			},
			shouldFire: false,
		},
		"wrong namespace does not match": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t5", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     testtriggersv1.TestTriggerEventModified,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "staging",
				eventType: "modified",
			},
			shouldFire: false,
		},
		"wrong event does not match": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t6", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     testtriggersv1.TestTriggerEventCreated,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "production",
				eventType: "deleted",
			},
			shouldFire: false,
		},
		"disabled trigger does not match": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t7", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     testtriggersv1.TestTriggerEventModified,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
					Disabled: true,
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "production",
				eventType: "modified",
			},
			shouldFire: false,
		},
		"deployment-specific cause matches": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t8", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:     "deployment-image-update",
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "production",
				eventType: "modified",
				causes:    []testtrigger.Cause{"deployment-image-update"},
			},
			shouldFire: true,
		},
		"pod created matches": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t9", Namespace: "production"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourcePod,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name: "my-pod",
					},
					Event:     testtriggersv1.TestTriggerEventCreated,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "smoke-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "pod",
				name:      "my-pod",
				Namespace: "production",
				eventType: "created",
			},
			shouldFire: true,
		},
		"configmap matches": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t10", Namespace: "default"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource: testtriggersv1.TestTriggerResourceConfigMap,
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name: "feature-flags",
					},
					Event:     testtriggersv1.TestTriggerEventModified,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "e2e-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "configmap",
				name:      "feature-flags",
				Namespace: "default",
				eventType: "modified",
			},
			shouldFire: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fired := false
			key := newStatusKey(triggerSourceV1, tc.trigger.Namespace, tc.trigger.Name)
			s := &Service{
				triggerStatus: map[statusKey]*triggerStatus{
					key: {trigger: convertV1ToInternal(tc.trigger)},
				},
				triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
					fired = true
					return nil
				},
				logger:  log.DefaultLogger,
				metrics: metrics.NewMetrics(),
			}

			err := s.match(context.Background(), tc.event)
			require.NoError(t, err)
			assert.Equal(t, tc.shouldFire, fired, "trigger should%s have fired", map[bool]string{true: "", false: " not"}[tc.shouldFire])
		})
	}
}
