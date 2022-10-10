package triggers

import (
	"context"

	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
)

func (s *Service) match(ctx context.Context, e *watcherEvent) error {
	for _, status := range s.triggerStatus {
		t := status.testTrigger
		if t.Spec.Resource != string(e.resource) {
			continue
		}
		if !matchEventOrCause(t.Spec.Event, e) {
			continue
		}
		if !matchSelector(&t.Spec.ResourceSelector, t.Namespace, e, s.logger) {
			continue
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
		return selector.Name == event.name && namespace == event.namespace
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
