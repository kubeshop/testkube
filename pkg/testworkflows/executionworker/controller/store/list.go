package store

import (
	"context"
	"sync"
)

type list[T any] struct {
	items     []*T
	update    Update
	ctx       context.Context
	ctxCancel context.CancelFunc
	mu        sync.RWMutex
}

type List[T any] interface {
	Put(item *T)
	Exists() bool
	Get(index int) *T
	Latest() []*T
	Channel() <-chan *T
	Count() int
	Next() <-chan struct{}

	Cancel()
	Canceled() bool
	Done() <-chan struct{}
}

func NewList[T any](parentCtx context.Context, update Update) List[T] {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	v := &list[T]{ctx: ctx, ctxCancel: ctxCancel, update: update}
	go func() {
		<-ctx.Done()
		v.mu.Lock()
		defer v.mu.Unlock()
		v.update.Close()
	}()
	return v
}

func (v *list[T]) Put(item *T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.items = append(v.items, item)
	v.update.Emit()
}

func (v *list[T]) Get(index int) *T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if index < 0 {
		return nil
	}
	if index >= len(v.items) {
		return nil
	}
	return v.items[index]
}

func (v *list[T]) Latest() []*T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return append([]*T(nil), v.items...)
}

func (v *list[T]) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.items)
}

func (v *list[T]) Exists() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.items != nil
}

func (v *list[T]) Next() <-chan struct{} {
	return v.update.Next()
}

func (v *list[T]) Channel() <-chan *T {
	ch := make(chan *T)
	go func() {
		defer close(ch)
		i := 0
		for {
			// Read all immediate values
			for i < v.Count() {
				ch <- v.Get(i)
				i++
			}

			// End if the context is closed
			if v.ctx.Err() != nil {
				return
			}

			// Wait for updates
			select {
			case <-v.update.Next():
			}
		}
	}()
	return ch
}

func (v *list[T]) Cancel() {
	v.ctxCancel()
}

func (v *list[T]) Canceled() bool {
	return v.ctx.Err() != nil
}

func (v *list[T]) Done() <-chan struct{} {
	return v.ctx.Done()
}
