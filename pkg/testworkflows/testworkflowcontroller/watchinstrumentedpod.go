package testworkflowcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gookit/color"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	constants2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	InitContainerName    = "tktw-init"
	IdleTimeout          = 100 * time.Millisecond
	ExpectedDelayTimeout = 1 * time.Second
)

type WatchInstrumentedPodOptions struct {
	JobEvents Channel[*corev1.Event]
	Job       Channel[*batchv1.Job]
	Follow    *bool
}

func WatchInstrumentedPod(parentCtx context.Context, clientSet kubernetes.Interface, signature []stage.Signature, scheduledAt time.Time, pod Channel[*corev1.Pod], podEvents Channel[*corev1.Event], opts WatchInstrumentedPodOptions) (<-chan ChannelMessage[Notification], error) {
	// Avoid missing data
	if pod == nil {
		return nil, errors.New("pod watcher is required")
	}

	// Initialize controller state
	ctx, ctxCancel := context.WithCancel(parentCtx)
	s := newNotifier(ctx, signature, scheduledAt)

	// Initialize pod state
	state := initializePodState(ctx, pod, podEvents, opts.Job, opts.JobEvents, s.Error)

	// Start watching
	go func() {
		defer func() {
			s.Flush()
			ctxCancel()
		}()

		// Watch for the basic initialization warnings
		for v := range state.PreStart("") {
			if v.Value.Queued != nil {
				s.Queue("", state.QueuedAt(""))
			} else if v.Value.Started != nil {
				s.Queue("", state.QueuedAt(""))
				s.Start("", state.StartedAt(""))
			} else if v.Value.Event != nil {
				ts := maxTime(v.Value.Event.CreationTimestamp.Time, v.Value.Event.FirstTimestamp.Time, v.Value.Event.LastTimestamp.Time)
				s.Event("", ts, v.Value.Event.Type, v.Value.Event.Reason, v.Value.Event.Message)
			}
		}

		// Ensure the queue/start time has been saved
		if (s.result.QueuedAt.IsZero() || s.result.StartedAt.IsZero()) && state.FinishedAt("").IsZero() {
			s.Error(errors.New("missing information about scheduled pod"))
			return
		}

		// Load the namespace information
		var podObj *corev1.Pod
		select {
		// Obtain the Pod information for further execution
		case p := <-pod.Peek(ctx):
			podObj = p
		// Handle when the execution have been finished, but there may be no Pod
		case <-state.Finished(""):
			select {
			case <-time.After(ExpectedDelayTimeout):
				s.Error(fmt.Errorf("the execution is finished, but failed to get pod"))
				return
			case p := <-pod.Peek(ctx):
				podObj = p
			}
		}

		// Load the references
		var refs, endRefs [][]string
		var actions actiontypes.ActionGroups
		err := json.Unmarshal([]byte(podObj.Annotations[constants2.SpecAnnotationName]), &actions)
		if err != nil {
			s.Error(fmt.Errorf("invalid instructions: %v", err))
			return
		}
		refs = make([][]string, len(actions))
		endRefs = make([][]string, len(actions))
		for i := range actions {
			for j := range actions[i] {
				if actions[i][j].Setup != nil {
					refs[i] = append(refs[i], InitContainerName)
					endRefs[i] = append(endRefs[i], InitContainerName)
				}
				if actions[i][j].Start != nil && *actions[i][j].Start != "" {
					refs[i] = append(refs[i], *actions[i][j].Start)
				}
				if actions[i][j].End != nil && *actions[i][j].End != "" {
					endRefs[i] = append(endRefs[i], *actions[i][j].End)
				}
			}
		}

		// For each container:
		lastTs := s.result.Initialization.FinishedAt
		for _, container := range append(podObj.Spec.InitContainers, podObj.Spec.Containers...) {
			// Ignore non-standard TestWorkflow containers
			number, err := strconv.Atoi(container.Name)
			if err != nil || number > len(refs) {
				continue
			}
			index := number - 1
			containerName := container.Name
			initialRef := refs[index][0]

			// Update queue time
			s.Queue(initialRef, lastTs)

			// Watch the container events, along with final finish too
			preStartCh := state.PreStart(containerName)
			finishedCh := state.Finished("")
		loop:
			for {
				var v ChannelMessage[podStateUpdate]
				select {
				case vv, ok := <-preStartCh:
					if !ok {
						break loop
					}
					v = vv
				default:
					select {
					case vv, ok := <-preStartCh:
						if !ok {
							break loop
						}
						v = vv
					case <-finishedCh:
						break loop
					}
				}

				if v.Value.Queued != nil {
					s.Queue(initialRef, state.QueuedAt(containerName))
				} else if v.Value.Started != nil {
					s.Queue(initialRef, state.QueuedAt(containerName))
					s.Start(initialRef, state.StartedAt(containerName))
				} else if v.Value.Event != nil {
					ts := maxTime(v.Value.Event.CreationTimestamp.Time, v.Value.Event.FirstTimestamp.Time, v.Value.Event.LastTimestamp.Time)
					s.Event(initialRef, ts, v.Value.Event.Type, v.Value.Event.Reason, v.Value.Event.Message)
				}
			}

			// Ensure the queue/start time has been saved
			if (s.GetStepResult(initialRef).QueuedAt.IsZero() || s.GetStepResult(initialRef).StartedAt.IsZero()) && state.FinishedAt("").IsZero() {
				s.Error(fmt.Errorf("missing information about scheduled '%s' step in '%s' container", initialRef, container.Name))
				return
			}

			// Watch the container logs
			follow := common.ResolvePtr(opts.Follow, true) && !state.IsFinished(containerName) && !state.IsFinished("")
			aborted := false
			lastStarted := initialRef
			executionStatuses := map[string]constants.ExecutionResult{}
			for v := range WatchContainerLogs(ctx, clientSet, podObj.Namespace, podObj.Name, containerName, follow, 10, pod).Channel() {
				if v.Error != nil {
					s.Error(v.Error)
				}

				switch v.Value.Type() {
				case ContainerLogTypeLog:
					s.Raw(lastStarted, v.Value.Time, string(v.Value.Log), false)
				case ContainerLogTypeOutput:
					s.Output(v.Value.Output.Ref, v.Value.Time, v.Value.Output)
				case ContainerLogTypeHint:
					if v.Value.Hint.Ref == constants2.RootOperationName {
						continue
					}
					switch v.Value.Hint.Name {
					case constants.InstructionStart:
						lastStarted = v.Value.Hint.Ref
						s.Start(v.Value.Hint.Ref, v.Value.Time)
					case constants.InstructionEnd:
						status := testkube.TestWorkflowStepStatus(v.Value.Hint.Value.(string))
						if status == "" {
							status = testkube.PASSED_TestWorkflowStepStatus
						}
						s.FinishStep(v.Value.Hint.Ref, ContainerResultStep{
							Status:     status,
							Details:    executionStatuses[v.Value.Hint.Ref].Details,
							ExitCode:   int(executionStatuses[v.Value.Hint.Ref].ExitCode),
							FinishedAt: v.Value.Time,
						})

						// Escape when the job was aborted
						if status == testkube.ABORTED_TestWorkflowStepStatus {
							aborted = true
							break
						}
					case constants.InstructionExecution:
						serialized, _ := json.Marshal(v.Value.Hint.Value)
						var executionResult constants.ExecutionResult
						_ = json.Unmarshal(serialized, &executionResult)
						executionStatuses[v.Value.Hint.Ref] = executionResult
					case constants.InstructionPause:
						ts, _ := v.Value.Hint.Value.(string)
						start, err := time.Parse(constants.PreciseTimeFormat, ts)
						if err != nil {
							start = v.Value.Time
							s.Error(fmt.Errorf("invalid timestamp provided with pausing instruction: %v", v.Value.Hint.Value))
						}
						s.Pause(v.Value.Hint.Ref, start)
					case constants.InstructionResume:
						ts, _ := v.Value.Hint.Value.(string)
						end, err := time.Parse(constants.PreciseTimeFormat, ts)
						if err != nil {
							end = v.Value.Time
							s.Error(fmt.Errorf("invalid timestamp provided with resuming instruction: %v", v.Value.Hint.Value))
						}
						s.Resume(v.Value.Hint.Ref, end)
					}
				}
			}

			if aborted {
				// Don't wait for any other statuses if we already know that some task has been aborted
			} else if follow {
				select {
				case <-state.Finished(container.Name):
				case <-state.Finished(""):
					// Finish fast when the whole execution has been finished
				}
			} else {
				select {
				case <-state.Finished(container.Name):
				case <-state.Finished(""):
					// Finish fast when the whole execution has been finished
				case <-time.After(IdleTimeout):
					return
				}
			}

			// Fall back results to the termination log
			if !aborted {
				result, err := state.ContainerResult(container.Name)
				if err != nil {
					s.Error(err)
					break
				}

				for i, ref := range endRefs[index] {
					// Ignore tree root hints
					if ref == "root" {
						continue
					}
					status := ContainerResultStep{
						Status:     testkube.ABORTED_TestWorkflowStepStatus,
						ExitCode:   -1,
						Details:    result.Details,
						FinishedAt: s.GetStepResult(ref).FinishedAt,
					}
					if status.FinishedAt.IsZero() {
						status.FinishedAt = result.FinishedAt
					}
					if status.FinishedAt.IsZero() {
						status.FinishedAt = state.FinishedAt("")
					}
					if status.FinishedAt.IsZero() {
						status.FinishedAt = s.GetLastTimestamp(lastStarted)
					}

					if len(result.Steps) > i {
						status = result.Steps[i]
						if status.Details == "" {
							status.Details = result.Details
						}
						finishedAt := s.GetStepResult(ref).FinishedAt
						if !finishedAt.IsZero() {
							status.FinishedAt = finishedAt
						}
					}
					s.FinishStep(ref, status)
					if status.Status == testkube.ABORTED_TestWorkflowStepStatus {
						lastStarted = ref
						break
					}
				}
			}

			// Update the last timestamp
			nextLastTs := s.GetLastTimestamp(lastStarted)
			if nextLastTs.After(lastTs) {
				lastTs = nextLastTs
			}

			// Break the function if the step has been aborted.
			// Breaking only to the loop is not enough,
			// because due to GKE bug, the Job is still pending,
			// so it will get stuck there.
			if s.IsAnyAborted() {
				result, _ := state.ContainerResult(container.Name)
				reason := s.result.Steps[lastStarted].ErrorMessage
				if reason == "" {
					reason = result.Details
				}
				message := "Aborted"
				if reason == "" {
					message = fmt.Sprintf("\n%s Aborted", lastTs.Format(KubernetesLogTimeFormat))
				} else {
					message = fmt.Sprintf("\n%s Aborted (%s)", lastTs.Format(KubernetesLogTimeFormat), reason)
				}
				s.Raw(lastStarted, lastTs, message, false)

				// Mark all not started steps as skipped
				for ref := range s.result.Steps {
					if !s.IsFinished(ref) {
						status := testkube.SKIPPED_TestWorkflowStepStatus
						details := "The execution was aborted before"
						if s.result.Steps[ref].Status != nil && *s.result.Steps[ref].Status != testkube.QUEUED_TestWorkflowStepStatus {
							status = testkube.ABORTED_TestWorkflowStepStatus
							details = result.Details
						} else if result.Details != "" {
							details = fmt.Sprintf("The execution was aborted before %s", color.FgDarkGray.Render("("+result.Details+")"))
						}
						if details != "" {
							s.Raw(ref, lastTs, fmt.Sprintf("%s %s", lastTs.Format(KubernetesLogTimeFormat), details), false)
						}
						s.FinishStep(ref, ContainerResultStep{
							Status:     status,
							ExitCode:   -1,
							Details:    "",
							FinishedAt: lastTs,
						})
					}
				}

				break
			}
		}

		// Watch the completion time
		if s.result.FinishedAt.IsZero() {
			<-state.Finished("")
			f := state.FinishedAt("")
			s.Finish(f)
		}
	}()

	return s.ch, nil
}

func maxTime(times ...time.Time) time.Time {
	var result time.Time
	for _, t := range times {
		if t.After(result) {
			result = t
		}
	}
	return result
}
