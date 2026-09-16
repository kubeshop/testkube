package controller

import (
	"context"
	"fmt"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	watchers2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	ForceFinalizationDelay = 30 * time.Second
)

type WatchInstrumentedPodOptions struct {
	DisableFollow       bool
	LogAbortedDetails   bool
	ContainerLogOptions ContainerLogOptions
}

func WatchInstrumentedPod(parentCtx context.Context, clientSet kubernetes.Interface, signature []stage.Signature, scheduledAt time.Time, watcher watchers2.ExecutionWatcher, opts WatchInstrumentedPodOptions) (<-chan ChannelMessage[Notification], error) {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	notifier := newNotifier(ctx, testkube.TestWorkflowResult{}, scheduledAt)
	signatureSeq := stage.MapSignatureToSequence(signature)
	executionId := getExecutionId(watcher.State())

	updatesCh := watcher.Updated(ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				notifier.Error(fmt.Errorf("fatal error watching data: %v", r))
			}

			notifier.Align(watcher.State())

			if ctx.Err() != nil {
				log.DefaultLogger.Warnw("canceled watching execution", "executionId", watcher.State().ResourceId(), "err", ctx.Err(), "debug", watcher.State().Debug())
				close(notifier.ch)
				return
			}

			if !watcher.State().Completed() {
				log.DefaultLogger.Warnw("execution was not detected as complete", "executionId", watcher.State().ResourceId(), "err", ctx.Err(), "debug", watcher.State().Debug())
				close(notifier.ch)
				return
			}

			notifier.End()
			ctxCancel()
			close(notifier.ch)

			if opts.LogAbortedDetails && notifier.result.IsAborted() {
				log.DefaultLogger.Warnw("execution (watch) detected as aborted", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			}
			if opts.LogAbortedDetails && notifier.result.IsCanceled() {
				log.DefaultLogger.Warnw("execution (watch) detected as canceled", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			}
		}()

		// Mark Job as started
		notifier.Align(watcher.State())

		// The initialization timeout covers the wait for the pod and for the first step container
		initializationDeadline, stopInitializationDeadline := newInitializationDeadline(watcher.State())
		defer stopInitializationDeadline()

		// Wait until the Pod is scheduled
		currentJobEventsIndex := 0
		currentPodEventsIndex := 0
		for ok := true; ok; ok = waitForUpdate(updatesCh, &initializationDeadline, notifier) {
			for _, ev := range watcher.State().JobEvents().Original()[currentJobEventsIndex:] {
				currentJobEventsIndex++

				if ev.Reason != "BackoffLimitExceeded" {
					ts := watchers2.GetEventTimestamp(ev)
					notifier.Event("", ts, ev.Type, ev.Reason, ev.Message, executionId)
				}
			}
			for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				name := GetEventContainerName(ev)
				if name == "" {
					notifier.Event("", watchers2.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message, executionId)
				}
			}

			if watcher.State().PodStarted() || watcher.State().Completed() || opts.DisableFollow {
				break
			}
			alignChangedCause(notifier, watcher.State())
		}

		// Stop immediately after the operation is canceled
		if ctx.Err() != nil {
			return
		}

		// Handle the case when it has been complete without pod start
		if !watcher.State().PodStarted() && (watcher.State().Completed() || opts.DisableFollow) {
			notifier.Align(watcher.State())
			log.DefaultLogger.Warnw("execution complete without pod start", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			return
		}

		// Load the pod information
		if watcher.State().EstimatedPodStartTimestamp().IsZero() {
			notifier.Error(fmt.Errorf("cannot estimate Pod start"))
			log.DefaultLogger.Warnw("cannot estimate execution pod start", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			return
		}

		notifier.Align(watcher.State())

		// Read the execution instructions
		actions, err := watcher.State().ActionGroups()
		if err != nil {
			notifier.Error(fmt.Errorf("cannot read execution instructions: %v", err))
			log.DefaultLogger.Warnw("cannot read execution instructions", "executionId", watcher.State().ResourceId(), "debug", watcher.State().Debug())
			return
		}
		refs, endRefs := ExtractRefsFromActionGroup(actions)
		initialRefs := make([]string, len(actions))
		for i := range refs {
			for j := range refs[i] {
				if refs[i][j] == constants.InitStepName {
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
		containersReady := false
		for containerIndex := 0; containerIndex < len(refs); containerIndex++ {
			aborted := false
			canceled := false
			container := fmt.Sprintf("%d", containerIndex+1)

			// Determine the last ref in this container, so we can confirm that the logs have been read until end
			lastRef := endRefs[containerIndex][len(endRefs[containerIndex])-1]
			if lastRef == "" && len(endRefs[containerIndex]) > 1 {
				lastRef = endRefs[containerIndex][len(endRefs[containerIndex])-2]
			}

			// Wait until the Container is started
			currentPodEventsIndex = 0
			for ok := true; ok; ok = waitForUpdate(updatesCh, &initializationDeadline, notifier) {
				// Read the Pod Events for the Container Events
				for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
					currentPodEventsIndex++

					// Display only events that are unrelated to further containers
					name := GetEventContainerName(ev)
					if name == container && ev.Reason != "Created" && ev.Reason != "Started" {
						notifier.Event(initialRefs[containerIndex], watchers2.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message, executionId)
					}
				}

				// Determine if the container should be already accessible
				if watcher.State().ContainerStarted(container) || watcher.State().Completed() || opts.DisableFollow {
					break
				}
				alignChangedCause(notifier, watcher.State())
			}

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// The initialization ends when a container with a step starts. With a custom image, a container can hold only the setup.
			if containsLeafStep(refs[containerIndex], signatureSeq) {
				endInitializationDeadline(&initializationDeadline, watcher.State().ContainerStartTimestamp(container), notifier)
			}

			// Start the initial one
			lastStarted := refs[containerIndex][0]

			// Read the Container logs
			isLastHint := func(hint *instructions.Instruction) bool {
				return hint != nil && hint.Ref == lastRef && hint.Name == constants.InstructionEnd
			}
			isDone := func() bool {
				return opts.DisableFollow || watcher.State().ContainerFinished(container) || watcher.State().Completed()
			}
			logsCh := WatchContainerLogs(ctx, clientSet, watcher.State().Namespace(), watcher.State().PodName(), container, 10, isDone, isLastHint, opts.ContainerLogOptions)
			containersReady = watcher.State().ContainersReady()
		logs:
			for {
				select {
				case <-updatesCh:
					// Force empty notification on container ready (for services)
					nextContainersReady := watcher.State().ContainersReady()
					if containersReady != nextContainersReady {
						containersReady = nextContainersReady
						notifier.send(Notification{Ref: lastStarted, Temporary: true}) // TODO: apply timestamp
					}
				case v, ok := <-logsCh:
					if !ok {
						break logs
					}
					if v.Error != nil {
						ts := time.Now() // TODO: get latest timestamp instead?
						notifier.Raw(lastRef, ts, fmt.Sprintf("%s error while fetching container logs: %s\n", ts.Format(constants.PreciseTimeFormat), v.Error.Error()), false)
						continue
					}

					switch v.Value.Type() {
					case ContainerLogTypeLog:
						notifier.Raw(lastStarted, v.Value.Time, string(v.Value.Log), false)
					case ContainerLogTypeOutput:
						notifier.Output(v.Value.Output.Ref, v.Value.Time, v.Value.Output)
					case ContainerLogTypeHint:
						if v.Value.Hint.Name == constants.InstructionStart {
							lastStarted = v.Value.Hint.Ref
						}
						if v.Value.Hint.Name == constants.InstructionEnd && testkube.TestWorkflowStepStatus(v.Value.Hint.Value.(string)) == testkube.ABORTED_TestWorkflowStepStatus {
							aborted = true
						}
						if v.Value.Hint.Name == constants.InstructionEnd && testkube.TestWorkflowStepStatus(v.Value.Hint.Value.(string)) == testkube.CANCELED_TestWorkflowStepStatus {
							canceled = true
						}
						notifier.Instruction(v.Value.Time, *v.Value.Hint, executionId)
					}
				}
			}

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// Wait until the Container is terminated
			for ok := true; ok; _, ok = <-updatesCh {
				// Determine if the container should be already stopped
				if watcher.State().ContainerFinished(container) || watcher.State().Completed() || opts.DisableFollow {
					break
				}
			}

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// TODO: Include Container/Pod events after the finish (?)

			// Load the correlation data about status
			notifier.Align(watcher.State())

			// Don't iterate over further containers if this one has failed completely
			if aborted || canceled || watcher.State().ContainerFailed(container) {
				break
			}
		}

		// Wait until everything is finished
	loop:
		for {
			if watcher.State().Completed() || ctx.Err() != nil || opts.DisableFollow {
				break loop
			}

			select {
			case <-ctx.Done():
				return
			case _, ok := <-updatesCh:
				if !ok || watcher.State().Completed() {
					break loop
				}
			case <-time.After(ForceFinalizationDelay):
				watcher.RefreshPod(ctx)
				watcher.RefreshJob(ctx)

				// Fallback in case of missing data
				if watcher.State().Completed() {
					break loop
				}
				// TODO: shouldn't be just a critical error?
			}
		}

		// Stop immediately after the operation is canceled
		if ctx.Err() != nil {
			return
		}

		// Kubernetes can stop the pod after the last step ended, for example when it removes a sidecar.
		// The watch can end before the events of that stop arrive, so it reads the events one more time.
		watcher.RefreshPodEvents(ctx)
		notifyTeardownEvents(notifier, eventsSince(watcher.State().PodEvents().Original(), currentPodEventsIndex), executionId)

		// Mark as finished
		notifier.Align(watcher.State())
	}()

	return notifier.ch, nil
}

// teardownEventReasons are the pod events that report a stop of the pod, after the step containers ran.
var teardownEventReasons = []string{"Killing", "FailedPreStopHook"}

// eventsSince returns the events after the index that the watch reached, and no event when the
// index is at the end. The caller then needs no bound check of its own.
func eventsSince(events []*corev1.Event, from int) []*corev1.Event {
	if from >= len(events) {
		return nil
	}
	return events[from:]
}

// notifyTeardownEvents sends the events that report a stop of the pod.
// The result of the steps stands, so these events are warnings that explain the stop, and not a cause.
// Kubernetes marks the Killing event as normal, and a normal event only reaches the log while the watch runs.
func notifyTeardownEvents(n *notifier, events []*corev1.Event, executionId string) {
	for _, ev := range events {
		if slices.Contains(teardownEventReasons, ev.Reason) {
			n.Event("", watchers2.GetEventTimestamp(ev), corev1.EventTypeWarning, ev.Reason, ev.Message, executionId)
		}
	}
}

// alignChangedCause sends the result only when the cause changes, because a result on each update is too many.
// It does not align the result. An alignment marks a created pod as running, and a waiting pod must stay scheduling.
func alignChangedCause(n *notifier, state watchers2.ExecutionState) {
	if n.alignCause(state) {
		n.sendResult()
	}
}

// initializationDeadline is the end of the initialization timeout, with a timer that receives at that time.
// The timer is nil when the workflow has no initialization timeout, or when the watch does not need it anymore.
// Expired is true after the timer received, until the watch decides about the abort.
type initializationDeadline struct {
	timer   <-chan time.Time
	at      time.Time
	expired bool
}

// newInitializationDeadline returns the deadline of the initialization timeout of the workflow.
// The timeout counts from the job creation, because the queue time has its own timeout.
func newInitializationDeadline(state watchers2.ExecutionState) (initializationDeadline, func()) {
	timeout := state.InitializationTimeout()
	if timeout <= 0 {
		return initializationDeadline{}, func() {}
	}
	start := state.EstimatedJobCreationTimestamp()
	if start.IsZero() {
		start = time.Now()
	}
	at := start.Add(timeout)
	timer := time.NewTimer(time.Until(at))
	return initializationDeadline{timer: timer.C, at: at}, func() { timer.Stop() }
}

// waitForUpdate waits for the next update of the watcher. It returns false when the updates end.
// When the deadline ends first, it returns true without an update, so the caller reads the state one more time.
// An update that shows a started step container can arrive together with the timer. When the caller waits again
// after the deadline ended, it asks one time for the abort.
func waitForUpdate(updatesCh <-chan struct{}, deadline *initializationDeadline, n *notifier) bool {
	if deadline.expired {
		deadline.expired = false
		n.requestAbort(testkube.StopReasonInitTimeout)
	}
	select {
	case _, ok := <-updatesCh:
		return ok
	case <-deadline.timer:
		deadline.timer = nil
		deadline.expired = true
		return true
	}
}

// containsLeafStep reports whether the references contain a step that has no children.
func containsLeafStep(refs []string, signatureSeq []stage.Signature) bool {
	for _, ref := range refs {
		if ref == "" || ref == constants.InitStepName {
			continue
		}
		if slices.ContainsFunc(signatureSeq, func(sig stage.Signature) bool { return sig.Ref() == ref && len(sig.Children()) == 0 }) {
			return true
		}
	}
	return false
}

// endInitializationDeadline stops the initialization deadline when a step container started or the execution completed.
// The start time of the container decides and not the timer, because a watch that connects late finds a timer that
// already fired. A container without a start time did not start, so it does not end the initialization in time.
func endInitializationDeadline(deadline *initializationDeadline, startedAt time.Time, n *notifier) {
	if (deadline.timer != nil || deadline.expired) && !startedAt.IsZero() && startedAt.After(deadline.at) {
		n.requestAbort(testkube.StopReasonInitTimeout)
	}
	deadline.timer, deadline.expired = nil, false
}
