package triggers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

func TestService_matchConditionsRetry(t *testing.T) {

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
			Execution:         "testworkflow",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			Disabled:          false,
		},
	}
	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
			Execution:         "testworkflow",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			Disabled:          false,
		},
	}
	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
			Execution:         "testworkflow",
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

	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	s := &Service{
		defaultProbesCheckBackoff: defaultProbesCheckBackoff,
		defaultProbesCheckTimeout: defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
			Execution:         "testworkflow",
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

	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	s := &Service{
		defaultProbesCheckBackoff: defaultProbesCheckBackoff,
		defaultProbesCheckTimeout: defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
			Execution:         "testworkflow",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			Disabled:          false,
		},
	}
	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
			Execution:         "testworkflow",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{NameRegex: "some.*"},
			Disabled:          false,
		},
	}
	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	s := &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
			Execution:         "testworkflow",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
			Disabled:          false,
		},
	}
	statusKey1 := newStatusKey(triggerSourceV1, testTrigger1.Namespace, testTrigger1.Name)
	triggerStatus1 := &triggerStatus{trigger: convertV1ToInternal(testTrigger1)}
	testExecutorF := func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
	key := newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name)
	status := &triggerStatus{trigger: convertV1ToInternal(trigger)}
	return &Service{
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		t.Error("should not trigger")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchResourceSelector_matchExpression(t *testing.T) {

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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		t.Error("should not trigger executor")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_nilSelector(t *testing.T) {

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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		t.Error("should not match")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_emptySelector(t *testing.T) {

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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		t.Error("should not match")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_matchLabels(t *testing.T) {

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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		triggerCount++
		assert.Equal(t, "testkube", trigger.Namespace)
		assert.Equal(t, "test-trigger", trigger.Name)
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, 1, triggerCount)
}

func TestService_matchSelector_matchLabels_resourceKindCaseInsensitive(t *testing.T) {

	cases := []struct {
		name              string
		selectorKindValue string
	}{
		{name: "lowercase selector value", selectorKindValue: "deployment"},
		{name: "capitalized selector value", selectorKindValue: "Deployment"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &watcherEvent{
				resource: "deployment",
				EventLabels: map[string]string{
					eventLabelKeyResourceKind:      "Deployment",
					eventLabelKeyResourceName:      "backend-api",
					eventLabelKeyResourceNamespace: "sandbox-cron-schedules",
				},
			}

			testTrigger := &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							eventLabelKeyResourceKind:      tc.selectorKindValue,
							eventLabelKeyResourceName:      "backend-api",
							eventLabelKeyResourceNamespace: "sandbox-cron-schedules",
						},
					},
				},
			}

			s := newDefaultTestTriggersService(t, testTrigger)
			triggerCount := 0
			s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
				triggerCount++
				assert.Equal(t, "testkube", trigger.Namespace)
				assert.Equal(t, "test-trigger", trigger.Name)
				return nil
			}

			err := s.match(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, 1, triggerCount)
		})
	}
}

func TestService_matchSelector_matchExpression(t *testing.T) {

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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		t.Error("should not trigger")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

func TestService_matchSelector_matchResourceSelector(t *testing.T) {

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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
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
	s.triggerExecutor = func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
		t.Error("should not match")
		return nil
	}

	err := s.match(context.Background(), e)
	assert.NoError(t, err)
}

// TestService_match_v1Scenarios is a table-driven guard that exercises the
// end-to-end Service.match() path against representative v1 TestTrigger
// CRDs — both the ResourceSelector and the legacy Spec.Selector
// (EventLabelSelector) paths. These scenarios complement the unit-level
// TestService_matchSelector_* / TestService_matchResourceSelector_* tests
// by verifying the full match pipeline after the internalTrigger refactor.
func TestService_match_v1Scenarios(t *testing.T) {
	tests := map[string]struct {
		trigger    *testtriggersv1.TestTrigger
		event      *watcherEvent
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
		"custom resource via resourceRef matches": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-ref", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					ResourceRef: &testtriggersv1.TestTriggerResourceRef{
						Group:   "kafka.strimzi.io",
						Version: "v1beta2",
						Kind:    "KafkaTopic",
					},
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "my-topic",
						Namespace: "kafka",
					},
					Event:     testtriggersv1.TestTriggerEventCreated,
					Execution: testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{
						Name: "kafka-test",
					},
				},
			},
			event: &watcherEvent{
				resource:  "kafkatopic",
				name:      "my-topic",
				Namespace: "kafka",
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
		// Spec.Selector (v1 legacy EventLabelSelector) path — matches against
		// the merged event + resource labels map, with normalizeResourceKindSelector
		// lowercasing the magic testkube.io/resource-kind key on both sides.
		"spec.Selector matches by resource-kind event label": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-sel-1", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{eventLabelKeyResourceKind: "Deployment"},
					},
					Event:        testtriggersv1.TestTriggerEventModified,
					Execution:    testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			},
			event: &watcherEvent{
				resource:  "deployment",
				name:      "api-server",
				Namespace: "production",
				eventType: "modified",
				EventLabels: map[string]string{
					eventLabelKeyResourceKind: "Deployment",
				},
			},
			shouldFire: true,
		},
		"spec.Selector does not match when resource-kind differs": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-sel-2", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{eventLabelKeyResourceKind: "Deployment"},
					},
					Event:        testtriggersv1.TestTriggerEventModified,
					Execution:    testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			},
			event: &watcherEvent{
				resource:  "pod",
				name:      "some-pod",
				Namespace: "production",
				eventType: "modified",
				EventLabels: map[string]string{
					eventLabelKeyResourceKind: "Pod",
				},
			},
			shouldFire: false,
		},
		"spec.Selector matches against resource label": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-sel-3", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"team": "platform"},
					},
					Event:        testtriggersv1.TestTriggerEventModified,
					Execution:    testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			},
			event: &watcherEvent{
				resource:       "deployment",
				name:           "api-server",
				Namespace:      "production",
				eventType:      "modified",
				resourceLabels: map[string]string{"team": "platform"},
				EventLabels:    map[string]string{eventLabelKeyResourceKind: "Deployment"},
			},
			shouldFire: true,
		},
		"spec.Selector does not match when resource label value differs": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-sel-4", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"team": "platform"},
					},
					Event:        testtriggersv1.TestTriggerEventModified,
					Execution:    testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			},
			event: &watcherEvent{
				resource:       "deployment",
				name:           "api-server",
				Namespace:      "production",
				eventType:      "modified",
				resourceLabels: map[string]string{"team": "storage"},
				EventLabels:    map[string]string{eventLabelKeyResourceKind: "Deployment"},
			},
			shouldFire: false,
		},
		// When both Spec.Selector AND ResourceSelector.Name are set, both must
		// match — guards against a refactor that accidentally ORs the two paths.
		"spec.Selector AND ResourceSelector.Name — both match": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-sel-5", Namespace: "production"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"team": "platform"},
					},
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:        testtriggersv1.TestTriggerEventModified,
					Execution:    testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			},
			event: &watcherEvent{
				resource:       "deployment",
				name:           "api-server",
				Namespace:      "production",
				eventType:      "modified",
				resourceLabels: map[string]string{"team": "platform"},
				EventLabels:    map[string]string{eventLabelKeyResourceKind: "Deployment"},
			},
			shouldFire: true,
		},
		"spec.Selector AND ResourceSelector.Name — label matches but name does not": {
			trigger: &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t-sel-6", Namespace: "production"},
				Spec: testtriggersv1.TestTriggerSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"team": "platform"},
					},
					ResourceSelector: testtriggersv1.TestTriggerSelector{
						Name:      "api-server",
						Namespace: "production",
					},
					Event:        testtriggersv1.TestTriggerEventModified,
					Execution:    testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector: testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			},
			event: &watcherEvent{
				resource:       "deployment",
				name:           "other-service",
				Namespace:      "production",
				eventType:      "modified",
				resourceLabels: map[string]string{"team": "platform"},
				EventLabels:    map[string]string{eventLabelKeyResourceKind: "Deployment"},
			},
			shouldFire: false,
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
			if tc.shouldFire {
				assert.True(t, fired, "trigger should have fired")
			} else {
				assert.False(t, fired, "trigger should not have fired")
			}
		})
	}
}

// TestService_match_V1_ExecutionFilter guards the pre-refactor behavior that
// v1 TestTriggers with Execution set to "test" or "testsuite" must be skipped
// by the matcher — we only run TestWorkflow executions. The filter was briefly
// dropped during the internalTrigger refactor; this table documents the
// expected contract so a future refactor can't silently re-break it.
func TestService_match_V1_ExecutionFilter(t *testing.T) {
	tests := map[string]struct {
		execution  testtriggersv1.TestTriggerExecution
		shouldFire bool
	}{
		"testworkflow fires":       {execution: testtriggersv1.TestTriggerExecutionTestWorkflow, shouldFire: true},
		"empty defaults to fire":   {execution: "", shouldFire: true},
		"legacy test is skipped":   {execution: testtriggersv1.TestTriggerExecutionTest, shouldFire: false},
		"legacy testsuite skipped": {execution: testtriggersv1.TestTriggerExecutionTestsuite, shouldFire: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			trigger := &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource:         testtriggersv1.TestTriggerResourceDeployment,
					ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "api-server", Namespace: "production"},
					Event:            testtriggersv1.TestTriggerEventModified,
					Execution:        tc.execution,
					TestSelector:     testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			}
			event := &watcherEvent{resource: "deployment", name: "api-server", Namespace: "production", eventType: "modified"}
			key := newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name)
			fired := false
			s := &Service{
				triggerStatus: map[statusKey]*triggerStatus{key: {trigger: convertV1ToInternal(trigger)}},
				triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
					fired = true
					return nil
				},
				logger:  log.DefaultLogger,
				metrics: metrics.NewMetrics(),
			}
			require.NoError(t, s.match(context.Background(), event))
			assert.Equal(t, tc.shouldFire, fired)
		})
	}
}

// TestService_match_V1_AllBuiltinResources_CaseFolded guards the case-folded
// comparison in matchInternalResource that lets v1 TestTriggers keep firing
// after the internalTrigger refactor. convertV1ToInternal stores the canonical
// PascalCase Kind (e.g. "Pod") in internalTrigger; the v1 watcher dispatches
// events with the lowercase enum value (e.g. "pod"). strings.EqualFold bridges
// the two. If EqualFold is replaced with == by a future refactor, all 8
// subtests here must fail.
func TestService_match_V1_AllBuiltinResources_CaseFolded(t *testing.T) {
	resources := []testtriggersv1.TestTriggerResource{
		testtriggersv1.TestTriggerResourcePod,
		testtriggersv1.TestTriggerResourceDeployment,
		testtriggersv1.TestTriggerResourceStatefulSet,
		testtriggersv1.TestTriggerResourceDaemonSet,
		testtriggersv1.TestTriggerResourceService,
		testtriggersv1.TestTriggerResourceIngress,
		testtriggersv1.TestTriggerResourceEvent,
		testtriggersv1.TestTriggerResourceConfigMap,
	}
	for _, r := range resources {
		t.Run(string(r), func(t *testing.T) {
			trigger := &testtriggersv1.TestTrigger{
				ObjectMeta: metav1.ObjectMeta{Name: "t", Namespace: "testkube"},
				Spec: testtriggersv1.TestTriggerSpec{
					Resource:         r,
					ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "target", Namespace: "ns"},
					Event:            testtriggersv1.TestTriggerEventModified,
					Execution:        testtriggersv1.TestTriggerExecutionTestWorkflow,
					TestSelector:     testtriggersv1.TestTriggerSelector{Name: "smoke-test"},
				},
			}
			event := &watcherEvent{resource: testtrigger.ResourceType(r), name: "target", Namespace: "ns", eventType: "modified"}
			key := newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name)
			fired := false
			s := &Service{
				triggerStatus: map[statusKey]*triggerStatus{key: {trigger: convertV1ToInternal(trigger)}},
				triggerExecutor: func(ctx context.Context, e *watcherEvent, trigger *internalTrigger) error {
					fired = true
					return nil
				},
				logger:  log.DefaultLogger,
				metrics: metrics.NewMetrics(),
			}
			require.NoError(t, s.match(context.Background(), event))
			assert.True(t, fired, "v1 %q trigger must match an event with resource=%q (case-folded)", r, r)
		})
	}
}
