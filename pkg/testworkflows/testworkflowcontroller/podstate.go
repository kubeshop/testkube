package testworkflowcontroller

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

const (
	eventBufferSize  = 20
	alignmentTimeout = 2 * time.Second
)

var (
	ErrNotTerminatedYet = errors.New("the container is not terminated yet")
)

type podState struct {
	pod        *corev1.Pod
	queued     map[string]time.Time
	started    map[string]time.Time
	finished   map[string]time.Time
	warnings   map[string][]*corev1.Event
	events     map[string][]*corev1.Event
	prestart   map[string]*channel[podStateUpdate]
	finishedCh map[string]chan struct{}
	mu         sync.RWMutex
	ctx        context.Context
	ctxCancel  context.CancelFunc
}

type podStateUpdate struct {
	Queued  *time.Time
	Started *time.Time
	Warning *corev1.Event
	Event   *corev1.Event
}

func newPodState(parentCtx context.Context) *podState {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	state := &podState{
		queued:     map[string]time.Time{},
		started:    map[string]time.Time{},
		finished:   map[string]time.Time{},
		warnings:   map[string][]*corev1.Event{},
		events:     map[string][]*corev1.Event{},
		prestart:   map[string]*channel[podStateUpdate]{},
		finishedCh: map[string]chan struct{}{},
		ctx:        ctx,
		ctxCancel:  ctxCancel,
	}
	go func() {
		<-ctx.Done()
		state.mu.Lock()
		defer state.mu.Unlock()
		for name, c := range state.finishedCh {
			if c != nil {
				state.finished[name] = time.Time{}
				close(c)
				delete(state.finishedCh, name)
			}
		}
		for _, c := range state.prestart {
			if c != nil {
				c.Close()
			}
		}
	}()
	return state
}

func (p *podState) preStartWatcher(name string) *channel[podStateUpdate] {
	if _, ok := p.prestart[name]; !ok {
		p.prestart[name] = newChannel[podStateUpdate](p.ctx, eventBufferSize)
		if p.ctx.Err() != nil || p.unsafeIsStarted(name) || p.unsafeIsFinished(name) {
			p.prestart[name].Close()
		}
	}
	return p.prestart[name]
}

func (p *podState) finishedChannel(name string) chan struct{} {
	if _, ok := p.finished[name]; p.ctx.Err() != nil || ok {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	if _, ok := p.finishedCh[name]; !ok {
		p.finishedCh[name] = make(chan struct{})
	}
	return p.finishedCh[name]
}

func (p *podState) setQueuedAt(name string, ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.queued[name].Equal(ts) {
		ts = ts.UTC()
		p.queued[name] = ts
		p.preStartWatcher(name).Send(podStateUpdate{Queued: &ts})
	}
}

func (p *podState) alignQueuedAt(name string, ts time.Time) {
	go func() {
		select {
		case <-time.After(alignmentTimeout):
		case <-p.ctx.Done():
			return
		}

		p.mu.Lock()
		hasQueued := p.queued[name].IsZero() || !p.queued[name].Before(ts)
		p.mu.Unlock()
		if hasQueued {
			p.setQueuedAt(name, ts)
		}
	}()
}

func (p *podState) alignStartedAt(name string, ts time.Time) {
	go func() {
		select {
		case <-time.After(alignmentTimeout):
		case <-p.ctx.Done():
			return
		}

		p.mu.Lock()
		hasQueued := p.queued[name].IsZero() || !p.queued[name].Before(ts)
		hasStarted := p.started[name].IsZero() || !p.started[name].Before(ts)
		p.mu.Unlock()
		if hasQueued {
			p.setQueuedAt(name, ts)
		}
		if hasStarted {
			p.setStartedAt(name, ts)
		}
	}()
}

func (p *podState) setStartedAt(name string, ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started[name].Equal(ts) {
		ts = ts.UTC()
		p.started[name] = ts
		w := p.preStartWatcher(name)
		w.Send(podStateUpdate{Started: &ts})
		w.Close()
		p.alignQueuedAt(name, ts)
	}
}

func (p *podState) setFinishedAt(name string, ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.containerResult(name)
	if !errors.Is(err, ErrNotTerminatedYet) && !p.finished[name].Equal(ts) {
		p.finished[name] = ts.UTC()
		if _, ok := p.finishedCh[name]; ok {
			close(p.finishedCh[name])
			delete(p.finishedCh, name)
		}
		p.alignStartedAt(name, ts)
	}
}

func (p *podState) unsafeAddEvent(name string, event *corev1.Event) {
	if !slices.ContainsFunc(p.events[name], common.DeepEqualCmp(event)) {
		p.events[name] = append(p.events[name], event)
		p.preStartWatcher(name).Send(podStateUpdate{Event: event})
	}
}

func (p *podState) addEvent(name string, event *corev1.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.unsafeAddEvent(name, event)
	if name == "" {
		p.unsafeAddEvent(InitContainerName, event)
	}
}

func (p *podState) containerStatus(name string) *corev1.ContainerStatus {
	if p.pod == nil {
		return nil
	}
	for i := range p.pod.Status.InitContainerStatuses {
		if p.pod.Status.InitContainerStatuses[i].Name == name {
			return &p.pod.Status.InitContainerStatuses[i]
		}
	}
	for i := range p.pod.Status.ContainerStatuses {
		if p.pod.Status.ContainerStatuses[i].Name == name {
			return &p.pod.Status.ContainerStatuses[i]
		}
	}
	return nil
}

func (p *podState) RegisterEvent(event *corev1.Event) {
	name := GetEventContainerName(event)
	switch event.Reason {
	case "SuccessfulCreate", "Created":
		p.setQueuedAt(name, event.CreationTimestamp.Time)
	case "Scheduled", "Started":
		p.setStartedAt(name, event.CreationTimestamp.Time)
	}
	if p.StartedAt(name).IsZero() &&
		event.Reason != "Created" && event.Reason != "SuccessfulCreate" &&
		(event.Reason != "Pulled" || (!strings.Contains(event.Message, constants.DefaultInitImage) && !strings.Contains(event.Message, constants.DefaultToolkitImage))) {
		p.addEvent(name, event)
	}
}

func (p *podState) RegisterPod(pod *corev1.Pod) {
	if pod == nil {
		return
	}
	pod.ManagedFields = nil
	p.mu.Lock()
	p.pod = pod
	p.mu.Unlock()

	done := IsPodDone(pod)
	doneTs := time.Time{}
	if done {
		for _, c := range pod.Status.Conditions {
			// TODO: Filter to only finished values
			if c.LastTransitionTime.After(doneTs) {
				doneTs = c.LastTransitionTime.Time
			}
		}
		if pod.DeletionTimestamp != nil && doneTs.IsZero() {
			doneTs = pod.DeletionTimestamp.Time
		} else if doneTs.IsZero() {
			// TODO: Consider getting the latest timestamp from the Pod object
			doneTs = time.Now()
		}
	}

	// Register pod creation
	p.setStartedAt("", pod.CreationTimestamp.Time)

	// Register container statuses
	for _, s := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		if s.State.Terminated != nil {
			p.setStartedAt(s.Name, s.State.Terminated.StartedAt.Time)
			p.setFinishedAt(s.Name, s.State.Terminated.FinishedAt.Time)
		} else if s.State.Running != nil {
			p.setStartedAt(s.Name, s.State.Running.StartedAt.Time)
		}
	}

	// Register pod finish time
	if done {
		for _, s := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
			if s.State.Terminated == nil {
				p.setFinishedAt(s.Name, doneTs)
			}
		}
		p.setFinishedAt("", doneTs)
	}
}

func (p *podState) RegisterJob(job *batchv1.Job) {
	p.setQueuedAt("", job.CreationTimestamp.Time)
	if job.Status.CompletionTime != nil {
		p.setFinishedAt("", job.Status.CompletionTime.Time)
	} else if slices.ContainsFunc(job.Status.Conditions, isJobConditionEnd) {
		for i := range job.Status.Conditions {
			if isJobConditionEnd(job.Status.Conditions[i]) {
				p.setFinishedAt("", job.Status.Conditions[i].LastTransitionTime.Time)
				break
			}
		}
	} else if job.DeletionTimestamp != nil {
		p.setFinishedAt("", job.DeletionTimestamp.Time)
	}
}

func (p *podState) Wait() {
	<-p.ctx.Done()
}

func (p *podState) PreStart(name string) <-chan ChannelMessage[podStateUpdate] {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.preStartWatcher(name).Channel()
}

func (p *podState) unsafeIsStarted(name string) bool {
	return !p.started[name].IsZero()
}

func (p *podState) unsafeIsFinished(name string) bool {
	return !p.finished[name].IsZero()
}

func (p *podState) IsFinished(name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.unsafeIsFinished(name)
}

func (p *podState) Finished(name string) chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.finishedChannel(name)
}

func (p *podState) QueuedAt(name string) time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.queued[name]; ok {
		return v
	}
	// Fallback to "started" if there is no "queued" time known
	return p.started[name]
}

func (p *podState) StartedAt(name string) time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started[name]
}

func (p *podState) FinishedAt(name string) time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.finished[name]
}

func (p *podState) containerResult(name string) (ContainerResult, error) {
	status := p.containerStatus(name)
	if status == nil || status.State.Terminated == nil {
		if p.pod != nil && IsPodDone(p.pod) {
			result := UnknownContainerResult
			for _, c := range p.pod.Status.Conditions {
				if c.Type == corev1.DisruptionTarget && c.Status == corev1.ConditionTrue {
					if c.Reason == "EvictionByEvictionAPI" {
						result.Details = "Pod has been requested for deletion using the Kubernetes API"
					} else if c.Message == "" {
						result.Details = c.Reason
					} else {
						result.Details = fmt.Sprintf("%s: %s", c.Reason, c.Message)
					}
					break
				}
			}
			return result, nil
		}
		return UnknownContainerResult, ErrNotTerminatedYet
	}

	result := ContainerResult{
		ExitCode:   int(status.State.Terminated.ExitCode),
		FinishedAt: status.State.Terminated.FinishedAt.Time,
	}

	// Workaround - GKE sends SIGKILL after the container is already terminated,
	// and the pod gets stuck then.
	if status.State.Terminated.Reason != "Completed" {
		result.Details = status.State.Terminated.Reason
	}

	re := regexp.MustCompile(`^([^,]),(0|[1-9]\d*)$`)
	for _, message := range strings.Split(status.State.Terminated.Message, "/") {
		match := re.FindStringSubmatch(message)
		if match == nil {
			result.Steps = append(result.Steps, ContainerResultStep{
				Status:     testkube.ABORTED_TestWorkflowStepStatus,
				FinishedAt: result.FinishedAt,
				ExitCode:   -1,
			})
		} else {
			exitCode, _ := strconv.Atoi(match[2])
			result.Steps = append(result.Steps, ContainerResultStep{
				Status:     testkube.TestWorkflowStepStatus(data.StepStatusFromCode(match[1])),
				Details:    result.Details,
				FinishedAt: result.FinishedAt,
				ExitCode:   exitCode,
			})
		}
	}

	return result, nil
}

func (p *podState) ContainerResult(name string) (ContainerResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.containerResult(name)
}

func initializePodState(parentCtx context.Context, pod Channel[*corev1.Pod], podEvents Channel[*corev1.Event], job Channel[*batchv1.Job], jobEvents Channel[*corev1.Event], errorHandler func(error)) *podState {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	state := newPodState(ctx)

	// Fill optional channels
	if job == nil {
		job = newChannel[*batchv1.Job](ctx, 0)
		job.Close()
	}
	if jobEvents == nil {
		jobEvents = newChannel[*corev1.Event](ctx, 0)
		jobEvents.Close()
	}

	go func() {
		defer ctxCancel()

		// Build channels for the streams
		left := 4
		jobCh := job.Channel()
		jobEventsCh := jobEvents.Channel()
		podCh := pod.Channel()
		podEventsCh := podEvents.Channel()

		// Loop for the data
		for {
			if left == 0 {
				return
			}

			// Prioritize pod & events
			select {
			case <-parentCtx.Done():
				return
			case v, ok := <-podCh:
				if !ok {
					podCh = nil
					left--
					continue
				}
				if v.Error != nil {
					errorHandler(v.Error)
					continue
				}
				state.RegisterPod(v.Value)
			case v, ok := <-jobEventsCh:
				if !ok {
					jobEventsCh = nil
					left--
					continue
				}
				if v.Error != nil {
					errorHandler(v.Error)
					continue
				}
				state.RegisterEvent(v.Value)
			case v, ok := <-podEventsCh:
				if !ok {
					podEventsCh = nil
					left--
					continue
				}
				if v.Error != nil {
					errorHandler(v.Error)
					continue
				}
				state.RegisterEvent(v.Value)
			case v, ok := <-jobCh:
				if !ok {
					jobCh = nil
					left--
					continue
				}
				if v.Error != nil {
					errorHandler(v.Error)
					continue
				}

				// Try to firstly finish with the Pod information when it's possible
				if IsJobDone(v.Value) && state.FinishedAt("").IsZero() && HadPodScheduled(v.Value) {
					select {
					case p, ok := <-podCh:
						if p.Error != nil {
							errorHandler(p.Error)
						} else if ok {
							state.RegisterPod(p.Value)
						}
					case <-time.After(alignmentTimeout):
						// Continue - likely we won't receive Pod status
					}
				}
				state.RegisterJob(v.Value)
			}
		}
	}()
	return state
}
