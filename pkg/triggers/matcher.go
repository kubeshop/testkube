package triggers

import (
	"context"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (s *Service) match(ctx context.Context, e *watcherEvent) error {
	for _, t := range s.triggers {
		if t.Spec.Resource != string(e.resource) {
			continue
		}
		if !matchEventOrCause(t.Spec.Event, e) {
			continue
		}
		if !matchSelector(&t.Spec.ResourceSelector, t.Namespace, e) {
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

func matchSelector(selector *testtriggersv1.TestTriggerSelector, namespace string, event *watcherEvent) bool {
	if selector.Name != "" {
		return selector.Name == event.name && namespace == event.namespace
	}
	if selector.LabelSelector != nil && len(event.labels) > 0 {
		return labels.Equals(selector.LabelSelector.MatchLabels, event.labels)
	}
	return false
}
