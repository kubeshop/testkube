package controller

import (
	"context"
	"fmt"
	"slices"
	"time"

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
	DisableFollow     bool
	LogAbortedDetails bool
}

func WatchInstrumentedPod(parentCtx context.Context, clientSet kubernetes.Interface, signature []stage.Signature, scheduledAt time.Time, watcher watchers2.KubernetesExecutionWatcher, opts WatchInstrumentedPodOptions) (<-chan ChannelMessage[Notification], error) {
	return WatchInstrumented(parentCtx, signature, scheduledAt, watcher, opts, func(ctx context.Context, container string, isDone func() bool, isLastHint func(instruction *instructions.Instruction) bool) <-chan ChannelMessage[ContainerLog] {
		return WatchContainerLogs(ctx, clientSet, watcher.KubernetesState().Namespace(), watcher.KubernetesState().PodName(), container, 10, isDone, isLastHint)
	})
}

func WatchInstrumented(parentCtx context.Context, signature []stage.Signature, scheduledAt time.Time, watcher watchers2.ExecutionWatcher, opts WatchInstrumentedPodOptions, getLogs func(ctx context.Context, container string, isDone func() bool, isLastHint func(instruction *instructions.Instruction) bool) <-chan ChannelMessage[ContainerLog]) (<-chan ChannelMessage[Notification], error) {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	notifier := NewNotifier(ctx, testkube.TestWorkflowResult{}, scheduledAt)
	signatureSeq := stage.MapSignatureToSequence(signature)

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
		}()

		// Mark Job as started
		notifier.Align(watcher.State())

		// Wait until the Pod is scheduled
		currentJobEventsIndex := 0
		currentPodEventsIndex := 0
		for ok := true; ok; _, ok = <-updatesCh {
			for _, ev := range watcher.State().JobEvents().Original()[currentJobEventsIndex:] {
				currentJobEventsIndex++

				if ev.Reason != "BackoffLimitExceeded" {
					notifier.Event("", watchers2.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}
			for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				name := GetEventContainerName(ev)
				if name == "" {
					notifier.Event("", watchers2.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}

			if watcher.State().PodStarted() || watcher.State().Completed() || opts.DisableFollow {
				break
			}
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
		lastStarted := constants.InitStepName
		containersReady := false
		for containerIndex := 0; containerIndex < len(refs); containerIndex++ {
			aborted := false
			container := fmt.Sprintf("%d", containerIndex+1)

			// Determine the last ref in this container, so we can confirm that the logs have been read until end
			lastRef := endRefs[containerIndex][len(endRefs[containerIndex])-1]
			if lastRef == "" && len(endRefs[containerIndex]) > 1 {
				lastRef = endRefs[containerIndex][len(endRefs[containerIndex])-2]
			}

			// Wait until the Container is started
			currentPodEventsIndex = 0
			for ok := true; ok; _, ok = <-updatesCh {
				// Read the Pod Events for the Container Events
				for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
					currentPodEventsIndex++

					// Display only events that are unrelated to further containers
					name := GetEventContainerName(ev)
					if name == container && ev.Reason != "Created" && ev.Reason != "Started" {
						notifier.Event(initialRefs[containerIndex], watchers2.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
					}
				}

				// Determine if the container should be already accessible
				if watcher.State().ContainerStarted(container) || watcher.State().Completed() || opts.DisableFollow {
					break
				}
			}

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// Start the initial one
			lastStarted = refs[containerIndex][0]

			// Read the Container logs
			isLastHint := func(hint *instructions.Instruction) bool {
				return hint != nil && hint.Ref == lastRef && hint.Name == constants.InstructionEnd
			}
			isDone := func() bool {
				return opts.DisableFollow || watcher.State().ContainerFinished(container) || watcher.State().Completed()
			}
			logsCh := getLogs(ctx, container, isDone, isLastHint)
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
						notifier.Instruction(v.Value.Time, *v.Value.Hint)
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
			if aborted || watcher.State().ContainerFailed(container) {
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

		// Mark as finished
		notifier.Align(watcher.State())
	}()

	return notifier.ch, nil
}
