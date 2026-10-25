package triggers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

const (
	// httpContentReconcileInterval is how often the watcher picks up added, changed and removed
	// HTTP content triggers.
	httpContentReconcileInterval = 5 * time.Second
	// maxHTTPContentBody bounds the response a poll reads; a larger one is a failed poll.
	maxHTTPContentBody = 10 << 20
	// maxHTTPContentRedirects bounds the redirects a poll follows, as the default policy does.
	maxHTTPContentRedirects = 10
)

var errHTTPTriggerTargetNotReady = errors.New("http trigger target not ready")

// httpObservation is the part of a polled response an HTTP content trigger watches.
type httpObservation struct {
	Value      string
	StatusCode int
}

// httpContentState remembers the last value a poller observed. It lives in memory only, so
// after a restart or a leader change the first poll records a baseline again.
type httpContentState struct {
	observed bool
	value    string
}

// observe records a value and reports whether it fires the trigger, with the value it replaced.
func (st *httpContentState) observe(value string, fireOnFirst bool) (fire bool, previous *string) {
	if !st.observed {
		st.observed, st.value = true, value
		return fireOnFirst, nil
	}
	if value == st.value {
		return false, nil
	}
	prev := st.value
	st.value = value
	return true, &prev
}

// restore puts back the value observe replaced, so that a change whose firing failed is
// retried on the next poll.
func (st *httpContentState) restore(previous *string) {
	if previous == nil {
		st.observed, st.value = false, ""
		return
	}
	st.value = *previous
}

type httpContentPoller struct {
	fingerprint string
	cancel      context.CancelFunc
	done        chan struct{}
}

// httpContentWatcher polls the endpoints of the HTTP content triggers the service knows about.
// Each trigger gets its own poller, restarted with a fresh baseline when its HTTP spec changes.
type httpContentWatcher struct {
	service           *Service
	reconcileInterval time.Duration
	minInterval       time.Duration
	client            *http.Client
	insecureClient    *http.Client
	pollers           map[statusKey]*httpContentPoller
}

func newHTTPContentWatcher(s *Service) *httpContentWatcher {
	insecure := http.DefaultTransport.(*http.Transport).Clone()
	insecure.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // only when the trigger sets insecureSkipVerify
	return &httpContentWatcher{
		service:           s,
		reconcileInterval: httpContentReconcileInterval,
		minInterval:       v1.MinHTTPContentInterval,
		client:            &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone(), CheckRedirect: sameOriginRedirect},
		insecureClient:    &http.Client{Transport: insecure, CheckRedirect: sameOriginRedirect},
		pollers:           map[statusKey]*httpContentPoller{},
	}
}

// sameOriginRedirect follows a redirect only within the origin the trigger was configured with.
// The request carries the trigger's headers, including values read from a Secret, and Go would
// forward custom headers to any host a redirect points at.
func sameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxHTTPContentRedirects {
		return fmt.Errorf("stopped after %d redirects", maxHTTPContentRedirects)
	}
	if first := via[0].URL; req.URL.Scheme != first.Scheme || req.URL.Host != first.Host {
		return fmt.Errorf("refusing to follow a redirect to another origin (%s://%s)", req.URL.Scheme, req.URL.Host)
	}
	return nil
}

// runHTTPContentWatcher polls the HTTP content triggers until ctx is done. It runs as a leader
// task, so only one agent polls a trigger.
func (s *Service) runHTTPContentWatcher(ctx context.Context) {
	newHTTPContentWatcher(s).run(ctx)
}

func (w *httpContentWatcher) run(ctx context.Context) {
	defer w.stopAll()
	ticker := time.NewTicker(w.reconcileInterval)
	defer ticker.Stop()
	for {
		w.reconcile(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *httpContentWatcher) reconcile(ctx context.Context) {
	targets := w.service.httpContentTargets()
	for key, poller := range w.pollers {
		trigger, ok := targets[key]
		if !ok || httpSpecFingerprint(trigger) != poller.fingerprint {
			poller.cancel()
			<-poller.done
			delete(w.pollers, key)
		}
	}
	for key, trigger := range targets {
		if _, ok := w.pollers[key]; ok {
			continue
		}
		pollCtx, cancel := context.WithCancel(ctx)
		poller := &httpContentPoller{fingerprint: httpSpecFingerprint(trigger), cancel: cancel, done: make(chan struct{})}
		w.pollers[key] = poller
		go func(key statusKey, namespace string, spec v1.TestTriggerContentHTTPSpec) {
			defer close(poller.done)
			w.poll(pollCtx, key, namespace, spec)
		}(key, trigger.Namespace, *trigger.ContentSelector.HTTP)
	}
}

func (w *httpContentWatcher) stopAll() {
	for key, poller := range w.pollers {
		poller.cancel()
		<-poller.done
		delete(w.pollers, key)
	}
}

func (w *httpContentWatcher) poll(ctx context.Context, key statusKey, namespace string, spec v1.TestTriggerContentHTTPSpec) {
	interval := v1.DefaultHTTPContentInterval
	if spec.Interval != nil {
		interval = max(spec.Interval.Duration, w.minInterval)
	}
	logger := w.service.logger.With("trigger", string(key), "url", spec.URL)

	var state httpContentState
	failing := false
	for {
		observation, err := w.fetch(ctx, namespace, spec)
		switch {
		case ctx.Err() != nil:
			return
		case err != nil:
			// a failed poll is never a change; log the transition rather than every attempt
			if !failing {
				logger.Warnw("trigger service: http content poll failed", "error", err)
			}
			failing = true
		default:
			if failing {
				logger.Infow("trigger service: http content poll recovered")
			}
			failing = false
			if fire, previous := state.observe(observation.Value, spec.FireOnFirstObservation); fire {
				if err := w.service.matchHTTPTrigger(withScheduleErrors(ctx), key, spec.URL, observation, previous); err != nil {
					logger.Errorw("trigger service: firing http content trigger failed, retrying on the next poll", "error", err)
					state.restore(previous)
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// fetch polls the endpoint once and extracts the watched value.
func (w *httpContentWatcher) fetch(ctx context.Context, namespace string, spec v1.TestTriggerContentHTTPSpec) (httpObservation, error) {
	timeout := v1.DefaultHTTPContentTimeout
	if spec.Timeout != nil && spec.Timeout.Duration > 0 {
		timeout = spec.Timeout.Duration
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if spec.Body != "" {
		body = strings.NewReader(spec.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, spec.URL, body)
	if err != nil {
		return httpObservation{}, fmt.Errorf("building request: %w", err)
	}
	for name, value := range spec.Headers {
		req.Header.Set(name, value)
	}
	for _, header := range spec.HeadersFrom {
		value, ok, err := w.service.resolveHeaderValue(ctx, namespace, header.ValueFrom)
		if err != nil {
			return httpObservation{}, fmt.Errorf("resolving header %s: %w", header.Name, err)
		}
		if ok {
			req.Header.Set(header.Name, value)
		}
	}

	client := w.client
	if spec.InsecureSkipVerify {
		client = w.insecureClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return httpObservation{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPContentBody+1))
	if err != nil {
		return httpObservation{}, fmt.Errorf("reading response: %w", err)
	}
	if len(data) > maxHTTPContentBody {
		return httpObservation{}, fmt.Errorf("response is larger than %d bytes", maxHTTPContentBody)
	}
	if !expectedHTTPStatus(spec.ExpectedStatus, resp.StatusCode) {
		return httpObservation{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	value, err := selectHTTPContent(spec.Select, resp.Header, data)
	if err != nil {
		return httpObservation{}, err
	}
	return httpObservation{Value: value, StatusCode: resp.StatusCode}, nil
}

func expectedHTTPStatus(expected []int32, status int) bool {
	if len(expected) == 0 {
		return status == http.StatusOK
	}
	return slices.Contains(expected, int32(status))
}

// selectHTTPContent extracts the watched value of a response: a JSON field, a header, or, when
// nothing is selected, the SHA-256 of the whole body.
func selectHTTPContent(sel *v1.TestTriggerContentHTTPSelect, header http.Header, body []byte) (string, error) {
	switch {
	case sel != nil && sel.Header != "":
		values := header.Values(sel.Header)
		if len(values) == 0 {
			return "", fmt.Errorf("response has no %s header", sel.Header)
		}
		return strings.Join(values, ", "), nil
	case sel != nil && sel.JSONPath != "":
		return selectJSONPath(sel.JSONPath, body)
	default:
		sum := sha256.Sum256(body)
		return "sha256:" + hex.EncodeToString(sum[:]), nil
	}
}

// selectJSONPath resolves a dot-path such as .data.version in a JSON document. A scalar is
// returned as text, and an object or array as its JSON encoding, with sorted object keys so
// that a reordering is not a change.
func selectJSONPath(path string, body []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var current any
	if err := decoder.Decode(&current); err != nil {
		return "", fmt.Errorf("response is not JSON: %w", err)
	}
	for _, segment := range strings.Split(strings.TrimPrefix(path, "."), ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: %q is not inside an object", path, segment)
		}
		if current, ok = object[segment]; !ok {
			return "", fmt.Errorf("%s: response has no %q field", path, segment)
		}
	}
	switch value := current.(type) {
	case string:
		return value, nil
	case json.Number:
		return value.String(), nil
	case bool:
		return strconv.FormatBool(value), nil
	case nil:
		return "null", nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("%s: %w", path, err)
		}
		return string(encoded), nil
	}
}

// resolveHeaderValue reads a header value from a Secret or ConfigMap in the trigger's namespace.
// An optional reference to a missing object or key sends no header.
func (s *Service) resolveHeaderValue(ctx context.Context, namespace string, source corev1.EnvVarSource) (string, bool, error) {
	if s.clientset == nil {
		return "", false, errors.New("no Kubernetes client to read the header value with")
	}
	switch {
	case source.SecretKeyRef != nil:
		ref := source.SecretKeyRef
		secret, err := s.clientset.CoreV1().Secrets(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return optionalRefValue(ref.Optional, fmt.Errorf("reading secret %s: %w", ref.Name, err))
		}
		value, ok := secret.Data[ref.Key]
		if !ok {
			return optionalRefValue(ref.Optional, fmt.Errorf("secret %s has no key %s", ref.Name, ref.Key))
		}
		return string(value), true, nil
	case source.ConfigMapKeyRef != nil:
		ref := source.ConfigMapKeyRef
		configMap, err := s.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return optionalRefValue(ref.Optional, fmt.Errorf("reading config map %s: %w", ref.Name, err))
		}
		value, ok := configMap.Data[ref.Key]
		if !ok {
			return optionalRefValue(ref.Optional, fmt.Errorf("config map %s has no key %s", ref.Name, ref.Key))
		}
		return value, true, nil
	default:
		return "", false, errors.New("valueFrom requires secretKeyRef or configMapKeyRef")
	}
}

func optionalRefValue(optional *bool, err error) (string, bool, error) {
	if optional != nil && *optional {
		return "", false, nil
	}
	return "", false, err
}

// httpContentTargets returns the enabled HTTP content triggers the service knows about.
func (s *Service) httpContentTargets() map[statusKey]*internalTrigger {
	s.triggerStatusMu.RLock()
	defer s.triggerStatusMu.RUnlock()
	targets := map[statusKey]*internalTrigger{}
	for key, status := range s.triggerStatus {
		if status == nil || !isHTTPContentTrigger(status.trigger) {
			continue
		}
		targets[key] = status.trigger
	}
	return targets
}

func isHTTPContentTrigger(trigger *internalTrigger) bool {
	return trigger != nil &&
		!trigger.Disabled &&
		strings.EqualFold(trigger.ResourceKind, string(testtrigger.ResourceContent)) &&
		trigger.ContentSelector != nil &&
		trigger.ContentSelector.HTTP != nil
}

func httpSpecFingerprint(trigger *internalTrigger) string {
	encoded, _ := json.Marshal(trigger.ContentSelector.HTTP)
	return string(encoded)
}

// matchHTTPTrigger fires an HTTP content trigger whose watched value changed. The event's object
// is the new observation (value, statusCode, url) and its old object the previous value, so that
// match[] and the run's config expressions can read both (resource.value, oldResource.value).
func (s *Service) matchHTTPTrigger(ctx context.Context, key statusKey, url string, observation httpObservation, previous *string) error {
	s.triggerStatusMu.RLock()
	status := s.triggerStatus[key]
	s.triggerStatusMu.RUnlock()
	if status == nil || !isHTTPContentTrigger(status.trigger) {
		return fmt.Errorf("%w: %s", errHTTPTriggerTargetNotReady, key)
	}
	trigger := status.trigger
	if trigger.Execution != "" && trigger.Execution != ExecutionTestWorkflow {
		return nil
	}

	event := s.newWatcherEvent(
		testtrigger.EventHTTPChanged,
		&metav1.ObjectMeta{Name: trigger.Name, Namespace: trigger.Namespace},
		map[string]any{"value": observation.Value, "statusCode": observation.StatusCode, "url": url},
		testtrigger.ResourceType(testtrigger.ResourceContent),
	)
	if previous != nil {
		event.OldObject = map[string]any{"value": *previous}
	}

	if !matchInternalResource(trigger, event, s.logger) || !matchEventOrCause(trigger.Events, event) {
		return nil
	}
	if !matchFieldSelector(trigger.FieldConditions, event.Object, event.OldObject) {
		return nil
	}
	return s.fireContentTrigger(ctx, event, trigger, status)
}
