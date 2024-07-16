package testworkflowcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

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
	InitContainerName = "tktw-init"
	IdleTimeout       = 100 * time.Millisecond
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
		if s.result.QueuedAt.IsZero() || s.result.StartedAt.IsZero() {
			s.Error(errors.New("missing information about scheduled pod"))
			return
		}

		// Load the namespace information
		podObj := <-pod.Peek(ctx)

		// Load the references
		refs := make([][]string, 0)
		var instructions [][]actiontypes.Action
		err := json.Unmarshal([]byte(podObj.Annotations[constants2.SpecAnnotationName]), &instructions)
		if err != nil {
			// TODO: Don't panic
			panic(fmt.Sprintf("invalid instructions: %v", err))
		} else {
			refs = make([][]string, len(instructions))
			for i := range instructions {
				for j := range instructions[i] {
					if instructions[i][j].Setup != nil {
						refs[i] = append(refs[i], InitContainerName)
					}
					if instructions[i][j].Start != nil && *instructions[i][j].Start != "" {
						refs[i] = append(refs[i], *instructions[i][j].Start)
					}
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

			// Watch the container events
			for v := range state.PreStart(containerName) {
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
			if s.GetStepResult(initialRef).QueuedAt.IsZero() || s.GetStepResult(initialRef).StartedAt.IsZero() {
				s.Error(fmt.Errorf("missing information about scheduled '%s' step in '%s' container", initialRef, container.Name))
				return
			}

			// Watch the container logs
			follow := common.ResolvePtr(opts.Follow, true) && !state.IsFinished(containerName)
			lastStarted := initialRef
			executionStatuses := map[string]constants.ExecutionResult{}
			for v := range WatchContainerLogs(ctx, clientSet, podObj.Namespace, podObj.Name, containerName, follow, 10, pod).Channel() {
				if v.Error != nil {
					s.Error(v.Error)
				} else if v.Value.Output != nil {
					s.Output(v.Value.Output.Ref, v.Value.Time, v.Value.Output)
				} else if v.Value.Hint != nil {
					switch v.Value.Hint.Name {
					case constants.InstructionStart:
						lastStarted = v.Value.Hint.Ref
						s.Start(v.Value.Hint.Ref, v.Value.Time)
					case constants.InstructionEnd:
						status := testkube.TestWorkflowStepStatus(v.Value.Hint.Value.(string))
						if status == "" {
							status = testkube.PASSED_TestWorkflowStepStatus
						}
						if v.Value.Hint.Ref == InitContainerName {
							s.finishInit(ContainerResult{
								Status:     status,
								Details:    executionStatuses[v.Value.Hint.Ref].Details,
								ExitCode:   int(executionStatuses[v.Value.Hint.Ref].ExitCode),
								FinishedAt: v.Value.Time,
							})
						} else {
							s.FinishStep(v.Value.Hint.Ref, ContainerResult{
								Status:     status,
								Details:    executionStatuses[v.Value.Hint.Ref].Details,
								ExitCode:   int(executionStatuses[v.Value.Hint.Ref].ExitCode),
								FinishedAt: v.Value.Time,
							})
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
				} else {
					s.Raw(lastStarted, v.Value.Time, string(v.Value.Log), false)
				}
			}

			// TODO? - use the Termination Log as fall back
			//// Get the final result
			//if follow {
			//	<-state.Finished(lastStarted)
			//} else {
			//	select {
			//	case <-state.Finished(lastStarted):
			//	case <-time.After(IdleTimeout):
			//		return
			//	}
			//}
			//status, err := state.ContainerResult(containerIndex)
			//if err != nil {
			//	s.Error(err)
			//	break
			//}
			//s.FinishStep(containerIndex, status)

			// Update the last timestamp
			lastTs = s.GetLastTimestamp(lastStarted)

			// TODO
			//// Break the function if the step has been aborted.
			//// Breaking only to the loop is not enough,
			//// because due to GKE bug, the Job is still pending,
			//// so it will get stuck there.
			//if status.Status == testkube.ABORTED_TestWorkflowStepStatus {
			//	if status.Details == "" {
			//		status.Details = "Manual"
			//	}
			//	s.Raw(containerIndex, s.GetLastTimestamp(containerIndex), fmt.Sprintf("\n%s Aborted (%s)", s.GetLastTimestamp(containerIndex).Format(KubernetesLogTimeFormat), status.Details), false)
			//	break
			//}
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
