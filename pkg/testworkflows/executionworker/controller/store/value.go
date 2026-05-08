package store

import (
	"context"
	"sync"
)

type value[T any] struct {
	last      *T
	update    Update
	ctx       context.Context
	ctxCancel context.CancelFunc
	mu        sync.RWMutex
}

type Value[T any] interface {
	Put(value *T)
	Latest() *T
	Exists() bool
	Next() <-chan struct{}
	Channel(ctx context.Context) <-chan *T

	Cancel()
	Canceled() bool
	Done() <-chan struct{}
}

func NewValue[T any](parentCtx context.Context, update Update) Value[T] {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	v := &value[T]{ctx: ctx, ctxCancel: ctxCancel, update: update}
	go func() {
		<-ctx.Done()
		v.mu.Lock()
		v.update.Close()
		v.mu.Unlock()
	}()
	return v
}

func (v *value[T]) Put(value *T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.last = value
	v.update.Emit()
}

func (v *value[T]) Latest() *T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.last
}

func (v *value[T]) Exists() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.last != nil
}

func (v *value[T]) Next() <-chan struct{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.update.Next()
}

func (v *value[T]) Channel(ctx context.Context) <-chan *T {
	updateCh := v.update.Channel(ctx)
	ch := make(chan *T)
	go func() {
		defer close(ch)
		latest := v.Latest()
		ch <- latest
		for range updateCh {
			next := v.Latest()
			if next != latest {
				latest = next
				ch <- latest
			}
		}
	}()
	return ch
}

func (v *value[T]) Cancel() {
	v.ctxCancel()
}

func (v *value[T]) Canceled() bool {
	return v.ctx.Err() != nil
}

func (v *value[T]) Done() <-chan struct{} {
	return v.ctx.Done()
}
