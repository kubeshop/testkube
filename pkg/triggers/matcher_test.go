package triggers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestService_matchConditionsRetry(t *testing.T) {
	t.Parallel()

	retry := 0
	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled:          false,
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
		metrics:       metrics.NewMetrics(),
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, retry)
}

func TestService_matchConditionsTimeout(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled:          false,
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
		metrics:       metrics.NewMetrics(),
	}

	err := s.match(context.Background(), e)
	assert.ErrorIs(t, err, ErrConditionTimeout)
}

func TestService_matchProbesMultiple(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled: false,
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
		metrics:       metrics.NewMetrics(),
	}

	err = s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchProbesTimeout(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled: false,
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
		metrics:       metrics.NewMetrics(),
	}

	err = s.match(context.Background(), e)
	assert.ErrorIs(t, err, ErrProbeTimeout)
}

func TestService_match(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled:          false,
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
		metrics:       metrics.NewMetrics(),
	}

	err = s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchRegex(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled:          false,
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
		metrics:       metrics.NewMetrics(),
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_noMatch(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:       "deployment",
		name:           "test-deployment",
		Namespace:      "testkube",
		resourceLabels: nil,
		objectMeta:     nil,
		Object:         nil,
		eventType:      "modified",
		causes:         nil,
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
			Disabled:          false,
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
		metrics:         metrics.NewMetrics(),
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func newDefaultTestTriggersService(t *testing.T, trigger *testtriggersv1.TestTrigger) *Service {
	key := newStatusKey(trigger.Namespace, trigger.Name)
	status := &triggerStatus{testTrigger: trigger}
	return &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
			t.Log("default test trigger executor")
			return nil
		},
		triggerStatus: map[statusKey]*triggerStatus{key: status},
		logger:        log.DefaultLogger,
		httpClient:    http.DefaultClient,
		metrics:       metrics.NewMetrics(),
	}
}

func TestService_matchResourceSelector_matchLabels(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resourceLabels: map[string]string{
			"app": "test",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	triggerCount := 0
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		triggerCount++
		assert.Equal(t, "testkube", trigger.Namespace)
		assert.Equal(t, "test-trigger", trigger.Name)
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, triggerCount)
}

func TestService_matchResourceSelector_matchLabels_noMatch(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resourceLabels: map[string]string{
			"app": "test",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "not-test",
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		t.Error("should not trigger")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchResourceSelector_matchExpression(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resourceLabels: map[string]string{
			"app": "test",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"test", "dev", "staging"},
						},
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	triggerCount := 0
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		triggerCount++
		assert.Equal(t, "testkube", trigger.Namespace)
		assert.Equal(t, "test-trigger", trigger.Name)
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, triggerCount)
}

func TestService_matchResourceSelector_matchExpression_noMatch(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resourceLabels: map[string]string{
			"app": "test",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		t.Error("should not trigger executor")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_nilSelector(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource: "deployment",
		resourceLabels: map[string]string{
			"app": "test",
		},
		EventLabels: map[string]string{
			eventLabelKeyResourceKind: "Deployment",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Selector: nil,
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		t.Error("should not match")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_emptySelector(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource: "deployment",
		resourceLabels: map[string]string{
			"app": "test",
		},
		EventLabels: map[string]string{
			eventLabelKeyResourceKind: "Deployment",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Selector: &metav1.LabelSelector{},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		t.Error("should not match")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_matchLabels(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource: "deployment",
		// Event labels should take precedence over the resource labels
		EventLabels: map[string]string{
			"label-source":            "listener",
			eventLabelKeyResourceKind: "Deployment",
		},
		resourceLabels: map[string]string{
			"label-source": "resource",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"label-source": "listener",
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	triggerCount := 0
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		triggerCount++
		assert.Equal(t, "testkube", trigger.Namespace)
		assert.Equal(t, "test-trigger", trigger.Name)
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, triggerCount)
}

func TestService_matchSelector_matchExpression(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource: "deployment",
		EventLabels: map[string]string{
			"label-source":            "listener",
			eventLabelKeyResourceKind: "Deployment",
		},
		resourceLabels: map[string]string{
			"label-source": "resource",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "label-source",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"listener"},
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	triggerCount := 0
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		triggerCount++
		assert.Equal(t, "testkube", trigger.Namespace)
		assert.Equal(t, "test-trigger", trigger.Name)
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, triggerCount)
}

func TestService_matchSelector_noMatch(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource: "deployment",
		EventLabels: map[string]string{
			"label-source":            "listener",
			eventLabelKeyResourceKind: "Deployment",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "label-source",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"not-listener"},
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		t.Error("should not trigger")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_matchResourceSelector(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:  "deployment",
		name:      "test-deployment",
		Namespace: "testkube",
		EventLabels: map[string]string{
			"label-source":            "listener",
			eventLabelKeyResourceKind: "Deployment",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource: "deployment",
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				NameRegex: "test-deploy.*",
			},
			Selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "label-source",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"listener"},
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	triggerCount := 0
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		triggerCount++
		assert.Equal(t, "testkube", trigger.Namespace)
		assert.Equal(t, "test-trigger", trigger.Name)
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, triggerCount)
}

func TestService_matchSelector_noMatchResourceSelector(t *testing.T) {
	t.Parallel()

	e := &watcherEvent{
		resource:  "deployment",
		name:      "test-deployment",
		Namespace: "testkube",
		EventLabels: map[string]string{
			"label-source":            "listener",
			eventLabelKeyResourceKind: "Deployment",
		},
	}

	testTrigger := &testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
		Spec: testtriggersv1.TestTriggerSpec{
			ResourceSelector: testtriggersv1.TestTriggerSelector{
				Name: "not-test-deployment",
			},
			Selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "label-source",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"listener"},
					},
				},
			},
		},
	}

	s := newDefaultTestTriggersService(t, testTrigger)
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *testtriggersv1.TestTrigger) error {
		t.Error("should not match")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}
