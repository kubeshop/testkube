package triggers

import (
	"context"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
)

func (s *Service) Match(ctx context.Context, e *Event) error {
	for _, t := range s.triggers {
		if t.Spec.Resource != string(e.Resource) {
			continue
		}
		if !matchEventOrCause(t.Spec.Event, e) {
			continue
		}
		if !matchSelector(&t.Spec.ResourceSelector, t.Namespace, e) {
			continue
		}
		status := s.getStatusForTrigger(t)
		if status.ActiveTests {
			s.l.Infof(
				"trigger service: matcher component: skipping trigger execution for trigger %s/%s by event %s on resource %s because it is currently running tests",
				t.Namespace, t.Name, e.Type, e.Resource,
			)
			return nil
		}
		s.l.Infof("trigger service: matcher component: event %s matches trigger %s/%s for resource %s", e.Type, t.Namespace, t.Name, e.Resource)
		s.l.Infof("trigger service: matcher component: triggering %s action for %s execution", t.Spec.Action, t.Spec.Execution)
		if err := s.Execute(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func matchEventOrCause(targetEvent string, event *Event) bool {
	if targetEvent == string(event.Type) {
		return true
	}
	for _, c := range event.Causes {
		if targetEvent == string(c) {
			return true
		}
	}
	return false
}

func matchSelector(selector *testtriggersv1.TestTriggerSelector, namespace string, event *Event) bool {
	if selector.Name != "" {
		if selector.Name == event.Name && namespace == event.Namespace {
			return true
		}
	}
	if len(selector.Labels) > 0 && len(event.Labels) > 0 {
		for targetLabel, targetValue := range selector.Labels {
			value, ok := event.Labels[targetLabel]
			if ok {
				if targetValue == value {
					return true
				}
			}
		}
	}
	return false
}
