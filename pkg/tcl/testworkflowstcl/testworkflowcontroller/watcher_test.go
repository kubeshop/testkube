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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type test struct {
	value string
}

func queue(fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		fn()
	}()
	wg.Wait()
}

func TestWatcherSync(t *testing.T) {
	w := newWatcher[test](context.Background(), 0)

	go func() {
		w.SendValue(test{value: "A"})
		w.SendValue(test{value: "B"})
		w.Close()
	}()
	a := <-w.Next()
	b := <-w.Next()
	c := <-w.Next()
	_, ok := <-w.Next()

	assert.Equal(t, WatcherValue[test]{Value: test{value: "A"}}, a)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "B"}}, b)
	assert.Equal(t, WatcherValue[test]{}, c)
	assert.Equal(t, false, ok)
}

func TestWatcherDistributed(t *testing.T) {
	w := newWatcher[test](context.Background(), 0)

	queue(func() {
		w.SendValue(test{value: "A"})
		w.SendValue(test{value: "B"})
		w.Close()
	})

	w.Pause()
	aCh, bCh := w.Next(), w.Next()
	w.Resume()
	a, b := <-aCh, <-bCh

	c := <-w.Next()
	d := <-w.Next()

	assert.Equal(t, WatcherValue[test]{Value: test{value: "A"}}, a)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "A"}}, b)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "B"}}, c)
	assert.Equal(t, WatcherValue[test]{}, d)
}

func TestWatcherSyncAdvanced(t *testing.T) {
	w := newWatcher[test](context.Background(), 0)

	go func() {
		time.Sleep(500 * time.Microsecond)
		w.SendValue(test{value: "A"})
		w.SendValue(test{value: "B"})
		w.Close()
	}()

	aCh := w.Next()
	w.SendValue(test{value: "A"})
	go w.SendValue(test{value: "B"})
	bCh := w.Next()
	a, b := <-aCh, <-bCh

	assert.Equal(t, WatcherValue[test]{Value: test{value: "A"}}, a)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "B"}}, b)
}

func TestWatcherPause(t *testing.T) {
	w := newWatcher[test](context.Background(), 0)

	w.Pause()
	aCh := w.Next()
	queue(func() {
		w.SendValue(test{value: "A"})
	})
	bCh := w.Next()
	time.Sleep(500 * time.Microsecond)
	var a, b WatcherValue[test]
	select {
	case a = <-aCh:
	default:
	}
	select {
	case b = <-bCh:
	default:
	}

	assert.Equal(t, WatcherValue[test]{}, a)
	assert.Equal(t, WatcherValue[test]{}, b)
}

func TestWatcherCache(t *testing.T) {
	w := newWatcher[test](context.Background(), 2)

	a := w.Stream()
	queue(func() {
		w.SendValue(test{value: "A"})
		w.SendValue(test{value: "B"})
		w.SendValue(test{value: "C"})
		time.Sleep(500 * time.Microsecond)
		w.SendValue(test{value: "D"})
		w.Close()
	})
	av1 := <-a.Channel()
	av2 := <-a.Channel()
	av3 := <-a.Channel()
	a.Stop()

	b := w.Stream()
	bv1 := <-b.Channel()
	bv2 := <-b.Channel()
	bv3 := <-b.Channel()
	_, ok := <-b.Channel()

	assert.Equal(t, WatcherValue[test]{Value: test{value: "A"}}, av1)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "B"}}, av2)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "C"}}, av3)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "B"}}, bv1)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "C"}}, bv2)
	assert.Equal(t, WatcherValue[test]{Value: test{value: "D"}}, bv3)
	assert.Equal(t, false, ok)
}
