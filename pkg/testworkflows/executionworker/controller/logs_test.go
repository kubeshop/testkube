package controller

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
)

func Test_ReadTimestamp_UTC_Initial(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T12:41:49.037275300Z "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + message))
	err := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

func Test_ReadTimestamp_NonUTC_Initial(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T15:41:49.037275300+03:00 "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + message))
	err := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

func Test_ReadTimestamp_UTC_Recurring(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T12:41:49.037275300Z "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + prefix + message))
	err1 := reader.Read(buf)
	err2 := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

func Test_ReadTimestamp_NonUTC_Recurring(t *testing.T) {
	reader := newTimestampReader()
	prefix := "2024-06-07T15:41:49.037275300+03:00 "
	message := "some-message"
	buf := bufio.NewReader(bytes.NewBufferString(prefix + prefix + message))
	err1 := reader.Read(buf)
	err2 := reader.Read(buf)
	rest, _ := io.ReadAll(buf)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, []byte(prefix), reader.Prefix())
	assert.Equal(t, []byte(message), rest)
	assert.Equal(t, time.Date(2024, 6, 7, 12, 41, 49, 37275300, time.UTC), reader.ts)
}

type blockingReader struct {
	ctx context.Context
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

type lateReadAfterTimeoutReader struct {
	ctx      context.Context
	release  <-chan struct{}
	mu       sync.Mutex
	stage    int
	released bool
	first    *bytes.Reader
	second   *bytes.Reader
}

func newLateReadAfterTimeoutReader(ctx context.Context, release <-chan struct{}) *lateReadAfterTimeoutReader {
	return &lateReadAfterTimeoutReader{
		ctx:     ctx,
		release: release,
		first:   bytes.NewReader([]byte("2024-06-07T12:41:49.037275300Z first\n")),
		second:  bytes.NewReader([]byte("2024-06-07T12:41:50.037275300Z second\n")),
	}
}

func (r *lateReadAfterTimeoutReader) Read(p []byte) (int, error) {
	for {
		r.mu.Lock()
		switch r.stage {
		case 0:
			n, err := r.first.Read(p)
			if err == io.EOF {
				r.stage = 1
				r.mu.Unlock()
				if n > 0 {
					return n, nil
				}
				continue
			}
			r.mu.Unlock()
			return n, err
		case 1:
			if !r.released {
				r.mu.Unlock()
				<-r.release
				r.mu.Lock()
				r.released = true
			}
			r.stage = 2
			r.mu.Unlock()
			continue
		case 2:
			n, err := r.second.Read(p)
			if err == io.EOF {
				r.stage = 3
				r.mu.Unlock()
				if n > 0 {
					return n, nil
				}
				continue
			}
			r.mu.Unlock()
			return n, err
		default:
			r.mu.Unlock()
			<-r.ctx.Done()
			return 0, r.ctx.Err()
		}
	}
}

func TestWatchContainerLogsIdleTimeoutCancelsWhenDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	idleTimeout := 50 * time.Millisecond
	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return &blockingReader{ctx: ctx}, nil
		},
		nil,
		"default",
		"pod",
		"container",
		1,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		idleTimeout,
	)

	deadline := time.NewTimer(500 * time.Millisecond)
	defer deadline.Stop()

	var gotErr bool
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				assert.True(t, gotErr, "expected idle timeout error before channel close")
				return
			}
			if msg.Error != nil {
				gotErr = true
				assert.Contains(t, msg.Error.Error(), "idle timeout")
				assert.True(t, errors.Is(msg.Error, ErrLogStreamIdleTimeout))
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for idle timeout to close the channel")
		}
	}
}

func TestWatchContainerLogsIdleTimeoutRetriesBeforeClose(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int32
	idleTimeout := 30 * time.Millisecond
	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			atomic.AddInt32(&calls, 1)
			return &blockingReader{ctx: ctx}, nil
		},
		nil,
		"default",
		"pod",
		"container",
		1,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		idleTimeout,
	)

	deadline := time.NewTimer(1500 * time.Millisecond)
	defer deadline.Stop()

	idleErrors := 0
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				assert.Equal(t, int32(LogStreamIdleMaxRetries+1), atomic.LoadInt32(&calls))
				assert.GreaterOrEqual(t, idleErrors, 1)
				return
			}
			if msg.Error != nil && errors.Is(msg.Error, ErrLogStreamIdleTimeout) {
				idleErrors++
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for retries, calls=%d idleErrors=%d", atomic.LoadInt32(&calls), idleErrors)
		}
	}
}

func TestWatchContainerLogsIdleTimeoutRetriesAfterLateRead(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int32
	var releaseOnce sync.Once
	release := make(chan struct{})

	idleTimeout := 30 * time.Millisecond
	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			call := atomic.AddInt32(&calls, 1)
			if call == 1 {
				return newLateReadAfterTimeoutReader(ctx, release), nil
			}
			return bytes.NewBuffer(nil), nil
		},
		nil,
		"default",
		"pod",
		"container",
		1,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		idleTimeout,
	)

	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()

	sawIdleError := false
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				assert.True(t, sawIdleError, "expected idle timeout error before channel close")
				assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2), "expected stream reopen after timeout-triggered cancellation")
				return
			}
			if msg.Error != nil && errors.Is(msg.Error, ErrLogStreamIdleTimeout) {
				sawIdleError = true
				releaseOnce.Do(func() {
					close(release)
				})
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for retry after late read, calls=%d sawIdleError=%t", atomic.LoadInt32(&calls), sawIdleError)
		}
	}
}

func TestWatchContainerLogsReopensOnEOF(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int32
	line := "2024-06-07T12:41:49.037275300Z hello\n"
	ch := watchContainerLogsWithStream(
		ctx,
		func(_ context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			call := atomic.AddInt32(&calls, 1)
			if call == 1 {
				return bytes.NewBufferString(line), nil
			}
			return bytes.NewBuffer(nil), nil
		},
		nil,
		"default",
		"pod",
		"container",
		5,
		func() bool { return false },
		func(*instructions.Instruction) bool { return false },
		500*time.Millisecond,
	)

	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()

	var gotLog bool
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				assert.True(t, gotLog, "expected at least one log message before channel close")
				assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2))
				return
			}
			if msg.Error != nil {
				t.Fatalf("unexpected error from logs channel: %v", msg.Error)
			}
			if bytes.Contains(msg.Value.Log, []byte("hello")) {
				gotLog = true
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for log stream to close")
		}
	}
}

func TestWatchContainerLogsDoneWithNoLogsCloses(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := watchContainerLogsWithStream(
		ctx,
		func(_ context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return bytes.NewBuffer(nil), nil
		},
		nil,
		"default",
		"pod",
		"container",
		1,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		500*time.Millisecond,
	)

	deadline := time.NewTimer(500 * time.Millisecond)
	defer deadline.Stop()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Error != nil {
				t.Fatalf("unexpected error from logs channel: %v", msg.Error)
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for log stream to close")
		}
	}
}
