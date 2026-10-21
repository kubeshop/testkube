package triggers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	v1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestSelectHTTPContent(t *testing.T) {
	body := []byte(`{"data": {"version": "2.1.0", "build": 42, "stable": true, "missing": null, "meta": {"b": 1, "a": [1, 2]}}}`)
	header := http.Header{"Etag": []string{`"abc"`}}

	tests := map[string]struct {
		sel     *v1.TestTriggerContentHTTPSelect
		want    string
		wantErr bool
	}{
		"string field":         {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.version"}, want: "2.1.0"},
		"number field":         {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.build"}, want: "42"},
		"bool field":           {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.stable"}, want: "true"},
		"null field":           {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.missing"}, want: "null"},
		"object, keys sorted":  {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.meta"}, want: `{"a":[1,2],"b":1}`},
		"absent field":         {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.nope"}, wantErr: true},
		"field of a scalar":    {sel: &v1.TestTriggerContentHTTPSelect{JSONPath: ".data.version.major"}, wantErr: true},
		"header":               {sel: &v1.TestTriggerContentHTTPSelect{Header: "ETag"}, want: `"abc"`},
		"absent header":        {sel: &v1.TestTriggerContentHTTPSelect{Header: "Last-Modified"}, wantErr: true},
		"whole body by digest": {want: "sha256:"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := selectHTTPContent(tt.sel, header, body)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.sel == nil {
				assert.Len(t, got, len("sha256:")+64)
				assert.Contains(t, got, tt.want)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("a body that is not JSON", func(t *testing.T) {
		_, err := selectHTTPContent(&v1.TestTriggerContentHTTPSelect{JSONPath: ".version"}, nil, []byte("<html>"))
		assert.Error(t, err)
	})
}

func TestHTTPContentState(t *testing.T) {
	t.Run("the first observation is a baseline", func(t *testing.T) {
		var st httpContentState
		fire, previous := st.observe("1", false)
		assert.False(t, fire)
		assert.Nil(t, previous)

		fire, _ = st.observe("1", false)
		assert.False(t, fire, "an unchanged value does not fire")

		fire, previous = st.observe("2", false)
		assert.True(t, fire)
		assert.Equal(t, "1", *previous)
	})

	t.Run("fireOnFirstObservation fires the baseline", func(t *testing.T) {
		var st httpContentState
		fire, previous := st.observe("1", true)
		assert.True(t, fire)
		assert.Nil(t, previous)
	})

	t.Run("a failed firing is retried", func(t *testing.T) {
		var st httpContentState
		st.observe("1", false)
		_, previous := st.observe("2", false)

		st.restore(previous)

		fire, previous := st.observe("2", false)
		assert.True(t, fire)
		assert.Equal(t, "1", *previous)
	})
}

func TestHTTPContentWatcher_Fetch(t *testing.T) {
	var got *http.Request
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		w.Header().Set("ETag", `"v7"`)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	s := &Service{
		clientset: fake.NewClientset(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "api-token", Namespace: "testkube"},
			Data:       map[string][]byte{"bearer": []byte("Bearer s3cret")},
		}),
		logger: log.DefaultLogger,
	}
	w := newHTTPContentWatcher(s)
	spec := v1.TestTriggerContentHTTPSpec{
		URL:     server.URL,
		Method:  http.MethodPost,
		Headers: map[string]string{"Accept": "application/json"},
		HeadersFrom: []v1.TestTriggerContentHTTPHeaderSource{{
			Name: "Authorization",
			ValueFrom: corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api-token"},
				Key:                  "bearer",
			}},
		}},
		Body:           `{"query": "{ version }"}`,
		ExpectedStatus: []int32{http.StatusAccepted},
		Select:         &v1.TestTriggerContentHTTPSelect{Header: "ETag"},
	}

	observation, err := w.fetch(context.Background(), "testkube", spec)

	require.NoError(t, err)
	assert.Equal(t, httpObservation{Value: `"v7"`, StatusCode: http.StatusAccepted}, observation)
	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "application/json", got.Header.Get("Accept"))
	assert.Equal(t, "Bearer s3cret", got.Header.Get("Authorization"))
	assert.Equal(t, `{"query": "{ version }"}`, gotBody)

	t.Run("an unexpected status is a failed poll", func(t *testing.T) {
		spec := spec
		spec.ExpectedStatus = nil

		_, err := w.fetch(context.Background(), "testkube", spec)

		assert.ErrorContains(t, err, "unexpected status 202")
	})

	t.Run("a missing secret is a failed poll unless optional", func(t *testing.T) {
		spec := spec
		ref := &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "absent"}, Key: "bearer"}
		spec.HeadersFrom = []v1.TestTriggerContentHTTPHeaderSource{{Name: "Authorization", ValueFrom: corev1.EnvVarSource{SecretKeyRef: ref}}}

		_, err := w.fetch(context.Background(), "testkube", spec)
		assert.Error(t, err)

		optional := true
		ref.Optional = &optional
		_, err = w.fetch(context.Background(), "testkube", spec)
		assert.NoError(t, err)
		assert.Empty(t, got.Header.Get("Authorization"))
	})
}

// versionServer serves {"version": <current>} and lets the test change the version.
type versionServer struct {
	*httptest.Server
	version atomic.Value
}

func newVersionServer(version string) *versionServer {
	s := &versionServer{}
	s.version.Store(version)
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version": "` + s.version.Load().(string) + `"}`))
	}))
	return s
}

type firedEvents struct {
	mu     sync.Mutex
	events []*watcherEvent
}

func (f *firedEvents) executor(_ context.Context, e *watcherEvent, _ *internalTrigger) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
	return nil
}

func (f *firedEvents) snapshot() []*watcherEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*watcherEvent(nil), f.events...)
}

func httpContentTrigger(url string, match ...workflowtriggersv1.WorkflowTriggerFieldCondition) *v1.TestTrigger {
	return &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "on-version", Namespace: "testkube"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventHTTPChanged,
			Match:    match,
			ContentSelector: &v1.TestTriggerContentSelector{HTTP: &v1.TestTriggerContentHTTPSpec{
				URL:      url,
				Interval: &metav1.Duration{Duration: 10 * time.Millisecond},
				Select:   &v1.TestTriggerContentHTTPSelect{JSONPath: ".version"},
			}},
			Execution: v1.TestTriggerExecutionTestWorkflow,
		},
	}
}

func runHTTPWatcher(t *testing.T, trigger *v1.TestTrigger, fired *firedEvents) *Service {
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: fired.executor,
		logger:          log.DefaultLogger,
		metrics:         metrics.NewMetrics(),
	}
	w := newHTTPContentWatcher(s)
	w.reconcileInterval = 10 * time.Millisecond
	w.minInterval = 0

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return s
}

func TestHTTPContentWatcher_FiresWhenTheValueChanges(t *testing.T) {
	server := newVersionServer("1.0.0")
	defer server.Close()
	fired := &firedEvents{}
	runHTTPWatcher(t, httpContentTrigger(server.URL), fired)

	// the first poll is only a baseline
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, fired.snapshot())

	server.version.Store("1.1.0")
	require.Eventually(t, func() bool { return len(fired.snapshot()) == 1 }, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	events := fired.snapshot()
	require.Len(t, events, 1, "an unchanged value fires nothing more")
	assert.Equal(t, "http-changed", string(events[0].eventType))
	assert.Equal(t, "1.1.0", events[0].Object.(map[string]any)["value"])
	assert.Equal(t, http.StatusOK, events[0].Object.(map[string]any)["statusCode"])
	assert.Equal(t, "1.0.0", events[0].OldObject.(map[string]any)["value"])
}

func TestHTTPContentWatcher_FiresOnlyWhenTheMatchConditionsPass(t *testing.T) {
	server := newVersionServer("1.0.0")
	defer server.Close()
	fired := &firedEvents{}
	runHTTPWatcher(t, httpContentTrigger(server.URL, workflowtriggersv1.WorkflowTriggerFieldCondition{
		Path:     ".value",
		Operator: workflowtriggersv1.FieldOperatorEquals,
		Value:    "2.0.0",
	}), fired)
	time.Sleep(100 * time.Millisecond)

	server.version.Store("1.1.0")
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, fired.snapshot(), "1.1.0 does not pass the condition")

	server.version.Store("2.0.0")
	require.Eventually(t, func() bool { return len(fired.snapshot()) == 1 }, 5*time.Second, 10*time.Millisecond)
}

func TestHTTPContentWatcher_StopsPollingARemovedTrigger(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"version": "1"}`))
	}))
	defer server.Close()
	trigger := httpContentTrigger(server.URL)
	s := runHTTPWatcher(t, trigger, &firedEvents{})
	require.Eventually(t, func() bool { return requests.Load() > 0 }, 5*time.Second, 10*time.Millisecond)

	s.triggerStatusMu.Lock()
	delete(s.triggerStatus, newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name))
	s.triggerStatusMu.Unlock()
	time.Sleep(100 * time.Millisecond)
	after := requests.Load()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, after, requests.Load())
}

func TestHTTPContentTargets_SkipsOtherTriggers(t *testing.T) {
	httpTrigger := httpContentTrigger("https://api.example.com")
	disabled := httpContentTrigger("https://api.example.com")
	disabled.Name, disabled.Spec.Disabled = "disabled", true
	git := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "git", Namespace: "testkube"},
		Spec: v1.TestTriggerSpec{
			Resource:        v1.TestTriggerResourceContent,
			Event:           v1.TestTriggerEventGitPush,
			ContentSelector: &v1.TestTriggerContentSelector{Git: &v1.TestTriggerContentGitSpec{Uri: "https://github.com/kubeshop/testkube"}},
		},
	}
	s := &Service{triggerStatus: map[statusKey]*triggerStatus{}}
	for _, trigger := range []*v1.TestTrigger{httpTrigger, disabled, git} {
		s.triggerStatus[newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name)] = &triggerStatus{trigger: convertV1ToInternal(trigger)}
	}

	targets := s.httpContentTargets()

	assert.Len(t, targets, 1)
	assert.Contains(t, targets, newStatusKey(triggerSourceV1, "testkube", "on-version"))
}
