package triggers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
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
	for _, status := range s.triggerStatus {
		t := status.testTrigger
		if t.Spec.Resource != testtriggersv1.TestTriggerResource(e.resource) {
			continue
		}
		if !matchEventOrCause(string(t.Spec.Event), e) {
			continue
		}
		if !matchSelector(&t.Spec.ResourceSelector, t.Namespace, e, s.logger) {
			continue
		}
		hasConditions := t.Spec.ConditionSpec != nil && len(t.Spec.ConditionSpec.Conditions) != 0
		if hasConditions && e.conditionsGetter != nil {
			matched, err := s.matchConditions(ctx, e, t, s.logger)
			if err != nil {
				return err
			}

			if !matched {
				continue
			}
		}

		hasProbes := t.Spec.ProbeSpec != nil && len(t.Spec.ProbeSpec.Probes) != 0
		if hasProbes {
			matched, err := s.matchProbes(ctx, e, t, s.logger)
			if err != nil {
				return err
			}

			if !matched {
				continue
			}
		}

		status := s.getStatusForTrigger(t)
		if t.Spec.ConcurrencyPolicy == testtriggersv1.TestTriggerConcurrencyPolicyForbid {
			if status.hasActiveTests() {
				s.logger.Infof(
					"trigger service: matcher component: skipping trigger execution for trigger %s/%s by event %s on resource %s because it is currently running tests",
					t.Namespace, t.Name, e.eventType, e.resource,
				)
				return nil
			}
		}

		if t.Spec.ConcurrencyPolicy == testtriggersv1.TestTriggerConcurrencyPolicyReplace {
			if status.hasActiveTests() {
				s.logger.Infof(
					"trigger service: matcher component: aborting trigger execution for trigger %s/%s by event %s on resource %s because it is currently running tests",
					t.Namespace, t.Name, e.eventType, e.resource,
				)
				s.abortExecutions(ctx, t.Name, status)
			}
		}

		s.logger.Infof("trigger service: matcher component: event %s matches trigger %s/%s for resource %s", e.eventType, t.Namespace, t.Name, e.resource)
		s.logger.Infof("trigger service: matcher component: triggering %s action for %s execution", t.Spec.Action, t.Spec.Execution)
		if err := s.triggerExecutor(ctx, e, t); err != nil {
			return err
		}
	}
	return nil
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

func matchSelector(selector *testtriggersv1.TestTriggerSelector, namespace string, event *watcherEvent, logger *zap.SugaredLogger) bool {
	if selector.Name != "" {
		isSameName := selector.Name == event.name
		isSameNamespace := selector.Namespace == event.namespace
		isSameTestTriggerNamespace := selector.Namespace == "" && namespace == event.namespace
		return isSameName && (isSameNamespace || isSameTestTriggerNamespace)
	}
	if selector.NameRegex != "" {
		re, err := regexp.Compile(selector.NameRegex)
		if err != nil {
			logger.Errorf("error compiling %v name regex: %v", selector.NameRegex, err)
			return false
		}

		isSameName := re.MatchString(event.name)
		isSameNamespace := selector.Namespace == event.namespace
		isSameTestTriggerNamespace := selector.Namespace == "" && namespace == event.namespace
		return isSameName && (isSameNamespace || isSameTestTriggerNamespace)
	}
	if selector.LabelSelector != nil && len(event.labels) > 0 {
		k8sSelector, err := v1.LabelSelectorAsSelector(selector.LabelSelector)
		if err != nil {
			logger.Errorf("error creating k8s selector from label selector: %v", err)
			return false
		}
		resourceLabelSet := labels.Set(event.labels)
		_, err = resourceLabelSet.AsValidatedSelector()
		if err != nil {
			logger.Errorf("%s %s/%s labels are invalid: %v", event.resource, event.namespace, event.name, err)
			return false
		}

		return k8sSelector.Matches(resourceLabelSet)
	}
	return false
}

func (s *Service) matchConditions(ctx context.Context, e *watcherEvent, t *testtriggersv1.TestTrigger, logger *zap.SugaredLogger) (bool, error) {
	timeout := s.defaultConditionsCheckTimeout
	if t.Spec.ConditionSpec.Timeout > 0 {
		timeout = time.Duration(t.Spec.ConditionSpec.Timeout) * time.Second
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
				t.Namespace, t.Name, e.eventType, e.resource, e.namespace, e.name,
			)
			return false, errors.WithStack(ErrConditionTimeout)
		default:
			logger.Debugf(
				"trigger service: matcher component: running conditions check iteration for %s %s/%s",
				e.resource, e.namespace, e.name,
			)
			conditions, err := e.conditionsGetter()
			if err != nil {
				logger.Errorf(
					"trigger service: matcher component: error getting conditions for %s %s/%s because of %v",
					e.resource, e.namespace, e.name, err,
				)
				return false, err
			}

			conditionMap := make(map[string]testtriggersv1.TestTriggerCondition, len(conditions))
			for _, condition := range conditions {
				conditionMap[condition.Type_] = condition
			}

			matched := true
			for _, triggerCondition := range t.Spec.ConditionSpec.Conditions {
				resourceCondition, ok := conditionMap[triggerCondition.Type_]
				if !ok || resourceCondition.Status == nil || triggerCondition.Status == nil ||
					*resourceCondition.Status != *triggerCondition.Status ||
					(triggerCondition.Reason != "" && triggerCondition.Reason != resourceCondition.Reason) ||
					(triggerCondition.Ttl != 0 && triggerCondition.Ttl < resourceCondition.Ttl) {
					matched = false
					break
				}
			}

			if matched {
				break outer
			}

			delay := s.defaultConditionsCheckBackoff
			if t.Spec.ConditionSpec.Delay > 0 {
				delay = time.Duration(t.Spec.ConditionSpec.Delay) * time.Second
			}
			time.Sleep(delay)
		}
	}

	return true, nil
}

func checkProbes(ctx context.Context, httpClient thttp.HttpClient, probes []testtriggersv1.TestTriggerProbe, logger *zap.SugaredLogger) bool {
	var wg sync.WaitGroup
	ch := make(chan bool, len(probes))
	defer close(ch)

	wg.Add(len(probes))
	for i := range probes {
		go func(probe testtriggersv1.TestTriggerProbe) {
			defer wg.Done()

			host := probe.Host
			if probe.Port != 0 {
				host = fmt.Sprintf("%s:%d", host, probe.Port)
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

func (s *Service) matchProbes(ctx context.Context, e *watcherEvent, t *testtriggersv1.TestTrigger, logger *zap.SugaredLogger) (bool, error) {
	timeout := s.defaultProbesCheckTimeout
	if t.Spec.ProbeSpec.Timeout > 0 {
		timeout = time.Duration(t.Spec.ProbeSpec.Timeout) * time.Second
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	host := ""
	if e.addressGetter != nil {
		var err error
		host, err = e.addressGetter(timeoutCtx, s.defaultProbesCheckBackoff)
		if err != nil {
			logger.Errorf(
				"trigger service: matcher component: error getting addess for %s %s/%s because of %v",
				e.resource, e.namespace, e.name, err,
			)
			return false, err
		}
	}

	for i := range t.Spec.ProbeSpec.Probes {
		if t.Spec.ProbeSpec.Probes[i].Scheme == "" {
			t.Spec.ProbeSpec.Probes[i].Scheme = defaultScheme
		}
		if t.Spec.ProbeSpec.Probes[i].Host == "" {
			t.Spec.ProbeSpec.Probes[i].Host = host
		}
		if t.Spec.ProbeSpec.Probes[i].Path == "" {
			t.Spec.ProbeSpec.Probes[i].Path = defaultPath
		}
	}

outer:
	for {
		select {
		case <-timeoutCtx.Done():
			logger.Errorf(
				"trigger service: matcher component: error waiting for probes to match for trigger %s/%s by event %s on resource %s %s/%s"+
					" because context got canceled by timeout or exit signal",
				t.Namespace, t.Name, e.eventType, e.resource, e.namespace, e.name,
			)
			return false, errors.WithStack(ErrProbeTimeout)
		default:
			logger.Debugf(
				"trigger service: matcher component: running probes check iteration for %s %s/%s",
				e.resource, e.namespace, e.name,
			)

			matched := checkProbes(timeoutCtx, s.httpClient, t.Spec.ProbeSpec.Probes, logger)
			if matched {
				break outer
			}

			delay := s.defaultProbesCheckBackoff
			if t.Spec.ProbeSpec.Delay > 0 {
				delay = time.Duration(t.Spec.ProbeSpec.Delay) * time.Second
			}
			time.Sleep(delay)
		}
	}

	return true, nil
}
