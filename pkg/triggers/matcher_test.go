package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestService_matchConditionsRetry(t *testing.T) {
	t.Parallel()

	retry := 0
	e := &watcherEvent{
		resource:  "pod",
		name:      "test-pod",
		namespace: "testkube",
		labels:    nil,
		object:    nil,
		eventType: "modified",
		causes:    nil,
		conditionsGetter: func() ([]testtriggersv1.TestTriggerCondition, error) {
			retry++
			status := testtriggersv1.FALSE_TestTriggerConditionStatuses
			if retry == 1 {
				status = testtriggersv1.TRUE_TestTriggerConditionStatuses
			}

			return []testtriggersv1.TestTriggerCondition{
				{
					Type_:  "Progressing",
					Status: &status,
					Reason: "NewReplicaSetAvailable",
				},
				{
					Type_:  "Available",
					Status: &status,
				},
			}, nil
		},
	}

	var timeout int32 = 1
	status := testtriggersv1.TRUE_TestTriggerConditionStatuses
	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:            "modified",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Timeout: timeout,
				Conditions: []testtriggersv1.TestTriggerCondition{
					{
						Type_:  "Progressing",
						Status: &status,
						Reason: "NewReplicaSetAvailable",
					},
					{
						Type_:  "Available",
						Status: &status,
					},
				},
			},
			Action:       "run",
			Execution:    "test",
			TestSelector: testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		executor: func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, retry)
}

func TestService_matchConditionsTimeout(t *testing.T) {
	t.Parallel()

	retry := 0
	e := &watcherEvent{
		resource:  "pod",
		name:      "test-pod",
		namespace: "testkube",
		labels:    nil,
		object:    nil,
		eventType: "modified",
		causes:    nil,
		conditionsGetter: func() ([]testtriggersv1.TestTriggerCondition, error) {
			retry++
			status := testtriggersv1.FALSE_TestTriggerConditionStatuses
			return []testtriggersv1.TestTriggerCondition{
				{
					Type_:  "Progressing",
					Status: &status,
					Reason: "NewReplicaSetAvailable",
				},
				{
					Type_:  "Available",
					Status: &status,
				},
			}, nil
		},
	}

	var timeout int32 = 1
	status := testtriggersv1.TRUE_TestTriggerConditionStatuses
	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:            "modified",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Timeout: timeout,
				Conditions: []testtriggersv1.TestTriggerCondition{
					{
						Type_:  "Progressing",
						Status: &status,
						Reason: "NewReplicaSetAvailable",
					},
					{
						Type_:  "Available",
						Status: &status,
					},
				},
			},
			Action:       "run",
			Execution:    "test",
			TestSelector: testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		executor: func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.ErrorIs(t, err, ErrConditionTimeout)
	assert.Equal(t, 2, retry)
}

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
		conditionsGetter: func() ([]testtriggersv1.TestTriggerCondition, error) {
			status := testtriggersv1.TRUE_TestTriggerConditionStatuses
			return []testtriggersv1.TestTriggerCondition{
				{
					Type_:  "Progressing",
					Status: &status,
					Reason: "NewReplicaSetAvailable",
				},
				{
					Type_:  "Available",
					Status: &status,
				},
			}, nil
		},
	}

	status := testtriggersv1.TRUE_TestTriggerConditionStatuses
	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:            "modified",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Conditions: []testtriggersv1.TestTriggerCondition{
					{
						Type_:  "Progressing",
						Status: &status,
						Reason: "NewReplicaSetAvailable",
					},
					{
						Type_:  "Available",
						Status: &status,
					},
				},
			},
			Action:       "run",
			Execution:    "test",
			TestSelector: testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		executor: func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
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
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	testExecutorF := func(ctx context.Context, trigger *testtriggersv1.TestTrigger) error {
		assert.Fail(t, "should not match event")
		return nil
	}
	s := &Service{
		executor:      testExecutorF,
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}
