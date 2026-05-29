package triggers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/pkg/operator/validation/tests/v1/testtrigger"
)

var errGitTriggerTargetNotReady = errors.New("git trigger target not ready")
var errGitTriggerConditionsUnavailable = errors.New("git trigger conditions unavailable for synthetic content event")
var errGitTriggerProbesUnavailable = errors.New("git trigger probes unavailable for synthetic content event")

// MatchGitTrigger creates a synthetic watcherEvent for a git content trigger
// and runs it through the matcher. Called by the git informer when new commits
// are detected that match the trigger's content selector paths.
func (s *Service) MatchGitTrigger(ctx context.Context, triggerName, namespace string, gitMeta map[string]string) error {
	return s.matchGitTriggerBySource(ctx, triggerName, namespace, triggerSourceV1, gitMeta)
}

func (s *Service) matchGitTriggerBySource(ctx context.Context, triggerName, namespace, source string, gitMeta map[string]string) error {
	event := s.newWatcherEvent(
		testtrigger.EventModified,
		&metav1.ObjectMeta{Name: triggerName, Namespace: namespace},
		nil,
		testtrigger.ResourceType(testtrigger.ResourceContent),
	)

	// Attach git metadata to the event for downstream use by the executor.
	if len(gitMeta) > 0 {
		event.GitMetadata = &GitMetadata{
			Commit:          gitMeta["TESTKUBE_GIT_COMMIT"],
			Ref:             gitMeta["TESTKUBE_GIT_REF"],
			Branch:          gitMeta["TESTKUBE_GIT_BRANCH"],
			Tag:             gitMeta["TESTKUBE_GIT_TAG"],
			CommitMessage:   gitMeta["TESTKUBE_GIT_COMMIT_MESSAGE"],
			Author:          gitMeta["TESTKUBE_GIT_AUTHOR"],
			CommitTimestamp: gitMeta["TESTKUBE_GIT_COMMIT_TIMESTAMP"],
		}
	}

	key := newStatusKey(source, namespace, triggerName)
	s.triggerStatusMu.RLock()
	status, exists := s.triggerStatus[key]
	var trigger *internalTrigger
	if exists {
		trigger = status.trigger
	}
	s.triggerStatusMu.RUnlock()
	if !exists || trigger == nil {
		return fmt.Errorf("%w: %s/%s", errGitTriggerTargetNotReady, namespace, triggerName)
	}
	if !isGitSyntheticTargetReady(trigger) {
		return fmt.Errorf("%w: %s/%s", errGitTriggerTargetNotReady, namespace, triggerName)
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
	if trigger.Conditions != nil && len(trigger.Conditions.Items) > 0 {
		if event.conditionsGetter == nil {
			return fmt.Errorf("%w: %s/%s", errGitTriggerConditionsUnavailable, namespace, triggerName)
		}
		matched, err := s.matchInternalConditions(ctx, event, trigger, s.logger)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
	}
	if trigger.Probes != nil && len(trigger.Probes.Items) > 0 {
		if event.addressGetter == nil {
			return fmt.Errorf("%w: %s/%s", errGitTriggerProbesUnavailable, namespace, triggerName)
		}
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
	s.logger.Infof(
		"trigger service: matcher component: event %s matches trigger %s/%s for resource %s",
		event.eventType,
		trigger.Namespace,
		trigger.Name,
		event.resource,
	)
	causes := make([]string, len(event.causes))
	for idx, cause := range event.causes {
		causes[idx] = string(cause)
	}
	s.metrics.IncTestTriggerEventCount(trigger.Name, string(event.resource), string(event.eventType), causes)
	return s.triggerExecutor(ctx, event, trigger)

}

func isGitSyntheticTargetReady(trigger *internalTrigger) bool {
	return strings.EqualFold(trigger.ResourceKind, string(testtrigger.ResourceContent)) &&
		!trigger.Disabled &&
		strings.EqualFold(trigger.Event, string(testtrigger.EventModified))
}
