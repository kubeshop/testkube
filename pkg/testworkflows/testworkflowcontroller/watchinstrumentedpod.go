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
)

const (
	InitContainerName = "tktw-init"
)

type WatchInstrumentedPodOptions struct {
	DisableFollow bool
}

//// FIXME: DEBUG
//// FIXME: DEBUG
//// FIXME: DEBUG
//// FIXME: DEBUG
//// FIXME: DEBUG
//// FIXME: DEBUG
//type currentList struct {
//	mu   sync.Mutex
//	list []string
//}
//
//func (c *currentList) add(name string) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	c.list = append(c.list, name)
//	fmt.Println("ADD  watching:", strings.Join(c.list, ", "))
//}
//func (c *currentList) del(name string) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	i := 0
//	for i = 0; i < len(c.list); i++ {
//		if c.list[i] == name {
//			c.list = append(c.list[0:i], c.list[i+1:]...)
//			fmt.Println("DEL  watching:", strings.Join(c.list, ", "))
//			return
//		}
//	}
//}
//func (c *currentList) print() {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	fmt.Println("PING watching:", strings.Join(c.list, ", "))
//}
//
//var debug = iterate()
//
//func iterate() *currentList {
//	c := &currentList{}
//
//	go func() {
//		for {
//			time.Sleep(10 * time.Second)
//			c.print()
//		}
//	}()
//	return c
//}

func WatchInstrumentedPod(parentCtx context.Context, clientSet kubernetes.Interface, signature []stage.Signature, scheduledAt time.Time, watcher watchers.ExecutionWatcher, opts WatchInstrumentedPodOptions) (<-chan ChannelMessage[Notification], error) {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	notifier := newNotifier(ctx, signature, scheduledAt)
	signatureSeq := stage.MapSignatureToSequence(signature)

	log := func(data ...interface{}) {
		// FIXME delete?
		//data = append([]interface{}{ui.Green(watcher.State().Job().ResourceId())}, data...)
		//fmt.Println(data...)
	}

	go func() {
		//debug.add(watcher.State().ResourceId()) FIXME
		defer func() {
			if r := recover(); r != nil {
				notifier.Error(fmt.Errorf("fatal error watching data: %v", r))
			}

			//debug.del(watcher.State().ResourceId()) FIXME
			notifier.Finalize()
			notifier.Flush()
			ctxCancel()
			log("closed")
		}()

		// Mark Job as started
		notifier.Queue("", watcher.State().EstimatedJobCreationTimestamp())
		log("queued")

		// Wait until the Pod is scheduled
		currentJobEventsIndex := 0
		for ok := true; ok; _, ok = <-watcher.Updated() {
			log("checking for scheduled pod")
			for _, ev := range watcher.State().JobEvents().Original()[currentJobEventsIndex:] {
				log("reading event", ev)
				if ev.Reason != "BackoffLimitExceeded" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
				log("finished reading event", ev)
			}
			log("job events read")

			if watcher.State().PodCreated() || watcher.State().Completed() {
				break
			}
			log("checking for scheduled pod: iteration end")
		}
		log("pod is scheduled")

		// Wait until the Pod is started
		currentPodEventsIndex := 0
		for ok := true; ok; _, ok = <-watcher.Updated() {
			log("waiting for started pod")
			// TODO: Watch the Job events too still?
			for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				name := GetEventContainerName(ev)
				if name == "" { // TODO: name == "1"?
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}

			if watcher.State().PodStarted() || watcher.State().Completed() {
				break
			}
			log("waiting for started pod: iteration end")
		}
		log("pod likely started")

		// Load the pod information
		// TODO: when it's complete without pod start, try to check if maybe job was not aborted etc
		if watcher.State().EstimatedPodStartTimestamp().IsZero() {
			log("cannot estimate pod start")
			notifier.Error(fmt.Errorf("cannot estimate Pod start"))
			return
		}

		notifier.Start("", watcher.State().EstimatedPodStartTimestamp())
		log("pod started")

		// Read the execution instructions
		actions, err := watcher.State().ActionGroups()
		if err != nil {
			// FIXME:
			notifier.Error(fmt.Errorf("cannot read execution instructions: %v", err))
			return
		}
		refs, endRefs := ExtractRefsFromActionGroup(actions)
		initialRefs := make([]string, len(actions))
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

			// Wait until the Container is started
			currentPodEventsIndex = 0
			for ok := true; ok; _, ok = <-watcher.Updated() {
				log("waiting for container start", container)

				// Read the Pod Events for the Container Events
				for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
					currentPodEventsIndex++

					// Display only events that are unrelated to further containers
					// TODO: Try to attach it to first recognizable step?
					name := GetEventContainerName(ev)
					if name == container {
						//if name == container && ev.Reason != "Created" && ev.Reason != "Started" {
						notifier.Event(initialRefs[containerIndex], watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
					}
				}

				// Determine if the container should be already accessible
				if watcher.State().ContainerStarted(container) || watcher.State().Completed() {
					break
				}

				log("waiting for container start: iteration end")
			}
			log("container started", container)

			// Start the initial one
			lastStarted = refs[containerIndex][0]

			// Read the Container logs
			isDone := func() bool {
				return opts.DisableFollow || watcher.State().ContainerFinished(container) || watcher.State().Completed()
			}
			for v := range WatchContainerLogs(ctx, clientSet, watcher.State().Namespace(), watcher.State().PodName(), container, 10, isDone).Channel() {
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
			for ok := true; ok; _, ok = <-watcher.Updated() {
				log("waiting for terminated container", container)

				// Determine if the container should be already stopped
				if watcher.State().ContainerFinished(container) || watcher.State().Completed() {
					break
				}

				log("waiting for terminated container: iteration end", container)
			}
			log("container terminated", container)

			// Load the correlation data about status
			// TODO: Should not wait for the actual container result?
			result := watcher.State().MustEstimatedPod().ContainerResult(container, watcher.State().Job().ExecutionError())
			log("container result", container, result)

			for i, ref := range endRefs[containerIndex] {
				if ref == "root" {
					continue
				}

				// TODO: Avoid passing that information?
				finishedAt := notifier.GetStepResult(ref).FinishedAt
				if finishedAt.IsZero() {
					finishedAt = watcher.State().MustEstimatedPod().ContainerFinishTimestamp(container)
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
			if watcher.State().Completed() {
				break loop
			}

			select {
			case _, ok := <-watcher.Updated():
				if !ok || watcher.State().Completed() {
					break loop
				}
			case <-time.After(30 * time.Second):
				// Fallback in case of missing data
				if watcher.State().Completed() {
					break loop
				}
				// TODO: shouldn't be just a critical error?
			}
		}

		// Mark as finished
		// TODO: Calibrate with top timestamp?
		notifier.Finish(watcher.State().CompletionTimestamp())
	}()

	return notifier.ch, nil
}
