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
	notifier := newNotifier(ctx, signature)
	signatureSeq := stage.MapSignatureToSequence(signature)
	resultState := watchers.NewResultState(testkube.TestWorkflowResult{}) // TODO: Use already acknowledge result as the initial one

	updatesCh := watcher.Updated(ctx)

	//r := rand.String(10)
	log := func(data ...interface{}) {
		// FIXME delete?
		if debug != "" {
			//data = append([]interface{}{ui.Green(watcher.State().Job().ResourceId()), ui.Blue(r)}, data...)
			//fmt.Println(data...)
		}
	}

	go func() {
		//debug.add(watcher.State().ResourceId()) FIXME
		defer func() {
			if r := recover(); r != nil {
				notifier.Error(fmt.Errorf("fatal error watching data: %v", r))
			}

			//debug.del(watcher.State().ResourceId()) FIXME
			resultState.Align(watcher.State()) // TODO: IS IT NEEDED? OR MAYBE SHOULD BE STH LIKE FINISH?
			resultState.End()
			notifier.Result(resultState.Result())
			ctxCancel()
			close(notifier.ch)
			log("closed")
		}()

		// Mark Job as started
		resultState.Align(watcher.State())
		notifier.Result(resultState.Result())
		log("queued")

		// Wait until the Pod is scheduled
		currentJobEventsIndex := 0
		currentPodEventsIndex := 0
		for ok := true; ok; _, ok = <-updatesCh {
			log("checking for scheduled pod")
			for _, ev := range watcher.State().JobEvents().Original()[currentJobEventsIndex:] {
				currentJobEventsIndex++

				log("reading event", ev)
				if ev.Reason != "BackoffLimitExceeded" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
				log("finished reading event", ev)
			}
			log("job events read")
			for _, ev := range watcher.State().PodEvents().Original()[currentPodEventsIndex:] {
				currentPodEventsIndex++

				// Display only events that are unrelated to further containers
				name := GetEventContainerName(ev)
				if name == "" {
					notifier.Event("", watchers.GetEventTimestamp(ev), ev.Type, ev.Reason, ev.Message)
				}
			}
			log("pod events read")

			if watcher.State().PodStarted() || watcher.State().Completed() {
				break
			}
			log("checking for scheduled pod: iteration end")
		}
		log("pod likely started")

		// Stop immediately after the operation is canceled
		if ctx.Err() != nil {
			return
		}

		// Handle the case when it has been complete without pod start
		if !watcher.State().PodStarted() && watcher.State().Completed() {
			log("complete without pod")
			resultState.Align(watcher.State())
			notifier.Result(resultState.Result())
			return
		}

		// Load the pod information
		// TODO: when it's complete without pod start, try to check if maybe job was not aborted etc
		if watcher.State().EstimatedPodStartTimestamp().IsZero() {
			log("cannot estimate pod start")
			notifier.Error(fmt.Errorf("cannot estimate Pod start"))
			return
		}

		resultState.Align(watcher.State())
		notifier.Result(resultState.Result())
		log("pod started")

		// Read the execution instructions
		actions, err := watcher.State().ActionGroups()
		if err != nil {
			// FIXME:
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

			log("iterating containers", container)

			// Wait until the Container is started
			currentPodEventsIndex = 0
			for ok := true; ok; _, ok = <-updatesCh {
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
			// TODO: Avoid fetching log for the containers that are known to never start
			for v := range WatchContainerLogs(ctx, clientSet, watcher.State().Namespace(), watcher.State().PodName(), container, 10, isDone) {
				if v.Error != nil {
					log("container error", container, v.Error)
					notifier.Error(v.Error)
					continue
				}
				log("container log", container, v.Value.Type())

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
					log("hint finished")
				}
			}
			log("container log finished", container, watcher.State().CompletionTimestamp().String(), watcher.State().Completed(), watcher.State().ContainerFinished(container))

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// Wait until the Container is terminated
			for ok := true; ok; _, ok = <-updatesCh {
				log("waiting for terminated container", container)

				// Determine if the container should be already stopped
				if watcher.State().ContainerFinished(container) || watcher.State().Completed() {
					break
				}

				log("waiting for terminated container: iteration end", container)
			}
			log("container terminated", container)

			// Stop immediately after the operation is canceled
			if ctx.Err() != nil {
				return
			}

			// TODO: Include Container/Pod events after the finish (?)

			// Load the correlation data about status
			resultState.Align(watcher.State())
			notifier.Result(resultState.Result())
			log("container result", container)

			// Don't iterate over further containers if this one has failed completely
			if aborted || watcher.State().ContainerFailed(container) {
				break
			}
		}
		log("finished iterating over containers")

		// Wait until everything is finished
		// TODO: Ignore when it's for services?
	loop:
		for {
			// FIXME?
			//if watcher.State().Completed() || !resultState.Result().FinishedAt.IsZero() {
			if watcher.State().Completed() || ctx.Err() != nil {
				break loop
			}

			log("looping over completion", watcher.State().CompletionTimestamp(), resultState.Result().FinishedAt)

			select {
			case <-ctx.Done():
				return
			case _, ok := <-updatesCh:
				if !ok || watcher.State().Completed() {
					break loop
				}
			case <-time.After(30 * time.Second):
				log("reloading pod & job")
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
