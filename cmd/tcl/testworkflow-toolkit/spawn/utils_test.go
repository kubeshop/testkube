// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package spawn

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecuteParallel(t *testing.T) {
	t.Run("all items run when context is not cancelled", func(t *testing.T) {
		items := []string{"a", "b", "c", "d"}
		namespaces := []string{"ns1", "ns1", "ns1", "ns1"}
		var executed atomic.Int64

		run := func(index int64, ns string, item *string) bool {
			executed.Add(1)
			return true
		}

		failed := ExecuteParallel(context.Background(), run, items, namespaces, 2)
		assert.Equal(t, int64(0), failed)
		assert.Equal(t, int64(4), executed.Load())
	})

	t.Run("cancelled context prevents queued items from starting", func(t *testing.T) {
		items := []string{"a", "b", "c", "d", "e", "f"}
		namespaces := []string{"ns1", "ns1", "ns1", "ns1", "ns1", "ns1"}
		var executed atomic.Int64

		ctx, cancel := context.WithCancel(context.Background())

		run := func(index int64, ns string, item *string) bool {
			executed.Add(1)
			if index == 0 {
				// First worker cancels the context, simulating fail-fast
				cancel()
				// Give a moment for cancellation to propagate
				time.Sleep(10 * time.Millisecond)
			}
			return index != 0 // first worker "fails"
		}

		// Parallelism of 1 ensures sequential execution so cancellation
		// prevents subsequent items from starting
		failed := ExecuteParallel(ctx, run, items, namespaces, 1)

		// Only the first worker should have executed
		assert.Equal(t, int64(1), executed.Load())
		// Failed count should only reflect actually-run workers that failed,
		// not skipped ones
		assert.Equal(t, int64(1), failed)
	})

	t.Run("counts failures correctly", func(t *testing.T) {
		items := []int{0, 1, 2, 3}
		namespaces := []string{"ns1", "ns1", "ns1", "ns1"}

		run := func(index int64, ns string, item *int) bool {
			return *item%2 == 0 // items 0 and 2 pass, 1 and 3 fail
		}

		failed := ExecuteParallel(context.Background(), run, items, namespaces, 4)
		assert.Equal(t, int64(2), failed)
	})

	t.Run("respects parallelism limit", func(t *testing.T) {
		items := []string{"a", "b", "c", "d"}
		namespaces := []string{"ns1", "ns1", "ns1", "ns1"}
		var concurrent atomic.Int64
		var maxConcurrent atomic.Int64

		run := func(index int64, ns string, item *string) bool {
			cur := concurrent.Add(1)
			// Track max concurrency
			for {
				max := maxConcurrent.Load()
				if cur <= max || maxConcurrent.CompareAndSwap(max, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			concurrent.Add(-1)
			return true
		}

		failed := ExecuteParallel(context.Background(), run, items, namespaces, 2)
		assert.Equal(t, int64(0), failed)
		assert.LessOrEqual(t, maxConcurrent.Load(), int64(2))
	})

	t.Run("already cancelled context skips all items", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		namespaces := []string{"ns1", "ns1", "ns1"}
		var executed atomic.Int64

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before execution

		run := func(index int64, ns string, item *string) bool {
			executed.Add(1)
			return true
		}

		failed := ExecuteParallel(ctx, run, items, namespaces, 2)
		assert.Equal(t, int64(0), executed.Load())
		assert.Equal(t, int64(0), failed) // skipped items don't count as failures
	})
}
