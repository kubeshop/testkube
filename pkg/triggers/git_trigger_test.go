package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/api/testtriggers/v1"
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
