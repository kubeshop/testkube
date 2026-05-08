package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestMatchGitTrigger_ExecutesOnlyTargetTrigger(t *testing.T) {
	triggerA := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
		},
	}
	triggerB := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-b", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, triggerA.Namespace, triggerA.Name): {trigger: convertV1ToInternal(triggerA)},
			newStatusKey(triggerSourceV1, triggerB.Namespace, triggerB.Name): {trigger: convertV1ToInternal(triggerB)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Name)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), "trigger-a", "default")
	require.NoError(t, err)
	assert.Equal(t, []string{"trigger-a"}, executed)
}

func TestMatchGitTrigger_UsesV1StatusKeyWhenV2HasSameName(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {
				trigger: convertV1ToInternal(trigger),
			},
			newStatusKey(triggerSourceV2, trigger.Namespace, trigger.Name): {
				trigger: &internalTrigger{
					Name:         trigger.Name,
					Namespace:    trigger.Namespace,
					Source:       triggerSourceV2,
					ResourceKind: string(v1.TestTriggerResourceContent),
					Event:        string(v1.TestTriggerEventModified),
				},
			},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Source)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.NoError(t, err)
	assert.Equal(t, []string{triggerSourceV1}, executed)
}

func TestMatchGitWorkflowTrigger_TargetsV2Trigger(t *testing.T) {
	trigger := &workflowtriggersv1.WorkflowTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "workflow-trigger-a", Namespace: "default"},
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			When: workflowtriggersv1.WorkflowTriggerWhen{Event: "modified"},
			Watch: &workflowtriggersv1.WorkflowTriggerWatch{
				Resource: workflowtriggersv1.WorkflowTriggerResource{Kind: "content"},
			},
			Run: workflowtriggersv1.WorkflowTriggerRun{
				Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{Name: "wf"},
			},
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {
				trigger: &internalTrigger{
					Name:         trigger.Name,
					Namespace:    trigger.Namespace,
					Source:       triggerSourceV1,
					ResourceKind: string(v1.TestTriggerResourceContent),
					Event:        string(v1.TestTriggerEventModified),
				},
			},
			newStatusKey(triggerSourceV2, trigger.Namespace, trigger.Name): {
				trigger: convertV2ToInternal(trigger),
			},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Source)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitWorkflowTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.NoError(t, err)
	assert.Equal(t, []string{triggerSourceV2}, executed)
}
