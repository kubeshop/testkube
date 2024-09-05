// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"sync"
	"sync/atomic"

	"golang.org/x/exp/maps"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
)

type registry struct {
	controllers map[int64]testworkflowcontroller.Controller
	statuses    map[int64]testkube.TestWorkflowStatus
	mu          sync.RWMutex
}

func NewRegistry() *registry {
	return &registry{
		controllers: make(map[int64]testworkflowcontroller.Controller),
		statuses:    make(map[int64]testkube.TestWorkflowStatus),
	}
}

func (r *registry) Set(index int64, ctrl testworkflowcontroller.Controller) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.controllers[index]; ok && v != ctrl {
		v.StopController()
	}
	r.controllers[index] = ctrl
}

func (r *registry) SetStatus(index int64, status *testkube.TestWorkflowStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if status == nil {
		r.statuses[index] = testkube.QUEUED_TestWorkflowStatus
	} else {
		r.statuses[index] = *status
	}
}

func (r *registry) Count() int64 {
	return int64(len(r.statuses))
}

func (r *registry) AllPaused() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.statuses {
		if u != testkube.PAUSED_TestWorkflowStatus {
			return false
		}
	}
	return true
}

func (r *registry) Get(index int64) testworkflowcontroller.Controller {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.controllers[index]
}

func (r *registry) Destroy(index int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.controllers[index]; ok {
		r.controllers[index].StopController()
		delete(r.controllers, index)
	}
	delete(r.statuses, index)
}

func (r *registry) EachAsync(fn func(int64, testworkflowcontroller.Controller)) {
	r.mu.RLock()
	indexes := maps.Keys(r.controllers)
	r.mu.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(indexes))
	for _, index := range indexes {
		go func(index int64) {
			ctrl := r.Get(index)
			if ctrl != nil {
				fn(index, ctrl)
			}
			wg.Done()
		}(index)
	}
	wg.Wait()
}

func (r *registry) EachAsyncAtOnce(fn func(int64, testworkflowcontroller.Controller, func())) {
	r.mu.RLock()
	indexes := maps.Keys(r.controllers)
	r.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	counter := atomic.Int32{}
	ready := func() {
		v := counter.Add(1)
		if v < int32(len(indexes)) {
			cond.Wait()
		} else {
			cond.Broadcast()
		}
	}

	wg.Add(len(indexes))
	for _, index := range indexes {
		go func(index int64) {
			ctrl := r.Get(index)
			cond.L.Lock()
			defer cond.L.Unlock()
			if ctrl != nil {
				fn(index, ctrl, ready)
			}
			wg.Done()
		}(index)
	}
	wg.Wait()
}
