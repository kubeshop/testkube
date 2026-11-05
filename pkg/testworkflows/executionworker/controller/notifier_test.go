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

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
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
	const pullError = `Failed to pull image "org/private:1": pull access denied, repository does not exist or may require authorization`
	const pullBackOff = `Back-off pulling image "org/private:1"`
	waitingPod := func(reason, message string) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, InitContainerStatuses: []corev1.ContainerStatus{{
			Name:  "1",
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}},
		}}}}
	}
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
			name:                 "keeps the error of a failed pull when Kubernetes waits to pull again",
			initializationStatus: testkube.RUNNING_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{waitingPod("ErrImagePull", pullError), waitingPod("ImagePullBackOff", pullBackOff)},
			wantInitialization:   cause{"the image could not be pulled: " + pullError, "image-pull-failed"},
		},
		{
			name:                 "writes the wait to pull again when no error came before it",
			initializationStatus: testkube.RUNNING_TestWorkflowStepStatus,
			pods:                 []*corev1.Pod{waitingPod("ImagePullBackOff", pullBackOff)},
			wantInitialization:   cause{"the image could not be pulled: " + pullBackOff, "image-pull-failed"},
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
	const scheduler = "0/1 nodes are available: 1 Insufficient cpu."
	const mountMessage = `MountVolume.SetUp failed for volume "missing-secret" : secret "absent" not found`
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	scheduledPod := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}
	// deletedPod is a pod that the job deleted before it started, so it reports no waiting cause of its own.
	deletedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: start.Add(time.Minute)}},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	event := func(eventType, reason, message string, after time.Duration) *corev1.Event {
		return &corev1.Event{Type: eventType, Reason: reason, Message: message, LastTimestamp: metav1.NewTime(start.Add(after))}
	}
	mountEvents := []*corev1.Event{
		event(corev1.EventTypeNormal, "Scheduled", "Successfully assigned", 0),
		event(corev1.EventTypeWarning, "FailedMount", mountMessage, 32*time.Second),
	}
	deadlineJob := func(annotations map[string]string) *batchv1.Job {
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Annotations: annotations},
			Spec:       batchv1.JobSpec{ActiveDeadlineSeconds: common.Ptr(int64(60))},
			Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
				Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: watchers.ReasonDeadlineExceeded,
			}}},
		}
	}
	plainJob := func(annotations map[string]string) *batchv1.Job {
		return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Annotations: annotations}}
	}
	initTimeout := map[string]string{
		constants.AnnotationTerminationCode:   string(testkube.ABORTED_TestWorkflowStatus),
		constants.AnnotationTerminationActor:  string(testkube.StopActorRunner),
		constants.AnnotationTerminationReason: string(testkube.StopReasonInitTimeout),
	}
	userCancel := map[string]string{
		constants.AnnotationTerminationCode:  string(testkube.CANCELED_TestWorkflowStatus),
		constants.AnnotationTerminationActor: string(testkube.StopActorUser),
	}

	tests := []struct {
		name        string
		job         *batchv1.Job
		pod         *corev1.Pod
		podEvents   []*corev1.Event
		initMessage string
		initReason  string
		// then runs after the notifier wrote the cause of the pod, and before End
		then        func(n *notifier)
		wantStatus  testkube.TestWorkflowStepStatus
		wantMessage string
		wantReason  string
		wantDetails string
	}{
		{
			name:        "puts the recorded cause into the cancel message",
			job:         plainJob(map[string]string{constants.AnnotationTerminationCode: string(testkube.CANCELED_TestWorkflowStatus)}),
			pod:         unschedulablePod,
			wantStatus:  testkube.CANCELED_TestWorkflowStepStatus,
			wantMessage: "The execution has been canceled. (no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
			wantReason:  string(testkube.StopReasonUnschedulable),
			wantDetails: scheduler,
		},
		{
			name:        "puts the recorded cause into the abort message of a job without a termination code",
			job:         plainJob(nil),
			pod:         unschedulablePod,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
			wantReason:  string(testkube.StopReasonUnschedulable),
			wantDetails: scheduler,
		},
		{
			name:        "gives the words of a stop of the runner before the recorded cause",
			job:         plainJob(initTimeout),
			pod:         unschedulablePod,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (by the runner: the first step did not start before the initialization timeout of the workflow: no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
			wantReason:  string(testkube.StopReasonUnschedulable),
			wantDetails: "the first step did not start before the initialization timeout of the workflow: " + scheduler,
		},
		{
			name:        "gives only the recorded cause when a person canceled",
			job:         plainJob(userCancel),
			pod:         unschedulablePod,
			wantStatus:  testkube.CANCELED_TestWorkflowStepStatus,
			wantMessage: "The execution has been canceled. (by the user: no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
			wantReason:  string(testkube.StopReasonUserCancel),
			wantDetails: scheduler,
		},
		{
			name:        "gives the deadline of the job when the step holds no cause",
			job:         deadlineJob(nil),
			pod:         scheduledPod,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Job timed out after 60 seconds)",
			wantReason:  string(testkube.StopReasonDeadlineExceeded),
			wantDetails: "Job timed out after 60 seconds",
		},
		{
			name:        "gives the deadline of the job before the recorded cause",
			job:         deadlineJob(nil),
			pod:         unschedulablePod,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Job timed out after 60 seconds: no node can run the pod: 0/1 nodes are available: 1 Insufficient cpu)",
			wantReason:  string(testkube.StopReasonUnschedulable),
			wantDetails: "Job timed out after 60 seconds: " + scheduler,
		},
		{
			name:        "gives no message when nothing explains the stop",
			job:         plainJob(nil),
			pod:         scheduledPod,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Job has been aborted)",
			wantReason:  string(testkube.StopReasonUnknown),
			wantDetails: "",
		},
		{
			name:        "writes the waiting cause that only the final events of a deleted pod hold",
			job:         plainJob(nil),
			pod:         deletedPod,
			podEvents:   mountEvents,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Kubernetes cannot mount a volume of the pod: " + mountMessage + ")",
			wantReason:  string(testkube.StopReasonVolumeMountFailed),
			wantDetails: mountMessage,
		},
		{
			name:        "gives the deadline of the job before the waiting cause of the final events",
			job:         deadlineJob(nil),
			pod:         deletedPod,
			podEvents:   mountEvents,
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Job timed out after 60 seconds: Kubernetes cannot mount a volume of the pod: " + mountMessage + ")",
			wantReason:  string(testkube.StopReasonVolumeMountFailed),
			wantDetails: "Job timed out after 60 seconds: " + mountMessage,
		},
		{
			name:        "keeps a cause that another writer put into the step as it was written",
			job:         plainJob(nil),
			pod:         deletedPod,
			podEvents:   mountEvents,
			initMessage: "the image could not be pulled: not found",
			initReason:  string(testkube.StartReasonImagePullFailed),
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (the image could not be pulled: not found)",
			wantReason:  string(testkube.StartReasonImagePullFailed),
			wantDetails: "the image could not be pulled: not found",
		},
		{
			name: "forgets the cause after the container of the step started",
			job:  plainJob(nil),
			pod:  unschedulablePod,
			then: func(n *notifier) {
				n.Instruction(time.Now(), instructions.Instruction{Ref: initconstants.InitStepName, Name: initconstants.InstructionStart}, "exec-1")
			},
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Job has been aborted)",
			wantReason:  string(testkube.StopReasonUnknown),
			wantDetails: "",
		},
		{
			name: "forgets the cause after another writer cleared the message of the step",
			job:  plainJob(nil),
			pod:  unschedulablePod,
			then: func(n *notifier) {
				n.result.Initialization.ErrorMessage, n.result.Initialization.ErrorReason = "", ""
				n.Align(watchers.NewExecutionState(watchers.NewJob(plainJob(nil)), watchers.NewPod(scheduledPod), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil))
			},
			wantStatus:  testkube.ABORTED_TestWorkflowStepStatus,
			wantMessage: "The execution has been aborted. (Job has been aborted)",
			wantReason:  string(testkube.StopReasonUnknown),
			wantDetails: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newTestNotifier(t, testkube.RUNNING_TestWorkflowStepStatus, tt.initMessage)
			n.result.Initialization.ErrorReason = tt.initReason
			n.Align(watchers.NewExecutionState(watchers.NewJob(tt.job), watchers.NewPod(tt.pod), watchers.NewJobEvents(nil), watchers.NewPodEvents(tt.podEvents), nil))
			if tt.then != nil {
				tt.then(n)
			}

			n.End()

			assert.Equal(t, tt.wantStatus, *n.result.Initialization.Status)
			assert.Equal(t, tt.wantMessage, n.result.Initialization.ErrorMessage)
			require.NotNil(t, n.result.StatusDetails)
			assert.Equal(t, tt.wantReason, n.result.StatusDetails.Reason)
			assert.Equal(t, tt.wantDetails, n.result.StatusDetails.Message)
		})
	}
}

func TestNotifier_End_InitProcessCause(t *testing.T) {
	const ref = "rstep1"
	execution := func(details, reason string) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: initconstants.InstructionExecution, Value: initconstants.ExecutionResult{ExitCode: 1, Details: details, Reason: reason}}
	}
	end := instructions.Instruction{Ref: ref, Name: initconstants.InstructionEnd, Value: string(testkube.ABORTED_TestWorkflowStepStatus)}
	retry := instructions.Instruction{Ref: ref, Name: initconstants.InstructionIteration, Value: 1}

	tests := []struct {
		name        string
		hints       []instructions.Instruction
		wantMessage string
		wantDetails string
	}{
		{
			name:        "a code without details names the step and the phrase of the code",
			hints:       []instructions.Instruction{execution("", string(testkube.StopReasonStepTimeout)), end},
			wantMessage: "the step did not finish within its timeout",
			wantDetails: `The step "Run tests" did not finish within its timeout.`,
		},
		{
			name:        "details from the init process stay as the cause",
			hints:       []instructions.Instruction{execution("the test process was killed (signal: killed)", string(testkube.StopReasonProcessKilled)), end},
			wantMessage: "the test process was killed (signal: killed)",
			wantDetails: "the test process was killed (signal: killed)",
		},
		{
			name:        "later details replace a code without details",
			hints:       []instructions.Instruction{execution("", string(testkube.StopReasonStepTimeout)), execution("the step stopped: exit 137", string(testkube.StopReasonStepTimeout)), end},
			wantMessage: "the step stopped: exit 137",
			wantDetails: "the step stopped: exit 137",
		},
		{
			name:        "a retry forgets the code of the previous attempt",
			hints:       []instructions.Instruction{execution("", string(testkube.StopReasonStepTimeout)), retry, execution("the step stopped: exit 137", string(testkube.StopReasonProcessKilled)), end},
			wantMessage: "the step stopped: exit 137",
			wantDetails: "the step stopped: exit 137",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newTestNotifier(t, testkube.PASSED_TestWorkflowStepStatus, "")
			n.Align(watchers.NewExecutionState(watchers.NewJob(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1"}}), watchers.NewPod(&corev1.Pod{}), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil))
			// One container runs the step, so the notifier can read the order of the steps.
			n.sigSequence = []testkube.TestWorkflowSignature{{Ref: ref, Name: "Run tests"}}
			n.actions = make(actiontypes.ActionGroups, 1)
			n.endRefs = [][]string{{ref}}
			for _, hint := range tt.hints {
				n.Instruction(time.Now(), hint, "exec-1")
			}

			n.End()

			assert.Equal(t, tt.wantMessage, n.result.Steps[ref].ErrorMessage)
			require.NotNil(t, n.result.StatusDetails)
			assert.Equal(t, tt.wantDetails, n.result.StatusDetails.Message)
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
	executionWithReason := func(details, reason string) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: initconstants.InstructionExecution, Value: initconstants.ExecutionResult{ExitCode: 1, Details: details, Reason: reason}}
	}
	retry := func(iteration int) instructions.Instruction {
		return instructions.Instruction{Ref: ref, Name: initconstants.InstructionIteration, Value: iteration}
	}

	tests := []struct {
		name         string
		hints        []instructions.Instruction
		wantMessage  string
		wantReason   string
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
			name:         "writes the reason code that the init process sends with its message",
			hints:        []instructions.Instruction{executionWithReason(timeout, "step-timeout")},
			wantMessage:  timeout,
			wantReason:   "step-timeout",
			wantAttempts: 1,
		},
		{
			name:         "writes the sentence of the code that the init process sends without details",
			hints:        []instructions.Instruction{executionWithReason("", "step-timeout")},
			wantMessage:  timeout,
			wantReason:   "step-timeout",
			wantAttempts: 1,
		},
		{
			name:         "writes no reason code for a message that a command wrote",
			hints:        []instructions.Instruction{execution(0, "error cloning repository")},
			wantMessage:  "error cloning repository",
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
			assert.Equal(t, tt.wantReason, n.result.Steps[ref].ErrorReason)
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

// newBundledNotifier builds a notifier over a realistic pod from the processor. It deliberately
// does not use newTestNotifier: that helper seeds a "rstep1" step that bundledPod's signature
// never contains, and HealAbortedOrCanceled aborts unrecognised steps, which would decide the
// verdict before the initialization ever mattered.
func newBundledNotifier(t *testing.T, pod *corev1.Pod, job *batchv1.Job, scheduledAt time.Time) *notifier {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	n := newNotifier(ctx, testkube.TestWorkflowResult{}, scheduledAt)
	n.ch = make(chan ChannelMessage[Notification], 500)
	n.Align(watchers.NewExecutionState(
		watchers.NewJob(job), watchers.NewPod(pod),
		watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil))
	return n
}

// TestNotifier_End_RecoversInitializationWithoutHint covers the case where the initialization
// hint never reached the watcher but the container termination message did. Every step passed,
// so the execution must pass: an initialization with no hint must not decide the verdict alone.
func TestNotifier_End_RecoversInitializationWithoutHint(t *testing.T) {
	workflow := &testworkflowsv1.TestWorkflow{Spec: testworkflowsv1.TestWorkflowSpec{
		Steps: []testworkflowsv1.Step{{StepOperations: testworkflowsv1.StepOperations{Shell: "echo hello"}}},
	}}
	scheduled := time.Date(2026, 10, 26, 5, 16, 34, 0, time.UTC)
	startedAt := metav1.NewTime(scheduled.Add(5 * time.Second))
	finishedAt := metav1.NewTime(scheduled.Add(30 * time.Second))
	always := corev1.ContainerRestartPolicyAlways

	// The container ran every operation of its group and reported each as passed. The group holds
	// the initialization, the step, the root and the trailing ref, so the message carries four.
	passedAll := corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
		Reason: "Completed", Message: "p,0/p,0/p,0/p,0", StartedAt: startedAt, FinishedAt: finishedAt}}

	tests := []struct {
		name      string
		withHint  bool
		withIstio bool
	}{
		{name: "no initialization hint reached the watcher"},
		{name: "the initialization hint reached the watcher", withHint: true},
		{name: "an istio native sidecar is still running", withIstio: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, pod, _ := bundledPod(t, workflow, scheduled,
				corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: startedAt}})
			for i := range pod.Status.ContainerStatuses {
				pod.Status.ContainerStatuses[i].State = passedAll
			}
			for i := range pod.Status.InitContainerStatuses {
				pod.Status.InitContainerStatuses[i].State = passedAll
			}
			if tt.withIstio {
				// Istio injects its proxy as a native sidecar, so it is an init container that
				// keeps running after the step containers terminated and leaves the pod not ready.
				pod.Spec.InitContainers = append([]corev1.Container{
					{Name: "istio-proxy", RestartPolicy: &always, ReadinessProbe: &corev1.Probe{}},
				}, pod.Spec.InitContainers...)
				pod.Status.InitContainerStatuses = append([]corev1.ContainerStatus{{
					Name: "istio-proxy", Ready: false, Started: common.Ptr(true),
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: startedAt}},
				}}, pod.Status.InitContainerStatuses...)
				pod.Status.Conditions = []corev1.PodCondition{
					{Type: corev1.ContainersReady, Status: corev1.ConditionFalse, LastTransitionTime: finishedAt},
				}
			}
			pod.Status.Phase = corev1.PodRunning
			pod.Status.StartTime = &metav1.Time{Time: scheduled}

			n := newBundledNotifier(t, pod, job, scheduled)
			if tt.withHint {
				n.Instruction(finishedAt.Time, instructions.Instruction{
					Ref:   initconstants.InitStepName,
					Name:  initconstants.InstructionEnd,
					Value: string(testkube.PASSED_TestWorkflowStepStatus),
				}, "exec-1")
			}
			n.End()

			assert.Equal(t, testkube.PASSED_TestWorkflowStepStatus, *n.result.Initialization.Status,
				"the initialization carries a passed result in the termination message")
			for ref, step := range n.result.Steps {
				assert.Equal(t, testkube.PASSED_TestWorkflowStepStatus, *step.Status, "step %q", ref)
			}
			require.NotNil(t, n.result.Status)
			assert.Equal(t, testkube.PASSED_TestWorkflowStatus, *n.result.Status,
				"every step passed, so the execution passed")
		})
	}
}

// TestNotifier_End_InitializationRecovery covers how the finalisation resolves the initialization
// from the container termination message, including the shapes where it must not resolve at all.
func TestNotifier_End_InitializationRecovery(t *testing.T) {
	const ref = "rstep1"
	start := time.Date(2026, 10, 26, 5, 16, 34, 0, time.UTC)

	terminated := func(message string) corev1.ContainerState {
		return corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			Reason: "Completed", Message: message,
			StartedAt: metav1.NewTime(start), FinishedAt: metav1.NewTime(start.Add(time.Minute))}}
	}
	// withInit mirrors production: the setup action puts tktw-init first in the group.
	withInit := actiontypes.ActionGroups{{{Setup: &lite.ActionSetup{}}, {End: common.Ptr(ref)}}}
	withoutInit := actiontypes.ActionGroups{{{End: common.Ptr(ref)}}}

	tests := []struct {
		name          string
		initStatus    testkube.TestWorkflowStepStatus
		state         corev1.ContainerState
		actions       actiontypes.ActionGroups
		reconcileOnly bool
		wantStatus    testkube.TestWorkflowStepStatus
		wantExitCode  float64
	}{
		{
			// The init operation carries exit code 3 and the step 7, so a lookup that resolved to
			// the wrong ref would surface as the wrong exit code rather than passing silently.
			name:       "recovers the initialization and its own exit code",
			initStatus: testkube.RUNNING_TestWorkflowStepStatus,
			state:      terminated("p,3/p,7"), actions: withInit,
			wantStatus: testkube.PASSED_TestWorkflowStepStatus, wantExitCode: 3,
		},
		{
			name:       "keeps an initialization already passed by a hint",
			initStatus: testkube.PASSED_TestWorkflowStepStatus,
			state:      terminated("f,1/p,0"), actions: withInit,
			wantStatus: testkube.PASSED_TestWorkflowStepStatus,
		},
		{
			name:       "keeps an initialization already failed by a hint",
			initStatus: testkube.FAILED_TestWorkflowStepStatus,
			state:      terminated("p,0/p,0"), actions: withInit,
			wantStatus: testkube.FAILED_TestWorkflowStepStatus,
		},
		{
			// The init process writes the whole word for an operation it never finished, which the
			// single-character pattern rejects, so the parse stops before any status is recorded.
			name:       "does not recover the production abort shape",
			initStatus: testkube.RUNNING_TestWorkflowStepStatus,
			state:      terminated("aborted,137/p,0"), actions: withInit,
			wantStatus: testkube.ABORTED_TestWorkflowStepStatus,
		},
		{
			name:       "does not recover the coded abort shape",
			initStatus: testkube.RUNNING_TestWorkflowStepStatus,
			state:      terminated("a,1/p,0"), actions: withInit,
			wantStatus: testkube.ABORTED_TestWorkflowStepStatus,
		},
		{
			name:       "does not recover from a container that is still running",
			initStatus: testkube.RUNNING_TestWorkflowStepStatus,
			state:      corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(start)}},
			actions:    withInit,
			wantStatus: testkube.ABORTED_TestWorkflowStepStatus,
		},
		{
			// Without a setup action the init ref is absent from the maps. A bare lookup would read
			// as index 0 and take the step's result, so the status must stay untouched.
			name:       "takes no other step when the init ref is absent",
			initStatus: testkube.RUNNING_TestWorkflowStepStatus,
			state:      terminated("p,7"), actions: withoutInit,
			wantStatus: testkube.ABORTED_TestWorkflowStepStatus,
		},
		{
			name:       "leaves the initialization alone outside a forced finalisation",
			initStatus: testkube.RUNNING_TestWorkflowStepStatus,
			state:      terminated("p,3/p,7"), actions: withInit, reconcileOnly: true,
			wantStatus: testkube.RUNNING_TestWorkflowStepStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "1"}}},
				Status: corev1.PodStatus{
					Phase:             corev1.PodRunning,
					StartTime:         &metav1.Time{Time: start},
					ContainerStatuses: []corev1.ContainerStatus{{Name: "1", State: tt.state}},
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			n := newNotifier(ctx, testkube.TestWorkflowResult{
				Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(tt.initStatus)},
			}, start)
			n.ch = make(chan ChannelMessage[Notification], 500)
			n.Align(watchers.NewExecutionState(
				watchers.NewJob(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1"}}),
				watchers.NewPod(pod), watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil))
			n.useActionGroups(tt.actions)

			if tt.reconcileOnly {
				n.reconcile()
			} else {
				n.End()
			}

			require.NotNil(t, n.result.Initialization.Status)
			assert.Equal(t, tt.wantStatus, *n.result.Initialization.Status)
			assert.Equal(t, tt.wantExitCode, n.result.Initialization.ExitCode)
		})
	}
}

// TestNotifier_End_RecoveredInitializationStillAborts pins the blast radius of the recovery.
// A run that really was aborted mid-flight must still abort, and the cause must stay visible:
// once the initialization resolves as passed it no longer carries the termination message, so
// the message has to land on the first step that never finished.
func TestNotifier_End_RecoveredInitializationStillAborts(t *testing.T) {
	const ref = "rstep1"
	start := time.Date(2026, 10, 26, 5, 16, 34, 0, time.UTC)

	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "exec-1", Annotations: map[string]string{
		constants.AnnotationTerminationCode:   string(testkube.ABORTED_TestWorkflowStatus),
		constants.AnnotationTerminationActor:  string(testkube.StopActorRunner),
		constants.AnnotationTerminationReason: string(testkube.StopReasonInitTimeout),
	}}}
	// The setup finished and reported itself, then the run stopped before the step ended, so the
	// message holds one status only.
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "1"}}},
		Status: corev1.PodStatus{
			Phase:     corev1.PodRunning,
			StartTime: &metav1.Time{Time: start},
			ContainerStatuses: []corev1.ContainerStatus{{Name: "1", State: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					Reason: "Completed", Message: "p,0",
					StartedAt: metav1.NewTime(start), FinishedAt: metav1.NewTime(start.Add(time.Minute))},
			}}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	n := newNotifier(ctx, testkube.TestWorkflowResult{
		Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)},
	}, start)
	n.ch = make(chan ChannelMessage[Notification], 500)
	// The watch knows the signature before it aligns, so Align seeds the step results from it.
	n.sigSequence = []testkube.TestWorkflowSignature{{Ref: ref, Name: "Run tests"}}
	n.Align(watchers.NewExecutionState(
		watchers.NewJob(job), watchers.NewPod(pod),
		watchers.NewJobEvents(nil), watchers.NewPodEvents(nil), nil))
	n.useActionGroups(actiontypes.ActionGroups{{{Setup: &lite.ActionSetup{}}, {End: common.Ptr(ref)}}})

	n.End()

	assert.Equal(t, testkube.PASSED_TestWorkflowStepStatus, *n.result.Initialization.Status,
		"the setup reported itself, so it is recovered as passed")
	require.NotNil(t, n.result.Status)
	assert.Equal(t, testkube.ABORTED_TestWorkflowStatus, *n.result.Status,
		"the run was aborted before the step finished, so the execution still aborts")
	assert.Equal(t, testkube.ABORTED_TestWorkflowStepStatus, *n.result.Steps[ref].Status)

	require.NotNil(t, n.result.StatusDetails)
	assert.Equal(t, string(testkube.StopReasonInitTimeout), n.result.StatusDetails.Reason,
		"the reason code of the stop survives the recovery")

	// The message moves with the abort: a passed initialization no longer carries it.
	assert.Empty(t, n.result.Initialization.ErrorMessage,
		"a recovered initialization carries no termination message")
	assert.NotEmpty(t, n.result.Steps[ref].ErrorMessage,
		"the first unfinished step carries the termination message instead")
}
