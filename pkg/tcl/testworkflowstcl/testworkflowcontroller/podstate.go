// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowcontroller

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/internal/common"
)

const eventBufferSize = 20

type podState struct {
	pod        *corev1.Pod
	queued     map[string]time.Time
	started    map[string]time.Time
	finished   map[string]time.Time
	warnings   map[string][]*corev1.Event
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
}

func newPodState(parentCtx context.Context) *podState {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	state := &podState{
		queued:     map[string]time.Time{},
		started:    map[string]time.Time{},
		finished:   map[string]time.Time{},
		warnings:   map[string][]*corev1.Event{},
		prestart:   map[string]*channel[podStateUpdate]{},
		finishedCh: map[string]chan struct{}{},
		ctx:        ctx,
		ctxCancel:  ctxCancel,
	}
	go func() {
		<-ctx.Done()
		state.mu.Lock()
		defer state.mu.Unlock()
		for _, c := range state.finishedCh {
			if c != nil {
				close(c)
			}
		}
	}()
	return state
}

func (p *podState) preStartWatcher(name string) *channel[podStateUpdate] {
	if _, ok := p.prestart[name]; !ok {
		p.prestart[name] = newChannel[podStateUpdate](p.ctx, eventBufferSize)
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

func (p *podState) setStartedAt(name string, ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started[name].Equal(ts) {
		ts = ts.UTC()
		p.started[name] = ts
		w := p.preStartWatcher(name)
		w.Send(podStateUpdate{Started: &ts})
		w.Close()
	}
}

func (p *podState) setFinishedAt(name string, ts time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.finished[name].Equal(ts) {
		p.finished[name] = ts.UTC()
		if _, ok := p.finishedCh[name]; ok {
			close(p.finishedCh[name])
			delete(p.finishedCh, name)
		}
	}
}

func (p *podState) addWarning(name string, event *corev1.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !slices.ContainsFunc(p.warnings[name], common.DeepEqualCmp(event)) {
		p.warnings[name] = append(p.warnings[name], event)
		p.preStartWatcher(name).Send(podStateUpdate{Warning: event})
	}
}

func (p *podState) containerStatus(name string) *corev1.ContainerStatus {
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
	if event.Type != "Normal" {
		p.addWarning(name, event)
	}
}

func (p *podState) RegisterPod(pod *corev1.Pod) {
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

func (p *podState) ContainerResult(name string) (ContainerResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	status := p.containerStatus(name)
	if status == nil || status.State.Terminated == nil {
		// TODO: Handle it nicer
		if IsPodDone(p.pod) {
			return UnknownContainerResult, nil
		}
		return UnknownContainerResult, errors.New("the container is not terminated yet")
	}
	return GetContainerResult(*status), nil
}

func initializePodState(parentCtx context.Context, pod Channel[*corev1.Pod], podEvents Channel[*corev1.Event], job Channel[*batchv1.Job], jobEvents Channel[*corev1.Event], errorHandler func(error)) *podState {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	state := newPodState(ctx)
	go func() {
		defer ctxCancel()
		var wg sync.WaitGroup
		if job != nil {
			wg.Add(1)
			go func() {
				for v := range job.Channel() {
					if v.Error != nil {
						errorHandler(v.Error)
					} else {
						state.RegisterJob(v.Value)
					}
				}
				wg.Done()
			}()
		}
		if jobEvents != nil {
			wg.Add(1)
			go func() {
				for v := range jobEvents.Channel() {
					if v.Error != nil {
						errorHandler(v.Error)
					} else {
						state.RegisterEvent(v.Value)
					}
				}
				wg.Done()
			}()
		}
		wg.Add(1)
		go func() {
			for v := range podEvents.Channel() {
				if v.Error != nil {
					errorHandler(v.Error)
				} else {
					state.RegisterEvent(v.Value)
				}
			}
		}()
		wg.Add(1)
		go func() {
			for v := range pod.Channel() {
				if v.Error != nil {
					errorHandler(v.Error)
				} else {
					state.RegisterPod(v.Value)
				}
			}
		}()
		wg.Wait()
	}()
	return state
}
