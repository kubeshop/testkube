package triggers

import (
	"context"
	"maps"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	thttp "github.com/kubeshop/testkube/pkg/http"
)

const (
	defaultScheme = "http"
	defaultPath   = "/"
)

var (
	ErrConditionTimeout = errors.New("timed-out waiting for trigger conditions")
	ErrProbeTimeout     = errors.New("timed-out waiting for trigger probes")
)

func (s *Service) match(ctx context.Context, e *watcherEvent) error {
	for _, entry := range s.snapshotStatuses() {
		status := entry.status
		t := entry.trigger
		if t.Disabled {
			continue
		}
		if t.Execution != "" && t.Execution != ExecutionTestWorkflow {
			continue
		}

		// Resource matching
		if !matchInternalResource(t, e, s.logger) {
			continue
		}

		// Event matching (v1 triggers also match on deployment-specific causes)
		if !matchEventOrCause(t.Event, e) {
			continue
		}

		// Field matching (v2 only, empty for v1 so always passes)
		if !matchFieldSelector(t.FieldConditions, e.Object, e.OldObject) {
			continue
		}

		// Condition matching
		if t.Conditions != nil && len(t.Conditions.Items) > 0 && e.conditionsGetter != nil {
			matched, err := s.matchInternalConditions(ctx, e, t, s.logger)
			if err != nil {
				return err
			}
			if !matched {
				continue
			}
		}

		// Probe matching
		if t.Probes != nil && len(t.Probes.Items) > 0 {
			matched, err := s.matchInternalProbes(ctx, e, t, s.logger)
			if err != nil {
				return err
			}
			if !matched {
				continue
			}
		}

		// Concurrency policy
		if t.ConcurrencyPolicy == concurrencyPolicyForbid {
			if status.hasActiveTests() {
				s.logger.Infof(
					"trigger service: matcher component: skipping trigger execution for trigger %s/%s by event %s on resource %s because it is currently running tests",
					t.Namespace, t.Name, e.eventType, e.resource,
				)
				return nil
			}
		}

		if t.ConcurrencyPolicy == concurrencyPolicyReplace {
			if status.hasActiveTests() {
				s.logger.Infof(
					"trigger service: matcher component: aborting trigger execution for trigger %s/%s by event %s on resource %s because it is currently running tests",
					t.Namespace, t.Name, e.eventType, e.resource,
				)
				s.abortExecutions(ctx, t.Name, status)
			}
		}

		s.logger.Infof("trigger service: matcher component: event %s matches trigger %s/%s for resource %s", e.eventType, t.Namespace, t.Name, e.resource)

		var causes []string
		for _, cause := range e.causes {
			causes = append(causes, string(cause))
		}

		s.metrics.IncTestTriggerEventCount(t.Name, string(e.resource), string(e.eventType), causes)
		if err := s.triggerExecutor(ctx, e, t); err != nil {
			return err
		}
	}
	return nil
}

// matchInternalResource checks if the event's resource matches the trigger's resource criteria.
// For v1 triggers with EventLabelSelector, uses the legacy label matching path.
// Otherwise matches by resource kind + name/namespace/selector.
func matchInternalResource(t *internalTrigger, e *watcherEvent, logger *zap.SugaredLogger) bool {
	hasEventLabelSelector := t.EventLabelSelector != nil &&
		(len(t.EventLabelSelector.MatchLabels) > 0 || len(t.EventLabelSelector.MatchExpressions) > 0)
	hasSelectorOrName := t.Selector != nil || t.ResourceName != "" || t.ResourceNamespace != ""

	// v1 legacy path: EventLabelSelector matches against merged event + resource labels
	if hasEventLabelSelector {
		selectorMatched := matchSelector(t.EventLabelSelector, e, logger)
		if hasSelectorOrName {
			// Both specified: both must match
			return selectorMatched && matchResourceCriteria(t, e, logger)
		}
		return selectorMatched
	}

	// No EventLabelSelector: check resource kind first
	if t.ResourceKind != "" && !strings.EqualFold(t.ResourceKind, string(e.resource)) {
		return false
	}

	return matchResourceCriteria(t, e, logger)
}

// matchResourceCriteria checks name, namespace, and selector criteria.
func matchResourceCriteria(t *internalTrigger, e *watcherEvent, logger *zap.SugaredLogger) bool {
	// Check exact name match
	if t.ResourceName != "" && t.ResourceName != e.name {
		return false
	}

	// Check exact namespace match
	if t.ResourceNamespace != "" && t.ResourceNamespace != e.Namespace {
		return false
	}

	// If no name/namespace specified, default to trigger's own namespace
	if t.ResourceName == "" && t.ResourceNamespace == "" && t.Namespace != e.Namespace {
		if t.Selector == nil {
			return false
		}
	}

	// Check advanced selector (regex, labels)
	if t.Selector != nil {
		return matchInternalSelector(t.Selector, t.Namespace, e, logger)
	}

	return true
}

func matchInternalSelector(sel *internalTriggerSelector, triggerNamespace string, e *watcherEvent, logger *zap.SugaredLogger) bool {
	hasNameCriteria := sel.Name != "" || sel.NameRegex != ""
	hasNamespaceCriteria := sel.Namespace != "" || sel.NamespaceRegex != ""

	// Label selector
	if sel.LabelSelector != nil {
		k8sSelector, err := v1.LabelSelectorAsSelector(sel.LabelSelector)
		if err != nil {
			logger.Errorf("error creating k8s selector from label selector: %v", err)
			return false
		}
		resourceLabelSet := labels.Set(e.resourceLabels)
		if len(e.resourceLabels) > 0 {
			if _, err := resourceLabelSet.AsValidatedSelector(); err != nil {
				logger.Errorf("%s %s/%s labels are invalid: %v", e.resource, e.Namespace, e.name, err)
				return false
			}
		}
		if !k8sSelector.Matches(resourceLabelSet) {
			return false
		}
		// Labels matched. If no name/namespace criteria, labels are sufficient.
		if !hasNameCriteria && !hasNamespaceCriteria {
			return true
		}
		// Otherwise fall through to AND with name/namespace criteria.
	}

	// Default: if no name/regex criteria specified, match any name
	nameMatched := !hasNameCriteria
	namespaceMatched := false

	// Name matching
	if sel.Name != "" {
		nameMatched = sel.Name == e.name
	}
	if sel.NameRegex != "" {
		re, err := regexp.Compile(sel.NameRegex)
		if err != nil {
			logger.Errorf("error compiling name regex %q: %v", sel.NameRegex, err)
			return false
		}
		nameMatched = re.MatchString(e.name)
	}

	// Namespace matching
	if sel.Namespace != "" {
		namespaceMatched = sel.Namespace == e.Namespace
	}
	if sel.NamespaceRegex != "" {
		re, err := regexp.Compile(sel.NamespaceRegex)
		if err != nil {
			logger.Errorf("error compiling namespace regex %q: %v", sel.NamespaceRegex, err)
			return false
		}
		namespaceMatched = re.MatchString(e.Namespace)
	}

	// Default to trigger's namespace when none specified
	if sel.Namespace == "" && sel.NamespaceRegex == "" {
		namespaceMatched = triggerNamespace == e.Namespace
	}

	return nameMatched && namespaceMatched
}

func matchEventOrCause(targetEvent string, event *watcherEvent) bool {
	if targetEvent == string(event.eventType) {
		return true
	}
	for _, c := range event.causes {
		if targetEvent == string(c) {
			return true
		}
	}
	return false
}

// matchSelector is kept for v1 backward compatibility with the legacy Selector field.
// It matches against merged resource + event labels.
func matchSelector(selector *v1.LabelSelector, event *watcherEvent, logger *zap.SugaredLogger) bool {
	if selector == nil {
		return true
	}

	normalizedSelector := selector.DeepCopy()
	mergedLabels := make(map[string]string)
	maps.Copy(mergedLabels, event.resourceLabels)
	maps.Copy(mergedLabels, event.EventLabels)
	normalizeResourceKindSelector(normalizedSelector, mergedLabels)

	k8sSelector, err := v1.LabelSelectorAsSelector(normalizedSelector)
	if err != nil {
		logger.Errorf("error creating k8s selector from label selector: %v", err)
		return false
	}
	labelsSet := labels.Set(mergedLabels)
	_, err = labelsSet.AsValidatedSelector()
	if err != nil {
		logger.Errorf("%s %s/%s labels are invalid: %v", event.resource, event.Namespace, event.name, err)
		return false
	}
	return k8sSelector.Matches(labelsSet)
}

func normalizeResourceKindSelector(selector *v1.LabelSelector, labels map[string]string) {
	if selector == nil {
		return
	}

	if selector.MatchLabels != nil {
		if value, ok := selector.MatchLabels[eventLabelKeyResourceKind]; ok {
			selector.MatchLabels[eventLabelKeyResourceKind] = strings.ToLower(value)
		}
	}

	for i := range selector.MatchExpressions {
		if selector.MatchExpressions[i].Key != eventLabelKeyResourceKind {
			continue
		}
		for j := range selector.MatchExpressions[i].Values {
			selector.MatchExpressions[i].Values[j] = strings.ToLower(selector.MatchExpressions[i].Values[j])
		}
	}

	if value, ok := labels[eventLabelKeyResourceKind]; ok && value != "" {
		labels[eventLabelKeyResourceKind] = strings.ToLower(value)
	}
}

func (s *Service) matchInternalConditions(ctx context.Context, e *watcherEvent, t *internalTrigger, logger *zap.SugaredLogger) (bool, error) {
	timeout := s.defaultConditionsCheckTimeout
	if t.Conditions.Timeout > 0 {
		timeout = time.Duration(t.Conditions.Timeout) * time.Second
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

outer:
	for {
		select {
		case <-timeoutCtx.Done():
			logger.Errorf(
				"trigger service: matcher component: error waiting for conditions to match for trigger %s/%s by event %s on resource %s %s/%s"+
					" because context got canceled by timeout or exit signal",
				t.Namespace, t.Name, e.eventType, e.resource, e.Namespace, e.name,
			)
			return false, errors.WithStack(ErrConditionTimeout)
		default:
			logger.Debugf(
				"trigger service: matcher component: running conditions check iteration for %s %s/%s",
				e.resource, e.Namespace, e.name,
			)
			conditions, err := e.conditionsGetter()
			if err != nil {
				logger.Errorf(
					"trigger service: matcher component: error getting conditions for %s %s/%s because of %v",
					e.resource, e.Namespace, e.name, err,
				)
				return false, err
			}

			conditionMap := make(map[string]testtriggersv1.TestTriggerCondition, len(conditions))
			for _, condition := range conditions {
				conditionMap[condition.Type_] = condition
			}

			matched := true
			for _, triggerCondition := range t.Conditions.Items {
				resourceCondition, ok := conditionMap[triggerCondition.Type]
				if !ok {
					matched = false
					break
				}
				if resourceCondition.Status == nil || triggerCondition.Status == nil {
					matched = false
					break
				}
				if string(*resourceCondition.Status) != *triggerCondition.Status {
					matched = false
					break
				}
				if triggerCondition.Reason != "" && triggerCondition.Reason != resourceCondition.Reason {
					matched = false
					break
				}
				if triggerCondition.TTL != 0 && triggerCondition.TTL < resourceCondition.Ttl {
					matched = false
					break
				}
			}

			if matched {
				break outer
			}

			delay := s.defaultConditionsCheckBackoff
			if t.Conditions.Delay > 0 {
				delay = time.Duration(t.Conditions.Delay) * time.Second
			}
			time.Sleep(delay)
		}
	}

	return true, nil
}

func checkProbes(ctx context.Context, httpClient thttp.HttpClient, probes []internalProbe, logger *zap.SugaredLogger) bool {
	var wg sync.WaitGroup
	ch := make(chan bool, len(probes))
	defer close(ch)

	wg.Add(len(probes))
	for i := range probes {
		go func(probe internalProbe) {
			defer wg.Done()

			host := probe.Host
			if probe.Port != 0 {
				host = net.JoinHostPort(probe.Host, strconv.Itoa(int(probe.Port)))
			} else if ip := net.ParseIP(probe.Host); ip != nil && ip.To4() == nil {
				host = "[" + probe.Host + "]"
			}

			if host == "" {
				ch <- false
				return
			}

			uri := url.URL{
				Scheme: probe.Scheme,
				Host:   host,
				Path:   probe.Path,
			}
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
			if err != nil {
				logger.Debugw("probe request creating error", "error", err)
				ch <- false
				return
			}

			for key, value := range probe.Headers {
				request.Header.Set(key, value)
			}

			resp, err := httpClient.Do(request)
			if err != nil {
				logger.Debugw("probe send error", "error", err)
				ch <- false
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				logger.Debugw("probe response with bad status code", "status", resp.StatusCode)
				ch <- false
				return
			}

			ch <- true
		}(probes[i])
	}

	wg.Wait()

	for i := 0; i < len(probes); i++ {
		result := <-ch
		if !result {
			return false
		}
	}

	return true
}

func (s *Service) matchInternalProbes(ctx context.Context, e *watcherEvent, t *internalTrigger, logger *zap.SugaredLogger) (bool, error) {
	timeout := s.defaultProbesCheckTimeout
	if t.Probes.Timeout > 0 {
		timeout = time.Duration(t.Probes.Timeout) * time.Second
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	host := ""
	if e.addressGetter != nil {
		var err error
		host, err = e.addressGetter(timeoutCtx, s.defaultProbesCheckBackoff)
		if err != nil {
			logger.Errorf(
				"trigger service: matcher component: error getting address for %s %s/%s because of %v",
				e.resource, e.Namespace, e.name, err,
			)
			return false, err
		}
	}

	// Fill defaults for probes
	probes := make([]internalProbe, len(t.Probes.Items))
	copy(probes, t.Probes.Items)
	for i := range probes {
		if probes[i].Scheme == "" {
			probes[i].Scheme = defaultScheme
		}
		if probes[i].Host == "" {
			probes[i].Host = host
		}
		if probes[i].Path == "" {
			probes[i].Path = defaultPath
		}
	}

outer:
	for {
		select {
		case <-timeoutCtx.Done():
			logger.Errorf(
				"trigger service: matcher component: error waiting for probes to match for trigger %s/%s by event %s on resource %s %s/%s"+
					" because context got canceled by timeout or exit signal",
				t.Namespace, t.Name, e.eventType, e.resource, e.Namespace, e.name,
			)
			return false, errors.WithStack(ErrProbeTimeout)
		default:
			logger.Debugf(
				"trigger service: matcher component: running probes check iteration for %s %s/%s",
				e.resource, e.Namespace, e.name,
			)

			matched := checkProbes(timeoutCtx, s.httpClient, probes, logger)
			if matched {
				break outer
			}

			delay := s.defaultProbesCheckBackoff
			if t.Probes.Delay > 0 {
				delay = time.Duration(t.Probes.Delay) * time.Second
			}
			time.Sleep(delay)
		}
	}

	return true, nil
}
