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
	"sync"
)

type ChannelMessage[T any] struct {
	Error error
	Value T
}

type Peekable[T any] interface {
	Peek(ctx context.Context) <-chan T
	PeekMessage(ctx context.Context) <-chan ChannelMessage[T]
}

type Channel[T any] interface {
	Peekable[T]
	Channel() <-chan ChannelMessage[T]
	Close()
	Done() <-chan struct{}
}

type channel[T any] struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	ch                chan ChannelMessage[T]
	last              *T
	lastMessage       *ChannelMessage[T]
	lastExists        chan struct{}
	lastMessageExists chan struct{}
	lastMu            sync.RWMutex
}

func newChannel[T any](parentCtx context.Context, bufferSize int) *channel[T] {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	c := &channel[T]{
		ctx:               ctx,
		ctxCancel:         ctxCancel,
		lastExists:        make(chan struct{}),
		lastMessageExists: make(chan struct{}),
		ch:                make(chan ChannelMessage[T], bufferSize),
	}
	go func() {
		<-c.ctx.Done()

		// Ignore when the channel is already closed
		defer func() {
			recover()
			c.lastMu.Unlock()
		}()
		c.lastMu.Lock()
		close(c.ch)
		if c.last == nil {
			close(c.lastExists)
		}
		if c.lastMessage == nil {
			close(c.lastMessageExists)
		}
	}()
	return c
}

func (c *channel[T]) _send(msg ChannelMessage[T]) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()

	// Save the last value and notify about it
	if msg.Error == nil {
		c.lastMu.Lock()
		empty := c.lastMessage == nil
		c.lastMessage = &msg
		if empty {
			close(c.lastMessageExists)
		}
		if msg.Error == nil {
			empty := c.last == nil
			c.last = &msg.Value
			if empty {
				close(c.lastExists)
			}
		}
		c.lastMu.Unlock()
	}

	// Pass the value down
	c.ch <- msg
}

func (c *channel[T]) Send(value T) {
	c._send(ChannelMessage[T]{Value: value})
}

func (c *channel[T]) Error(err error) {
	if err != nil {
		c._send(ChannelMessage[T]{Error: err})
	}
}

func (c *channel[T]) Peek(ctx context.Context) <-chan T {
	// Wait until there is any value
	select {
	case <-ctx.Done():
		ch := make(chan T)
		close(ch)
		return ch
	case <-c.lastExists:
	}

	// Read lock
	c.lastMu.RLock()
	defer c.lastMu.RUnlock()

	// Return the last value if available
	if c.last != nil {
		ch := make(chan T, 1)
		ch <- *c.last
		close(ch)
		return ch
	}

	// Return empty if it's already closed and there is no value
	ch := make(chan T)
	close(ch)
	return ch
}

func (c *channel[T]) PeekMessage(ctx context.Context) <-chan ChannelMessage[T] {
	ch := make(chan ChannelMessage[T])
	go func() {
		// Handle case when there are no messages yet
		// Wait until there is any value
		select {
		case <-ctx.Done():
			ch <- ChannelMessage[T]{Error: ctx.Err()}
			close(ch)
			return
		case <-c.lastMessageExists:
		}

		// Read lock
		c.lastMu.RLock()
		defer c.lastMu.RUnlock()

		// Return the last value if available
		if c.lastMessage != nil {
			ch <- *c.lastMessage
		}

		// Return empty if it's already closed and there is no value
		close(ch)
	}()
	return ch
}

func (c *channel[T]) Channel() <-chan ChannelMessage[T] {
	return c.ch
}

func (c *channel[T]) Close() {
	c.ctxCancel()
}

func (c *channel[T]) Done() <-chan struct{} {
	return c.ctx.Done()
}
