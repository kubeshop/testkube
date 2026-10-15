package channels

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWatcherSendWithContextDeliversToReader(t *testing.T) {
	w := NewWatcher[int]()
	received := make(chan int, 1)
	go func() { received <- <-w.Channel() }()

	sent := w.SendWithContext(context.Background(), 42)

	assert.True(t, sent)
	assert.Equal(t, 42, <-received)
}

func TestWatcherSendWithContextGivesUpWhenReaderIsGone(t *testing.T) {
	w := NewWatcher[int]()
	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan bool, 1)
	go func() { result <- w.SendWithContext(ctx, 42) }()

	select {
	case <-result:
		t.Fatal("send without a reader should block until cancelled")
	case <-time.After(20 * time.Millisecond):
	}
	cancel()

	select {
	case sent := <-result:
		assert.False(t, sent)
	case <-time.After(5 * time.Second):
		t.Fatal("send did not give up after cancellation")
	}
}
