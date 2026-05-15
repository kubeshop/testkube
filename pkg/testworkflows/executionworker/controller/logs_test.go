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

func TestBuildPodLogOptionsDefaultsToVerifiedKubeletBackend(t *testing.T) {
	t.Parallel()

	opts := buildPodLogOptions("container", func() bool { return false }, nil, ContainerLogOptions{})

	assert.Equal(t, "container", opts.Container)
	assert.True(t, opts.Follow)
	assert.True(t, opts.Timestamps)
	assert.Nil(t, opts.SinceTime)
	assert.False(t, opts.InsecureSkipTLSVerifyBackend)
}

func TestBuildPodLogOptionsCanSkipKubeletBackendTLSVerification(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 4, 27, 12, 30, 0, 0, time.UTC)
	opts := buildPodLogOptions("container", func() bool { return true }, &since, ContainerLogOptions{
		InsecureSkipTLSVerifyBackend: true,
	})

	assert.Equal(t, "container", opts.Container)
	assert.False(t, opts.Follow)
	assert.True(t, opts.Timestamps)
	assert.NotNil(t, opts.SinceTime)
	assert.True(t, since.Equal(opts.SinceTime.Time))
	assert.True(t, opts.InsecureSkipTLSVerifyBackend)
}

func TestTLSRetryConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg := TLSRetryConfig{}
	assert.Equal(t, LogTLSRetryMaxAttempts, cfg.maxAttempts())
	assert.Equal(t, LogTLSRetryInitialDelay, cfg.initialDelay())
	assert.Equal(t, LogTLSRetryMaxDelay, cfg.maxDelay())
}

func TestTLSRetryConfigCustom(t *testing.T) {
	t.Parallel()

	cfg := TLSRetryConfig{
		MaxAttempts:  50,
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
	}
	assert.Equal(t, 50, cfg.maxAttempts())
	assert.Equal(t, 1*time.Second, cfg.initialDelay())
	assert.Equal(t, 60*time.Second, cfg.maxDelay())
}

func TestTLSRetryConfigBackoffDelay(t *testing.T) {
	t.Parallel()

	cfg := TLSRetryConfig{
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
	}

	// Verify exponential progression
	assert.Equal(t, 500*time.Millisecond, cfg.backoffDelay(1))
	assert.Equal(t, 1*time.Second, cfg.backoffDelay(2))
	assert.Equal(t, 2*time.Second, cfg.backoffDelay(3))
	assert.Equal(t, 4*time.Second, cfg.backoffDelay(4))
	assert.Equal(t, 8*time.Second, cfg.backoffDelay(5))
	assert.Equal(t, 16*time.Second, cfg.backoffDelay(6))
	// Capped at maxDelay
	assert.Equal(t, 30*time.Second, cfg.backoffDelay(7))
	assert.Equal(t, 30*time.Second, cfg.backoffDelay(10))
}

func TestTLSRetryConfigBackoffDelayOverflowProtection(t *testing.T) {
	t.Parallel()

	// Large initial delay that would overflow with many retries
	cfg := TLSRetryConfig{
		InitialDelay: 60 * time.Second,
		MaxDelay:     5 * time.Minute,
		MaxAttempts:  30,
	}

	// High shift values should clamp to maxDelay, not overflow to negative
	delay := cfg.backoffDelay(29)
	assert.True(t, delay > 0, "delay should be positive, got %v", delay)
	assert.Equal(t, 5*time.Minute, delay)

	// Extremely high shift (>= 63) should be safe
	delay = cfg.backoffDelay(64)
	assert.Equal(t, 5*time.Minute, delay)
}

func TestTLSRetryConfigBackoffDelayInitialExceedsMax(t *testing.T) {
	t.Parallel()

	// Misconfiguration: InitialDelay > MaxDelay should still be clamped
	cfg := TLSRetryConfig{
		InitialDelay: 5 * time.Minute,
		MaxDelay:     30 * time.Second,
		MaxAttempts:  10,
	}

	// First retry should be clamped to MaxDelay, not return InitialDelay
	assert.Equal(t, 30*time.Second, cfg.backoffDelay(1))
	assert.Equal(t, 30*time.Second, cfg.backoffDelay(2))
	assert.Equal(t, 30*time.Second, cfg.backoffDelay(5))
}

func TestGetContainerLogsStreamWithStreamerTLSRetry(t *testing.T) {
	t.Parallel()

	// Use small delays for fast testing
	cfg := TLSRetryConfig{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     80 * time.Millisecond,
	}
	opts := ContainerLogOptions{TLSRetry: cfg}

	var attempts int
	var timestamps []time.Time
	tlsErr := fmt.Errorf("Get \"https://10.0.0.1:10250/containerLogs/ns/pod/c\": remote error: tls: internal error")

	streamer := func(ctx context.Context) (io.ReadCloser, error) {
		attempts++
		timestamps = append(timestamps, time.Now())
		return nil, tlsErr
	}

	ctx := context.Background()
	_, err := getContainerLogsStreamWithStreamer(ctx, streamer, "pod", "container", func() bool { return false }, opts)

	// Should have exhausted all retries (1 initial + maxAttempts retries)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tls: internal error")
	assert.Equal(t, cfg.MaxAttempts+1, attempts, "expected 1 initial attempt + MaxAttempts retries")

	// Verify exponential backoff delays between attempts
	// Expected: 10ms, 20ms, 40ms, 80ms (capped)
	for i := 1; i < len(timestamps); i++ {
		elapsed := timestamps[i].Sub(timestamps[i-1])
		expectedDelay := cfg.backoffDelay(i)
		// Allow 50% tolerance for timing jitter in CI
		minDelay := expectedDelay * 50 / 100
		assert.True(t, elapsed >= minDelay,
			"retry %d: elapsed %v should be >= %v (expected delay %v)", i, elapsed, minDelay, expectedDelay)
	}
}

func TestGetContainerLogsStreamWithStreamerTLSSucceedsAfterRetries(t *testing.T) {
	t.Parallel()

	cfg := TLSRetryConfig{
		MaxAttempts:  5,
		InitialDelay: 5 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
	}
	opts := ContainerLogOptions{TLSRetry: cfg}

	var attempts int
	tlsErr := fmt.Errorf("remote error: tls: internal error")

	streamer := func(ctx context.Context) (io.ReadCloser, error) {
		attempts++
		if attempts <= 3 {
			return nil, tlsErr
		}
		return io.NopCloser(bytes.NewBufferString("success")), nil
	}

	ctx := context.Background()
	reader, err := getContainerLogsStreamWithStreamer(ctx, streamer, "pod", "container", func() bool { return false }, opts)

	assert.NoError(t, err)
	assert.Equal(t, 4, attempts, "expected 3 failed attempts + 1 successful")
	content, _ := io.ReadAll(reader)
	assert.Equal(t, "success", string(content))
}

func TestGetContainerLogsStreamWithStreamerContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := TLSRetryConfig{
		MaxAttempts:  10,
		InitialDelay: 1 * time.Second, // Long delay to ensure context cancels first
		MaxDelay:     5 * time.Second,
	}
	opts := ContainerLogOptions{TLSRetry: cfg}

	var attempts int
	tlsErr := fmt.Errorf("remote error: tls: internal error")

	streamer := func(ctx context.Context) (io.ReadCloser, error) {
		attempts++
		return nil, tlsErr
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay to interrupt the backoff wait
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := getContainerLogsStreamWithStreamer(ctx, streamer, "pod", "container", func() bool { return false }, opts)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	// Should have been interrupted before exhausting all retries
	assert.Less(t, attempts, cfg.MaxAttempts+1)
}

func TestTLSRetryErrorPropagation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Verify that TLS errors from the stream opener propagate as errors through the channel.
	// The retry/backoff logic inside getContainerLogsStream is tested via
	// TestTLSRetryConfigBackoffDelay and TestTLSRetryConfigBackoffDelayOverflowProtection.
	ch := watchContainerLogsWithStream(
		ctx,
		func(streamCtx context.Context, _ kubernetes.Interface, _, _, _ string, _ func() bool, _ *time.Time) (io.Reader, error) {
			return nil, fmt.Errorf("Get \"https://10.0.0.1:10250/containerLogs/ns/pod/c\": remote error: tls: internal error")
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
				assert.True(t, gotErr, "expected TLS error before channel close")
				return
			}
			if msg.Error != nil {
				gotErr = true
				assert.Contains(t, msg.Error.Error(), "tls: internal error")
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for TLS error to propagate")
		}
	}
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
