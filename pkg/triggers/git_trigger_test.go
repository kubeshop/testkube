package triggers

import (
	"context"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestMatchGitTrigger_ExecutesOnlyTargetTrigger(t *testing.T) {
	triggerA := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
		},
	}
	triggerB := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-b", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, triggerA.Namespace, triggerA.Name): {trigger: convertV1ToInternal(triggerA)},
			newStatusKey(triggerSourceV1, triggerB.Namespace, triggerB.Name): {trigger: convertV1ToInternal(triggerB)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Name)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), "trigger-a", "default", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"trigger-a"}, executed)
}

func TestMatchGitTrigger_IncrementsEventMetric(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
		},
	}
	m := metrics.NewMetrics()
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: m,
	}

	counter := m.TestTriggerEventCount.WithLabelValues(trigger.Name, "content", string(v1.TestTriggerEventGitPush), "")
	metricBefore := &dto.Metric{}
	require.NoError(t, counter.Write(metricBefore))
	before := metricBefore.GetCounter().GetValue()

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.NoError(t, err)

	metricAfter := &dto.Metric{}
	require.NoError(t, counter.Write(metricAfter))
	after := metricAfter.GetCounter().GetValue()
	assert.Equal(t, before+1, after)
}

func TestMatchGitTrigger_ExecutesBoundV1Source(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {
				trigger: convertV1ToInternal(trigger),
			},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Source)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.NoError(t, err)
	assert.Contains(t, executed, triggerSourceV1)
}

func TestMatchGitTrigger_DoesNotExecuteUnrelatedV2TriggerWithSameName(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "deploy", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
		},
	}

	unrelatedV2 := &internalTrigger{
		Name:              "deploy",
		Namespace:         "default",
		Source:            triggerSourceV2,
		ResourceKind:      "content",
		Event:             string(v1.TestTriggerEventGitPush),
		ResourceName:      "another-content",
		ResourceNamespace: "default",
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
			newStatusKey(triggerSourceV2, trigger.Namespace, trigger.Name): {trigger: unrelatedV2},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Source)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{triggerSourceV1}, executed)
}

func TestMatchGitTrigger_IgnoresFieldConditionsForContentEvents(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
			Match: []workflowtriggersv1.WorkflowTriggerFieldCondition{
				{
					Path:     ".metadata.name",
					Operator: workflowtriggersv1.FieldOperatorExists,
				},
			},
		},
	}

	var executed []string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {
				trigger: convertV1ToInternal(trigger),
			},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, trigger *internalTrigger) error {
			executed = append(executed, trigger.Name)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{trigger.Name}, executed)
}

func TestMatchGitTrigger_ReturnsErrorWhenTargetTriggerNotReady(t *testing.T) {
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{},
		logger:        log.DefaultLogger,
		metrics:       metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), "trigger-a", "default", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerTargetNotReady)
}

func TestMatchGitTrigger_ReturnsErrorWhenTargetStatusIsStaleNonContent(t *testing.T) {
	staleTrigger := &internalTrigger{
		Name:         "trigger-a",
		Namespace:    "default",
		Source:       triggerSourceV1,
		ResourceKind: "tests",
		Event:        string(v1.TestTriggerEventCreated),
	}

	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, staleTrigger.Namespace, staleTrigger.Name): {trigger: staleTrigger},
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), staleTrigger.Name, staleTrigger.Namespace, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerTargetNotReady)
}

func TestMatchGitTrigger_ReturnsErrorWhenConditionsConfiguredForSyntheticEvent(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
			ConditionSpec: &v1.TestTriggerConditionSpec{
				Conditions: []v1.TestTriggerCondition{
					{Type_: "Ready", Status: conditionStatusPtr(v1.TRUE_TestTriggerConditionStatuses)},
				},
			},
		},
	}

	executed := false
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			executed = true
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerConditionsUnavailable)
	assert.False(t, executed)
}

func TestMatchGitTrigger_ReturnsErrorWhenProbesConfiguredForSyntheticEvent(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
			ProbeSpec: &v1.TestTriggerProbeSpec{
				Probes: []v1.TestTriggerProbe{
					{Path: "/health", Port: 8080},
				},
			},
		},
	}

	executed := false
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			executed = true
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errGitTriggerProbesUnavailable)
	assert.False(t, executed)
}

func TestMatchGitTrigger_SkipsNonTestWorkflowExecution(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource:  v1.TestTriggerResourceContent,
			Event:     v1.TestTriggerEventGitPush,
			Execution: v1.TestTriggerExecutionTest,
		},
	}

	executed := false
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, _ *watcherEvent, _ *internalTrigger) error {
			executed = true
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, nil)
	require.NoError(t, err)
	assert.False(t, executed)
}

func conditionStatusPtr(v v1.TestTriggerConditionStatuses) *v1.TestTriggerConditionStatuses {
	return &v
}

func TestGitEventTypeFromMeta(t *testing.T) {
	tests := []struct {
		name     string
		meta     map[string]string
		expected string
	}{
		{"nil meta returns git-push", nil, string(v1.TestTriggerEventGitPush)},
		{"empty meta returns git-push", map[string]string{}, string(v1.TestTriggerEventGitPush)},
		{"branch meta returns git-push", map[string]string{"TESTKUBE_GIT_BRANCH": "main"}, string(v1.TestTriggerEventGitPush)},
		{"tag meta returns git-tag-push", map[string]string{"TESTKUBE_GIT_TAG": "v1.0"}, string(v1.TestTriggerEventGitTagPush)},
		{"both branch and tag prefers tag", map[string]string{"TESTKUBE_GIT_BRANCH": "main", "TESTKUBE_GIT_TAG": "v1.0"}, string(v1.TestTriggerEventGitTagPush)},
		{"PR number returns git-pull-request", map[string]string{"TESTKUBE_GIT_PR_NUMBER": "42"}, string(v1.TestTriggerEventGitPullRequest)},
		{"PR with tag prefers PR", map[string]string{"TESTKUBE_GIT_PR_NUMBER": "42", "TESTKUBE_GIT_TAG": "v1.0"}, string(v1.TestTriggerEventGitPullRequest)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, gitEventTypeFromMeta(tt.meta))
		})
	}
}

func TestIsGitSyntheticTargetReady(t *testing.T) {
	tests := []struct {
		name     string
		trigger  *internalTrigger
		expected bool
	}{
		{
			"git-push event is ready",
			&internalTrigger{ResourceKind: "content", Event: string(v1.TestTriggerEventGitPush)},
			true,
		},
		{
			"git-tag-push event is ready",
			&internalTrigger{ResourceKind: "content", Event: string(v1.TestTriggerEventGitTagPush)},
			true,
		},
		{
			"git-pull-request event is ready",
			&internalTrigger{ResourceKind: "content", Event: string(v1.TestTriggerEventGitPullRequest)},
			true,
		},
		{
			"modified event is not ready",
			&internalTrigger{ResourceKind: "content", Event: "modified"},
			false,
		},
		{
			"created event is not ready",
			&internalTrigger{ResourceKind: "content", Event: "created"},
			false,
		},
		{
			"disabled trigger is not ready",
			&internalTrigger{ResourceKind: "content", Event: string(v1.TestTriggerEventGitPush), Disabled: true},
			false,
		},
		{
			"non-content resource is not ready",
			&internalTrigger{ResourceKind: "deployment", Event: string(v1.TestTriggerEventGitPush)},
			false,
		},
		{
			"case insensitive resource kind",
			&internalTrigger{ResourceKind: "Content", Event: string(v1.TestTriggerEventGitPush)},
			true,
		},
		{
			"case insensitive event",
			&internalTrigger{ResourceKind: "content", Event: "Git-Push"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isGitSyntheticTargetReady(tt.trigger))
		})
	}
}

func TestMatchGitTrigger_GitTagPushEvent(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitTagPush,
		},
	}

	var executedEvent string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, event *watcherEvent, _ *internalTrigger) error {
			executedEvent = string(event.eventType)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	meta := map[string]string{"TESTKUBE_GIT_TAG": "v1.0.0"}
	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, meta)
	require.NoError(t, err)
	assert.Equal(t, string(v1.TestTriggerEventGitTagPush), executedEvent)
}

func TestMatchGitTrigger_AttachesGitMetadata(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "trigger-a", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPush,
		},
	}

	var capturedMeta *GitMetadata
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, event *watcherEvent, _ *internalTrigger) error {
			capturedMeta = event.GitMetadata
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	meta := map[string]string{
		"TESTKUBE_GIT_COMMIT": "abc123",
		"TESTKUBE_GIT_REF":    "refs/heads/main",
		"TESTKUBE_GIT_BRANCH": "main",
	}
	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, meta)
	require.NoError(t, err)
	require.NotNil(t, capturedMeta)
	assert.Equal(t, "abc123", capturedMeta.Commit)
	assert.Equal(t, "refs/heads/main", capturedMeta.Ref)
	assert.Equal(t, "main", capturedMeta.Branch)
}

func TestMatchGitTrigger_GitPullRequestEvent(t *testing.T) {
	trigger := &v1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Name: "pr-trigger", Namespace: "default"},
		Spec: v1.TestTriggerSpec{
			Resource: v1.TestTriggerResourceContent,
			Event:    v1.TestTriggerEventGitPullRequest,
		},
	}

	var capturedMeta *GitMetadata
	var executedEvent string
	s := &Service{
		triggerStatus: map[statusKey]*triggerStatus{
			newStatusKey(triggerSourceV1, trigger.Namespace, trigger.Name): {trigger: convertV1ToInternal(trigger)},
		},
		triggerExecutor: func(_ context.Context, event *watcherEvent, _ *internalTrigger) error {
			capturedMeta = event.GitMetadata
			executedEvent = string(event.eventType)
			return nil
		},
		logger:  log.DefaultLogger,
		metrics: metrics.NewMetrics(),
	}

	meta := map[string]string{
		"TESTKUBE_GIT_COMMIT":      "deadbeef",
		"TESTKUBE_GIT_REF":         "refs/pull/42/head",
		"TESTKUBE_GIT_BRANCH":      "feature/x",
		"TESTKUBE_GIT_PR_NUMBER":   "42",
		"TESTKUBE_GIT_PR_ACTION":   "synchronize",
		"TESTKUBE_GIT_PR_BASE_REF": "main",
		"TESTKUBE_GIT_PR_HEAD_REF": "feature/x",
		"TESTKUBE_GIT_PR_HEAD_SHA": "deadbeef",
		"TESTKUBE_GIT_PR_URL":      "https://github.com/org/repo/pull/42",
		"TESTKUBE_GIT_PR_TITLE":    "Add feature",
		"TESTKUBE_GIT_PR_AUTHOR":   "developer",
	}
	err := s.MatchGitTrigger(context.Background(), trigger.Name, trigger.Namespace, meta)
	require.NoError(t, err)
	assert.Equal(t, string(v1.TestTriggerEventGitPullRequest), executedEvent)
	require.NotNil(t, capturedMeta)
	assert.Equal(t, "deadbeef", capturedMeta.Commit)
	assert.Equal(t, "42", capturedMeta.PRNumber)
	assert.Equal(t, "synchronize", capturedMeta.PRAction)
	assert.Equal(t, "main", capturedMeta.PRBaseRef)
	assert.Equal(t, "feature/x", capturedMeta.PRHeadRef)
	assert.Equal(t, "deadbeef", capturedMeta.PRHeadSHA)
	assert.Equal(t, "https://github.com/org/repo/pull/42", capturedMeta.PRURL)
	assert.Equal(t, "Add feature", capturedMeta.PRTitle)
	assert.Equal(t, "developer", capturedMeta.PRAuthor)
}
