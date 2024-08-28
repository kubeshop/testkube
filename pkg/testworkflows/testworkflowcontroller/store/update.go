package store

import (
	"sync"
	"sync/atomic"
)

type updateImmediate struct {
	ch     chan struct{}
	nextCh chan struct{}
	mu     sync.Mutex
	closed atomic.Bool
}

type Update interface {
	Channel() <-chan struct{}
	Next() <-chan struct{}
	Emit()
	Close()
}

func (u *updateImmediate) Channel() <-chan struct{} {
	// TODO: use individual channels with Next()
	return u.ch
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
	if len(u.ch) == 0 {
		u.ch <- struct{}{}
		nextCh := u.nextCh
		u.nextCh = make(chan struct{})
		close(nextCh)
	}
}

func (u *updateImmediate) Close() {
	defer func() {
		recover() // ignore closed channel
	}()
	if u.closed.CompareAndSwap(false, true) {
		close(u.ch)
		<-u.ch
		close(u.nextCh)
	}
}

func NewUpdate() Update {
	return &updateImmediate{ch: make(chan struct{}, 1), nextCh: make(chan struct{})}
}
