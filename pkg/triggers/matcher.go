package triggers

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
)

var ErrConditionTimeout = errors.New("timed-out waiting for trigger conditions")

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

		status := s.getStatusForTrigger(t)
		if status.hasActiveTests() {
			s.logger.Infof(
				"trigger service: matcher component: skipping trigger execution for trigger %s/%s by event %s on resource %s because it is currently running tests",
				t.Namespace, t.Name, e.eventType, e.resource,
			)
			return nil
		}
		s.logger.Infof("trigger service: matcher component: event %s matches trigger %s/%s for resource %s", e.eventType, t.Namespace, t.Name, e.resource)
		s.logger.Infof("trigger service: matcher component: triggering %s action for %s execution", t.Spec.Action, t.Spec.Execution)
		if err := s.executor(ctx, t); err != nil {
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

			time.Sleep(s.defaultConditionsCheckBackoff)
		}
	}

	return true, nil
}
