package triggers

import (
	"context"
	"testing"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestService_match(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:  "pod",
		name:      "test-pod",
		namespace: "testkube",
		labels:    nil,
		object:    nil,
		eventType: "modified",
		causes:    nil,
	}

	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:            "modified",
			Action:           "run",
			Execution:        "test",
			TestSelector:     testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{}
	s := &Service{
		executor: func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggers:      []*testtriggersv1.TestTrigger{testTrigger1},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_noMatch(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:  "deployment",
		name:      "test-deployment",
		namespace: "testkube",
		labels:    nil,
		object:    nil,
		eventType: "modified",
		causes:    nil,
	}

	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:            "modified",
			Action:           "run",
			Execution:        "test",
			TestSelector:     testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{}
	testExecutorF := func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
		assert.Fail(t, "should not match event")
		return nil
	}
	s := &Service{
		executor:      testExecutorF,
		triggers:      []*testtriggersv1.TestTrigger{testTrigger1},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}
