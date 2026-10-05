package channels

import (
	"context"
	"sync/atomic"
)

type watcher[T any] struct {
	ch       chan T
	finished atomic.Bool
	err      atomic.Value
}

type Watcher[T any] interface {
	Channel() <-chan T
	All() ([]T, error)
	Err() error
}

type WritableWatcher[T any] interface {
	Watcher[T]
	Send(value T)
	SendWithContext(ctx context.Context, value T) bool
	Close(err error)
}

func (n *watcher[T]) Send(value T) {
	n.ch <- value
}

// SendWithContext sends value unless ctx is done first, and reports whether it was sent.
// Unlike Send, it cannot block forever when the reader has stopped reading.
func (n *watcher[T]) SendWithContext(ctx context.Context, value T) bool {
	select {
	case n.ch <- value:
		return true
	case <-ctx.Done():
		return false
	}
}

func (n *watcher[T]) Close(err error) {
	if n.finished.CompareAndSwap(false, true) {
		if err != nil {
			n.err.Store(err)
		}
		close(n.ch)
	}
}

func (n *watcher[T]) Channel() <-chan T {
	return n.ch
}

func (n *watcher[T]) All() ([]T, error) {
	var result []T
	for v := range n.ch {
		result = append(result, v)
	}
	return result, n.Err()
}

func (n *watcher[T]) Err() error {
	err := n.err.Load()
	if err == nil {
		return nil
	}
	return err.(error)
}

func NewWatcher[T any]() WritableWatcher[T] {
	return &watcher[T]{ch: make(chan T)}
}

func NewError[T any](err error) Watcher[T] {
	res := NewWatcher[T]()
	res.Close(err)
	return res
}

func Transform[T any, U any](w Watcher[T], transform func(T) (U, bool)) Watcher[U] {
	res := NewWatcher[U]()
	go func() {
		for v := range w.Channel() {
			next, ok := transform(v)
			if ok {
				res.Send(next)
			}
		}
		res.Close(w.Err())
	}()
	return res
}
