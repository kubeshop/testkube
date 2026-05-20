package triggers

import (
	"context"
	"testing"

	dto "github.com/prometheus/client_model/go"
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

func TestMatchGitTrigger_IncrementsEventMetric(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
		},
	}
	m := metrics.NewMetrics()
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: m,
	}

	counter := m.TestTriggerEventCount.WithLabelValues(trigger.Name, "content", "modified", "")
	metricBefore := &dto.Metric{}
	require.NoError(t, counter.Write(metricBefore))
	before := metricBefore.GetCounter().GetValue()

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.NoError(t, err)

	metricAfter := &dto.Metric{}
	require.NoError(t, counter.Write(metricAfter))
	after := metricAfter.GetCounter().GetValue()
	assert.Equal(t, before+1, after)
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

func TestMatchGitTrigger_IgnoresFieldConditionsForContentEvents(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
			Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
				{
					Path:     ".metadata.name",
					Operator: workflowtriggersv1.FieldOperatorExists,
				},
			},
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {
				trigger: convertV1ToInternal(trigger),
			},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Name)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.NoError(t, err)
	assert.Equal(t, []string{trigger.Name}, executed)
}

func TestMatchGitTrigger_ReturnsErrorWhenTargetTriggerNotReady(t *testing.T) {
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{},
		logger:        log.DefaultLogger,
		metrics:       metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), "trigger-a", "default")
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerTargetNotReady)
}

func TestMatchGitTrigger_ReturnsErrorWhenTargetStatusIsStaleNonContent(t *testing.T) {
	staleTrigger := &internalTrigger{
		Name:         "trigger-a",
		Namespace:    "default",
		Source:       triggerSourceV1,
		ResourceKind: "tests",
		Event:        string(v1.TestTriggerEventCreated),
	}

	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, staleTrigger.Namespace, staleTrigger.Name): {trigger: staleTrigger},
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), staleTrigger.Name, staleTrigger.Namespace)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerTargetNotReady)
}

func TestMatchGitTrigger_ReturnsErrorWhenConditionsConfiguredForSyntheticEvent(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
			ConditionSpec: &v1.TestTriggerConditionSpec{
				Conditions: []v1.TestTriggerCondition{
					{Type_: "Ready", Status: conditionStatusPtr(v1.TRUE_TestTriggerConditionStatuses)},
				},
			},
		},
	}

	executed := false
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			executed = true
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerConditionsUnavailable)
	assert.False(t, executed)
}

func TestMatchGitTrigger_ReturnsErrorWhenProbesConfiguredForSyntheticEvent(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventModified,
			ProbeSpec: &v1.TestTriggerProbeSpec{
				Probes: []v1.TestTriggerProbe{
					{Path: "/health", Port: 8080},
				},
			},
		},
	}

	executed := false
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			executed = true
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerProbesUnavailable)
	assert.False(t, executed)
}

func TestMatchGitTrigger_SkipsNonTestWorkflowExecution(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource:  v1.TestTriggerResourceContent,
			Event:     v1.TestTriggerEventModified,
			Execution: v1.TestTriggerExecutionTest,
		},
	}

	executed := false
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			executed = true
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace)
	require.NoError(t, err)
	assert.False(t, executed)
}

func conditionStatusPtr(v v1.TestTriggerConditionStatuses) *v1.TestTriggerConditionStatuses {
	return &v
}
