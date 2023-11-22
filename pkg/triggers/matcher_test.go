package triggers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestService_matchConditionsRetry(t *testing.T) {
	t.Parallel()

	retry := 0
	e := &watcherEvent{
		resource:  "deployment",
		name:      "test-deployment",
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
					Ttl:    60,
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
			Resource:         "deployment",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:            "modified",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Timeout: timeout,
				Conditions: []testtriggersv1.TestTriggerCondition{
					{
						Type_:  "Progressing",
						Status: &status,
						Reason: "NewReplicaSetAvailable",
						Ttl:    60,
					},
					{
						Type_:  "Available",
						Status: &status,
					},
				},
			},
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
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

	e := &watcherEvent{
		resource:  "deployment",
		name:      "test-deployment",
		namespace: "testkube",
		labels:    nil,
		object:    nil,
		eventType: "modified",
		causes:    nil,
		conditionsGetter: func() ([]testtriggersv1.TestTriggerCondition, error) {
			status := testtriggersv1.FALSE_TestTriggerConditionStatuses
			return []testtriggersv1.TestTriggerCondition{
				{
					Type_:  "Progressing",
					Status: &status,
					Reason: "NewReplicaSetAvailable",
					Ttl:    60,
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
			Resource:         "deployment",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:            "modified",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Timeout: timeout,
				Conditions: []testtriggersv1.TestTriggerCondition{
					{
						Type_:  "Progressing",
						Status: &status,
						Reason: "NewReplicaSetAvailable",
						Ttl:    60,
					},
					{
						Type_:  "Available",
						Status: &status,
					},
				},
			},
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.ErrorIs(t, err, ErrConditionTimeout)
}

func TestService_matchProbesMultiple(t *testing.T) {
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

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer srv1.Close()

	url1, err := url.Parse(srv1.URL)
	assert.NoError(t, err)

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer srv2.Close()

	url2, err := url.Parse(srv2.URL)
	assert.NoError(t, err)

	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:          "deployment",
			ResourceSelector:  testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:             "modified",
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			ProbeSpec: &testtriggersv1.TestTriggerProbeSpec{
				Probes: []testtriggersv1.TestTriggerProbe{
					{
						Scheme: url1.Scheme,
						Host:   url1.Host,
						Path:   url1.Path,
					},
					{
						Scheme: url2.Scheme,
						Host:   url2.Host,
						Path:   url2.Path,
					},
				},
			},
		},
	}

	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultProbesCheckBackoff: defaultProbesCheckBackoff,
		defaultProbesCheckTimeout: defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
		httpClient:    http.DefaultClient,
	}

	err = s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchProbesTimeout(t *testing.T) {
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

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer srv1.Close()

	url1, err := url.Parse(srv1.URL)
	assert.NoError(t, err)

	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:          "deployment",
			ResourceSelector:  testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:             "modified",
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			ProbeSpec: &testtriggersv1.TestTriggerProbeSpec{
				Timeout: 2,
				Delay:   1,
				Probes: []testtriggersv1.TestTriggerProbe{
					{
						Scheme: url1.Scheme,
						Host:   url1.Host,
						Path:   url1.Path,
					},
					{
						Host: "fakehost",
					},
				},
			},
		},
	}

	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultProbesCheckBackoff: defaultProbesCheckBackoff,
		defaultProbesCheckTimeout: defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
		httpClient:    http.DefaultClient,
	}

	err = s.match(context.Background(), e)
	assert.ErrorIs(t, err, ErrProbeTimeout)

}

func TestService_match(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:  "deployment",
		name:      "test-deployment",
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
					Ttl:    60,
				},
				{
					Type_:  "Available",
					Status: &status,
				},
			}, nil
		},
	}

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer srv1.Close()

	url1, err := url.Parse(srv1.URL)
	assert.NoError(t, err)

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer srv2.Close()

	url2, err := url.Parse(srv2.URL)
	assert.NoError(t, err)

	status := testtriggersv1.TRUE_TestTriggerConditionStatuses
	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "deployment",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:            "modified",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Conditions: []testtriggersv1.TestTriggerCondition{
					{
						Type_:  "Progressing",
						Status: &status,
						Reason: "NewReplicaSetAvailable",
						Ttl:    60,
					},
					{
						Type_:  "Available",
						Status: &status,
					},
				},
			},
			ProbeSpec: &testtriggersv1.TestTriggerProbeSpec{
				Probes: []testtriggersv1.TestTriggerProbe{
					{
						Scheme: url1.Scheme,
						Host:   url1.Host,
						Path:   url1.Path,
					},
					{
						Scheme: url2.Scheme,
						Host:   url2.Host,
						Path:   url2.Path,
					},
				},
			},
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
		httpClient:    http.DefaultClient,
	}

	err = s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchRegex(t *testing.T) {
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

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer srv1.Close()

	testTrigger1 := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:          "deployment",
			ResourceSelector:  testtriggersv1.TestTriggerSelector{NameRegex: "test.*"},
			Event:             "modified",
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{NameRegex: "some.*"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			assert.Equal(t, "testkube", trigger.Namespace)
			assert.Equal(t, "test-trigger-1", trigger.Name)
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:        log.DefaultLogger,
		httpClient:    http.DefaultClient,
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
			Resource:          "pod",
			ResourceSelector:  testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:             "modified",
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}
	statusKey1 := newStatusKey(testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{testTrigger: testTrigger1}
	testExecutorF := func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		assert.Fail(t, "should not match event")
		return nil
	}
	s := &Service{
		triggerExecutor: testExecutorF,
		triggerStatus:   map[statusKey]*triggerStatus{statusKey1: triggerStatus1},
		logger:          log.DefaultLogger,
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}
