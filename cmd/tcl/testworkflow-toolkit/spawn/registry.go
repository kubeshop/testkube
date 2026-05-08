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
)

type registry struct {
	addresses map[int64]string
	statuses  map[int64]testkube.TestWorkflowStatus
	mu        sync.RWMutex
}

func NewRegistry() *registry {
	return &registry{
		statuses:  make(map[int64]testkube.TestWorkflowStatus),
		addresses: make(map[int64]string),
	}
}

func (r *registry) Indexes() []int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return maps.Keys(r.statuses)
}

func (r *registry) SetAddress(index int64, address string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addresses[index] = address
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

func (r *registry) GetAddress(index int64) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.addresses[index]
}

func (r *registry) Destroy(index int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.statuses, index)
	delete(r.addresses, index)
}

func (r *registry) EachAsyncAtOnce(fn func(int64, string, func())) {
	r.mu.RLock()
	indexes := maps.Keys(r.statuses)
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
			address := r.GetAddress(index)
			cond.L.Lock()
			defer cond.L.Unlock()
			fn(index, address, ready)
			wg.Done()
		}(index)
	}
	wg.Wait()
}
