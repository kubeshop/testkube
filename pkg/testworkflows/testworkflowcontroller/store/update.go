package store

import (
	"context"
	"sync"
	"sync/atomic"
)

type updateImmediate struct {
	nextCh chan struct{}
	mu     sync.Mutex
	closed atomic.Bool
	ctx    context.Context
	cancel context.CancelFunc
}

type Update interface {
	Channel() <-chan struct{}
	Next() <-chan struct{}
	Emit()
	Close()
}

func (u *updateImmediate) Channel() <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		defer func() {
			close(ch)
			<-ch
		}()
		for {
			select {
			case <-u.ctx.Done():
				return
			default:
			}
			select {
			case <-u.ctx.Done():
				return
			case <-u.Next():
				select {
				case <-u.ctx.Done():
					return
				case ch <- struct{}{}:
				}
			}
		}
	}()
	return ch
}

func (u *updateImmediate) Next() <-chan struct{} {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.nextCh
}

func (u *updateImmediate) Emit() {
	if !u.mu.TryLock() {
		return
	}
	defer func() {
		recover() // ignore closed channel error
		u.mu.Unlock()
	}()
	nextCh := u.nextCh
	u.nextCh = make(chan struct{})
	close(nextCh)
}

func (u *updateImmediate) Close() {
	defer func() {
		recover() // ignore closed channel
	}()
	if u.closed.CompareAndSwap(false, true) {
		close(u.nextCh)
		u.cancel()
	}
}

func NewUpdate() Update {
	ctx, cancel := context.WithCancel(context.Background())
	return &updateImmediate{nextCh: make(chan struct{}), ctx: ctx, cancel: cancel}
}
