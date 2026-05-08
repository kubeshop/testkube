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

	for _, entry := range s.snapshotStatuses() {
		if entry.trigger.Name != triggerName || entry.trigger.Namespace != namespace {
			continue
		}
		if !strings.EqualFold(entry.trigger.ResourceKind, string(testtrigger.ResourceContent)) {
			continue
		}

		matcher := *s
		matcher.triggerStatus = map[statusKey]*triggerStatus{
			entry.key: entry.status,
		}
		return matcher.match(ctx, event)
	}

	return nil
}
