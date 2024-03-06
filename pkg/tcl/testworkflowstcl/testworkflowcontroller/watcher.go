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
)

type WatcherValue[T interface{}] struct {
	Value T
	Error error
}

type Watcher[T interface{}] interface {
	Next(ctx context.Context) <-chan WatcherValue[T]
	Any(ctx context.Context) <-chan WatcherValue[T]
	Done() <-chan struct{}
	Listen(fn func(WatcherValue[T], bool)) func()
	Stream(ctx context.Context) WatcherChannel[T]
	Close()
}

type watcher[T interface{}] struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	mu        sync.Mutex
	hasCh     chan struct{}
	ch        chan WatcherValue[T]
	listeners []*func(WatcherValue[T], bool)
	paused    bool
	closed    bool

	cacheSize   int
	cacheOffset int
	cache       []WatcherValue[T]

	readerCh chan<- struct{}
}

func newWatcher[T interface{}](ctx context.Context, cacheSize int) *watcher[T] {
	finalCtx, ctxCancel := context.WithCancel(ctx)
	return &watcher[T]{
		ctx:       finalCtx,
		ctxCancel: ctxCancel,
		hasCh:     make(chan struct{}),
		ch:        make(chan WatcherValue[T]),
		cacheSize: cacheSize,
	}
}

func (w *watcher[T]) Pause() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.paused = true
	if w.readerCh != nil {
		close(w.readerCh)
		w.readerCh = nil
	}
}

func (w *watcher[T]) Resume() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.paused = false
	w.recomputeReader()
}

func (w *watcher[T]) Next(ctx context.Context) <-chan WatcherValue[T] {
	ch := make(chan WatcherValue[T])
	var cancelListener func()
	finalCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Wait()
		<-finalCtx.Done()
		cancelListener()
	}()
	cancelListener = w.Listen(func(w WatcherValue[T], ok bool) {
		wg.Wait() // on finished channel, the listener may be called before the lock goes down
		cancelListener()
		cancel()
		if ok {
			ch <- w
		}
		close(ch)
	})
	wg.Done()
	return ch
}

func (w *watcher[T]) Any(ctx context.Context) <-chan WatcherValue[T] {
	ch := make(chan WatcherValue[T])
	go func() {
		w.mu.Lock()
		if len(w.cache) > 0 {
			v := w.cache[len(w.cache)-1]
			w.mu.Unlock()
			ch <- v
			close(ch)
			return
		}
		w.mu.Unlock()
		v, ok := <-w.Next(ctx)
		if ok {
			ch <- v
		}
		close(ch)
	}()
	return ch
}

func (w *watcher[T]) _send(v WatcherValue[T]) {
	w.mu.Lock()

	// Handle closed stream
	if w.closed {
		w.mu.Unlock()
		return
	}

	// Save in cache
	if w.cacheSize == 0 {
		// Ignore cache
	} else if w.cacheSize < 0 || w.cacheSize > len(w.cache) {
		// Unlimited cache or still cache size
		w.cache = append(w.cache, v)
	} else {
		// Emptying oldest entries in the cache
		for i := 1; i < len(w.cache); i++ {
			w.cache[i-1] = w.cache[i]
		}
		w.cache[len(w.cache)-1] = v
		w.cacheOffset++
	}
	w.mu.Unlock()

	// Ignore the panic due to the channel closed externally
	defer func() {
		recover()
	}()

	// Emit the data to the live stream
	w.hasCh <- struct{}{}
	w.ch <- v
}

func (w *watcher[T]) SendValue(value T) {
	w._send(WatcherValue[T]{Value: value})

}

func (w *watcher[T]) SendError(err error) {
	w._send(WatcherValue[T]{Error: err})
}

func (w *watcher[T]) Close() {
	w.mu.Lock()
	if !w.closed {
		w.ctxCancel()
		ch := w.ch
		w.closed = true
		close(ch)
		close(w.hasCh)
		w.mu.Unlock()
	} else {
		w.mu.Unlock()
	}
}

func (w *watcher[T]) recomputeReader() {
	if w.paused {
		return
	}
	shouldRead := !w.closed && len(w.listeners) > 0
	if shouldRead && w.readerCh == nil {
		// Start the reader
		ch := make(chan struct{})
		w.readerCh = ch
		go func() {
			// Prioritize cancel channels
			for {
				select {
				case <-ch:
					return
				default:
				}
				// Then wait for the results
				select {
				case <-ch:
					return
				case _, ok := <-w.hasCh:
					listeners := slices.Clone(w.listeners)
					if ok {
						select {
						case <-ch:
							go func() {
								defer func() {
									recover()
								}()
								w.hasCh <- struct{}{} // replay hasCh in case it is needed in next iteration
							}()
							return
						default:
						}
					}
					value, ok := <-w.ch
					var wg sync.WaitGroup
					for _, l := range listeners {
						wg.Add(1)
						go func(fn func(WatcherValue[T], bool)) {
							defer func() {
								recover()
								wg.Done()
							}()
							fn(value, ok)
						}(*l)
					}
					wg.Wait()
				}
			}
		}()
	} else if !shouldRead && w.readerCh != nil {
		// Stop the reader
		close(w.readerCh)
		w.readerCh = nil
	}
}

func (w *watcher[T]) stop(ptr *func(WatcherValue[T], bool)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	index := slices.Index(w.listeners, ptr)
	if index == -1 {
		return
	}
	// Delete the listener and stop a base channel reader if needed
	*w.listeners[index] = func(value WatcherValue[T], ok bool) {}
	w.listeners = append(w.listeners[0:index], w.listeners[index+1:]...)
	w.recomputeReader()
}

func (w *watcher[T]) listenUnsafe(fn func(WatcherValue[T], bool)) func() {
	// Fail immediately if the watcher is already closed
	if w.closed {
		go func() {
			fn(WatcherValue[T]{}, false)
		}()
		return func() {}
	}

	// Append new listener and start a base channel reader if needed
	ptr := &fn
	w.listeners = append(w.listeners, ptr)
	w.recomputeReader()
	return func() {
		w.stop(ptr)
	}
}

func (w *watcher[T]) Listen(fn func(WatcherValue[T], bool)) func() {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.listenUnsafe(fn)
}

func (w *watcher[T]) Done() <-chan struct{} {
	return w.ctx.Done()
}

func (w *watcher[T]) getAndLock(index int) (WatcherValue[T], int, bool) {
	w.mu.Lock()
	index -= w.cacheOffset
	if index < 0 {
		index = 0
	}
	next := index + w.cacheOffset + 1

	// Load value from cache
	if index < len(w.cache) {
		return w.cache[index], next, true
	}

	// Fetch next result
	return WatcherValue[T]{}, next, false
}

func (w *watcher[T]) Stream(ctx context.Context) WatcherChannel[T] {
	// Create the channel
	wCh := &watcherChannel[T]{
		ch: make(chan WatcherValue[T]),
	}

	// Handle context
	finalCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-finalCtx.Done()
		wCh.Stop()
	}()

	// Fast-track when there are no cached messages
	w.mu.Lock()
	if len(w.cache) == 0 {
		wCh.cancel = w.listenUnsafe(func(v WatcherValue[T], ok bool) {
			defer func() {
				// Ignore writing to already closed channel
				recover()
			}()
			if ok {
				wCh.ch <- v
			} else if wCh.ch != nil {
				wCh.Stop()
				cancel()
			}
		})
		w.mu.Unlock()
		return wCh
	}
	w.mu.Unlock()

	// Pick cache data
	go func() {
		defer func() {
			// Ignore writing to already closed channel
			recover()
		}()

		if wCh.ch == nil {
			cancel()
			return
		}

		// Send cache data
		wCh.cancel = func() { cancel() }
		var value WatcherValue[T]
		var ok bool
		index := 0
		for value, index, ok = w.getAndLock(index); ok; value, index, ok = w.getAndLock(index) {
			if wCh.ch == nil {
				w.mu.Unlock()
				cancel()
				return
			}
			w.mu.Unlock()
			wCh.ch <- value
		}

		if wCh.ch == nil {
			w.mu.Unlock()
			cancel()
			return
		}

		// Start actually listening
		wCh.cancel = w.listenUnsafe(func(v WatcherValue[T], ok bool) {
			defer func() {
				// Ignore writing to already closed channel
				recover()
			}()
			if ok {
				wCh.ch <- v
			} else if wCh.ch != nil {
				wCh.Stop()
				cancel()
			}
		})
		w.mu.Unlock()
	}()

	return wCh
}

type WatcherChannel[T interface{}] interface {
	Channel() <-chan WatcherValue[T]
	Stop()
}

type watcherChannel[T interface{}] struct {
	cancel func()
	ch     chan WatcherValue[T]
}

func (w *watcherChannel[T]) Channel() <-chan WatcherValue[T] {
	if w.ch == nil {
		ch := make(chan WatcherValue[T])
		close(ch)
		return ch
	}
	return w.ch
}

func (w *watcherChannel[T]) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
		if w.ch != nil {
			ch := w.ch
			w.ch = nil
			close(ch)
		}
	}
}
