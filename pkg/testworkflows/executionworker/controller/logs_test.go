package controller

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
)

const (
	testIdleTimeoutShort   = 50 * time.Millisecond
	testIdleTimeoutDefault = 500 * time.Millisecond
	testDeadlineShort      = 500 * time.Millisecond
	testDeadlineMedium     = 2 * time.Second
	testDeadlineLong       = 5 * time.Second
	testBufferSizeSmall    = 1
	testBufferSizeLarge    = 5
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

func TestWatchContainerLogsIdleTimeoutCancelsWhenDoneWithoutError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return &blockingReader{ctx: ctx}, nil
		},
		nil,
		"default",
		"pod",
		"container",
		testBufferSizeSmall,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutShort,
	)

	deadline := time.NewTimer(testDeadlineShort)
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
			t.Fatal("timed out waiting for idle timeout to close the channel")
		}
	}
}

func TestWatchContainerLogsIdleTimeoutCancelsWhileOpeningDoneStream(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	openerEntered := make(chan struct{})
	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			select {
			case <-openerEntered:
			default:
				close(openerEntered)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
		nil,
		"default",
		"pod",
		"container",
		testBufferSizeSmall,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutShort,
	)

	select {
	case <-openerEntered:
	case <-time.After(testDeadlineShort):
		t.Fatal("timed out waiting for log stream opener to start")
	}

	deadline := time.NewTimer(testDeadlineShort)
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
			t.Fatal("timed out waiting for idle timeout to cancel stream opener")
		}
	}
}

func TestWatchContainerLogsTerminalIdleReopensAndDrains(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var calls int32
	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, since *time.Time) (io.Reader, error) {
			call := atomic.AddInt32(&calls, 1)
			if call == 1 {
				assert.Nil(t, since)
				return &blockingReader{ctx: ctx}, nil
			}
			assert.NotNil(t, since)
			line := fmt.Sprintf("%s drained\n", since.Add(time.Second).UTC().Format(time.RFC3339Nano))
			return bytes.NewBufferString(line), nil
		},
		nil,
		"default",
		"pod",
		"container",
		testBufferSizeLarge,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutShort,
	)

	deadline := time.NewTimer(testDeadlineMedium)
	defer deadline.Stop()

	var gotDrainedLog bool
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				assert.True(t, gotDrainedLog, "expected drain log after terminal idle reopen")
				assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2))
				return
			}
			if msg.Error != nil {
				t.Fatalf("unexpected error from logs channel: %v", msg.Error)
			}
			if bytes.Contains(msg.Value.Log, []byte("drained")) {
				gotDrainedLog = true
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for terminal idle stream to reopen and drain")
		}
	}
}

func TestWatchContainerLogsDoesNotCancelQuietRunningStream(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ch := watchContainerLogsWithStream(
		ctx,
		func(ctx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return &blockingReader{ctx: ctx}, nil
		},
		nil,
		"default",
		"pod",
		"container",
		testBufferSizeSmall,
		func() bool { return false },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutShort,
	)

	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatal("quiet running stream closed before parent context was canceled")
		}
		if msg.Error != nil {
			t.Fatalf("unexpected error from quiet running stream: %v", msg.Error)
		}
	case <-time.After(3 * testIdleTimeoutShort):
	}

	cancel()

	select {
	case <-ch:
	case <-time.After(testDeadlineShort):
		t.Fatal("timed out waiting for stream to close after parent context cancellation")
	}
}

func TestWatchContainerLogsReopensOnEOF(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

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
		testBufferSizeLarge,
		func() bool { return false },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutDefault,
	)

	deadline := time.NewTimer(testDeadlineMedium)
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

func TestWatchContainerLogsProxyErrorPropagates(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ch := watchContainerLogsWithStream(
		ctx,
		func(_ context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return nil, fmt.Errorf("proxy error from 127.0.0.1:9345, code 502: Bad Gateway")
		},
		nil,
		"default",
		"pod",
		"container",
		testBufferSizeLarge,
		func() bool { return false },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutDefault,
	)

	deadline := time.NewTimer(testDeadlineLong)
	defer deadline.Stop()

	var gotErr bool
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				assert.True(t, gotErr, "expected proxy error before channel close")
				return
			}
			if msg.Error != nil {
				gotErr = true
				assert.Contains(t, msg.Error.Error(), "proxy error")
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for proxy error to propagate")
		}
	}
}

func TestWatchContainerLogsDoneWithNoLogsCloses(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ch := watchContainerLogsWithStream(
		ctx,
		func(_ context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return bytes.NewBuffer(nil), nil
		},
		nil,
		"default",
		"pod",
		"container",
		testBufferSizeSmall,
		func() bool { return true },
		func(*instructions.Instruction) bool { return false },
		testIdleTimeoutDefault,
	)

	deadline := time.NewTimer(testDeadlineShort)
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
