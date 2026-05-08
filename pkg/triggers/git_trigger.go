package triggers

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

// MatchGitTrigger creates a synthetic watcherEvent for a git content trigger
// and runs it through the matcher. Called by the git informer when new commits
// are detected that match the trigger's content selector paths.
func (s *Service) MatchGitTrigger(ctx context.Context, triggerName, namespace string) error {
	event := s.newWatcherEvent(
		testtrigger.EventModified,
		&metav1.ObjectMeta{Name: triggerName, Namespace: namespace},
		nil,
		testtrigger.ResourceType(testtrigger.ResourceContent),
	)

	key := newStatusKey(triggerSourceV1, namespace, triggerName)
	s.triggerStatusMu.RLock()
	status, exists := s.triggerStatus[key]
	var trigger *internalTrigger
	if exists {
		trigger = status.trigger
	}
	s.triggerStatusMu.RUnlock()
	if !exists || trigger == nil {
		return nil
	}
	if !strings.EqualFold(trigger.ResourceKind, string(testtrigger.ResourceContent)) {
		return nil
	}

	if trigger.Disabled {
		return nil
	}
	if trigger.Execution != "" && trigger.Execution != ExecutionTestWorkflow {
		return nil
	}
	if !matchInternalResource(trigger, event, s.logger) {
		return nil
	}
	if !matchEventOrCause(trigger.Event, event) {
		return nil
	}
	if !matchFieldSelector(trigger.FieldConditions, event.Object, event.OldObject) {
		return nil
	}
	if trigger.Conditions != nil && len(trigger.Conditions.Items) > 0 && event.conditionsGetter != nil {
		matched, err := s.matchInternalConditions(ctx, event, trigger, s.logger)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
	}
	if trigger.Probes != nil && len(trigger.Probes.Items) > 0 {
		matched, err := s.matchInternalProbes(ctx, event, trigger, s.logger)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
	}
	if trigger.ConcurrencyPolicy == concurrencyPolicyForbid && status.hasActiveTests() {
		return nil
	}
	if trigger.ConcurrencyPolicy == concurrencyPolicyReplace && status.hasActiveTests() {
		s.abortExecutions(ctx, trigger.Name, status)
	}
	return s.triggerExecutor(ctx, event, trigger)

}
