package controller

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	watchers2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	constants2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestNotifierEndWithoutTerminationCodeDefaultsToAbort(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	n := newTestNotifierWithJobTerminationCode("", now)
	if got := watchers2.GetTerminationCode(n.state.Job().Original()); got != string(testkube.ABORTED_TestWorkflowStatus) {
		t.Fatalf("expected missing termination code to default to aborted, got: %q", got)
	}

	n.End()

	if status := n.result.Steps["step-1"].Status; status == nil || (*status != testkube.ABORTED_TestWorkflowStepStatus && *status != testkube.SKIPPED_TestWorkflowStepStatus) {
		if status == nil {
			t.Fatalf("expected step-1 to be aborted/skipped, got nil status")
		}
		t.Fatalf("expected step-1 to be aborted/skipped, got: %s", *status)
	}
	if n.result.Initialization == nil || n.result.Initialization.Status == nil || *n.result.Initialization.Status != testkube.ABORTED_TestWorkflowStepStatus {
		t.Fatalf("expected initialization to be aborted, got: %v", n.result.Initialization)
	}
}

func TestNotifierEndWithExplicitAbortTerminationCodeAborts(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	n := newTestNotifierWithJobTerminationCode(string(testkube.ABORTED_TestWorkflowStatus), now)
	if got := watchers2.GetTerminationCode(n.state.Job().Original()); got != string(testkube.ABORTED_TestWorkflowStatus) {
		t.Fatalf("expected termination code to be propagated, got: %q", got)
	}

	n.End()

	if status := n.result.Steps["step-1"].Status; status == nil || (*status != testkube.ABORTED_TestWorkflowStepStatus && *status != testkube.SKIPPED_TestWorkflowStepStatus) {
		if status == nil {
			t.Fatalf("expected step-1 to be aborted/skipped, got nil status")
		}
		t.Fatalf("expected step-1 to be aborted/skipped, got: %s", *status)
	}
	if n.result.Initialization == nil || n.result.Initialization.Status == nil || *n.result.Initialization.Status != testkube.ABORTED_TestWorkflowStepStatus {
		t.Fatalf("expected initialization to be aborted, got: %v", n.result.Initialization)
	}
}

func newTestNotifierWithJobTerminationCode(terminationCode string, now time.Time) *notifier {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // avoid blocking send() in notifier during unit tests

	initial := testkube.TestWorkflowResult{
		QueuedAt:  now.Add(-3 * time.Minute),
		StartedAt: now.Add(-2 * time.Minute),
		Initialization: &testkube.TestWorkflowStepResult{
			Status:    common.Ptr(testkube.RUNNING_TestWorkflowStepStatus),
			QueuedAt:  now.Add(-2 * time.Minute),
			StartedAt: now.Add(-2 * time.Minute),
		},
		Steps: map[string]testkube.TestWorkflowStepResult{
			"step-1": {
				Status:    common.Ptr(testkube.RUNNING_TestWorkflowStepStatus),
				QueuedAt:  now.Add(-90 * time.Second),
				StartedAt: now.Add(-90 * time.Second),
			},
		},
	}
	n := newNotifier(ctx, initial, now.Add(-3*time.Minute))

	annotations := map[string]string{}
	if terminationCode != "" {
		annotations[constants2.AnnotationTerminationCode] = terminationCode
	}

	job := watchers2.NewJob(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "exec-id",
			Namespace:   "testkube",
			Annotations: annotations,
		},
	})

	n.state = watchers2.NewExecutionState(
		job,
		nil,
		watchers2.NewJobEvents(nil),
		watchers2.NewPodEvents(nil),
		&watchers2.ExecutionStateOptions{ScheduledAt: now.Add(-3 * time.Minute)},
	)
	n.sigSequence = []testkube.TestWorkflowSignature{
		{Ref: initconstants.InitStepName, Name: "Initializing"},
		{Ref: "step-1", Name: "Step 1"},
	}

	return n
}
