package bus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// makeTestBus starts an embedded NATS server and returns a NATSBus whose
// ConnectionConfig points at that server (so reconnect() can create a new
// connection without an external NATS installation).  It also returns the
// underlying *nats.Conn so tests can manipulate the connection lifecycle.
// The server is shut down by t.Cleanup.
func makeTestBus(t *testing.T) (*NATSBus, *nats.Conn) {
	t.Helper()

	ns, nc := TestServerWithConnection(t)
	t.Cleanup(func() { ns.Shutdown() })

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER) //nolint:staticcheck
	require.NoError(t, err)

	cfg := ConnectionConfig{NatsURI: ns.ClientURL()}
	b := NewNATSBus(ec, cfg)
	return b, nc
}

// waitDone blocks until wg.Wait() completes or the timeout elapses, failing
// the test with msg if it times out.
func waitDone(t *testing.T, wg *sync.WaitGroup, timeout time.Duration, msg string) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal(msg)
	}
}

// TestNATSBus_Reconnect_ReRegistersSubscriptions verifies that after the
// underlying connection is permanently closed, calling reconnect() swaps in a
// new connection AND re-registers all previously stored subscriptions so they
// continue receiving events.
func TestNATSBus_Reconnect_ReRegistersSubscriptions(t *testing.T) {
	b, nc := makeTestBus(t)

	var count int32
	var wg sync.WaitGroup
	wg.Add(1)

	require.NoError(t, b.SubscribeTopic("recon-topic", "recon-queue", func(evt testkube.Event) error {
		atomic.AddInt32(&count, 1)
		wg.Done()
		return nil
	}))

	// Baseline: verify the subscription works on the initial connection.
	event := testkube.NewEventStartTestWorkflow(testkube.NewQueuedExecution(), "")
	event.Id = "pre-reconnect"
	require.NoError(t, b.PublishTopic("recon-topic", event))
	waitDone(t, &wg, 2*time.Second, "timed out waiting for pre-reconnect event")
	require.Equal(t, int32(1), atomic.LoadInt32(&count))

	// Suppress the automatic background reconnect triggered by the ClosedHandler
	// so the test has explicit, race-free control over when reconnect() runs.
	nc.SetClosedHandler(func(_ *nats.Conn) {})

	// Force-close the underlying connection so IsClosed() == true.
	nc.Close()
	require.Eventually(t, func() bool { return b.getNC().Conn.IsClosed() },
		time.Second, 10*time.Millisecond, "underlying connection should be closed")

	// Reconnect; the embedded server is still running so a new connection is
	// established immediately.
	require.NoError(t, b.reconnect())
	assert.False(t, b.getNC().Conn.IsClosed(), "new connection should be open after reconnect")

	// Publish again — the subscription must have been re-registered on the new
	// connection, so the handler should fire.
	wg.Add(1)
	event.Id = "post-reconnect"
	require.NoError(t, b.PublishTopic("recon-topic", event))
	waitDone(t, &wg, 2*time.Second, "timed out waiting for post-reconnect event; subscription may not have been re-registered")
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

// TestNATSBus_PublishTopic_ReconnectsOnClosedConnection verifies that
// PublishTopic detects ErrConnectionClosed, triggers a reconnect, and
// successfully delivers the event on the new connection.
func TestNATSBus_PublishTopic_ReconnectsOnClosedConnection(t *testing.T) {
	b, nc := makeTestBus(t)

	var count int32
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe before forcing the close so that the subscription is in the map
	// and gets re-registered during the reconnect triggered by PublishTopic.
	require.NoError(t, b.SubscribeTopic("pub-recon-topic", "pub-recon-queue", func(evt testkube.Event) error {
		atomic.AddInt32(&count, 1)
		wg.Done()
		return nil
	}))

	// Suppress the automatic ClosedHandler reconnect.
	nc.SetClosedHandler(func(_ *nats.Conn) {})
	nc.Close()
	require.Eventually(t, func() bool { return b.getNC().Conn.IsClosed() },
		time.Second, 10*time.Millisecond, "underlying connection should be closed")

	event := testkube.NewEventStartTestWorkflow(testkube.NewQueuedExecution(), "")
	event.Id = "pub-recon-test"

	// PublishTopic should detect ErrConnectionClosed, reconnect, re-register
	// the subscription, then publish — all transparently.
	require.NoError(t, b.PublishTopic("pub-recon-topic", event),
		"PublishTopic should reconnect and succeed")

	waitDone(t, &wg, 2*time.Second, "timed out waiting for event after reconnect-triggered publish")
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))
}

// TestNATSBus_SubscribeTopic_EmptyQueueUsesPlainSubscribe verifies that
// SubscribeTopic falls back to a plain (non-queue) Subscribe when the
// sanitized queue name is empty, and that the resulting subscription receives
// events correctly.
func TestNATSBus_SubscribeTopic_EmptyQueueUsesPlainSubscribe(t *testing.T) {
	b, _ := makeTestBus(t)

	var count int32
	var wg sync.WaitGroup
	wg.Add(1)

	// Passing an empty queueName sanitizes to "" → should use plain Subscribe.
	require.NoError(t, b.SubscribeTopic("eq-topic", "", func(evt testkube.Event) error {
		atomic.AddInt32(&count, 1)
		wg.Done()
		return nil
	}))

	event := testkube.NewEventStartTestWorkflow(testkube.NewQueuedExecution(), "")
	event.Id = "empty-queue-test"
	require.NoError(t, b.PublishTopic("eq-topic", event))

	waitDone(t, &wg, 2*time.Second, "timed out waiting for event on empty-queue subscription")
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))

	// Confirm the subscription entry is stored with an empty queue field.
	var entry *subscriptionEntry
	b.subscriptions.Range(func(_, value any) bool {
		e := value.(*subscriptionEntry)
		if e.topic == "eq-topic" {
			entry = e
			return false
		}
		return true
	})
	require.NotNil(t, entry, "subscription entry not found in map")
	assert.Empty(t, entry.queue, "entry.queue should be empty for plain Subscribe")
}

// TestNATSBus_Close_DoesNotReconnect verifies that calling Close() sets the
// shutdown flag so the ClosedHandler does not spawn a background reconnect,
// leaving the bus permanently closed as the caller intended.
func TestNATSBus_Close_DoesNotReconnect(t *testing.T) {
	b, _ := makeTestBus(t)

	require.False(t, b.shutdown.Load(), "shutdown flag should be false before Close()")
	require.NoError(t, b.Close())

	assert.True(t, b.shutdown.Load(), "shutdown flag should be true after Close()")
	assert.True(t, b.getNC().Conn.IsClosed(), "underlying connection should be closed")

	// Give the ClosedHandler goroutine time to fire (if it incorrectly did).
	time.Sleep(50 * time.Millisecond)

	// The connection should remain closed — no reconnect was triggered.
	assert.True(t, b.getNC().Conn.IsClosed(), "bus should remain closed after Close(); unexpected reconnect detected")
}
