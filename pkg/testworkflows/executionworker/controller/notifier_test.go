package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

var unschedulablePod = &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, Conditions: []corev1.PodCondition{{
	Type:    corev1.PodScheduled,
	Status:  corev1.ConditionFalse,
	Reason:  corev1.PodReasonUnschedulable,
	Message: "0/1 nodes are available: 1 Insufficient cpu.",
}}}}

func TestNotifier_Align(t *testing.T) {
	const unschedulableMessage = "no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu."
	type cause struct {
		message string
		reason  string
	}
	scheduledPod := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}
	// scheduledAgainPod reports the same cause in a new pod object, so the notifier checks it again.
	scheduledAgainPod := unschedulablePod.DeepCopy()
	hint := func(name string, value interface{}) instructions.Instruction {
		return instructions.Instruction{Ref: "rstep1", Name: name, Value: value}
	}
	deletedPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}}, Status: corev1.PodStatus{Phase: corev1.PodPending}}
	state := func(pod *corev1.Pod) watchers.ExecutionState {
		return watchers.NewExecutionState(nil, watchers.NewPod(pod), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil)
	}

	tests := []struct {
		name                  string
		initializationStatus  testkube.TestWorkflowStepStatus
		initializationMessage string
		stepStatus            testkube.TestWorkflowStepStatus
		pods                  []*corev1.Pod
		// then sets the step status or applies the hints, and aligns one more pod
		thenStepStatus     testkube.TestWorkflowStepStatus
		thenHints          []instructions.Instruction
		thenPod            *corev1.Pod
		wantInitialization cause
		wantStep           cause
		wantNextStep       cause
		wantEvents         int
	}{
		{
			name:                 "writes the cause into the initialization step and sends one event while the cause stays the same",
			initializationStatus: testkube.RUNNING_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod, unschedulablePod},
			wantInitialization:   cause{unschedulableMessage, "unschedulable"},
			wantEvents:           1,
		},
		{
			name:                 "writes the cause into the first step that did not start when the initialization passed",
			initializationStatus: testkube.PASSED_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod},
			wantStep:             cause{unschedulableMessage, "unschedulable"},
			wantEvents:           1,
		},
		{
			name:                 "clears the cause when Kubernetes stops to report it",
			initializationStatus: testkube.RUNNING_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod, scheduledPod},
			wantEvents:           1,
		},
		{
			name:                 "keeps the cause when the pod of the ended execution is deleted",
			initializationStatus: testkube.RUNNING_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod, deletedPod},
			wantInitialization:   cause{unschedulableMessage, "unschedulable"},
			wantEvents:           1,
		},
		{
			name:                  "keeps a message that another component wrote",
			initializationStatus:  testkube.RUNNING_TestWorkflowStepStatus,
			initializationMessage: "the image could not be pulled",
			pods:                  []*corev1.Pod{unschedulablePod},
			wantInitialization:    cause{"the image could not be pulled", ""},
		},
		{
			name:                 "does not write a cause while a step runs",
			initializationStatus: testkube.PASSED_TestWorkflowStepStatus,
			stepStatus:           testkube.RUNNING_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod},
		},
		{
			name:                 "clears its own cause when the step finished after the cause, also when the execution is complete",
			initializationStatus: testkube.PASSED_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod},
			thenStepStatus:       testkube.PASSED_TestWorkflowStepStatus,
			thenPod:              deletedPod,
			wantEvents:           1,
		},
		{
			name:                 "clears its own cause and reason when the init process starts the step",
			initializationStatus: testkube.PASSED_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod},
			thenHints:            []instructions.Instruction{hint(initconstants.InstructionStart, nil)},
			wantEvents:           1,
		},
		{
			name:                 "gives up a cause that the step result replaced, so the next waiting step gets its own cause",
			initializationStatus: testkube.PASSED_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod},
			thenHints: []instructions.Instruction{
				hint(initconstants.InstructionStart, nil),
				hint(initconstants.InstructionExecution, initconstants.ExecutionResult{ExitCode: 1, Details: "the step ran for too long"}),
				hint(initconstants.InstructionEnd, string(testkube.FAILED_TestWorkflowStepStatus)),
			},
			thenPod:      scheduledAgainPod,
			wantStep:     cause{"the step ran for too long", ""},
			wantNextStep: cause{unschedulableMessage, "unschedulable"},
			wantEvents:   2,
		},
		{
			name:                 "clears its own cause when the step starts to run",
			initializationStatus: testkube.PASSED_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{unschedulablePod},
			thenStepStatus:       testkube.RUNNING_TestWorkflowStepStatus,
			thenPod:              unschedulablePod,
			wantEvents:           1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newTestNotifier(t, tt.initializationStatus, tt.initializationMessage)
			n.sigSequence = []testkube.TestWorkflowSignature{{Ref: "rstep1"}, {Ref: "rstep2"}}
			n.result.Steps["rstep2"] = testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus)}
			if tt.stepStatus != "" {
				n.result.Steps["rstep1"] = testkube.TestWorkflowStepResult{Status: common.Ptr(tt.stepStatus)}
			}

			for _, pod := range tt.pods {
				n.alignCause(state(pod))
			}
			for _, h := range tt.thenHints {
				n.Instruction(time.Now(), h, "exec-1")
			}
			if tt.thenStepStatus != "" {
				n.result.Steps["rstep1"] = testkube.TestWorkflowStepResult{Status: common.Ptr(tt.thenStepStatus), ErrorMessage: n.result.Steps["rstep1"].ErrorMessage}
			}
			if tt.thenPod != nil {
				n.alignCause(state(tt.thenPod))
			}

			assert.Equal(t, tt.wantInitialization, cause{n.result.Initialization.ErrorMessage, n.result.Initialization.ErrorReason})
			assert.Equal(t, tt.wantStep, cause{n.result.Steps["rstep1"].ErrorMessage, n.result.Steps["rstep1"].ErrorReason})
			assert.Equal(t, tt.wantNextStep, cause{n.result.Steps["rstep2"].ErrorMessage, n.result.Steps["rstep2"].ErrorReason})
			assert.Equal(t, tt.wantEvents, countLogs(n, "(unschedulable)"))
		})
	}
}

func TestNotifier_End(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantStatus  testkube.TestWorkflowStepStatus
		wantMessage string
	}{
		{
			name:        "puts the recorded cause into the cancel message",
			annotations: map[string]string{constants.AnnotationTerminationCode: string(testkube.CANCELED_TestWorkflowStatus)},
			wantStatus:  testkube.CANCELED_TestWorkflowStepStatus,
			wantMessage: "The execution has been canceled. (no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
		},
		{
			name:        "puts the recorded cause into the abort message of a job without a termination code",
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newTestNotifier(t, testkube.RUNNING_TestWorkflowStepStatus, "")
			job := watchers.NewJob(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Annotations: tt.annotations}})
			n.Align(watchers.NewExecutionState(job, watchers.NewPod(unschedulablePod), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil))

			n.End()

			assert.Equal(t, tt.wantStatus, *n.result.Initialization.Status)
			assert.Equal(t, tt.wantMessage, n.result.Initialization.ErrorMessage)
		})
	}
}

// newTestNotifier returns a notifier with one queued step and a buffered channel, so the notifications do not block the test.
func newTestNotifier(t *testing.T, initializationStatus testkube.TestWorkflowStepStatus, initializationMessage string) *notifier {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	n := newNotifier(ctx, testkube.TestWorkflowResult{
		Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(initializationStatus), ErrorMessage: initializationMessage},
		Steps:          map[string]testkube.TestWorkflowStepResult{"rstep1": {Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus)}},
	}, time.Time{})
	n.ch = make(chan ChannelMessage[Notification], 100)
	return n
}

func countLogs(n *notifier, text string) int {
	count := 0
	for len(n.ch) > 0 {
		if strings.Contains((<-n.ch).Value.Log, text) {
			count++
		}
	}
	return count
}

func TestNotifier_Instruction(t *testing.T) {
	const ref = "rstep1"
	const timeout = "the step did not finish within its timeout"
	execution := func(iteration int, details string) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: initconstants.InstructionExecution, Value: initconstants.ExecutionResult{ExitCode: 1, Iteration: iteration, Details: details}}
	}
	retry := func(iteration int) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: initconstants.InstructionIteration, Value: iteration}
	}

	tests := []struct {
		name         string
		hints        []instructions.Instruction
		wantMessage  string
		wantAttempts int32
	}{
		{
			name:         "reports 1 attempt for a step without retry",
			hints:        []instructions.Instruction{execution(0, "")},
			wantAttempts: 1,
		},
		{
			name:         "sets the attempts from the iteration of the last execution result",
			hints:        []instructions.Instruction{execution(0, ""), retry(1), execution(1, ""), retry(2), execution(2, "")},
			wantAttempts: 3,
		},
		{
			name:         "counts the attempts that time out and send no execution result",
			hints:        []instructions.Instruction{retry(1), retry(2)},
			wantAttempts: 3,
		},
		{
			name:         "keeps the step message when the execution result has no details",
			hints:        []instructions.Instruction{execution(0, timeout), execution(0, "")},
			wantMessage:  timeout,
			wantAttempts: 1,
		},
		{
			name:         "clears the message of the previous attempt when the step retries",
			hints:        []instructions.Instruction{execution(0, timeout), retry(1)},
			wantAttempts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			ch := make(chan ChannelMessage[Notification], 100)
			n := &notifier{
				ctx: ctx,
				ch:  ch,
				result: testkube.TestWorkflowResult{
					Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.PASSED_TestWorkflowStepStatus)},
					Steps: map[string]testkube.TestWorkflowStepResult{
						ref: {Status: common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)},
					},
				},
			}

			for _, hint := range tt.hints {
				n.Instruction(time.Now(), hint, "exec-1")
			}

			assert.Equal(t, tt.wantMessage, n.result.Steps[ref].ErrorMessage)
			// Read the sent result, because the notifier sends a copy of its state.
			var last *testkube.TestWorkflowResult
			for len(ch) > 0 {
				if message := <-ch; message.Value.Result != nil {
					last = message.Value.Result
				}
			}
			require.NotNil(t, last)
			assert.Equal(t, tt.wantAttempts, last.Steps[ref].Attempts)
		})
	}
}
