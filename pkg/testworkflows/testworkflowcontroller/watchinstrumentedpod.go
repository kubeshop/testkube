package testworkflowcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/gookit/color"
	"k8s.io/client-go/kubernetes"

	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	InitContainerName = "tktw-init"
)

type WatchInstrumentedPodOptions struct {
	DisableFollow bool
}

func WatchInstrumentedPod(parentCtx context.Context, clientSet kubernetes.Interface, signature []stage.Signature, scheduledAt time.Time, watcher watchers.ExecutionWatcher, opts WatchInstrumentedPodOptions) (<-chan ChannelMessage[Notification], error) {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	notifier := newNotifier(ctx, signature, scheduledAt)
	state := NewExecutionState(ctx, watcher)
	signatureSeq := stage.MapSignatureToSequence(signature)

	log := func(data ...interface{}) {
		// FIXME delete?
		data = append([]interface{}{ui.Green(state.Job().Name)}, data...)
		fmt.Println(data...)
	}

	go func() {
		defer func() {
			notifier.Finalize()
			notifier.Flush()
			ctxCancel()
			log("closed")
		}()

		// TODO: Think how to get rid of that, thanks to past TestWorkflowResult
		// Wait for the Job
		for ok := true; ok; _, ok = <-state.Updated() {
			if state.Job() != nil {
				break
			}
		}

		// Mark Job as started
		notifier.Queue("", watcher.JobCreationTimestamp(true))
		log("queued")

		// Wait until the Pod is scheduled
		currentJobEventsIndex := 0
		for ok := true; ok; _, ok = <-state.Updated() {
			log("checking for scheduled pod")
			for _, ev := range state.JobEvents()[currentJobEventsIndex:] {
				currentJobEventsIndex++
				if ev.Reason != "BackoffLimitExceeded" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}

			// Continue if the Pod has been scheduled
			if state.Pod() != nil {
				break
			}

			// Determine if the job is not failed already without the Pod
			if watchers.IsJobFinished(state.Job()) {
				break
			}
			log("checking for scheduled pod: iteration end")
		}
		log("pod is scheduled")

		// Wait until the Pod is started
		currentPodEventsIndex := 0
		for ok := true; ok; _, ok = <-state.Updated() {
			log("waiting for started pod")
			for _, ev := range state.PodEvents()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				name := GetEventContainerName(ev)
				if name == "" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}

			// Continue if the Pod has been started
			isPodStarted := state.Pod() != nil && watchers.IsPodStarted(state.Pod())
			if isPodStarted {
				break
			}

			// Determine if the pod is not failed without starting
			if watchers.IsPodFinished(state.Pod()) {
				break
			}

			// Determine if the job is not failed already without the Pod
			if watchers.IsJobFinished(state.Job()) {
				break
			}
			log("waiting for started pod: iteration end")
		}
		log("pod likely started")

		// Load the pod information
		if watcher.PodCreationTimestamp(true).IsZero() {
			log("no pod creation time found")
			notifier.Error(fmt.Errorf("pod is not there"))
			return
		}

		notifier.Start("", watcher.PodCreationTimestamp(true))
		log("pod started")

		// Read the execution instructions
		actions, err := state.ActionGroups()
		if err != nil {
			// FIXME:
			notifier.Error(fmt.Errorf("cannot read execution instructions: %v", err))
			return
		}
		refs := make([][]string, len(actions))
		initialRefs := make([]string, len(actions))
		endRefs := make([][]string, len(actions))
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
		for i := range refs {
			for j := range refs[i] {
				if refs[i][j] == InitContainerName {
					initialRefs[i] = ""
					break
				}
				if slices.ContainsFunc(signatureSeq, func(sig stage.Signature) bool {
					return len(sig.Children()) == 0
				}) {
					initialRefs[i] = refs[i][j]
					break
				}
			}
		}

		// Iterate over containers
		aborted := false
		registeredAbortedOperation := false
		lastStarted := InitContainerName
		executionStatuses := map[string]constants2.ExecutionResult{}
		for containerIndex := 0; containerIndex < len(refs); containerIndex++ {
			container := fmt.Sprintf("%d", containerIndex+1)
			log("iterating containers", container)

			// Read the Pod Events for the Container Events
			for _, ev := range state.PodEvents()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				// TODO: Try to attach it to first recognizable step?
				name := GetEventContainerName(ev)
				if name == container {
					//if name == container && ev.Reason != "Created" && ev.Reason != "Started" {
					notifier.Event(initialRefs[containerIndex], watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}

			// Wait until the Container is started
			currentPodEventsIndex = 0
			for ok := true; ok; _, ok = <-state.Updated() {
				log("waiting for container start", container)

				// Determine if the container should be already accessible
				if watchers.IsContainerStarted(state.Pod(), container) || watchers.IsContainerFinished(state.Pod(), container) {
					break
				}

				// Determine if the job is not failed already without the Pod
				if watchers.IsJobFinished(state.Job()) {
					break
				}

				log("waiting for container start: iteration end", container)
			}
			log("container started", container)

			// Start the initial one
			lastStarted = refs[containerIndex][0]

			// Read the Container logs
			isDone := func() bool {
				return opts.DisableFollow || watcher.PodFinished() || watcher.JobFinished()
			}
			for v := range WatchContainerLogs(ctx, clientSet, watcher.Namespace(), watcher.PodName(), container, 10, isDone).Channel() {
				if v.Error != nil {
					log("container error", container, v.Error)
					notifier.Error(v.Error)
					continue
				}
				log("container log", container)

				switch v.Value.Type() {
				case ContainerLogTypeLog:
					notifier.Raw(lastStarted, v.Value.Time, string(v.Value.Log), false)
				case ContainerLogTypeOutput:
					notifier.Output(v.Value.Output.Ref, v.Value.Time, v.Value.Output)
				case ContainerLogTypeHint:
					if v.Value.Hint.Ref == constants.RootOperationName {
						continue
					}
					switch v.Value.Hint.Name {
					case constants2.InstructionStart:
						lastStarted = v.Value.Hint.Ref
						if !aborted {
							notifier.Start(v.Value.Hint.Ref, v.Value.Time)
						}
					case constants2.InstructionEnd:
						status := testkube.TestWorkflowStepStatus(v.Value.Hint.Value.(string))
						if status == "" {
							status = testkube.PASSED_TestWorkflowStepStatus
						}
						if !aborted {
							notifier.FinishStep(v.Value.Hint.Ref, ContainerResultStep{
								Status:     status,
								Details:    executionStatuses[v.Value.Hint.Ref].Details,
								ExitCode:   int(executionStatuses[v.Value.Hint.Ref].ExitCode),
								FinishedAt: v.Value.Time,
							})
						}
						if status == testkube.ABORTED_TestWorkflowStepStatus {
							aborted = true
							continue
						}
					case constants2.InstructionExecution:
						serialized, _ := json.Marshal(v.Value.Hint.Value)
						var executionResult constants2.ExecutionResult
						_ = json.Unmarshal(serialized, &executionResult)
						executionStatuses[v.Value.Hint.Ref] = executionResult
					case constants2.InstructionPause:
						ts, _ := v.Value.Hint.Value.(string)
						start, err := time.Parse(constants2.PreciseTimeFormat, ts)
						if err != nil {
							start = v.Value.Time
							notifier.Error(fmt.Errorf("invalid timestamp provided with pausing instruction: %v", v.Value.Hint.Value))
						}
						notifier.Pause(v.Value.Hint.Ref, start)
					case constants2.InstructionResume:
						ts, _ := v.Value.Hint.Value.(string)
						end, err := time.Parse(constants2.PreciseTimeFormat, ts)
						if err != nil {
							end = v.Value.Time
							notifier.Error(fmt.Errorf("invalid timestamp provided with resuming instruction: %v", v.Value.Hint.Value))
						}
						notifier.Resume(v.Value.Hint.Ref, end)
					}
				}
			}
			log("container log finished", container)

			// Wait until the Container is terminated
			for ok := true; ok; _, ok = <-state.Updated() {
				log("waiting for terminated container", container)
				// Determine if the container should be already stopped
				if watchers.IsContainerFinished(state.Pod(), container) {
					break
				}

				// Determine if the pod is not failed already without the container stopped
				if watchers.IsPodFinished(state.Pod()) {
					break
				}

				// Determine if the job is not failed already without the Pod
				if watchers.IsJobFinished(state.Job()) {
					break
				}
				log("waiting for terminated container: iteration end", container)
			}
			log("container terminated", container)

			// Load the correlation data about status
			status := watchers.GetContainerStatus(state.Pod(), container)
			result := watchers.ReadContainerResult(status, watcher.ExecutionError())
			log("container result", container, status, result)

			for i, ref := range endRefs[containerIndex] {
				if ref == "root" {
					continue
				}

				// TODO: Avoid passing that information?
				finishedAt := notifier.GetStepResult(ref).FinishedAt
				if finishedAt.IsZero() && status != nil && status.State.Terminated != nil {
					finishedAt = status.State.Terminated.FinishedAt.Time
				}

				// Handle available result
				if len(result.Statuses) > i {
					// Send information about step finish
					notifier.FinishStep(ref, ContainerResultStep{
						Status:     result.Statuses[i].Status,
						ExitCode:   result.Statuses[i].ExitCode,
						Details:    "",
						FinishedAt: finishedAt,
					})
					continue
				}

				// Ignore when there is already the status available
				registeredStatus := notifier.GetStepResult(ref).Status
				if registeredStatus != nil && registeredStatus.Finished() {
					continue
				}

				// Handle missing result - first aborted task
				if !registeredAbortedOperation {
					registeredAbortedOperation = true

					details := "The execution has been aborted."
					if result.ErrorDetails != "" {
						details = fmt.Sprintf("The execution has been aborted. %s", color.FgDarkGray.Render("("+result.ErrorDetails+")"))
					}

					notifier.FinishStep(ref, ContainerResultStep{
						Status:     testkube.ABORTED_TestWorkflowStepStatus,
						ExitCode:   -1,
						Details:    details,
						FinishedAt: finishedAt,
					})
					continue
				}

				// Handle missing result - after aborted task
				// TODO: Consider if that should be displayed
				details := "The execution was aborted before."
				if result.ErrorDetails != "" {
					details = fmt.Sprintf("The execution was aborted before. %s", color.FgDarkGray.Render("("+result.ErrorDetails+")"))
				}
				notifier.FinishStep(ref, ContainerResultStep{
					Status:     testkube.SKIPPED_TestWorkflowStepStatus,
					ExitCode:   -1,
					Details:    details,
					FinishedAt: finishedAt,
				})
			}
		}
		log("finished iterating over containers")

		// Wait until everything is finished
	loop:
		for {
			if watchers.IsPodFinished(state.Pod()) {
				break loop
			}
			if watchers.IsJobFinished(state.Job()) {
				break loop
			}

			select {
			case _, ok := <-state.Updated():
				if !ok {
					break loop
				}
				if watchers.IsPodFinished(state.Pod()) {
					break loop
				}
				if watchers.IsJobFinished(state.Job()) {
					break loop
				}
			case <-time.After(3 * time.Second):
				// Fallback in case of missing data
				if watcher.PodFinished() || watcher.JobFinished() {
					break loop
				}
			}
		}

		// Mark as finished
		// TODO: Calibrate with top timestamp?
		notifier.Finish(watcher.CompletionTimestamp())
	}()

	return notifier.ch, nil
}
