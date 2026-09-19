package grpc

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cloudflare/backoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
)

// stubUpdatesClient answers GetExecutionUpdates from a script.
type stubUpdatesClient struct {
	executionv1.TestWorkflowExecutionServiceClient

	mu    sync.Mutex
	calls int
	// failUntil is how many leading calls fail; the rest succeed.
	failUntil int
}

func (s *stubUpdatesClient) GetExecutionUpdates(context.Context, *executionv1.GetExecutionUpdatesRequest, ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.failUntil < 0 || s.calls <= s.failUntil {
		return nil, errors.New("control plane unavailable")
	}
	return &executionv1.GetExecutionUpdatesResponse{}, nil
}

func (s *stubUpdatesClient) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// sleepRecorder captures the backoff ladder. Full jitter makes any assertion on
// real elapsed time flaky, so the durations are recorded instead of slept.
type sleepRecorder struct {
	mu        sync.Mutex
	durations []time.Duration
}

func (r *sleepRecorder) sleep(d time.Duration) <-chan time.Time {
	r.mu.Lock()
	r.durations = append(r.durations, d)
	r.mu.Unlock()

	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func (r *sleepRecorder) recorded() []time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Duration(nil), r.durations...)
}

func newPollClient(t *testing.T, stub executionv1.TestWorkflowExecutionServiceClient, recorder *sleepRecorder, maxBackoff time.Duration) *Client {
	t.Helper()
	return &Client{
		client:           stub,
		logger:           zap.NewNop().Sugar(),
		pollInterval:     time.Millisecond,
		callTimeout:      time.Second,
		maxPollBackoff:   maxBackoff,
		startConcurrency: defaultStartConcurrency,
		sleep:            recorder.sleep,
	}
}

// runUntil runs the poll loop until it has recorded at least n backoffs.
func runUntil(t *testing.T, c *Client, recorder *sleepRecorder, n int) []time.Duration {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = c.Start(ctx, "env-1")
	}()

	deadline := time.After(10 * time.Second)
	for {
		if len(recorder.recorded()) >= n {
			cancel()
			<-done
			return recorder.recorded()
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("only %d backoffs recorded, wanted %d", len(recorder.recorded()), n)
			return nil
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// Duration() advances the attempt counter, so it must be called exactly once per
// failure.
//
// It used to be called twice - once to build the log field and once for the
// sleep - so every failure advanced the exponent by two and the number in the
// log was not the number slept. That is why nine consecutive failures were
// enough to reach a multi-hour retry interval and stop the runner entirely.
//
// Jitter means each sample is uniform in [0, 2^n * interval), so the ceiling is
// what can be asserted: doubling per failure, not quadrupling.
func TestStart_BackoffAdvancesOncePerFailure(t *testing.T) {
	stub := &stubUpdatesClient{failUntil: -1}
	recorder := &sleepRecorder{}
	// A max far above anything the first few attempts can reach, so the ladder is
	// not clamped while we measure its growth.
	c := newPollClient(t, stub, recorder, time.Hour)

	durations := runUntil(t, c, recorder, 6)

	for i := 0; i < 6; i++ {
		// Ceiling for attempt i when the exponent advances once per failure.
		singleStep := time.Duration(1<<uint(i)) * c.pollInterval
		// Ceiling if the exponent advanced twice per failure, as it used to.
		doubleStep := time.Duration(1<<uint(2*i)) * c.pollInterval

		assert.LessOrEqualf(t, durations[i], singleStep,
			"backoff %d was %s, above the %s ceiling for one advance per failure (two advances would allow %s)",
			i, durations[i], singleStep, doubleStep)
	}
}

// A transient failure must not become a permanent one. The library default cap
// is six hours; with it, a runner that hit the cap stopped consuming work for
// the rest of the day.
func TestStart_BackoffIsCappedAtMax(t *testing.T) {
	stub := &stubUpdatesClient{failUntil: -1}
	recorder := &sleepRecorder{}
	maxBackoff := 30 * time.Millisecond
	c := newPollClient(t, stub, recorder, maxBackoff)

	durations := runUntil(t, c, recorder, 50)

	for i, d := range durations {
		require.LessOrEqualf(t, d, maxBackoff,
			"backoff %d was %s, above the %s ceiling", i, d, maxBackoff)
	}
}

// A successful poll resets the ladder, so a runner that recovers does not keep
// paying for failures that are over.
func TestStart_ResetsBackoffAfterSuccess(t *testing.T) {
	stub := &stubUpdatesClient{failUntil: 5}
	recorder := &sleepRecorder{}
	c := newPollClient(t, stub, recorder, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = c.Start(ctx, "env-1")
	}()

	// Wait for the five failures, the success, and then one more failure... there
	// is none, so instead wait until a poll has succeeded.
	deadline := time.After(10 * time.Second)
	for c.LastSuccessfulPoll().Before(time.Now().Add(-time.Hour)) || stub.callCount() <= 5 {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("no successful poll observed")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	<-done

	durations := recorder.recorded()
	require.Len(t, durations, 5, "only the five failing polls back off")
	assert.WithinDuration(t, time.Now(), c.LastSuccessfulPoll(), 10*time.Second,
		"a successful poll updates the readiness clock")
}

// A shutdown during a backoff has to return promptly.
//
// The sleep used to be a bare <-time.After, so with the six hour cap a runner
// could hold the errgroup that waits on it for hours after the context was
// cancelled.
func TestStart_ReturnsPromptlyDuringBackoff(t *testing.T) {
	stub := &stubUpdatesClient{failUntil: -1}
	// A sleep that never fires, standing in for a long backoff.
	blocking := func(time.Duration) <-chan time.Time { return make(chan time.Time) }

	c := &Client{
		client:           stub,
		logger:           zap.NewNop().Sugar(),
		pollInterval:     time.Millisecond,
		callTimeout:      time.Second,
		maxPollBackoff:   10 * time.Second,
		startConcurrency: defaultStartConcurrency,
		sleep:            blocking,
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Start(ctx, "env-1") }()

	// Let the first poll fail and enter the backoff.
	for stub.callCount() == 0 {
		time.Sleep(time.Millisecond)
	}
	cancel()

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return while blocked in a backoff")
	}
}

// The readiness clock only advances on a successful poll, so a stalled loop is
// visible rather than being masked by the process still being up.
func TestClient_LastSuccessfulPollAdvancesOnSuccessOnly(t *testing.T) {
	stub := &stubUpdatesClient{failUntil: -1}
	recorder := &sleepRecorder{}
	c := newPollClient(t, stub, recorder, time.Millisecond)

	seeded := time.Now().Add(-time.Hour)
	c.lastPollOK.Store(seeded.UnixNano())

	runUntil(t, c, recorder, 5)

	assert.Equal(t, seeded.UnixNano(), c.lastPollOK.Load(),
		"failing polls must not look like progress")
}

func TestClient_HealthyReportsStaleAfterThreshold(t *testing.T) {
	c := &Client{}
	last := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	c.lastPollOK.Store(last.UnixNano())

	assert.NoError(t, c.Healthy(last.Add(30*time.Second), 2*time.Minute))
	assert.NoError(t, c.Healthy(last.Add(2*time.Minute), 2*time.Minute))

	err := c.Healthy(last.Add(3*time.Minute), 2*time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no successful execution update poll")
}

// BenchmarkBackoffLadder puts a number on the headline defect: how long a runner
// is asleep across a run of consecutive failures.
//
// "once" is the fixed ladder: Duration() called once per failure against the
// capped max. "twice" reproduces the bug - Duration() called a second time to
// build the log field - against the library default of six hours. The reported
// metric is what matters, not ns/op.
func BenchmarkBackoffLadder(b *testing.B) {
	const failures = 20

	for _, tc := range []struct {
		name       string
		max        time.Duration
		perFailure int
	}{
		{"once-capped", defaultMaxPollBackoff, 1},
		{"twice-uncapped", backoff.DefaultMaxDuration, 2},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			var slept time.Duration
			for i := 0; i < b.N; i++ {
				bo := backoff.NewWithoutJitter(tc.max, time.Second)
				slept = 0
				for f := 0; f < failures; f++ {
					var d time.Duration
					for c := 0; c < tc.perFailure; c++ {
						d = bo.Duration()
					}
					slept += d
				}
			}
			b.StopTimer()
			b.ReportMetric(slept.Seconds(), "sleep-secs/20-failures")
		})
	}
}
