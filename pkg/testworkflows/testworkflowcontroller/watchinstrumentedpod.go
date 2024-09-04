package testworkflowcontroller

import (
	"context"
	"fmt"
	"slices"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	InitContainerName = "tktw-init"
)

type WatchInstrumentedPodOptions struct {
	DisableFollow bool
}

func WatchInstrumentedPod(parentCtx context.Context, clientSet kubernetes.Interface, signature []stage.Signature, scheduledAt time.Time, watcher watchers.ExecutionWatcher, opts WatchInstrumentedPodOptions) (<-chan ChannelMessage[Notification], error) {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	notifier := newNotifier(ctx, signature)
	signatureSeq := stage.MapSignatureToSequence(signature)
	resultState := watchers.NewResultState(testkube.TestWorkflowResult{}) // TODO: Use already acknowledge result as the initial one

	updatesCh := watcher.Updated(ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				notifier.Error(fmt.Errorf("fatal error watching data: %v", r))
			}

			resultState.Align(watcher.State())
			resultState.End()
			notifier.Result(resultState.Result())
			ctxCancel()
			close(notifier.ch)
		}()

		// Mark Job as started
		resultState.Align(watcher.State())
		notifier.Result(resultState.Result())

		// Wait until the Pod is scheduled
		currentJobEventsIndex := 0
		currentPodEventsIndex := 0
		for ok := true; ok; _, ok = <-updatesCh {
			for _, ev := range watcher.State().JobEvents().Original()[currentJobEventsIndex:] {
				currentJobEventsIndex++

				if ev.Reason != "BackoffLimitExceeded" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}
			for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				name := GetEventContainerName(ev)
				if name == "" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}

			if watcher.State().PodStarted() || watcher.State().Completed() {
				break
			}
		}

		// Stop immediately after the operation is canceled
		if ctx.Err() != nil {
			return
		}

		// Handle the case when it has been complete without pod start
		if !watcher.State().PodStarted() && watcher.State().Completed() {
			resultState.Align(watcher.State())
			notifier.Result(resultState.Result())
			return
		}

		// Load the pod information
		// TODO: when it's complete without pod start, try to check if maybe job was not aborted etc
		if watcher.State().EstimatedPodStartTimestamp().IsZero() {
			notifier.Error(fmt.Errorf("cannot estimate Pod start"))
			return
		}

		resultState.Align(watcher.State())
		notifier.Result(resultState.Result())

		// Read the execution instructions
		actions, err := watcher.State().ActionGroups()
		if err != nil {
			notifier.Error(fmt.Errorf("cannot read execution instructions: %v", err))
			return
		}
		refs, _ := ExtractRefsFromActionGroup(actions)
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
		lastStarted := InitContainerName
		for containerIndex := 0; containerIndex < len(refs); containerIndex++ {
			aborted := false
			container := fmt.Sprintf("%d", containerIndex+1)

			// Wait until the Container is started
			currentPodEventsIndex = 0
			for ok := true; ok; _, ok = <-updatesCh {
				// Read the Pod Events for the Container Events
				for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
					currentPodEventsIndex++

					// Display only events that are unrelated to further containers
					name := GetEventContainerName(ev)
					if name == container && ev.Reason != "Created" && ev.Reason != "Started" {
						notifier.Event(initialRefs[containerIndex], watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
					}
				}

				// Determine if the container should be already accessible
				if watcher.State().ContainerStarted(container) || watcher.State().Completed() {
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
			isDone := func() bool {
				return opts.DisableFollow || watcher.State().ContainerFinished(container) || watcher.State().Completed()
			}
			for v := range WatchContainerLogs(ctx, clientSet, watcher.State().Namespace(), watcher.State().PodName(), container, 10, isDone) {
				if v.Error != nil {
					notifier.Error(v.Error)
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
					resultState.Append(v.Value.Time, *v.Value.Hint)
					notifier.Result(resultState.Result())
				}
			}

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// Wait until the Container is terminated
			for ok := true; ok; _, ok = <-updatesCh {
				// Determine if the container should be already stopped
				if watcher.State().ContainerFinished(container) || watcher.State().Completed() {
					break
				}
			}

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// TODO: Include Container/Pod events after the finish (?)

			// Load the correlation data about status
			resultState.Align(watcher.State())
			notifier.Result(resultState.Result())

			// Don't iterate over further containers if this one has failed completely
			if aborted || watcher.State().ContainerFailed(container) {
				break
			}
		}

		// Wait until everything is finished
		// TODO: Ignore when it's for services?
	loop:
		for {
			// FIXME?
			//if watcher.State().Completed() || !resultState.Result().FinishedAt.IsZero() {
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
			case <-time.After(30 * time.Second):
				watcher.RefreshPod(30 * time.Second) // FIXME?
				watcher.RefreshJob(30 * time.Second) // FIXME?

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
		// TODO: Calibrate with top timestamp?
		resultState.Align(watcher.State())
		notifier.Result(resultState.Result())
	}()

	//return notifierProxyCh, nil
	return notifier.ch, nil
}
